/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2017-2018 CERN and copyright holders of ALICE O².
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

package environment

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/AliceO2Group/Control/common/utils/uid"
	"github.com/AliceO2Group/Control/common/controlmode"
	"github.com/AliceO2Group/Control/common/event"
	"github.com/AliceO2Group/Control/common/utils"
	"github.com/AliceO2Group/Control/core/task"
	"github.com/AliceO2Group/Control/core/task/taskop"
	"github.com/AliceO2Group/Control/core/workflow"
	pb "github.com/AliceO2Group/Control/executor/protos"
	"github.com/sirupsen/logrus"
)

type Manager struct {
	mu      sync.RWMutex
	m       map[uid.ID]*Environment
	taskman *task.ManagerV2
	incomingEventCh <-chan event.Event
}

func NewEnvManager(tm *task.ManagerV2, incomingEventCh <-chan event.Event) *Manager {
	envman := &Manager{
		m:               make(map[uid.ID]*Environment),
		taskman:         tm,
		incomingEventCh: incomingEventCh,
	}

	go func() {
		for ;; {
			select {
			case incomingEvent := <- envman.incomingEventCh:
				switch typedEvent := incomingEvent.(type) {
				case event.DeviceEvent:
					envman.handleDeviceEvent(typedEvent)
				default:
					// noop
				}
			}
		}
	}()
	return envman
}

func (envs *Manager) CreateEnvironment(workflowPath string, userVars map[string]string) (uid.ID, error) {
	envs.mu.Lock()
	defer envs.mu.Unlock()

	// userVar identifiers come in 2 forms:
	// environment user var: "someKey"
	// workflow user var:    "path.to.some.role:someKey"
	// We need to split them into 2 structures, the first of which is passed to newEnvironment and the other one
	// to loadWorkflow as its keys must be injected into one or more specific roles.

	envUserVars := make(map[string]string)
	workflowUserVars := make(map[string]string)
	for k, v := range userVars {
		// If the key contains a ':', means we have a var associated with a specific workflow role
		if strings.ContainsRune(k, task.TARGET_SEPARATOR_RUNE) {
			workflowUserVars[k] = v
		} else {
			envUserVars[k] = v
		}
	}

	env, err := newEnvironment(envUserVars)
	if err != nil {
		return uid.NilID(), err
	}
	env.hookHandlerF = func(hooks task.Tasks) error {
		return envs.taskman.TriggerHooks(hooks)
	}
	env.workflow, err = envs.loadWorkflow(workflowPath, env.wfAdapter, workflowUserVars)
	if err != nil {
		err = fmt.Errorf("cannot load workflow template: %w", err)
		return env.id, err
	}

	envs.m[env.id] = env

	err = env.TryTransition(NewConfigureTransition(
		envs.taskman,
		nil, //roles,
		nil,
		true	))
	if err != nil {
		envState := env.CurrentState()
		envTasks := env.Workflow().GetTasks()
		taskmanMessage := task.NewEnvironmentMessage(taskop.ReleaseTasks,env.id.Array(), envTasks, nil)
		envs.taskman.MessageChannel <- taskmanMessage
		// rlsErr := envs.taskman.ReleaseTasks(env.id.Array(), envTasks)
		// if rlsErr != nil {
		// 	log.WithError(rlsErr).Warning("environment configure failed, some tasks could not be released")
		// }

		delete(envs.m, env.id)

		// As this is a failed env deployment, we must clean up any running & released tasks.
		// This is an important step as env deploy failures are often caused by bad task
		// configuration. In this case, the task is likely to go STANDBY×CONFIGURE→ERROR,
		// so there's no point in keeping it for future recovery & reconfiguration.
		
		// draft mostly on Cleanup() and KillTasks() we need the killedTasks
		// either to print them or return them to server. We need to setup a way
		// to communicate back such info from taskman (channel ?)
		// var killedTasks task.Tasks
		// taskmanMessage = task.NewkillTasksMessage(envTasks.GetTaskIds())
		// envs.taskman.MessageChannel <- taskmanMessage

		killedTasks, _, rlsErr := envs.taskman.KillTasks(envTasks.GetTaskIds())
		if rlsErr != nil {
			log.WithError(rlsErr).Warn("task teardown error")
		}
		log.WithFields(logrus.Fields{
			"killedCount": len(killedTasks),
			"lastEnvState": envState,
		}).
		Warn("environment deployment failed, tasks were cleaned up")

		return env.id, err
	}

	return env.id, err
}

func (envs *Manager) TeardownEnvironment(environmentId uid.ID, force bool) error {
	envs.mu.Lock()
	defer envs.mu.Unlock()

	env, err := envs.environment(environmentId)
	if err != nil {
		return err
	}

	if env.CurrentState() != "STANDBY" && !force {
		return errors.New(fmt.Sprintf("cannot teardown environment in state %s", env.CurrentState()))
	}

	taskmanMessage := task.NewEnvironmentMessage(taskop.ReleaseTasks,environmentId.Array(), env.Workflow().GetTasks(), nil)
	envs.taskman.MessageChannel <- taskmanMessage
	// err = envs.taskman.ReleaseTasks(environmentId.Array(), env.Workflow().GetTasks())
	// if err != nil {
	// 	return err
	// }

	delete(envs.m, environmentId)
	return err
}

/*func (envs *Manager) Configuration(environmentId uuid.UUID) EnvironmentCfg {
	envs.mu.RLock()
	defer envs.mu.RUnlock()
	return envs.m[environmentId.Array()].cfg
}*/

func (envs *Manager) Ids() (keys []uid.ID) {
	envs.mu.RLock()
	defer envs.mu.RUnlock()
	keys = make([]uid.ID, len(envs.m))
	i := 0
	for k := range envs.m {
		keys[i] = k
		i++
	}
	return
}

func (envs *Manager) Environment(environmentId uid.ID) (env *Environment, err error) {
	envs.mu.RLock()
	defer envs.mu.RUnlock()
	return envs.environment(environmentId)
}

func (envs *Manager) environment(environmentId uid.ID) (env *Environment, err error) {
	if len(environmentId) == 0 { // invalid id
		return nil, fmt.Errorf("invalid id: %s", environmentId)
	}
	env, ok := envs.m[environmentId]
	if !ok {
		err = errors.New(fmt.Sprintf("no environment with id %s", environmentId))
	}
	return
}

func (envs *Manager) loadWorkflow(workflowPath string, parent workflow.Updatable, workflowUserVars map[string]string) (root workflow.Role, err error) {
	if strings.Contains(workflowPath, "://") {
		return nil, errors.New("workflow loading from file not implemented yet")
	}
	return workflow.Load(workflowPath, parent, envs.taskman, workflowUserVars)
}

func (envs *Manager) handleDeviceEvent(evt event.DeviceEvent) {
	if evt == nil {
		log.Error("cannot handle null DeviceEvent")
		return
	}

	switch evt.GetType() {
	case pb.DeviceEventType_BASIC_TASK_TERMINATED:
		if btt, ok := evt.(*event.BasicTaskTerminated); ok {
			log.WithPrefix("scheduler").
				WithFields(logrus.Fields{
					"exitCode": btt.ExitCode,
					"stdout": btt.Stdout,
					"stderr": btt.Stderr,
					"finalMesosState": btt.FinalMesosState.String(),
				}).
				Info("basic task terminated")

			// Propagate this information to the task/role
			taskId := evt.GetOrigin().TaskId
			t := envs.taskman.GetTask(taskId.Value)
			isHook := false
			if t != nil {
				if parentRole, ok := t.GetParentRole().(workflow.Role); ok {
					parentRole.SetRuntimeVars(map[string]string{
						"taskResult.exitCode": strconv.Itoa(btt.ExitCode),
						"taskResult.stdout": btt.Stdout,
						"taskResult.stderr": btt.Stderr,
						"taskResult.finalStatus": btt.FinalMesosState.String(),
						"taskResult.timestamp": utils.NewUnixTimestamp(),
					})

					// If it's an update following a HOOK execution
					if t.GetControlMode() == controlmode.HOOK {
						isHook = true
						env, err := envs.environment(t.GetEnvironmentId().UUID())
						if err != nil {
							log.WithPrefix("scheduler").WithError(err).Error("cannot find environment for DeviceEvent")
						}
						env.NotifyEvent(evt)
					}
				} else {
					log.WithPrefix("scheduler").Error("DeviceEvent BASIC_TASK_TERMINATED received for task with no parent role")
				}
			} else {
				log.WithPrefix("scheduler").Error("cannot find task for DeviceEvent BASIC_TASK_TERMINATED")
			}

			// If the task hasn't already been killed
			// AND it's not a hook
			if !isHook {
				goto doFallthrough
			}
		}
		return
	doFallthrough:
		fallthrough
	case pb.DeviceEventType_END_OF_STREAM:
		taskId := evt.GetOrigin().TaskId
		t := envs.taskman.GetTask(taskId.Value)
		if t == nil {
			log.WithPrefix("scheduler").Error("cannot find task for DeviceEvent END_OF_STREAM")
			return
		}
		env, err := envs.environment(t.GetEnvironmentId().UUID())
		if err != nil {
			log.WithPrefix("scheduler").WithError(err).Error("cannot find environment for DeviceEvent")
		}
		if env.CurrentState() == "RUNNING" {
			t.SetSafeToStop(true) // we mark this specific task as ok to STOP
			go func() {
				if env.IsSafeToStop() {     // but then we ask the env whether *all* of them are
					err = env.TryTransition(NewStopActivityTransition(envs.taskman))
					if err != nil {
						log.WithPrefix("scheduler").WithError(err).Error("cannot stop run after END_OF_STREAM event")
					}
				}
			}()
		}
	}
}
