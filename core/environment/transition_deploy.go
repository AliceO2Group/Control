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
	"fmt"
	"time"

	"github.com/AliceO2Group/Control/common/event"
	"github.com/AliceO2Group/Control/core/task"
	"github.com/AliceO2Group/Control/core/task/taskop"
	"github.com/AliceO2Group/Control/core/workflow"
	"github.com/pborman/uuid"
	"github.com/sirupsen/logrus"
)

func NewDeployTransition(taskman *task.Manager, addRoles []string, removeRoles []string) Transition {
	return &DeployTransition{
		baseTransition: baseTransition{
			name: "DEPLOY",
			taskman: taskman,
		},
		addRoles: addRoles,
		removeRoles: removeRoles,
	}
}

type DeployTransition struct {
	baseTransition
	addRoles		[]string
	removeRoles		[]string
}

func (t DeployTransition) do(env *Environment) (err error) {
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

	notifyStatus := make(chan task.Status)
	subscriptionId := uuid.NewUUID().String()
	env.wfAdapter.SubscribeToStatusChange(subscriptionId, notifyStatus)
	defer env.wfAdapter.UnsubscribeFromStatusChange(subscriptionId)

	// listen to workflow State changes
	notifyState := make(chan task.State)
	env.wfAdapter.SubscribeToStateChange(subscriptionId, notifyState)
	defer env.wfAdapter.UnsubscribeFromStateChange(subscriptionId)

	taskDescriptors := wf.GenerateTaskDescriptors()
	if len(taskDescriptors) != 0 {
		// err = t.taskman.AcquireTasks(env.Id().Array(), taskDescriptors)
		taskmanMessage := task.NewEnvironmentMessage(taskop.AcquireTasks, env.Id(), nil, taskDescriptors)
		t.taskman.MessageChannel <- taskmanMessage
	}
	if err != nil {
		return
	}

	// We set all callRoles to ACTIVE right now, because there's no task activation for them.
	// This is the callRole equivalent of AcquireTasks, which only pushes updates to taskRoles.
	allHooks := wf.GetHooksForTrigger("")	// no trigger = all hooks
	callHooks := allHooks.FilterCalls()							// get the calls
	if len(callHooks) > 0 {
		for _, h := range callHooks {
			pr, ok := h.GetParentRole().(workflow.PublicUpdatable)
			if !ok {
				continue
			}
			go pr.UpdateStatus(task.ACTIVE)
		}
	}

	deploymentTimeout := 90 * time.Second
	wfStatus := wf.GetStatus()
	if wfStatus != task.ACTIVE {
		WORKFLOW_ACTIVE_LOOP:
		for {
			log.Debug("waiting for workflow to become active")
			select {
			case wfStatus = <-notifyStatus:
				log.WithField("status", wfStatus.String()).
				    Debug("workflow status change")
				if wfStatus == task.ACTIVE {
					break WORKFLOW_ACTIVE_LOOP
				}
				continue
			case <-time.After(deploymentTimeout):
				err = errors.New(fmt.Sprintf("workflow deployment timed out. timeout: %s",deploymentTimeout.String()))
				break WORKFLOW_ACTIVE_LOOP
			// This is needed for when the workflow fails during the STAGING state(mesos status),mesos responds with the `REASON_COMMAND_EXECUTOR_FAILED`,
			// By listening to workflow state ERROR we can break the loop before reaching the timeout (1m30s), we can trigger the cleanup faster
			// in the CreateEnvironment (environment/manager.go) and the lock in the `envman` is reserved for a sorter period, which allows operations like
			// `environment list` to be done almost immediatelly after mesos informs with TASK_FAILED.
			case wfState := <-notifyState:
				if wfState == task.ERROR {
					workflow.LeafWalk(wf, func(role workflow.Role) {
						if st := role.GetState();  st == task.ERROR {
							log.WithField("state", st).
								WithField("role", role.GetPath()).
								WithField("environment", role.GetEnvironmentId().String()).
								Error("environment reached invalid state")
						}
					})
					log.WithField("state", wfState.String()).
				    	Debug("workflow state change")
					err = errors.New("workflow deployment failed, aborting and cleaning up")
					break WORKFLOW_ACTIVE_LOOP
				}
			}
		}
	}

	if err != nil {
		log.WithFields(logrus.Fields{"error": err.Error()}).
			Error("workflow deployment error")
		return
	}

	env.sendEnvironmentEvent(&event.EnvironmentEvent{EnvironmentID: env.Id().String(), State: "DEPLOYED"})
	return
}
