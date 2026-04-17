/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2024 CERN and copyright holders of ALICE O².
 * Author: Teo Mrnjavac <teo.mrnjavac@cern.ch>
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

package controller

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strconv"

	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	aliecsv1alpha1 "github.com/AliceO2Group/Control/operator/api/v1alpha1"
	"github.com/go-logr/logr"
)

// EnvironmentReconciler reconciles a Environment object
type EnvironmentReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *EnvironmentReconciler) runTasksFromReferenceOnNode(ctx context.Context, taskReferences []aliecsv1alpha1.TaskReference,
	nodename string, req ctrl.Request, environment *aliecsv1alpha1.Environment, log logr.Logger,
) (*ctrl.Result, error) {
	for _, taskReference := range taskReferences {
		log.Info("geting stored template for task", "task", taskReference.Name)
		template := &aliecsv1alpha1.TaskTemplate{}
		if err := r.Get(ctx, types.NamespacedName{Namespace: req.Namespace, Name: taskReference.Name}, template); err != nil {
			log.Error(err, "failed to get template for task", "task", taskReference.Name)
			return &ctrl.Result{}, nil
		}

		task := &aliecsv1alpha1.Task{}
		task.Namespace = req.Namespace
		task.Name = fmt.Sprintf("%s-%s", nodename, template.Name)
		if err := r.Get(ctx, types.NamespacedName{Name: task.Name, Namespace: task.Namespace}, task); err == nil {
			continue
		}

		// TODO: N^2 change
		for _, env := range template.Spec.EnvVars {
			if !slices.ContainsFunc(taskReference.Env, func(envVar v1.EnvVar) bool {
				return envVar.Name == env
			}) {
				log.Error(fmt.Errorf("didn't find required env: %s", env), "failed to fill in env vars from template")
				return &reconcile.Result{}, nil
			}
		}

		task.Spec.Pod = *template.Spec.Pod.DeepCopy()
		task.Spec.Control = *template.Spec.Control.DeepCopy()

		if foundIdx := slices.IndexFunc(taskReference.Env, func(envVar v1.EnvVar) bool { return envVar.Name == "OCC_CONTROL_PORT" }); foundIdx == -1 {
			log.Error(fmt.Errorf("didn't find OCC_CONTROL_PORT in env"), "failed to fill in env vars from template")
		} else {
			port, err := strconv.Atoi(taskReference.Env[foundIdx].Value)
			if err != nil {
				log.Error(fmt.Errorf("found OCC_CONTROL_PORT isn't convertible to number "), "failed to fill in env vars from template")
				return &reconcile.Result{}, nil
			}
			task.Spec.Control.Port = port
		}

		if task.Spec.Arguments == nil {
			task.Spec.Arguments = make(map[string]string)
		}

		// TODO: check for containers!
		task.Spec.Pod.Containers[0].Env = append(task.Spec.Pod.Containers[0].Env, taskReference.Env...)
		task.Spec.Pod.Containers[0].Args = append(task.Spec.Pod.Containers[0].Args, taskReference.ArgsCLI...)
		maps.Copy(task.Spec.Arguments, taskReference.ArgsTransition)

		task.Spec.Pod.NodeName = nodename
		task.Spec.State = environment.Spec.State

		task.Labels = labels(environment, nodename)

		if err := controllerutil.SetControllerReference(environment, task, r.Scheme); err != nil {
			log.Error(err, "failed to set controller reference", "task", task.Name)
			return &ctrl.Result{}, err
		}

		if err := r.Create(ctx, task); err != nil {
			if err = client.IgnoreAlreadyExists(err); err != nil {
				log.Error(err, "Failed to create task on node", "nodename", nodename, "task", task.Name)
				return &ctrl.Result{}, err
				// TODO: add error handling
			}
		}
	}
	return nil, nil
}

func (r *EnvironmentReconciler) runTaskFromDefinitionOnNode(ctx context.Context, taskDefs []aliecsv1alpha1.TaskDefinition,
	nodename string, req ctrl.Request, environment *aliecsv1alpha1.Environment, log logr.Logger,
) (*ctrl.Result, error) {
	for _, taskDef := range taskDefs {
		task := &aliecsv1alpha1.Task{}
		task.Namespace = req.Namespace
		task.Name = fmt.Sprintf("%s-%s", nodename, taskDef.Name)
		if err := r.Get(ctx, types.NamespacedName{Name: task.Name, Namespace: task.Namespace}, task); err == nil {
			continue
		}

		task.Spec.Pod = *taskDef.Spec.Pod.DeepCopy()
		task.Spec.Control = *taskDef.Spec.Control.DeepCopy()

		if len(taskDef.Spec.Arguments) > 0 {
			if task.Spec.Arguments == nil {
				task.Spec.Arguments = make(map[string]string)
			}
			maps.Copy(task.Spec.Arguments, taskDef.Spec.Arguments)
		}

		task.Spec.Pod.NodeName = nodename
		task.Spec.State = environment.Spec.State

		task.Labels = labels(environment, nodename)

		if err := controllerutil.SetControllerReference(environment, task, r.Scheme); err != nil {
			log.Error(err, "failed to set controller reference", "task", task.Name)
			return &ctrl.Result{}, err
		}

		if err := r.Create(ctx, task); err != nil {
			if err = client.IgnoreAlreadyExists(err); err != nil {
				log.Error(err, "Failed to create task on node", "nodename", nodename, "task", task.Name)
				return &ctrl.Result{}, err
				// TODO: add error handling
			}
		}

	}

	return nil, nil
}

// create label to bunch tasks to the environment and node
func labels(environment *aliecsv1alpha1.Environment, nodename string) map[string]string {
	return map[string]string{
		"environment": environment.Name,
		"node":        nodename,
	}
}

// +kubebuilder:rbac:groups=aliecs.alice.cern,resources=environments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=aliecs.alice.cern,resources=environments/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=aliecs.alice.cern,resources=environments/finalizers,verbs=update
// +kubebuilder:rbac:groups=aliecs.alice.cern,resources=tasktemplates,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=nodes,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Environment object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.23.1/pkg/reconcile
func (r *EnvironmentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	environment := &aliecsv1alpha1.Environment{}
	if err := r.Get(ctx, req.NamespacedName, environment); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if environment.Status.State == "" {
		for nodename, tasksReferences := range environment.TaskTemplates.Tasks {
			log.Info("creating tasks for hostname from references", "hostname", nodename, "number of tasks", len(tasksReferences))
			if res, err := r.checkNodeExistence(ctx, nodename); err != nil {
				return res, err
			}

			if res, err := r.runTasksFromReferenceOnNode(ctx, tasksReferences, nodename, req, environment, log); res != nil {
				return *res, err
			}
		}

		for nodename, taskTemplates := range environment.Spec.Tasks {
			log.Info("creating tasks for hostname from definitions", "hostname", nodename, "number of tasks", len(taskTemplates))
			if res, err := r.checkNodeExistence(ctx, nodename); err != nil {
				return res, err
			}

			if res, err := r.runTaskFromDefinitionOnNode(ctx, taskTemplates, nodename, req, environment, log); res != nil {
				return *res, err
			}
		}
	}

	tasks := &aliecsv1alpha1.TaskList{}
	if err := r.List(ctx, tasks, client.InNamespace(environment.Namespace), client.MatchingLabels{"environment": environment.Name}); err != nil {
		log.Error(err, "failed to get list of tasks for this environment")
		return ctrl.Result{}, err
	}

	if environment.Status.Tasks == nil {
		environment.Status.Tasks = make(map[string]map[string]string)
	}

	for _, task := range tasks.Items {
		nodename := task.Spec.Pod.NodeName
		if environment.Status.Tasks[nodename] == nil {
			environment.Status.Tasks[nodename] = make(map[string]string)
		}

		environment.Status.Tasks[nodename][task.Name] = task.Status.State
	}

	environment.Status.State = aggregateState(tasks.Items, environment.Status.State)

	for _, task := range tasks.Items {
		if task.Spec.State != environment.Spec.State {
			patch := client.MergeFrom(task.DeepCopy())
			task.Spec.State = environment.Spec.State
			if err := r.Patch(ctx, &task, patch); err != nil {
				log.Error(err, "failed to patch task state", "task", task.Name)
			}
		}
	}

	if err := r.Status().Update(ctx, environment); err != nil {
		if k8serrors.IsConflict(err) {
			return ctrl.Result{Requeue: true}, nil
		}
		log.Error(err, "failed to update status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *EnvironmentReconciler) checkNodeExistence(ctx context.Context, nodename string) (ctrl.Result, error) {
	node := &v1.Node{}
	if err := r.Get(ctx, types.NamespacedName{Name: nodename}, node); err != nil {
		if k8serrors.IsNotFound(err) {
			return ctrl.Result{}, fmt.Errorf("node %s not found in cluster", nodename)
		}
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func aggregateState(tasks []aliecsv1alpha1.Task, previousState string) string {
	if len(tasks) == 0 {
		return previousState
	}

	first := tasks[0].Status.State
	allSame := true
	for _, task := range tasks {
		if task.Status.State == "error" {
			return "error"
		}
		if task.Status.State != first {
			allSame = false
		}
	}

	if allSame {
		return first
	}
	return previousState
}

// SetupWithManager sets up the controller with the Manager.
func (r *EnvironmentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&aliecsv1alpha1.Environment{}).
		Owns(&aliecsv1alpha1.Task{}).
		Watches(&aliecsv1alpha1.TaskTemplate{}, handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, obj client.Object) []reconcile.Request {
			return nil
		})).
		Named("environment").
		Complete(r)
}
