/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2018 CERN and copyright holders of ALICE O².
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
	"time"

	"github.com/AliceO2Group/Control/core/task"
	"github.com/AliceO2Group/Control/core/task/taskop"
	"github.com/pborman/uuid"
	"github.com/sirupsen/logrus"
)

func NewConfigureTransition(taskman *task.ManagerV2, addRoles []string, removeRoles []string, reconfigureAll bool) Transition {
	return &ConfigureTransition{
		baseTransition: baseTransition{
			name: "CONFIGURE",
			taskman: taskman,
		},
		addRoles: addRoles,
		removeRoles: removeRoles,
		reconfigureAll: reconfigureAll,
	}
}

type ConfigureTransition struct {
	baseTransition
	addRoles		[]string
	removeRoles		[]string
	reconfigureAll	bool
}

func (t ConfigureTransition) do(env *Environment) (err error) {
	if env == nil {
		return errors.New("cannot transition in NIL environment")
	}

	wf := env.Workflow()


	// Role tree operations go here, and afterwards we'll generally get a role tree which
	// has
	// - some TaskRoles already deployed with Tasks
	// - some TaskRoles with no Tasks but with matching Tasks in the roster
	// - some TaskRoles with no Tasks and no matching running Tasks in the roster

/*
	// First we free the relevant roles, if any
	if len(t.removeRoles) != 0 {
		rolesThatStay := env.roles[:0]
		rolesToRelease := make([]string, 0)

		for _, role := range env.roles {
			for _, removeRole := range t.removeRoles {
				if role == removeRole {
					rolesToRelease = append(rolesToRelease, role)
					break
				}
				rolesThatStay = append(rolesThatStay, role)
			}
		}
		err = t.roleman.ReleaseRoles(env.Id().Array(), rolesToRelease)
		if err != nil {
			return
		}
		env.roles = rolesThatStay
	}
	// IDEA: instead of passing around m.state or roleman, pass around one or more of
	// roleman's channels. This way roleman could potentially be lockless, and we just pipe
	// him a list of rolenames to remove/add, or even a function or a struct that does so.
	// This struct would implement an interface of the type of his channel, and he could
	// use type assertion to check whether he needs to add, remove or do something else.

	// Alright, so now we have freed some roles (if required).
	// We proceed by deduplicating and attempting an acquire.
	if len(t.addRoles) != 0 {
		rolesToAcquire := make([]string, 0)

		for _, addRole := range t.addRoles {
			alreadyInEnv := false
			for _, role := range env.roles {
				if role == addRole {
					alreadyInEnv = true
					break
				}
			}
			if !alreadyInEnv {
				rolesToAcquire = append(rolesToAcquire, addRole)
			}
		}
		err = t.roleman.AcquireRoles(env.Id().Array(), rolesToAcquire)
		if err != nil {
			return
		}

		// We complete a move to CONFIGURED for all roles and we're done.
		err = t.roleman.ConfigureRoles(env.Id().Array(), rolesToAcquire)
		if err != nil {
			return
		}

		env.roles = append(env.roles, rolesToAcquire...)
	}

	// Finally, we configure.
	if t.reconfigureAll {
		err = t.roleman.ConfigureRoles(env.Id().Array(), env.roles)
		if err != nil {
			return
		}
	}

	return*/

	notify := make(chan task.Status)
	subscriptionId := uuid.NewUUID().String()
	env.wfAdapter.SubscribeToStatusChange(subscriptionId, notify)
	defer env.wfAdapter.UnsubscribeFromStatusChange(subscriptionId)

	taskDescriptors := wf.GenerateTaskDescriptors()
	if len(taskDescriptors) != 0 {
		// err = t.taskman.AcquireTasks(env.Id().Array(), taskDescriptors)
		taskmanMessage := task.NewEnvironmentMessage(taskop.AcquireTasks, env.Id(), nil, taskDescriptors)
		t.taskman.MessageChannel <- taskmanMessage
	}
	if err != nil {
		return
	}

	deploymentTimeout := 90 * time.Second
	wfStatus := wf.GetStatus()
	if wfStatus != task.ACTIVE {
		WORKFLOW_ACTIVE_LOOP:
		for {
			log.Debug("waiting for workflow to become active")
			select {
			case wfStatus = <-notify:
				log.WithField("status", wfStatus.String()).
				    Debug("workflow status change")
				if wfStatus == task.ACTIVE {
					break WORKFLOW_ACTIVE_LOOP
				}
				continue
			case <-time.After(deploymentTimeout):
				err = errors.New("workflow deployment timed out")
				break WORKFLOW_ACTIVE_LOOP
			}
		}
	}

	if err != nil {
		log.WithFields(logrus.Fields{"error": err.Error(), "timeout": deploymentTimeout.String()}).
			Error("workflow deployment error")
		return
	}

	tasks := wf.GetTasks()

	if len(tasks) != 0 {
		// err = t.taskman.ConfigureTasks(env.Id().Array(), tasks)
		taskmanMessage := task.NewEnvironmentMessage(taskop.ConfigureTasks, env.Id(), tasks, nil)
		t.taskman.MessageChannel <- taskmanMessage
		// if err != nil {
		// 	return
		// }
	}

	// This will subscribe to workflow state change. In case of workflow state
	// ERROR will transition environment to ERROR state. The goroutine starts 
	// after a successful transition to CONFIGURE in order to handle only ERROR
	// states triggered by mesos.(TASK_LOST,TASK_KILLED,TASK_FAILED,TASK_ERROR)
	go func() {
		wf := env.Workflow()
		notify := make(chan task.State)
		subscriptionId := uuid.NewUUID().String()
		env.wfAdapter.SubscribeToStateChange(subscriptionId, notify)
		defer env.wfAdapter.UnsubscribeFromStateChange(subscriptionId)

		wfState := wf.GetState()
		if wfState != task.ERROR {
			WORKFLOW_STATE_LOOP:
			for {
				select {
				case wfState = <-notify:
					if wfState == task.DONE {
						break WORKFLOW_STATE_LOOP
					}
					// We kill the goroutine on a reset or a teardown
					// of the environment
					if wfState == task.STANDBY {
						break WORKFLOW_STATE_LOOP
					}
					if wfState == task.ERROR {
						prvState := env.CurrentState()
						env.setState(wfState.String())
						if prvState == "RUNNING" {
							taskmanMessage := task.NewTransitionTaskMessage(
								env.Workflow().GetTasks(),
								task.RUNNING.String(),
								task.STOP.String(),
								task.CONFIGURED.String(),
								nil,
							)
							t.taskman.MessageChannel <- taskmanMessage
							// err = t.taskman.TransitionTasks(
							// 	env.Workflow().GetTasks(),
							// 	task.RUNNING.String(),
							// 	task.STOP.String(),
							// 	task.CONFIGURED.String(),
							// 	nil,
							// )
						}
						break WORKFLOW_STATE_LOOP
					}
				}
			}
		}
	}()

	return
}
