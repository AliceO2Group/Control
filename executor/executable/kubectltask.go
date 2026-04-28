/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2018-2025 CERN and copyright holders of ALICE O².
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

package executable

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/AliceO2Group/Control/core/controlcommands"
	"github.com/AliceO2Group/Control/executor/executorcmd"
	mesos "github.com/mesos/mesos-go/api/v1/lib"
	"github.com/sirupsen/logrus"
)

const (
	KUBECTL string = "kubectl"

	APPLY  string = "apply"
	DELETE string = "delete"
	PATCH  string = "patch"
	GET    string = "get"
	TASK   string = "task"
	CREATE string = "create"

	// TRANSITION_TIMEOUT = 10 * time.Second // inside controllable task
)

var kubeconfigDir string

func init() {
	if kc := os.Getenv("KUBECONFIG"); kc != "" {
		kubeconfigDir = kc
		return
	}
	if u, err := user.Current(); err == nil {
		tempPath := filepath.Join(u.HomeDir, ".kube", "config")
		if _, err = os.Stat(tempPath); err == nil {
			kubeconfigDir = tempPath
		}
	}
}

func kubectl(ctx context.Context, arg ...string) *exec.Cmd {
	if kubeconfigDir == "" {
		log.Warn(`kubectl config was not set, thus kubectl might not be able to find a cluster. 
Either KUBECONFIG env var was not found, or current user was not determined, or home/.kube/config does not exist. 
Using kubectl builtin defaults`)
		return exec.CommandContext(ctx, KUBECTL, arg...)
	} else {
		return exec.CommandContext(ctx, KUBECTL, append([]string{"--kubeconfig", kubeconfigDir}, arg...)...)
	}
}

type KubectlTask struct {
	taskBase
	rpc          *executorcmd.RpcClient
	configYaml   string
	running      bool
	latestStatus atomic.Value
}

func GetUserInfo(username string) (uid, gid int64, supplemental []int64, err error) {
	u, err := user.Lookup(username)
	if err != nil {
		return 0, 0, nil, err
	}

	// Convert UID
	uidInt, _ := strconv.ParseInt(u.Uid, 10, 64)

	// Convert Primary GID
	gidInt, _ := strconv.ParseInt(u.Gid, 10, 64)

	// Get Supplemental Groups (e.g., wheel, pda)
	groupStrings, _ := u.GroupIds()
	var supplementalInts []int64
	for _, g := range groupStrings {
		gInt, _ := strconv.ParseInt(g, 10, 64)
		// Avoid adding the primary GID to the supplemental list
		if gInt != gidInt {
			supplementalInts = append(supplementalInts, gInt)
		}
	}

	return uidInt, gidInt, supplementalInts, nil
}

func (task *KubectlTask) Launch() error {
	if len(task.Tci.Arguments) == 0 {
		log.WithFields(logrus.Fields{
			"controlmode": task.Tci.ControlMode,
			"name":        task.ti.Name,
		}).
			Error("no arguments in kubectl task. We need to have at least manifest location as the last argument")
		return errors.New("no arguments for kubectl task. Location for kubernetes manifest needed")
	}

	task.configYaml = task.Tci.Arguments[0]

	// Read the template file
	content, err := os.ReadFile(task.configYaml)
	if err != nil {
		log.WithFields(logrus.Fields{
			"controlmode": task.Tci.ControlMode,
			"name":        task.ti.Name,
			"file":        task.configYaml,
		}).WithError(err).Error("failed to read kubectl config file")
		return err
	}

	// Set the AliECS environment variables in the local process
	// so os.ExpandEnv can find them
	for _, envVar := range task.Tci.Env {
		parts := strings.SplitN(envVar, "=", 2)
		if len(parts) == 2 {
			os.Setenv(parts[0], parts[1])
		}
	}

	// Set arguments into the KUBE_ARGUMENTS os env leaving the kubemanifest file
	arguments := task.Tci.Arguments[1:]

	log.WithFields(logrus.Fields{
		"controlmode": task.Tci.ControlMode,
		"name":        task.ti.Name,
		"args":        arguments,
	}).Info("setting arguments as a KUBE_ARGUMENTS env var")

	os.Setenv("KUBE_ARGUMENTS", strings.Join(arguments, " "))

	log.WithFields(logrus.Fields{
		"controlmode": task.Tci.ControlMode,
		"name":        task.ti.Name,
		"command":     *task.Tci.Value,
	}).Info("setting command as a KUBE_COMMAND env var")
	os.Setenv("KUBE_COMMAND", *task.Tci.Value)

	if uid, gid, supplementalIds, err := GetUserInfo("flp"); err == nil {
		os.Setenv("FLP_UID", strconv.FormatInt(uid, 10))
		os.Setenv("FLP_GID", strconv.FormatInt(gid, 10))

		var strIds []string
		for _, id := range supplementalIds {
			strIds = append(strIds, strconv.FormatInt(id, 10))
		}
		supplementalString := "[" + strings.Join(strIds, ", ") + "]"

		os.Setenv("FLP_SUPPLEMENTAL_GROUPS", supplementalString)
	} else {
		log.Error("we cannot run kubectl task as flp user because we didn't find user details")
	}

	// Replace ${VAR} placeholders with actual values
	expandedYaml := os.ExpandEnv(string(content))

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	// Apply via Stdin (-)
	command := kubectl(ctx, APPLY, "-f", "-")
	command.Stdin = strings.NewReader(expandedYaml)

	log.WithFields(logrus.Fields{
		"controlmode":  task.Tci.ControlMode,
		"name":         task.ti.Name,
		"command":      command,
		"expandedYaml": expandedYaml,
	}).Info("Starting kubectl apply via Stdin")

	var stdoutBuf bytes.Buffer
	var stderrBuf bytes.Buffer

	command.Stdout = io.MultiWriter(os.Stdout, &stdoutBuf)
	command.Stderr = io.MultiWriter(os.Stderr, &stderrBuf)

	err = command.Run()
	if err != nil {
		log.WithFields(logrus.Fields{
			"controlmode": task.Tci.ControlMode,
			"name":        task.ti.Name,
		}).WithError(err).Errorf("kubectl apply failed stderr: %s , stdout: %s", stderrBuf.String(), stdoutBuf.String())
		return err
	}

	task.latestStatus.Store("")
	task.running = true
	go task.eventLoop()
	return nil
}

func (task *KubectlTask) Kill() error {
	task.running = false
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	command := kubectl(ctx, DELETE, "-f", task.configYaml)

	command.Stdout = os.Stdout
	command.Stderr = os.Stderr

	err := command.Run()
	if err != nil {
		log.WithFields(logrus.Fields{
			"controlmode": task.Tci.ControlMode,
			"name":        task.ti.Name,
		}).WithError(err).Error("kubectl delete failed")
		return err
	}

	task.sendStatus(task.knownEnvironmentId, mesos.TASK_FINISHED, "")

	return nil
}

func (task *KubectlTask) Transition(transition *executorcmd.ExecutorCommand_Transition) *controlcommands.MesosCommandResponse_Transition {
	// kubectl patch -f exampletask.yaml --type='json' -p='[{"op": "replace", "path": "/spec/state", "value": "running"}]'

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	// Extract the transition arguments (the 'Mix') to pipe them to Kubernetes
	log.WithFields(logrus.Fields{
		"controlmode": task.Tci.ControlMode,
		"name":        task.ti.Name,
		"args":        transition.Arguments,
	}).Info("Patching transition arguments to Kubernetes")

	argsJSON, err := json.Marshal(transition.Arguments)
	if err != nil {
		log.WithFields(logrus.Fields{
			"controlmode": task.Tci.ControlMode,
			"name":        task.ti.Name,
		}).WithError(err).Error("failed to marshal transition arguments")
		return transition.PrepareResponse(err, transition.Source, task.ti.TaskID.Value)
	}

	transitionJSON := fmt.Sprintf(`[
		{"op": "add", "path": "/spec/state", "value": "%s"},
		{"op": "add", "path": "/spec/arguments", "value": %s}
	]`, strings.ToLower(transition.Destination), string(argsJSON))

	command := kubectl(ctx, PATCH, "-f", task.configYaml, "--type=json", "-p", transitionJSON)

	command.Stdout = os.Stdout
	command.Stderr = os.Stderr

	log.WithFields(logrus.Fields{
		"controlmode": task.Tci.ControlMode,
		"name":        task.ti.Name,
		"command":     command,
	}).Info("Starting kubectl patch")

	statusBeforeTransition := task.latestStatus.Load().(string)

	err = command.Run()
	if err != nil {
		log.WithFields(logrus.Fields{
			"controlmode": task.Tci.ControlMode,
			"name":        task.ti.Name,
			"command":     command,
		}).WithError(err).Error("kubectl patch failed")
		return transition.PrepareResponse(err, transition.Source, task.ti.TaskID.Value)
	}

	log.WithFields(logrus.Fields{
		"controlmode": task.Tci.ControlMode,
		"name":        task.ti.Name,
		"command":     command,
	}).Info("kubectl patch suceeded, waiting for actual status change")
	actualStatus := ""
	timeout := time.After(TRANSITION_TIMEOUT)
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

loop:
	for {
		select {
		case <-ticker.C:
			actualStatus = task.latestStatus.Load().(string)
			if actualStatus != statusBeforeTransition {
				break loop
			}
		case <-timeout:
			return transition.PrepareResponse(errors.New("timeout waiting for status change"), statusBeforeTransition, task.ti.TaskID.Value)
		}
	}
	log.WithFields(logrus.Fields{
		"controlmode": task.Tci.ControlMode,
		"name":        task.ti.Name,
		"command":     command,
	}).Infof("status changed from %s to %s", statusBeforeTransition, actualStatus)

	// TODO: I am not sure what PID should I put here
	return transition.PrepareResponse(nil, actualStatus, task.ti.TaskID.Value)
}

func (task *KubectlTask) getTaskStatus() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	// command := exec.CommandContext(ctx, KUBECTL, GET, "-f", task.configYaml, "-o", "jsonpath={.status.state}")
	command := kubectl(ctx, GET, "-f", task.configYaml, "-o", "jsonpath={.status.state}")

	var stdoutBuf bytes.Buffer

	command.Stdout = io.MultiWriter(os.Stdout, &stdoutBuf)
	command.Stderr = os.Stderr

	log.WithFields(logrus.Fields{
		"controlmode": task.Tci.ControlMode,
		"name":        task.ti.Name,
		"command":     command,
	}).Debug("Starting kubectl get task")

	err := command.Run()
	if err != nil {
		log.WithFields(logrus.Fields{
			"controlmode": task.Tci.ControlMode,
			"name":        task.ti.Name,
			"command":     command,
		}).WithError(err).Error("kubectl get task failed")
		return "", err
	}

	// no newlines
	return strings.TrimSpace(stdoutBuf.String()), nil
}

func (task *KubectlTask) eventLoop() {
	errorCount := 0
	maxErrors := 5
	for task.running {
		time.Sleep(5 * time.Second)
		status, err := task.getTaskStatus()
		if err != nil {
			errorCount += 1
			if errorCount < maxErrors {
				log.WithError(err).Warnf("failed to get Task Status, retrying %d/%d", errorCount, maxErrors)
				continue
			}
			log.WithError(err).Errorf("failed to get Task Status, sending TASK_FAILED and breaking from the eventLoop")
			task.sendStatus(task.knownEnvironmentId, mesos.TASK_FAILED, "couldn't get task status via kubectl")
			task.running = false
			// TODO: remove when debugging done
			// _ = task.Kill()
			break
		}

		status = strings.ToUpper(status)

		if task.latestStatus.Load().(string) == status {
			continue
		}
		task.latestStatus.Store(status)

		var state mesos.TaskState
		switch status {
		case "CONFIGURED", "RUNNING", "STANDBY":
			state = mesos.TASK_RUNNING

		case "ERROR":
			state = mesos.TASK_FAILED
			log.WithError(err).Error("Received error from kubectl task, terminating everything and sending update")
			task.running = false
		// TODO: remove when debugging done
		// _ = task.Kill()
		//

		default:
			log.Errorf("received different status than expected: %s", status)
			continue
		}

		log.Debugf("sending new status from kubectl task %s", status)
		task.sendStatus(task.knownEnvironmentId, state, "")

	}
}

func (task *KubectlTask) UnmarshalTransition(data []byte) (cmd *executorcmd.ExecutorCommand_Transition, err error) {
	cmd = new(executorcmd.ExecutorCommand_Transition)
	if task.rpc != nil {
		cmd.Transitioner = task.rpc.Transitioner
	}
	err = json.Unmarshal(data, cmd)
	if err != nil {
		cmd = nil
	}
	return
}
