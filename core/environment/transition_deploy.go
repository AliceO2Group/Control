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
	"context"
	"errors"
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/AliceO2Group/Control/common/event"
	"github.com/AliceO2Group/Control/common/logger/infologger"
	"github.com/AliceO2Group/Control/core/task"
	"github.com/AliceO2Group/Control/core/task/sm"
	"github.com/AliceO2Group/Control/core/task/taskop"
	"github.com/AliceO2Group/Control/core/workflow"
	"github.com/hashicorp/go-multierror"
	"github.com/pborman/uuid"
)

func NewDeployTransition(taskman *task.Manager, addRoles []string, removeRoles []string) Transition {
	return &DeployTransition{
		baseTransition: baseTransition{
			name:    "DEPLOY",
			taskman: taskman,
		},
		addRoles:    addRoles,
		removeRoles: removeRoles,
	}
}

type DeployTransition struct {
	baseTransition
	addRoles    []string
	removeRoles []string
}

func (t DeployTransition) do(env *Environment) (err error) {
	if env == nil {
		return errors.New("cannot transition in NIL environment")
	}

	wf := env.Workflow()

	// Skip cleanup for anything other than readout-dataflow
	if wf.GetName() == "readout-dataflow" {

		flps := env.GetFLPs()
		var wg sync.WaitGroup
		wg.Add(len(flps))

		var scriptErrors *multierror.Error
		// Execute the cleanup script on all FLPs before the DEPLOY transition commences
		for _, flp := range flps {
			go func(flp string) {
				defer wg.Done()

				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()

				cmdString := "ssh -o StrictHostKeyChecking=no root@" + flp + " \"source /etc/profile.d/o2.sh && O2_PARTITION=" + env.id.String() + " O2_FACILITY=core/shmcleaner O2_ROLE=" + flp + " /opt/o2/bin/o2-aliecs-shmcleaner\""
				log.WithField("partition", env.Id().String()).
					Traceln("[cleanup binary] Executing: " + cmdString)
				cmd := exec.CommandContext(ctx, "/bin/bash", "-c", cmdString)
				var localErr error
				localErr = cmd.Run()
				if localErr != nil {
					log.WithField("partition", env.Id().String()).
						Warnf("cleanup script failed on %s : %s\n", flp, localErr.Error())
					scriptErrors = multierror.Append(scriptErrors, fmt.Errorf("cleanup unsuccessful on %s: %w", flp, localErr))
				} else {
					log.WithField("partition", env.Id().String()).
						Tracef("cleanup script executed successfully on %s\n", flp)
				}
			}(flp)
		}

		wg.Wait()
		err = scriptErrors.ErrorOrNil()
	}

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
	notifyState := make(chan sm.State)
	env.wfAdapter.SubscribeToStateChange(subscriptionId, notifyState)
	defer env.wfAdapter.UnsubscribeFromStateChange(subscriptionId)

	taskDescriptors := wf.GenerateTaskDescriptors()
	if len(taskDescriptors) != 0 {
		// err = t.taskman.AcquireTasks(env.Id().Array(), taskDescriptors)
		taskmanMessage := task.NewEnvironmentMessage(taskop.AcquireTasks, env.Id(), nil, taskDescriptors)
		t.taskman.MessageChannel <- taskmanMessage
	}
	if err != nil {
		log.WithField("partition", env.Id().String()).
			Warnf("pre-deployment cleanup, %s", err.Error())
		err = nil // we don't want to fail the deployment because of pre-deploy cleanup issues
	}

	// We set all callRoles to ACTIVE right now, because there's no task activation for them.
	// This is the callRole equivalent of AcquireTasks, which only pushes updates to taskRoles.
	allHooks := wf.GetAllHooks()
	callHooks := allHooks.FilterCalls() // get the calls
	if len(callHooks) > 0 {
		for _, h := range callHooks {
			pr, ok := h.GetParentRole().(workflow.PublicUpdatable)
			if !ok {
				continue
			}
			go pr.UpdateStatus(task.ACTIVE)
		}
	}

	deploymentTimeout := acquireDeploymentTimeout(wf)

	wfStatus := wf.GetStatus()
	if wfStatus != task.ACTIVE {
		log.WithField("partition", env.Id().String()).
			Infof("waiting %s for workflow to become active", deploymentTimeout.String())
	WORKFLOW_ACTIVE_LOOP:
		for {
			select {
			case wfStatus = <-notifyStatus:
				log.WithField("status", wfStatus.String()).
					WithField("partition", env.Id().String()).
					Debug("workflow status change")
				if wfStatus == task.ACTIVE {
					break WORKFLOW_ACTIVE_LOOP
				} else if wfStatus == task.UNDEPLOYABLE {
					err = errors.New("workflow is undeployable")
					undeployableTaskRoles := make([]string, 0)
					workflow.LeafWalk(wf, func(role workflow.Role) {
						if roleStatus := role.GetStatus(); roleStatus == task.UNDEPLOYABLE {
							detector, ok := role.GetUserVars().Get("detector")
							if !ok {
								detector = ""
							}
							log.WithField("state", role.GetState().String()).
								WithField("status", roleStatus.String()).
								WithField("role", role.GetPath()).
								WithField("partition", env.Id().String()).
								WithField("timeout", deploymentTimeout).
								WithField("detector", detector).
								Error("role failed to deploy within timeout")
							undeployableTaskRoles = append(undeployableTaskRoles, role.GetPath())
						}
					})

					sort.Strings(undeployableTaskRoles)
					inactiveTaskRolesS := strings.Join(undeployableTaskRoles, ", ")

					err = fmt.Errorf("workflow deployment failed (one or more roles undeployable), aborting and cleaning up [undeployable roles: %s]", inactiveTaskRolesS)

					break WORKFLOW_ACTIVE_LOOP
				}
				continue

			case <-time.After(deploymentTimeout):
				wfStatus = wf.GetStatus()
				inactiveTaskRoles := make([]string, 0)
				undeployableTaskRoles := make([]string, 0)
				workflow.LeafWalk(wf, func(role workflow.Role) {
					if roleStatus := role.GetStatus(); roleStatus == task.UNDEPLOYABLE {
						detector, ok := role.GetUserVars().Get("detector")
						if !ok {
							detector = ""
						}
						log.WithField("state", role.GetState().String()).
							WithField("status", roleStatus.String()).
							WithField("role", role.GetPath()).
							WithField("partition", env.Id().String()).
							WithField("timeout", deploymentTimeout).
							WithField("detector", detector).
							Error("role failed to deploy within timeout")
						undeployableTaskRoles = append(undeployableTaskRoles, role.GetPath())
					} else if roleStatus != task.ACTIVE {
						detector, ok := role.GetUserVars().Get("detector")
						if !ok {
							detector = ""
						}
						log.WithField("state", role.GetState().String()).
							WithField("status", roleStatus.String()).
							WithField("role", role.GetPath()).
							WithField("partition", env.Id().String()).
							WithField("timeout", deploymentTimeout).
							WithField("detector", detector).
							Error("role failed to deploy within timeout")
						inactiveTaskRoles = append(inactiveTaskRoles, role.GetPath())
					}
				})

				sort.Strings(inactiveTaskRoles)
				sort.Strings(undeployableTaskRoles)
				inactiveTaskRolesS := strings.Join(inactiveTaskRoles, ", ")
				undeployableTaskRolesS := strings.Join(undeployableTaskRoles, ", ")

				err = fmt.Errorf("workflow deployment timed out (%s), aborting and cleaning up [%d undeployable roles: %s; %d inactive roles: %s]",
					deploymentTimeout.String(),
					len(undeployableTaskRoles),
					undeployableTaskRolesS,
					len(inactiveTaskRoles),
					inactiveTaskRolesS)
				break WORKFLOW_ACTIVE_LOOP
			// This is needed for when the workflow fails during the STAGING state(mesos status),mesos responds with the `REASON_COMMAND_EXECUTOR_FAILED`,
			// By listening to workflow state ERROR we can break the loop before reaching the timeout (1m30s), we can trigger the cleanup faster
			// in the CreateEnvironment (environment/manager.go) and the lock in the `envman` is reserved for a sorter period, which allows operations like
			// `environment list` to be done almost immediately after mesos informs with TASK_FAILED.
			case wfState := <-notifyState:
				if wfState == sm.ERROR {
					failedRoles := make([]string, 0)
					workflow.LeafWalk(wf, func(role workflow.Role) {
						if st := role.GetState(); st == sm.ERROR {
							detector, ok := role.GetUserVars().Get("detector")
							if !ok {
								detector = ""
							}

							log.WithField("state", st).
								WithField("role", role.GetPath()).
								WithField("partition", env.Id().String()).
								WithField("detector", detector).
								WithField("level", infologger.IL_Trace).
								Error("environment reached invalid state")
							failedRoles = append(failedRoles, role.GetPath())
						}
					})
					log.WithField("state", wfState.String()).
						WithField("partition", env.Id().String()).
						Debug("workflow state change")
					err = fmt.Errorf("workflow deployment failed, aborting and cleaning up [failed roles: %s]", strings.Join(failedRoles, ", "))
					break WORKFLOW_ACTIVE_LOOP
				}
			}
		}
	}

	if err != nil {
		log.WithError(err).
			WithField("partition", env.Id().String()).
			Error("deployment error")
		return
	}

	env.sendEnvironmentEvent(&event.EnvironmentEvent{EnvironmentID: env.Id().String(), State: "DEPLOYED"})
	return
}

func acquireDeploymentTimeout(wf workflow.Role) time.Duration {
	envId := wf.GetEnvironmentId()

	var (
		cws map[string]string
		err error
	)
	deploymentTimeout := 90 * time.Second
	cws, err = wf.ConsolidatedVarStack()
	if err != nil {
		log.WithField("partition", envId).
			WithError(err).
			Warnf("could not get consolidated variable stack, deploy_timeout defaulting to %s", deploymentTimeout.String())
		err = nil
	} else {
		deploymentTimeoutS, ok := cws["deploy_timeout"]
		if ok {
			var timeout time.Duration
			timeout, err = time.ParseDuration(deploymentTimeoutS)
			if err != nil {
				log.WithField("partition", envId).
					WithError(err).
					Warnf("variable deploy_timeout (%s) could not be parsed, defaulting to %s", deploymentTimeoutS, deploymentTimeout.String())
				err = nil
			} else {
				deploymentTimeout = timeout
			}
		}
	}
	return deploymentTimeout
}
