/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2026 CERN and copyright holders of ALICE O².
 * Author: Michal Tichak <michal.tichak@cern.ch>
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 * In applying this license CERN does not waive the privileges and
 * immunities granted to it by virtue of its status as an
 * Intergovernmental Organization or submit itself to any jurisdiction.
 */

package task

import (
	"context"
	"fmt"
	"maps"
	"strings"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	k8sapierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"

	"github.com/AliceO2Group/Control/common/gera"
	"github.com/AliceO2Group/Control/common/utils/uid"
	"github.com/AliceO2Group/Control/control-operator/api/v1alpha1"
	k8sclient "github.com/AliceO2Group/Control/control-operator/pkg/client"
	"github.com/AliceO2Group/Control/core/task/channel"
	"github.com/AliceO2Group/Control/core/task/sm"
	"github.com/AliceO2Group/Control/core/task/taskclass"
	"github.com/spf13/viper"
)

// TODO: revisit all the timeouts in ECS. they shouldn't be hardcoded.
const (
	k8sDeployTimeout     = 80 * time.Second
	k8sTransitionTimeout = 80 * time.Second
	k8sWatchRetryDelay   = 5 * time.Second
)

// k8sEnvRegistry maps ECS environment IDs to K8s Environment CRD names.
type k8sEnvRegistry struct {
	mu   sync.RWMutex
	envs map[uid.ID]string
}

func newK8sEnvRegistry() *k8sEnvRegistry {
	return &k8sEnvRegistry{envs: make(map[uid.ID]string)}
}

func (r *k8sEnvRegistry) set(envId uid.ID, crdName string) {
	r.mu.Lock()
	r.envs[envId] = crdName
	r.mu.Unlock()
}

func (r *k8sEnvRegistry) get(envId uid.ID) (string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	crdName, ok := r.envs[envId]
	return crdName, ok
}

func (r *k8sEnvRegistry) delete(envId uid.ID) {
	r.mu.Lock()
	delete(r.envs, envId)
	r.mu.Unlock()
}

// newK8sClientFromViper creates a K8s client from viper config.
// Returns nil, nil if kubeNamespace is not configured (K8s disabled).
func newK8sClientFromViper() (*k8sclient.Client, error) {
	namespace := viper.GetString("kubeNamespace")
	if namespace == "" {
		return nil, nil
	}
	kubeconfig := viper.GetString("kubeconfig")
	return k8sclient.New(kubeconfig, namespace)
}

type k8sTaskEntry struct {
	task *Task
	desc *Descriptor
}

// deployKubernetesTasks deploys all KUBERNETES-mode descriptors by creating one Environment CRD that
// references the pre-deployed TaskTemplate CRDs. It blocks until the operator reports status.state == "standby".
func (m *Manager) deployKubernetesTasks(ctx context.Context, envId uid.ID, descriptors Descriptors) (DeploymentMap, error) {
	log.WithField("partition", envId).
		WithField("count", len(descriptors)).
		Info("deploying K8s tasks")

	entries, err := m.createK8sTaskEntries(envId, descriptors)
	if err != nil {
		log.WithField("partition", envId).WithError(err).Error("failed to create K8s task entries")
		return nil, err
	}

	nodeToRefs, err := m.buildK8sNodeTaskRefs(entries)
	if err != nil {
		log.WithField("partition", envId).WithError(err).Error("failed to build K8s node task refs")
		return nil, err
	}

	for _, e := range entries {
		m.roster.append(e.task)
	}

	envCRDName, err := m.createK8sEnvironmentCRD(ctx, envId, nodeToRefs)
	if err != nil {
		log.WithField("partition", envId).WithError(err).Error("failed to create K8s Environment CRD")
		return nil, err
	}

	log.WithField("partition", envId).
		WithField("crd", envCRDName).
		Infof("K8s Environment CRD created, waiting up to %s for standby", k8sDeployTimeout)

	if err := m.waitForK8sEnvState(ctx, envCRDName, "standby", k8sDeployTimeout); err != nil {
		log.WithField("partition", envId).WithField("crd", envCRDName).WithError(err).Error("K8s Environment did not reach standby")
		return nil, fmt.Errorf("waiting for Environment %s standby: %w", envCRDName, err)
	}

	log.WithField("partition", envId).
		WithField("crd", envCRDName).
		Infof("K8s Environment reached standby, %d tasks deployed", len(entries))

	deployed := make(DeploymentMap, len(entries))
	for _, e := range entries {
		deployed[e.task] = e.desc
	}
	return deployed, nil
}

func (m *Manager) createK8sTaskEntries(envId uid.ID, descriptors Descriptors) ([]k8sTaskEntry, error) {
	entries := make([]k8sTaskEntry, 0, len(descriptors))
	for _, desc := range descriptors {
		taskClass, ok := m.classes.GetClass(desc.TaskClassName)
		if !ok || taskClass == nil {
			return nil, fmt.Errorf("task class not found: %s", desc.TaskClassName)
		}

		nodeName, err := nodeNameFromDescriptor(desc, taskClass)
		if err != nil {
			return nil, err
		}
		t := m.newTaskForKubernetes(desc, nodeName)

		wants, err := m.GetWantsForDescriptor(desc, envId)
		if err != nil {
			return nil, fmt.Errorf("getting wants for %s: %w", desc.TaskClassName, err)
		}

		for _, ch := range wants.InboundChannels {
			if ch.Addressing == channel.IPC {
				t.localBindMap[ch.Name] = channel.NewBoundIpcEndpoint(ch.Transport)
			} else {
				log.WithField("task", desc.TaskClassName).
					WithField("channel", ch.Name).
					Warn("TCP inbound channel in K8s task — skipping port allocation")
			}
			if len(ch.Global) != 0 {
				t.localBindMap["::"+ch.Global] = t.localBindMap[ch.Name]
			}
		}

		entries = append(entries, k8sTaskEntry{task: t, desc: desc})
	}
	return entries, nil
}

func (m *Manager) buildK8sNodeTaskRefs(entries []k8sTaskEntry) (map[string][]v1alpha1.TaskReference, error) {
	nodeToRefs := make(map[string][]v1alpha1.TaskReference)
	for _, e := range entries {
		t := e.task
		desc := e.desc

		taskClass, _ := m.classes.GetClass(desc.TaskClassName)

		if err := t.BuildTaskCommand(desc.TaskRole); err != nil {
			return nil, fmt.Errorf("building task command for %s: %w", desc.TaskClassName, err)
		}
		cmd := t.GetTaskCommandInfo()

		for varName, defaultValue := range map[string]string{
			"O2_ROLE":   t.hostname,
			"O2_SYSTEM": "FLP",
		} {
			varIsSet := false
			for _, e := range cmd.Env {
				key, _, _ := strings.Cut(e, "=")
				if key == varName {
					varIsSet = true
					break
				}
			}
			if !varIsSet {
				cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", varName, defaultValue))
			}
		}

		argsCLI := make([]string, 0, len(cmd.Arguments))
		for _, arg := range cmd.Arguments {
			if strings.TrimSpace(arg) != "" {
				argsCLI = append(argsCLI, arg)
			}
		}

		ref := v1alpha1.TaskReference{
			Name:    taskClass.Identifier.Name,
			TaskID:  t.taskId,
			ArgsCLI: argsCLI,
			Env:     cmdEnvToK8sEnvVars(cmd.Env),
		}
		nodeToRefs[t.hostname] = append(nodeToRefs[t.hostname], ref)
	}
	return nodeToRefs, nil
}

func (m *Manager) createK8sEnvironmentCRD(ctx context.Context, envId uid.ID, nodeToRefs map[string][]v1alpha1.TaskReference) (string, error) {
	envCRDName := strings.ToLower(envId.String())

	env := &v1alpha1.Environment{
		ObjectMeta: metav1.ObjectMeta{Name: envCRDName},
		TaskTemplates: v1alpha1.TemplateSpecification{
			Tasks: nodeToRefs,
		},
		Spec: v1alpha1.EnvironmentSpec{
			Tasks: make(map[string][]v1alpha1.TaskDefinition),
			State: "standby",
		},
	}
	if err := m.k8sClient.CreateEnvironment(ctx, env); err != nil {
		return "", fmt.Errorf("creating Environment CRD %s: %w", envCRDName, err)
	}
	m.k8sEnvs.set(envId, envCRDName)
	return envCRDName, nil
}

// equivalent of newTaskForMesosOffer in manager.go
func (m *Manager) newTaskForKubernetes(desc *Descriptor, nodeName string) *Task {
	newId := uid.New().String()
	t := &Task{
		name:         fmt.Sprintf("%s#%s", desc.TaskClassName, newId),
		parent:       desc.TaskRole,
		className:    desc.TaskClassName,
		hostname:     nodeName,
		agentId:      "kubernetes",
		offerId:      "kubernetes",
		taskId:       newId,
		executorId:   "kubernetes",
		properties:   gera.MakeMap[string, string]().Wrap(m.GetTaskClass(desc.TaskClassName).Properties),
		localBindMap: make(channel.BindMap),
		state:        sm.STANDBY,
		status:       ACTIVE,
	}
	t.GetTaskClass = func() *taskclass.Class {
		return m.GetTaskClass(t.className)
	}
	return t
}

func (m *Manager) waitForK8sEnvState(ctx context.Context, envCRDName, expected string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Check current state before watching in case it's already reached.
	env, err := m.k8sClient.GetEnvironment(ctx, envCRDName)
	if err != nil {
		return fmt.Errorf("getting Environment %s: %w", envCRDName, err)
	}
	if env.Status.State == expected {
		return nil
	}

	watcher, err := m.k8sClient.WatchEnvironmentsFromVersion(ctx, env.ResourceVersion)
	if err != nil {
		return fmt.Errorf("watching Environments: %w", err)
	}
	defer watcher.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for Environment %s to reach state %q: %w", envCRDName, expected, ctx.Err())
		case ev, ok := <-watcher.ResultChan():
			if !ok {
				return fmt.Errorf("watch channel closed waiting for Environment %s", envCRDName)
			}
			env, ok := ev.Object.(*v1alpha1.Environment)
			if !ok {
				continue
			}
			log.WithField("partition", envCRDName).
				WithField("crd", envCRDName).Debug("received some environment update")
			if env.Name == envCRDName {
				log.WithField("crd", envCRDName).
					WithField("state", env.Status.State).
					WithField("expected", expected).
					Debug("K8s Environment state update received")
				if env.Status.State == expected {
					return nil
				}
			}
		}
	}
}

// configureK8sTasks is necessary addition to transitionAndWaitK8sEnvState because we need to pass generated
// channels to individual tasks. These are generated during Configure transition
func (m *Manager) configureK8sTasks(ctx context.Context, envId uid.ID, tasks Tasks, bindMap channel.BindMap) error {
	for _, t := range tasks {
		propMap, err := t.BuildPropertyMap(bindMap)
		if err != nil {
			return fmt.Errorf("building property map for K8s task %s: %w", t.GetClassName(), err)
		}

		k8sTaskList, err := m.k8sClient.ListTasksByLabel(ctx, map[string]string{"taskID": t.taskId})
		if err != nil {
			return fmt.Errorf("error while listing K8s Task CRD for task %s: %w", t.taskId, err)
		}
		if len(k8sTaskList) == 0 {
			return fmt.Errorf("there were not tasks for taskId %s", t.taskId)
		} else if len(k8sTaskList) > 1 {
			return fmt.Errorf("there was more than one task for taskId %s", t.taskId)
		}

		k8sTask := k8sTaskList[0]
		if k8sTask.Spec.Arguments == nil {
			k8sTask.Spec.Arguments = make(map[string]string)
		}
		maps.Copy(k8sTask.Spec.Arguments, propMap)
		if err := m.k8sClient.UpdateTask(ctx, &k8sTask); err != nil {
			return fmt.Errorf("updating arguments for K8s Task CRD %s: %w", k8sTask.Name, err)
		}
	}
	return m.transitionAndWaitK8sEnvState(ctx, envId, "configured")
}

// transitionAndWaitK8sEnvState is the easy way to transition tasks. Posibility for future: make this async,
// so we don't have to wait during deployment
func (m *Manager) transitionAndWaitK8sEnvState(ctx context.Context, envId uid.ID, targetState string) error {
	crdName, ok := m.k8sEnvs.get(envId)
	if !ok {
		return fmt.Errorf("no K8s Environment CRD registered for env %s", envId)
	}
	env, err := m.k8sClient.GetEnvironment(ctx, crdName)
	if err != nil {
		return fmt.Errorf("error getting Environment CRD %s: %w", crdName, err)
	}
	env.Spec.State = targetState
	if err := m.k8sClient.UpdateEnvironment(ctx, env); err != nil {
		return fmt.Errorf("updating Environment CRD %s: %w", crdName, err)
	}
	return m.waitForK8sEnvState(ctx, crdName, targetState, k8sTransitionTimeout)
}

func (m *Manager) killK8sEnvironment(ctx context.Context, envId uid.ID) error {
	crdName, ok := m.k8sEnvs.get(envId)
	if !ok {
		return nil
	}
	if err := m.k8sClient.DeleteEnvironment(ctx, crdName); err != nil {
		return fmt.Errorf("deleting Environment CRD %s: %w", crdName, err)
	}
	m.k8sEnvs.delete(envId)
	return nil
}

func (m *Manager) killK8sTask(ctx context.Context, task *Task) error {
	err := m.k8sClient.DeleteTask(ctx, task.taskId)
	if k8sapierrors.IsNotFound(err) {
		return nil
	}
	return err
}

func (m *Manager) watchK8sTasks(ctx context.Context) {
	for ctx.Err() == nil {
		watcher, err := m.k8sClient.WatchTasks(ctx)
		if err != nil {
			log.WithError(err).Warnf("failed to start K8s task watcher, retrying after %v", k8sWatchRetryDelay)
			time.Sleep(k8sWatchRetryDelay)
			continue
		}
	messages:
		for {
			select {
			case <-ctx.Done():
				watcher.Stop()
				return
			case ev, ok := <-watcher.ResultChan():
				if !ok {
					break messages
				}
				m.process(ev)
			}
		}
	}
}

func (m *Manager) process(ev watch.Event) {
	k8sTask, ok := ev.Object.(*v1alpha1.Task)
	if !ok || k8sTask.Status.State == "" {
		return
	}
	if taskId, ok := k8sTask.Labels["taskID"]; ok && taskId != "" {
		m.updateTaskState(taskId, strings.ToUpper(k8sTask.Status.State))
	}
}

// nodeNameFromDescriptor extracts the target K8s node name from the merged constraints.
// Workflows pin tasks to a host via a machine_id constraint (set by the per-host for loop
// in workflow templates). Returns an error if no machine_id constraint is found, since K8s
// tasks require an explicit node target (there is no offer mechanism to derive it from).
func nodeNameFromDescriptor(desc *Descriptor, taskClass *taskclass.Class) (string, error) {
	merged := desc.RoleConstraints.MergeParent(taskClass.Constraints)
	for _, c := range merged {
		if c.Attribute == "machine_id" {
			return c.Value, nil
		}
	}
	return "", fmt.Errorf("task class %q has no machine_id constraint: K8s tasks require a machine_id constraint to select the target node", desc.TaskClassName)
}

// cmdEnvToK8sEnvVars converts a slice of "KEY=VALUE" strings to Kubernetes EnvVar objects.
func cmdEnvToK8sEnvVars(env []string) []corev1.EnvVar {
	vars := make([]corev1.EnvVar, 0, len(env))
	for _, e := range env {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			vars = append(vars, corev1.EnvVar{Name: parts[0], Value: parts[1]})
		}
	}
	return vars
}
