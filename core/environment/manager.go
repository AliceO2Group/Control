/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2017-2022 CERN and copyright holders of ALICE O².
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
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/AliceO2Group/Control/common"
	"github.com/AliceO2Group/Control/common/controlmode"
	"github.com/AliceO2Group/Control/common/event"
	"github.com/AliceO2Group/Control/common/event/topic"
	"github.com/AliceO2Group/Control/common/logger/infologger"
	evpb "github.com/AliceO2Group/Control/common/protos"
	"github.com/AliceO2Group/Control/common/system"
	"github.com/AliceO2Group/Control/common/utils"
	"github.com/AliceO2Group/Control/common/utils/uid"
	lhcevent "github.com/AliceO2Group/Control/core/integration/lhc/event"
	event2 "github.com/AliceO2Group/Control/core/integration/odc/event"
	"github.com/AliceO2Group/Control/core/task"
	"github.com/AliceO2Group/Control/core/task/sm"
	"github.com/AliceO2Group/Control/core/task/taskop"
	"github.com/AliceO2Group/Control/core/the"
	"github.com/AliceO2Group/Control/core/workflow"
	pb "github.com/AliceO2Group/Control/executor/protos"
	"github.com/sirupsen/logrus"
)

type Manager struct {
	mu                   sync.RWMutex
	m                    map[uid.ID]*Environment
	taskman              *task.Manager
	incomingEventCh      chan event.Event
	pendingTeardownsCh   map[uid.ID]chan *event.TasksReleasedEvent
	pendingStateChangeCh map[uid.ID]chan *event.TasksStateChangedEvent
}

var instance *Manager

func ManagerInstance() *Manager {
	return instance
}

func NewEnvManager(tm *task.Manager, incomingEventCh chan event.Event) *Manager {
	instance = &Manager{
		m:                    make(map[uid.ID]*Environment),
		taskman:              tm,
		incomingEventCh:      incomingEventCh,
		pendingTeardownsCh:   make(map[uid.ID]chan *event.TasksReleasedEvent),
		pendingStateChangeCh: make(map[uid.ID]chan *event.TasksStateChangedEvent),
	}

	go func() {
		for {
			select {
			case incomingEvent := <-instance.incomingEventCh:
				switch typedEvent := incomingEvent.(type) {
				case event.DeviceEvent:
					instance.handleDeviceEvent(typedEvent)

				case event.IntegratedServiceEvent:
					instance.handleIntegratedServiceEvent(typedEvent)

				case *event.ExecutorFailedEvent:
					envIdsAffected := tm.HandleExecutorFailed(typedEvent)
					for envId := range envIdsAffected {
						env, err := instance.environment(envId)
						if err != nil {
							log.WithPrefix("scheduler").
								WithField("partition", envId.String()).
								WithField("executorId", typedEvent.GetId().Value).
								WithError(err).
								Error("cannot find environment for incoming executor failed event")
						}
						log.WithPrefix("scheduler").
							WithField("partition", envId.String()).
							WithField("executorId", typedEvent.GetId().Value).
							WithField("envState", env.CurrentState()).
							Debug("received executor failed event")
					}

				case *event.AgentFailedEvent:
					envIdsAffected := tm.HandleAgentFailed(typedEvent)
					for envId := range envIdsAffected {
						env, err := instance.environment(envId)
						if err != nil {
							log.WithPrefix("scheduler").
								WithField("partition", envId.String()).
								WithField("agentId", typedEvent.GetId().Value).
								WithError(err).
								Error("cannot find environment for incoming agent failed event")
						}
						log.WithPrefix("scheduler").
							WithField("partition", envId.String()).
							WithField("agentId", typedEvent.GetId().Value).
							WithField("envState", env.CurrentState()).
							Debug("received agent failed event")
					}

				case *event.TasksReleasedEvent:
					// If we got a TasksReleasedEvent, it must be matched with a pending
					// environment teardown.

					instance.mu.RLock()
					thisEnvCh, ok := instance.pendingTeardownsCh[typedEvent.GetEnvironmentId()]
					instance.mu.RUnlock()

					if ok {
						thisEnvCh <- typedEvent

						instance.mu.Lock()
						close(thisEnvCh)
						delete(instance.pendingTeardownsCh, typedEvent.GetEnvironmentId())
						instance.mu.Unlock()

					} else {
						// If there is no pending environment teardown, it means that the released task stopped
						// unexpectedly. In that case, the environment should get torn-down only if the task
						// is critical.
						releaseCriticalTask := false
						for _, v := range typedEvent.GetTaskIds() {
							if tm.GetTask(v) != nil {
								if tm.GetTask(v).GetTraits().Critical == true {
									//|| tm.GetTask(v).GetParent().GetTaskTraits().Critical == true
									releaseCriticalTask = true
								}
							}
						}
						if releaseCriticalTask {
							thisEnvCh <- typedEvent

							instance.mu.Lock()
							close(thisEnvCh)
							delete(instance.pendingTeardownsCh, typedEvent.GetEnvironmentId())
							instance.mu.Unlock()
						}
					}

				case *event.TasksStateChangedEvent:
					// If we got a TasksStateChangedEvent, it must be matched with a pending
					// environment transition.
					instance.mu.RLock()
					thisEnvCh, ok := instance.pendingStateChangeCh[typedEvent.GetEnvironmentId()]
					instance.mu.RUnlock()
					// If environment is not in state transition message is being propagated through task/manager
					if ok {
						thisEnvCh <- typedEvent
					}
				default:
					// noop
				}
			}
		}
	}()
	return instance
}

func (envs *Manager) NotifyIntegratedServiceEvent(event event.IntegratedServiceEvent) {
	envs.incomingEventCh <- event
}

func (envs *Manager) GetActiveDetectors() system.IDMap {
	envs.mu.RLock()
	defer envs.mu.RUnlock()

	response := make(system.IDMap)
	for _, env := range envs.m {
		if env.workflow == nil { // we can only query for detectors post-workflow-load
			continue
		}
		envDetectors := env.GetActiveDetectors()
		for det := range envDetectors {
			response[det] = struct{}{}
		}
	}
	return response
}

func (envs *Manager) CreateEnvironment(workflowPath string, userVars map[string]string, public bool, newId uid.ID, autoTransition bool) (resultEnvId uid.ID, resultErr error) {
	// Before we load the workflow, we get the list of currently active detectors. This query must be performed before
	// loading the workflow in order to compare the currently used detectors with the detectors required by the newly
	// created environment.
	alreadyActiveDetectors := envs.GetActiveDetectors()

	lastRequestUser := &evpb.User{}
	lastRequestUserJ, ok := userVars["last_request_user"]
	if ok {
		_ = json.Unmarshal([]byte(lastRequestUserJ), lastRequestUser)
	}

	// CreateEnvironment() is not transition from state machine, so we need to emit the same message as in TryTransition
	the.EventWriterWithTopic(topic.Environment).WriteEvent(&evpb.Ev_EnvironmentEvent{
		EnvironmentId:    newId.String(),
		State:            "PENDING",
		Transition:       "CREATE",
		TransitionStatus: evpb.OpStatus_STARTED,
		LastRequestUser:  lastRequestUser,
		Message:          "transition starting",
	})

	// report error of the CreateEnvironment() in the same way as in TryTransition
	defer func() {
		if resultErr != nil {
			the.EventWriterWithTopic(topic.Environment).WriteEvent(&evpb.Ev_EnvironmentEvent{
				EnvironmentId:    newId.String(),
				Error:            resultErr.Error(),
				LastRequestUser:  lastRequestUser,
				Message:          "transition error",
				State:            "PENDING",
				Transition:       "CREATE",
				TransitionStatus: evpb.OpStatus_DONE_ERROR,
			})
		}
	}()

	// in case of err==nil, env will be false unless user
	// set it to True which will be overwritten in server.go
	workflowPublicInfo, err := parseWorkflowPublicInfo(workflowPath)
	if err != nil {
		log.WithField("public info", public).
			WithField("workflow path", workflowPath).
			WithError(err).
			Warn("parse workflow public info failed.")

		resultEnvId = newId
		resultErr = fmt.Errorf("workflow public info parsing failed: %w", err)
		return
	}

	the.EventWriterWithTopic(topic.Environment).WriteEvent(&evpb.Ev_EnvironmentEvent{
		EnvironmentId:    newId.String(),
		State:            "PENDING",
		Transition:       "CREATE",
		TransitionStep:   "before_CREATE",
		TransitionStatus: evpb.OpStatus_STARTED,
		Message:          "instantiating",
		LastRequestUser:  lastRequestUser,
		WorkflowTemplateInfo: &evpb.WorkflowTemplateInfo{
			Path:        workflowPath,
			Public:      workflowPublicInfo.IsPublic,
			Name:        workflowPublicInfo.Name,
			Description: workflowPublicInfo.Description,
		},
	})

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

	cleanedUpTasks, runningTasks, err := envs.taskman.Cleanup()
	if err != nil {
		log.WithError(err).
			Warnf("pre-deployment cleanup failed, continuing anyway")
		err = nil
	} else {
		cleanedUpTaskIds, runningTaskIds := make([]string, 0), make([]string, 0)
		for _, t := range cleanedUpTasks {
			cleanedUpTaskIds = append(cleanedUpTaskIds, fmt.Sprintf("%s.%s#%s", t.GetHostname(), t.GetClassName(), t.GetTaskId()))
		}
		for _, t := range runningTasks {
			runningTaskIds = append(runningTaskIds, fmt.Sprintf("%s.%s#%s", t.GetHostname(), t.GetClassName(), t.GetTaskId()))
		}
		log.WithField("tasksCleanedUp", strings.Join(cleanedUpTaskIds, ", ")).
			WithField("level", infologger.IL_Devel).
			Debug("tasks cleaned up during pre-deployment cleanup")
		log.WithField("tasksStillRunning", strings.Join(runningTaskIds, ", ")).
			WithField("level", infologger.IL_Devel).
			Debug("tasks still running after pre-deployment cleanup")
		log.WithField("level", infologger.IL_Ops).
			Infof("pre-deployment cleanup completed (%d tasks cleaned up, %d tasks still running)", len(cleanedUpTasks), len(runningTasks))
	}

	var env *Environment
	env, err = newEnvironment(envUserVars, newId)
	if public {
		env.Public = true
	}

	gotEnvId := uid.NilID()
	if err == nil {
		if env != nil {
			gotEnvId = env.Id()
		} else {
			err = errors.New("newEnvironment returned nil environment")
			log.WithError(err).
				WithField("partition", newId.String()).
				Logf(logrus.FatalLevel, "environment creation failed")
		}
	}
	if err != nil || env == nil {
		if env == nil {
			err = errors.New("newEnvironment returned nil environment")
		}
		log.WithError(err).
			WithField("partition", gotEnvId.String()).
			Logf(logrus.FatalLevel, "environment creation failed")
		resultEnvId = gotEnvId
		resultErr = err
		return
	}

	log.WithFields(logrus.Fields{
		"workflow":  workflowPath,
		"partition": gotEnvId.String(),
	}).Info("creating new environment")

	env.Public = workflowPublicInfo.IsPublic
	env.name = workflowPublicInfo.Name
	env.Description = workflowPublicInfo.Description
	env.WorkflowPath = workflowPath

	the.EventWriterWithTopic(topic.Environment).WriteEvent(&evpb.Ev_EnvironmentEvent{
		EnvironmentId:        newId.String(),
		State:                "PENDING",
		Transition:           "CREATE",
		TransitionStep:       "before_CREATE",
		TransitionStatus:     evpb.OpStatus_ONGOING,
		Message:              "running hooks",
		LastRequestUser:      lastRequestUser,
		WorkflowTemplateInfo: env.GetWorkflowInfo(),
	})

	env.hookHandlerF = func(hooks task.Tasks) error {
		return envs.taskman.TriggerHooks(gotEnvId, hooks)
	}

	// Ensure the environment_id is available to all
	env.UserVars.Set("environment_id", env.id.String())

	the.EventWriterWithTopic(topic.Environment).WriteEvent(&evpb.Ev_EnvironmentEvent{
		EnvironmentId:        newId.String(),
		State:                "PENDING",
		Transition:           "CREATE",
		TransitionStep:       "CREATE",
		TransitionStatus:     evpb.OpStatus_ONGOING,
		Message:              "loading workflow",
		LastRequestUser:      lastRequestUser,
		WorkflowTemplateInfo: env.GetWorkflowInfo(),
	})

	// We load the workflow (includes template processing)
	env.workflow, err = envs.loadWorkflow(workflowPath, env.wfAdapter, workflowUserVars, env.BaseConfigStack)
	if err != nil {
		err = fmt.Errorf("cannot load workflow template: %w", err)

		resultEnvId = env.id
		resultErr = err
		return
	}

	// Ensure we provide a very defaulty `detectors` variable
	detectors, err := the.ConfSvc().GetDetectorsForHosts(env.GetFLPs())
	if err != nil {
		err = fmt.Errorf("cannot acquire detectors in loaded workflow template: %w", err)

		resultEnvId = env.id
		resultErr = err
		return
	}
	detectorsStr, err := SliceToJSONSlice(detectors)
	if err != nil {
		err = fmt.Errorf("cannot process detectors in loaded workflow template: %w", err)

		resultEnvId = env.id
		resultErr = err
		return
	}
	env.GlobalDefaults.Set("detectors", detectorsStr)

	log.WithFields(logrus.Fields{
		"partition": gotEnvId.String(),
	}).Infof("detectors in environment: %s", strings.Join(detectors, " "))

	// env.GetActiveDetectors() is valid starting now, so we can check for detector exclusion
	neededDetectors := env.GetActiveDetectors()
	for det := range neededDetectors {
		if _, contains := alreadyActiveDetectors[det]; contains {
			// required detector det is already active in some other environment

			resultEnvId = env.id
			resultErr = fmt.Errorf("detector %s is already in use", det.String())
			return
		}
	}

	cvs, _ := env.Workflow().ConsolidatedVarStack()
	the.EventWriterWithTopic(topic.Environment).WriteEvent(&evpb.Ev_EnvironmentEvent{
		EnvironmentId:        newId.String(),
		State:                env.CurrentState(),
		Transition:           "CREATE",
		TransitionStep:       "after_CREATE",
		TransitionStatus:     evpb.OpStatus_DONE_OK,
		Message:              "workflow loaded",
		Vars:                 cvs, // we push the full var stack of the root role in the workflow loaded event
		LastRequestUser:      lastRequestUser,
		WorkflowTemplateInfo: env.GetWorkflowInfo(),
	})

	the.EventWriterWithTopic(topic.Environment).WriteEvent(&evpb.Ev_EnvironmentEvent{
		EnvironmentId:    newId.String(),
		LastRequestUser:  lastRequestUser,
		Message:          "transition completed successfully",
		State:            env.CurrentState(),
		Transition:       "CREATE",
		TransitionStatus: evpb.OpStatus_DONE_OK,
	})

	log.WithField("method", "CreateEnvironment").
		WithField("level", infologger.IL_Devel).
		Debug("envman write lock")
	envs.mu.Lock()
	envs.m[env.id] = env
	envs.pendingStateChangeCh[env.id] = env.stateChangedCh
	envs.mu.Unlock()
	log.WithField("method", "CreateEnvironment").
		WithField("level", infologger.IL_Devel).
		Debug("envman write unlock")

	err = env.TryTransition(NewDeployTransition(
		envs.taskman,
		nil, // roles,
		nil),
	)

	if err == nil {
		err = env.TryTransition(NewConfigureTransition(
			envs.taskman),
		)
	}

	if err == nil {
		// CONFIGURE transition successful!
		env.subscribeToWfState(envs.taskman)

		if autoTransition {
			// We now return the configured environment from CreateEnvironment, but if autoTransition is set to true
			// then the auto-transitioner starts here.
			// Sequence:
			//   * CONFIGURED×START_ACTIVITY→RUNNING
			//   * wait for RUNNING to finish as required by tasks (RUNNING×STOP_ACTIVITY→CONFIGURED)
			//   * safe DESTROY path
			go func() {
				defer env.unsubscribeFromWfState()

				goErrorKillDestroy := func(op string) {
					envState := env.CurrentState()
					log.WithField("state", envState).
						WithField("partition", env.Id().String()).
						WithError(err).
						Warnf("auto-transitioning environment failed %s, cleanup in progress", op)

					the.EventWriterWithTopic(topic.Environment).WriteEvent(
						NewEnvGoErrorEvent(env, fmt.Sprintf("%s failed: %v", op, err)),
					)
					err := env.TryTransition(NewGoErrorTransition(
						envs.taskman),
					)
					if err != nil {
						log.WithField("partition", env.Id().String()).
							WithField("state", envState).
							Debug("could not transition failed auto-transitioning environment to ERROR, cleanup in progress")
						env.setState("ERROR")
					}

					envTasks := env.Workflow().GetTasks()
					// TeardownEnvironment manages the envs.mu internally
					_ = envs.TeardownEnvironment(env.Id(), true /*force*/)

					killedTasks, _, rlsErr := envs.taskman.KillTasks(envTasks.GetTaskIds())
					if rlsErr != nil {
						log.WithError(rlsErr).Warn("task teardown error")
					}
					log.WithFields(logrus.Fields{
						"killedCount":  len(killedTasks),
						"lastEnvState": envState,
						"level":        infologger.IL_Support,
						"partition":    env.Id().String(),
					}).
						Infof("auto-environment failed at %s, tasks were cleaned up", op)
					log.WithField("partition", env.Id().String()).Info("environment teardown complete")
				}

				// now we have the environment we should transition to start
				trans := NewStartActivityTransition(envs.taskman)
				if trans == nil {
					goErrorKillDestroy("transition START_ACTIVITY")
					return
				}

				err = env.TryTransition(trans)
				if err != nil {
					goErrorKillDestroy("transition START_ACTIVITY")
					return
				}

				for {
					// we know we performed START_ACTIVITY, so we poll at 1Hz for the run to finish
					time.Sleep(1 * time.Second)

					if env == nil {
						// must've died during the loop
						return
					}

					envState := env.CurrentState()
					switch envState {
					case "CONFIGURED":
						// RUN finished so we can reset and delete the environment
						err = env.TryTransition(NewResetTransition(envs.taskman))
						if err != nil {
							goErrorKillDestroy("transition RESET")
							return
						}

						err = envs.TeardownEnvironment(env.id, false)
						if err != nil {
							goErrorKillDestroy("teardown")
							return
						}
						tasksForEnv := env.Workflow().GetTasks().GetTaskIds()
						_, _, err = envs.taskman.KillTasks(tasksForEnv)
						if err != nil {
							return
						}

						return
					case "ERROR":
						fallthrough
					case "STANDBY":
						fallthrough
					case "DEPLOYED":
						goErrorKillDestroy("transition STOP_ACTIVITY")
						return
					case "MIXED":
						continue
					case "":
						continue
					}
				}
			}()
		}

		resultEnvId = env.id
		resultErr = err
		return
	}

	// Deployment/configuration failure code path starts here

	envState := env.CurrentState()
	log.WithField("partition", env.Id().String()).
		Errorf("environment deployment and configuration failed (%s)", workflowPath)
	log.WithField("state", envState).
		WithField("partition", env.Id().String()).
		WithError(err).
		WithField("level", infologger.IL_Devel).
		Error("environment deployment and configuration error, cleanup in progress")

	the.EventWriterWithTopic(topic.Environment).WriteEvent(
		NewEnvGoErrorEvent(env, fmt.Sprintf("deployment or configuration failed: %v", err)),
	)
	errTxErr := env.TryTransition(NewGoErrorTransition(
		envs.taskman),
	)
	if errTxErr != nil {
		log.WithField("partition", env.Id().String()).
			WithField("state", envState).
			WithError(errTxErr).
			Debug("could not transition to ERROR after failed deployment/configuration, cleanup in progress")
	}
	envTasks := env.Workflow().GetTasks()
	// TeardownEnvironment manages the envs.mu internally
	// We do not get the error here cause it overwrites the failed deployment error
	// with <nil> which results to server.go to report back
	// cannot get newly created environment: no environment with id <env id>
	_ = envs.TeardownEnvironment(env.Id(), true /*force*/)

	killedTasks, _, rlsErr := envs.taskman.KillTasks(envTasks.GetTaskIds())
	if rlsErr != nil {
		log.WithError(rlsErr).Warn("task teardown error")
	}
	log.WithFields(logrus.Fields{
		"killedCount":  len(killedTasks),
		"lastEnvState": envState,
		"level":        infologger.IL_Support,
		"partition":    env.Id().String(),
	}).
		Info("environment deployment failed, tasks were cleaned up")
	log.WithField("partition", env.Id().String()).Info("environment teardown complete")

	return env.id, err
}

func (envs *Manager) TeardownEnvironment(environmentId uid.ID, force bool) error {
	log.WithFields(logrus.Fields{
		"partition": environmentId.String(),
	}).Info("tearing down environment")

	envs.mu.RLock()
	env, err := envs.environment(environmentId)
	envs.mu.RUnlock()

	if err != nil {
		return err
	}

	if !env.transitionMutex.TryLock() {
		log.WithField("partition", environmentId.String()).
			Warnf("environment teardown attempt delayed: transition '%s' in progress. waiting for completion or failure", env.currentTransition)
		env.transitionMutex.Lock()
		log.WithField("level", infologger.IL_Support).
			WithField("partition", environmentId.String()).
			Infof("environment teardown attempt resumed")
	}
	defer env.transitionMutex.Unlock()

	if env.CurrentState() == "DONE" {
		return errors.New("attempting to teardown an environment which is already in DONE, doing nothing")
	}
	if env.CurrentState() != "STANDBY" && env.CurrentState() != "DEPLOYED" && !force {
		return errors.New(fmt.Sprintf("cannot teardown environment in state %s", env.CurrentState()))
	}

	the.EventWriterWithTopic(topic.Environment).WriteEvent(&evpb.Ev_EnvironmentEvent{
		EnvironmentId:        environmentId.String(),
		State:                env.CurrentState(),
		Transition:           "DESTROY",
		TransitionStep:       "before_DESTROY",
		TransitionStatus:     evpb.OpStatus_STARTED,
		Message:              "workflow teardown started",
		LastRequestUser:      env.GetLastRequestUser(),
		WorkflowTemplateInfo: env.GetWorkflowInfo(),
	})

	env.Mu.Lock()
	env.currentTransition = "DESTROY"
	env.Mu.Unlock()

	the.EventWriterWithTopic(topic.Environment).WriteEvent(&evpb.Ev_EnvironmentEvent{
		EnvironmentId:        environmentId.String(),
		State:                env.CurrentState(),
		Transition:           "DESTROY",
		TransitionStep:       "leave_" + env.CurrentState(),
		TransitionStatus:     evpb.OpStatus_ONGOING,
		Message:              "workflow teardown ongoing",
		LastRequestUser:      env.GetLastRequestUser(),
		WorkflowTemplateInfo: env.GetWorkflowInfo(),
	})

	err = env.handleAllHooks(env.Workflow(), "leave_"+env.CurrentState())
	if err != nil {
		log.WithFields(logrus.Fields{
			"partition": environmentId.String(),
		}).Error(fmt.Errorf("could not handle hooks for the trigger leave_%s, error: %w", env.CurrentState(), err))
	}

	if env.CurrentState() == "RUNNING" {
		endTime, ok := env.workflow.GetUserVars().Get("run_end_time_ms")
		if ok && endTime == "" {
			runEndTime := time.Now()
			runEndTimeS := strconv.FormatInt(runEndTime.UnixMilli(), 10)
			env.workflow.SetRuntimeVar("run_end_time_ms", runEndTimeS)

			the.EventWriterWithTopic(topic.Run).WriteEventWithTimestamp(&evpb.Ev_RunEvent{
				EnvironmentId:    environmentId.String(),
				RunNumber:        env.GetCurrentRunNumber(),
				State:            env.Sm.Current(),
				Error:            "",
				Transition:       "TEARDOWN",
				TransitionStatus: evpb.OpStatus_STARTED,
			}, runEndTime)
		} else {
			log.WithField("partition", environmentId.String()).
				Debug("O2 End time already set before DESTROY")
		}

		endCompletionTime, ok := env.workflow.GetUserVars().Get("run_end_completion_time_ms")
		if ok && endCompletionTime == "" {
			runEndCompletionTime := time.Now()
			runEndCompletionTimeS := strconv.FormatInt(runEndCompletionTime.UnixMilli(), 10)
			env.workflow.SetRuntimeVar("run_end_completion_time_ms", runEndCompletionTimeS)

			the.EventWriterWithTopic(topic.Run).WriteEventWithTimestamp(&evpb.Ev_RunEvent{
				EnvironmentId:    environmentId.String(),
				RunNumber:        env.GetCurrentRunNumber(),
				State:            env.Sm.Current(),
				Error:            "",
				Transition:       "TEARDOWN",
				TransitionStatus: evpb.OpStatus_STARTED,
			}, runEndCompletionTime)
		} else {
			log.WithField("partition", environmentId.String()).
				Debug("O2 End Completion time already set before DESTROY")
		}
	}

	tasksToRelease := env.Workflow().GetTasks()

	// we gather all DESTROY/after_DESTROY hooks, as these require special treatment
	hooksMapForDestroy := env.Workflow().GetHooksMapForTrigger("DESTROY")
	for k, v := range env.Workflow().GetHooksMapForTrigger("after_DESTROY") {
		hooksMapForDestroy[k] = v
	}

	allWeights := hooksMapForDestroy.GetWeights()

	// for each weight within DESTROY
	//     for each found DESTROY hook,
	//         for each of *all* tasks last-to-first
	//             if the pointed task is one of the known cleanup hooks, remove it
	for _, weight := range allWeights {
		hooksForWeight, ok := hooksMapForDestroy[weight]
		if ok {
			for _, hook := range hooksForWeight {
				for i := len(tasksToRelease) - 1; i >= 0; i-- {
					if hook == tasksToRelease[i] {
						tasksToRelease = append(tasksToRelease[:i], tasksToRelease[i+1:]...)
					}
				}
			}
		}
	}

	the.EventWriterWithTopic(topic.Environment).WriteEvent(&evpb.Ev_EnvironmentEvent{
		EnvironmentId:        environmentId.String(),
		State:                env.CurrentState(),
		Transition:           "DESTROY",
		TransitionStep:       "DESTROY",
		TransitionStatus:     evpb.OpStatus_ONGOING,
		Message:              "releasing tasks",
		LastRequestUser:      env.GetLastRequestUser(),
		WorkflowTemplateInfo: env.GetWorkflowInfo(),
	})

	log.WithField("method", "TeardownEnvironment").
		WithField("level", infologger.IL_Devel).
		Debug("envman write lock")
	envs.mu.Lock()
	// we kill all tasks that aren't cleanup hooks
	taskmanMessage := task.NewEnvironmentMessage(taskop.ReleaseTasks, environmentId, tasksToRelease, nil)
	// close state channel
	if ch := envs.pendingStateChangeCh[environmentId]; ch != nil {
		close(envs.pendingStateChangeCh[environmentId])
	}
	delete(envs.pendingStateChangeCh, environmentId)
	envs.mu.Unlock()
	log.WithField("method", "TeardownEnvironment").
		WithField("level", infologger.IL_Devel).
		Debug("envman write lock")

	// We set all callRoles to INACTIVE right now, because there's no task activation for them.
	// This is the callRole equivalent of AcquireTasks, which only pushes updates to taskRoles.
	allHooks := env.Workflow().GetAllHooks()
	callHooks := allHooks.FilterCalls() // get the calls
	if len(callHooks) > 0 {
		for _, h := range callHooks {
			pr, ok := h.GetParentRole().(workflow.PublicUpdatable)
			if !ok {
				continue
			}
			go pr.UpdateStatus(task.INACTIVE)
		}
	}

	pendingCh := make(chan *event.TasksReleasedEvent)
	log.WithField("method", "TeardownEnvironment").
		WithField("level", infologger.IL_Devel).
		Debug("envman write lock")
	envs.mu.Lock()
	envs.pendingTeardownsCh[environmentId] = pendingCh
	envs.mu.Unlock()
	log.WithField("method", "TeardownEnvironment").
		WithField("level", infologger.IL_Devel).
		Debug("envman write lock")

	envs.taskman.MessageChannel <- taskmanMessage

	incomingEv := <-pendingCh

	// If some tasks failed to release
	if taskReleaseErrors := incomingEv.GetTaskReleaseErrors(); len(taskReleaseErrors) > 0 {
		for taskId, err := range taskReleaseErrors {
			log.WithFields(logrus.Fields{
				"taskId":    taskId,
				"partition": environmentId,
			}).
				WithError(err).
				Warn("task failed to release")
		}
		err = fmt.Errorf("%d tasks failed to release for environment %s",
			len(taskReleaseErrors), environmentId)

		the.EventWriterWithTopic(topic.Environment).WriteEvent(&evpb.Ev_EnvironmentEvent{
			EnvironmentId:        environmentId.String(),
			State:                "DONE",
			Transition:           "DESTROY",
			TransitionStep:       "after_DESTROY",
			TransitionStatus:     evpb.OpStatus_DONE_ERROR,
			Message:              "environment teardown finished with error",
			Error:                err.Error(),
			LastRequestUser:      env.GetLastRequestUser(),
			WorkflowTemplateInfo: env.GetWorkflowInfo(),
		})

		return err
	}

	the.EventWriterWithTopic(topic.Environment).WriteEvent(&evpb.Ev_EnvironmentEvent{
		EnvironmentId:        environmentId.String(),
		State:                env.CurrentState(),
		Transition:           "DESTROY",
		TransitionStep:       "after_DESTROY",
		TransitionStatus:     evpb.OpStatus_ONGOING,
		Message:              "running DESTROY hooks",
		LastRequestUser:      env.GetLastRequestUser(),
		WorkflowTemplateInfo: env.GetWorkflowInfo(),
	})

	// we trigger all cleanup hooks, first calls, then tasks immediately after
	for _, weight := range allWeights {
		hooksForWeight, ok := hooksMapForDestroy[weight]
		if ok {
			hooksForWeight.FilterCalls().CallAll()

			// calls done, we start the task hooks...
			cleanupTaskHooks := hooksForWeight.FilterTasks()

			// ...but only if their parent role is still ACTIVE (i.e. not killed or executor failed)
			cleanupTaskHooks = cleanupTaskHooks.Filtered(func(t *task.Task) bool {
				if pr, prOk := t.GetParentRole().(workflow.Role); prOk {
					return pr.GetStatus() == task.ACTIVE
				}
				return false
			})
			err = envs.taskman.TriggerHooks(environmentId, cleanupTaskHooks)
			if err != nil {
				log.WithField("partition", environmentId.String()).
					WithError(err).
					Warn("environment post-destroy hooks failed")
			}

			// and then we kill them too
			taskmanMessage = task.NewEnvironmentMessage(taskop.ReleaseTasks, environmentId, cleanupTaskHooks, nil)
		}
	}

	envs.cancelCallsPendingAwait(env)

	// we remake the pending teardown channel too, because each completed TasksReleasedEvent
	// automatically closes it
	pendingCh = make(chan *event.TasksReleasedEvent)
	log.WithField("method", "TeardownEnvironment").
		WithField("level", infologger.IL_Devel).
		Debug("envman write lock")
	envs.mu.Lock()
	envs.pendingTeardownsCh[environmentId] = pendingCh
	envs.mu.Unlock()
	log.WithField("method", "TeardownEnvironment").
		WithField("level", infologger.IL_Devel).
		Debug("envman write lock")

	envs.taskman.MessageChannel <- taskmanMessage

	incomingEv = <-pendingCh

	// If some cleanup hooks failed to release
	if taskReleaseErrors := incomingEv.GetTaskReleaseErrors(); len(taskReleaseErrors) > 0 {
		for taskId, err := range taskReleaseErrors {
			log.WithFields(logrus.Fields{
				"taskId":    taskId,
				"partition": environmentId,
			}).
				WithError(err).
				Warn("task failed to release")
		}
		err = fmt.Errorf("%d tasks failed to release for environment %s",
			len(taskReleaseErrors), environmentId)

		the.EventWriterWithTopic(topic.Environment).WriteEvent(&evpb.Ev_EnvironmentEvent{
			EnvironmentId:        environmentId.String(),
			State:                "DONE",
			Transition:           "DESTROY",
			TransitionStep:       "after_DESTROY",
			TransitionStatus:     evpb.OpStatus_DONE_ERROR,
			Message:              "environment teardown finished with error",
			Error:                err.Error(),
			LastRequestUser:      env.GetLastRequestUser(),
			WorkflowTemplateInfo: env.GetWorkflowInfo(),
		})

		return err
	}

	env.setState("DONE")
	env.sendEnvironmentEvent(&event.EnvironmentEvent{EnvironmentID: env.Id().String(), Message: "teardown complete", State: "DONE"})

	log.WithField("method", "TeardownEnvironment").
		WithField("level", infologger.IL_Devel).
		Debug("envman write lock")
	envs.mu.Lock()
	defer envs.mu.Unlock()
	delete(envs.m, environmentId)
	env.unsubscribeFromWfState()
	log.WithField("method", "TeardownEnvironment").
		WithField("level", infologger.IL_Devel).
		Debug("envman write lock")

	the.EventWriterWithTopic(topic.Environment).WriteEvent(&evpb.Ev_EnvironmentEvent{
		EnvironmentId:        environmentId.String(),
		State:                "DONE",
		Transition:           "DESTROY",
		TransitionStep:       "after_DESTROY",
		TransitionStatus:     evpb.OpStatus_DONE_OK,
		Message:              "environment teardown complete",
		LastRequestUser:      env.GetLastRequestUser(),
		WorkflowTemplateInfo: env.GetWorkflowInfo(),
	})

	log.WithFields(logrus.Fields{
		"partition":      environmentId.String(),
		infologger.Level: infologger.IL_Ops,
	}).Info("environment teardown complete")
	return err
}

func (envs *Manager) cancelCallsPendingAwait(env *Environment) {
	// unblock all calls which are stuck waiting for an await trigger which never happened
	if env == nil {
		return
	}
	for _, callMapForAwait := range env.callsPendingAwait {
		for _, callsForWeight := range callMapForAwait {
			for _, call := range callsForWeight {
				if call != nil {
					call.Cancel()
				}
			}
		}
	}
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
	return envs.environment(environmentId)
}

func (envs *Manager) environment(environmentId uid.ID) (env *Environment, err error) {
	if len(environmentId) == 0 { // invalid id
		return nil, fmt.Errorf("empty env ID")
	}
	envs.mu.RLock()
	defer envs.mu.RUnlock()
	env, ok := envs.m[environmentId]
	if !ok {
		err = errors.New(fmt.Sprintf("no environment with id %s", environmentId))
	}
	return
}

func (envs *Manager) loadWorkflow(workflowPath string, parent workflow.Updatable, workflowUserVars map[string]string, baseConfigStack map[string]string) (root workflow.Role, err error) {
	if strings.Contains(workflowPath, "://") {
		return nil, errors.New("workflow loading from file not implemented yet")
	}
	return workflow.Load(workflowPath, parent, envs.taskman, workflowUserVars, baseConfigStack)
}

func (envs *Manager) handleIntegratedServiceEvent(evt event.IntegratedServiceEvent) {
	if evt == nil {
		log.Error("cannot handle null IntegratedServiceEvent")
		return
	}

	// for now we only handle ODC events
	if evt.GetServiceName() == "ODC" {
		if odcEvent, ok := evt.(*event2.OdcPartitionStateChangeEvent); ok && odcEvent.GetState() == "ERROR" {
			envId := odcEvent.GetEnvironmentId()
			env, err := envs.environment(envId)
			if err != nil {
				log.WithPrefix("scheduler").
					WithField("partition", envId.String()).
					WithField("odcState", odcEvent.GetState()).
					WithError(err).
					Error("cannot find environment for OdcPartitionStateChangeEvent")
			} else {
				log.WithPrefix("scheduler").
					WithField("partition", envId.String()).
					WithField("odcState", odcEvent.GetState()).
					WithField("envState", env.CurrentState()).
					Debug("received ODC_PARTITION_STATE_CHANGE event from ODC, trying to stop the run")
				if env.CurrentState() == "RUNNING" {
					go func() {
						log.WithPrefix("scheduler").
							WithField("partition", envId.String()).
							Log(logrus.FatalLevel, "ODC partition state changed to ERROR, the current run will be stopped")

						err = env.TryTransition(NewStopActivityTransition(envs.taskman))
						if err != nil {
							log.WithPrefix("scheduler").
								WithField("partition", envId.String()).
								WithError(err).
								Error("cannot stop run after ODC_PARTITION_STATE_CHANGE ERROR event")
						}

						if env.CurrentState() != "ERROR" {
							the.EventWriterWithTopic(topic.Environment).WriteEvent(
								NewEnvGoErrorEvent(env, "ODC partition went to ERROR during RUNNING"),
							)
							err = env.TryTransition(NewGoErrorTransition(envs.taskman))
							if err != nil {
								log.WithPrefix("scheduler").
									WithField("partition", envId.String()).
									WithError(err).
									Error("environment GO_ERROR transition failed after ODC_PARTITION_STATE_CHANGE ERROR event")
								env.setState("ERROR")
							}
						}
					}()
				}
			}
		}
	} else if evt.GetServiceName() == "LHC" {
		envs.handleLhcEvents(evt)
	}
}

func (envs *Manager) handleLhcEvents(evt event.IntegratedServiceEvent) {

	lhcEvent, ok := evt.(*lhcevent.LhcStateChangeEvent)
	if !ok {
		return
	}

	// stop all relevant environments when beams are dumped
	beamMode := lhcEvent.GetBeamInfo().BeamMode
	beamsDumped := beamMode == evpb.BeamMode_BEAM_DUMP || beamMode == evpb.BeamMode_LOST_BEAMS || beamMode == evpb.BeamMode_NO_BEAM
	if !beamsDumped {
		return
	}

	for envId, env := range envs.m {
		shouldStopAtBeamDump, _ := strconv.ParseBool(env.GetKV("", "stop_at_beam_dump"))
		if shouldStopAtBeamDump && env.CurrentState() == "RUNNING" {
			if currentTransition := env.CurrentTransition(); currentTransition != "" {
				log.WithPrefix("scheduler").
					WithField(infologger.Level, infologger.IL_Support).
					WithField("partition", envId.String()).
					WithField("run", env.currentRunNumber).
					Infof("run was supposed to be stopped at beam dump, but transition '%s' is already in progress, skipping (probably the operator was faster)", currentTransition)
				continue
			}

			go func(env *Environment) {
				log.WithPrefix("scheduler").
					WithField(infologger.Level, infologger.IL_Ops).
					WithField("partition", envId.String()).
					WithField("run", env.currentRunNumber).
					Info("stopping the run due to beam dump")

				env.SetLastRequestUser(evpb.SpecialUser(evpb.SpecialUserId_LHC))
				err := env.TryTransition(NewStopActivityTransition(envs.taskman))
				if err != nil {
					log.WithPrefix("scheduler").
						WithField("partition", envId.String()).
						WithField("run", env.currentRunNumber).
						WithError(err).
						Error("could not stop the run upon beam dump")

					if env.CurrentState() != "ERROR" {
						err = env.TryTransition(NewGoErrorTransition(envs.taskman))
						if err != nil {
							log.WithPrefix("scheduler").
								WithField("partition", envId.String()).
								WithField("run", env.currentRunNumber).
								WithError(err).
								Error("environment GO_ERROR transition failed after a beam dump event, forcing")
							env.setState("ERROR")
						}
					}
				}
			}(env)
		}
	}
}

func (envs *Manager) handleDeviceEvent(evt event.DeviceEvent) {
	if evt == nil {
		log.Error("cannot handle null DeviceEvent")
		return
	}

	envId := common.GetEnvironmentIdFromLabelerType(evt)

	switch evt.GetType() {
	case pb.DeviceEventType_BASIC_TASK_TERMINATED:
		if btt, ok := evt.(*event.BasicTaskTerminated); ok {
			log.WithPrefix("scheduler").
				WithFields(logrus.Fields{
					"exitCode":        btt.ExitCode,
					"stdout":          btt.Stdout,
					"stderr":          btt.Stderr,
					"finalMesosState": btt.FinalMesosState.String(),
					"level":           infologger.IL_Devel,
					"partition":       envId.String(),
				}).
				Debug("basic task terminated")

			// Propagate this information to the task/role
			taskId := evt.GetOrigin().TaskId
			t := envs.taskman.GetTask(taskId.Value)
			isHook := false
			if t != nil {
				t.SendEvent(&event.TaskEvent{Name: t.GetName(), TaskID: taskId.Value, Status: btt.FinalMesosState.String(), Hostname: t.GetHostname(), ClassName: t.GetClassName()})
				if parentRole, ok := t.GetParentRole().(workflow.Role); ok {
					parentRole.SetRuntimeVars(map[string]string{
						"taskResult.exitCode":    strconv.Itoa(btt.ExitCode),
						"taskResult.stdout":      btt.Stdout,
						"taskResult.stderr":      btt.Stderr,
						"taskResult.finalStatus": btt.FinalMesosState.String(),
						"taskResult.timestamp":   utils.NewUnixTimestamp(),
					})

					// If it's an update following a HOOK execution
					if t.GetControlMode() == controlmode.HOOK {
						isHook = true
						env, err := envs.environment(t.GetEnvironmentId())
						if err != nil {
							log.WithPrefix("scheduler").
								WithField(infologger.Level, infologger.IL_Devel).
								WithError(err).
								Error("cannot find environment for DeviceEvent")
						}
						if env != nil {
							env.NotifyEvent(evt)
						}
					}
				} else {
					// Task has no parent role - this can happen during environment teardown
					// when tasks are released before termination events are processed
					log.WithPrefix("scheduler").
						WithField("partition", envId.String()).
						WithField("taskId", taskId.Value).
						WithField(infologger.Level, infologger.IL_Devel).
						Debug("DeviceEvent BASIC_TASK_TERMINATED received for task with no parent role, likely due to environment teardown")
				}
			} else {
				log.WithPrefix("scheduler").
					WithField("partition", envId.String()).
					WithField(infologger.Level, infologger.IL_Devel).
					Debug("cannot find task for DeviceEvent BASIC_TASK_TERMINATED")
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
			log.WithPrefix("scheduler").
				WithField("partition", envId.String()).
				WithField("taskId", taskId.Value).
				WithField(infologger.Level, infologger.IL_Devel).
				Debug("cannot find task for DeviceEvent END_OF_STREAM")
			return
		}
		env, err := envs.environment(t.GetEnvironmentId())
		if err != nil {
			log.WithPrefix("scheduler").
				WithField("partition", envId.String()).
				WithField("taskId", taskId.Value).
				WithField(infologger.Level, infologger.IL_Devel).
				WithError(err).
				Error("cannot find environment for DeviceEvent")
		} else {
			log.WithPrefix("scheduler").
				WithField("partition", envId.String()).
				WithField("taskId", taskId.Value).
				WithField("role", t.GetParent().GetName()).
				WithField("envState", env.CurrentState()).
				Debug("received END_OF_STREAM event from task, trying to stop the run")
			if env.CurrentState() == "RUNNING" {
				t.SetSafeToStop(true) // we mark this specific task as ok to STOP
				go func() {
					if env.IsSafeToStop() { // but then we ask the env whether *all* of them are
						err = env.TryTransition(NewStopActivityTransition(envs.taskman))
						if err != nil {
							log.WithPrefix("scheduler").
								WithField("partition", envId.String()).
								WithError(err).
								Error("cannot stop run after END_OF_STREAM event")
						}
					}
				}()
			}
		}

	case pb.DeviceEventType_TASK_INTERNAL_ERROR:
		// a task has already internally transitioned to ERROR state
		taskId := evt.GetOrigin().TaskId
		t := envs.taskman.GetTask(taskId.Value)
		if t == nil {
			log.WithPrefix("scheduler").
				WithField("partition", envId.String()).
				WithField("taskId", taskId.Value).
				Debug("cannot find task for DeviceEvent TASK_INTERNAL_ERROR")
			return
		}
		env, err := envs.environment(t.GetEnvironmentId())
		if err != nil {
			log.WithPrefix("scheduler").
				WithField("partition", envId.String()).
				WithField("taskId", taskId.Value).
				WithField(infologger.Level, infologger.IL_Devel).
				WithError(err).
				Error("cannot find environment for DeviceEvent")
		} else {
			log.WithPrefix("scheduler").
				WithField("partition", envId.String()).
				WithField("taskId", taskId.Value).
				WithField("taskRole", t.GetParentRolePath()).
				WithField("envState", env.CurrentState()).
				WithField(infologger.Level, infologger.IL_Support).
				Debug("received TASK_INTERNAL_ERROR event from task, trying to stop the run")
			if env.CurrentState() == "RUNNING" {
				go func() {
					t.GetParent().UpdateState(sm.ERROR)
					err = env.TryTransition(NewStopActivityTransition(envs.taskman))
					if err != nil {
						log.WithPrefix("scheduler").
							WithField("partition", envId.String()).
							WithError(err).
							Error("cannot stop run after END_OF_STREAM event")
					}
				}()
			}
		}

	}
}

// FIXME: this function should be deduplicated with CreateEnvironment so detector resource matching works correctly
func (envs *Manager) CreateAutoEnvironment(workflowPath string, userVars map[string]string, newId uid.ID, sub Subscription) {
	envUserVars := make(map[string]string)
	workflowUserVars := make(map[string]string)
	for k, v := range userVars {
		if strings.ContainsRune(k, task.TARGET_SEPARATOR_RUNE) {
			workflowUserVars[k] = v
		} else {
			envUserVars[k] = v
		}
	}

	lastRequestUser := &evpb.User{}
	lastRequestUserJ, ok := userVars["last_request_user"]
	if ok {
		_ = json.Unmarshal([]byte(lastRequestUserJ), lastRequestUser)
	}

	// in case of err==nil, env will be false unless user
	// set it to True which will be overwritten in server.go
	workflowPublicInfo, err := parseWorkflowPublicInfo(workflowPath)
	if err != nil {
		log.WithField("public info", workflowPublicInfo.IsPublic).
			WithField("environment", newId.String()).
			WithError(err).
			Warn("parse workflow public info failed.")
	}

	the.EventWriterWithTopic(topic.Environment).WriteEvent(&evpb.Ev_EnvironmentEvent{
		EnvironmentId:    newId.String(),
		State:            "PENDING",
		Transition:       "CREATE",
		TransitionStep:   "before_CREATE",
		TransitionStatus: evpb.OpStatus_STARTED,
		Message:          "instantiating",
		LastRequestUser:  lastRequestUser,
		WorkflowTemplateInfo: &evpb.WorkflowTemplateInfo{
			Path:        workflowPath,
			Public:      workflowPublicInfo.IsPublic,
			Name:        workflowPublicInfo.Name,
			Description: workflowPublicInfo.Description,
		},
	})

	env, err := newEnvironment(envUserVars, newId)
	newEnvId := uid.NilID()
	if err == nil && env != nil {
		newEnvId = env.Id()
	} else {
		log.WithError(err).Logf(logrus.FatalLevel, "environment creation failed")
		env.sendEnvironmentEvent(&event.EnvironmentEvent{EnvironmentID: newEnvId.String(), Error: err})
		the.EventWriterWithTopic(topic.Environment).WriteEvent(&evpb.Ev_EnvironmentEvent{
			EnvironmentId:   newId.String(),
			State:           "ERROR",
			Error:           err.Error(),
			Message:         "cannot create new environment", // GUI listens for this concrete string
			LastRequestUser: lastRequestUser,
			WorkflowTemplateInfo: &evpb.WorkflowTemplateInfo{
				Path:        workflowPath,
				Public:      workflowPublicInfo.IsPublic,
				Name:        workflowPublicInfo.Name,
				Description: workflowPublicInfo.Description,
			},
		})
		return
	}

	log.WithFields(logrus.Fields{
		"workflow":  workflowPath,
		"partition": newEnvId.String(),
	}).Info("creating new automatic environment")

	env.Public = workflowPublicInfo.IsPublic
	env.name = workflowPublicInfo.Name
	env.Description = workflowPublicInfo.Description
	env.WorkflowPath = workflowPath

	the.EventWriterWithTopic(topic.Environment).WriteEvent(&evpb.Ev_EnvironmentEvent{
		EnvironmentId:        newId.String(),
		State:                "PENDING",
		Transition:           "CREATE",
		TransitionStep:       "before_CREATE",
		TransitionStatus:     evpb.OpStatus_ONGOING,
		Message:              "running hooks",
		LastRequestUser:      lastRequestUser,
		WorkflowTemplateInfo: env.GetWorkflowInfo(),
	})

	env.addSubscription(sub)
	defer env.closeStream()

	env.hookHandlerF = func(hooks task.Tasks) error {
		return envs.taskman.TriggerHooks(newEnvId, hooks)
	}

	// Ensure the environment_id is available to all
	env.UserVars.Set("environment_id", env.id.String())

	the.EventWriterWithTopic(topic.Environment).WriteEvent(&evpb.Ev_EnvironmentEvent{
		EnvironmentId:        newId.String(),
		State:                "PENDING",
		Transition:           "CREATE",
		TransitionStep:       "CREATE",
		TransitionStatus:     evpb.OpStatus_ONGOING,
		Message:              "loading workflow",
		LastRequestUser:      lastRequestUser,
		WorkflowTemplateInfo: env.GetWorkflowInfo(),
	})

	env.workflow, err = envs.loadWorkflow(workflowPath, env.wfAdapter, workflowUserVars, env.BaseConfigStack)
	if err != nil {
		err = fmt.Errorf("cannot load workflow template: %w", err)
		env.sendEnvironmentEvent(&event.EnvironmentEvent{EnvironmentID: env.Id().String(), Error: err})

		log.WithError(err).Error("failed to load workflow")

		the.EventWriterWithTopic(topic.Environment).WriteEvent(&evpb.Ev_EnvironmentEvent{
			EnvironmentId:   newId.String(),
			State:           "ERROR",
			Error:           err.Error(),
			Message:         "cannot load workflow", // GUI listens for this concrete string
			LastRequestUser: lastRequestUser,
			WorkflowTemplateInfo: &evpb.WorkflowTemplateInfo{
				Path:        workflowPath,
				Public:      workflowPublicInfo.IsPublic,
				Name:        workflowPublicInfo.Name,
				Description: workflowPublicInfo.Description,
			},
		})

		return
	}

	cvs, _ := env.Workflow().ConsolidatedVarStack()
	the.EventWriterWithTopic(topic.Environment).WriteEvent(&evpb.Ev_EnvironmentEvent{
		EnvironmentId:        newId.String(),
		State:                env.CurrentState(),
		Transition:           "CREATE",
		TransitionStep:       "after_CREATE",
		TransitionStatus:     evpb.OpStatus_DONE_OK,
		Message:              "workflow loaded",
		Vars:                 cvs, // we push the full var stack of the root role in the workflow loaded event
		LastRequestUser:      lastRequestUser,
		WorkflowTemplateInfo: env.GetWorkflowInfo(),
	})

	log.WithField("method", "CreateAutoEnvironment").
		WithField("level", infologger.IL_Devel).
		Debug("envman write lock")
	envs.mu.Lock()
	envs.m[env.id] = env
	envs.pendingStateChangeCh[env.id] = env.stateChangedCh
	envs.mu.Unlock()
	log.WithField("method", "CreateAutoEnvironment").
		WithField("level", infologger.IL_Devel).
		Debug("envman write unlock")

	err = env.TryTransition(NewDeployTransition(
		envs.taskman,
		nil, // roles,
		nil),
	)
	if err == nil {
		err = env.TryTransition(NewConfigureTransition(
			envs.taskman),
		)
	}

	goErrorKillDestroy := func(op string) {
		envState := env.CurrentState()
		env.sendEnvironmentEvent(&event.EnvironmentEvent{Message: fmt.Sprintf("environment is in %s, handling ERROR", envState), EnvironmentID: env.Id().String(), Error: err})
		log.WithField("state", envState).
			WithField("partition", env.Id().String()).
			WithError(err).
			Warnf("auto-transitioning environment failed %s, cleanup in progress", op)

		the.EventWriterWithTopic(topic.Environment).WriteEvent(
			NewEnvGoErrorEvent(env, fmt.Sprintf("%s failed: %v", op, err)),
		)
		err := env.TryTransition(NewGoErrorTransition(
			envs.taskman),
		)
		if err != nil {
			log.WithField("partition", env.Id().String()).
				WithField("state", envState).
				Debug("could not transition failed auto-transitioning environment to ERROR, cleanup in progress")
			env.sendEnvironmentEvent(&event.EnvironmentEvent{Message: "transition ERROR failed, forcing", EnvironmentID: env.Id().String(), Error: err})
			env.setState("ERROR")
		}

		envTasks := env.Workflow().GetTasks()
		// TeardownEnvironment manages the envs.mu internally
		err = envs.TeardownEnvironment(env.Id(), true /*force*/)
		if err != nil {
			env.sendEnvironmentEvent(&event.EnvironmentEvent{Message: "force teardown failed, handling", EnvironmentID: env.Id().String(), Error: err})
		}

		killedTasks, _, rlsErr := envs.taskman.KillTasks(envTasks.GetTaskIds())
		if rlsErr != nil {
			log.WithError(rlsErr).Warn("task teardown error")
		}
		log.WithFields(logrus.Fields{
			"killedCount":  len(killedTasks),
			"lastEnvState": envState,
			"level":        infologger.IL_Support,
			"partition":    env.Id().String(),
		}).
			Infof("auto-environment failed at %s, tasks were cleaned up", op)
		log.WithField("partition", env.Id().String()).Info("environment teardown complete")

		env.sendEnvironmentEvent(&event.EnvironmentEvent{Message: "environment teardown complete", State: "DONE", EnvironmentID: env.Id().String()})
	}

	if err != nil {
		goErrorKillDestroy("transition CONFIGURE")

		return
	}

	env.subscribeToWfState(envs.taskman)
	defer env.unsubscribeFromWfState()

	// now we have the environment we should transition to start
	trans := NewStartActivityTransition(envs.taskman)
	if trans == nil {
		env.sendEnvironmentEvent(&event.EnvironmentEvent{EnvironmentID: env.Id().String(), Error: err, Message: "transition START_ACTIVITY failed, handling"})
		goErrorKillDestroy("transition START_ACTIVITY")

		return
	}

	err = env.TryTransition(trans)
	if err != nil {
		goErrorKillDestroy("transition START_ACTIVITY")
		return
	}

	for {
		// we know we performed START_ACTIVITY, so we poll at 1Hz for the run to finish
		time.Sleep(1 * time.Second)

		if env == nil {
			// must've died during the loop
			return
		}

		envState := env.CurrentState()
		switch envState {
		case "CONFIGURED":
			// RUN finished so we can reset and delete the environment
			err = env.TryTransition(NewResetTransition(envs.taskman))
			if err != nil {
				env.sendEnvironmentEvent(&event.EnvironmentEvent{Message: "transition RESET failed, handling", EnvironmentID: env.Id().String(), Error: err})
				goErrorKillDestroy("transition RESET")
				return
			}
			err = envs.TeardownEnvironment(env.id, false)
			if err != nil {
				env.sendEnvironmentEvent(&event.EnvironmentEvent{Message: "teardown failed, handling", EnvironmentID: env.Id().String(), Error: err})
				goErrorKillDestroy("teardown")
				return
			}
			tasksForEnv := env.Workflow().GetTasks().GetTaskIds()
			_, _, err = envs.taskman.KillTasks(tasksForEnv)
			if err != nil {
				env.sendEnvironmentEvent(&event.EnvironmentEvent{Message: "task kill failed, handling", EnvironmentID: env.Id().String(), Error: err})
				return
			}

			env.sendEnvironmentEvent(&event.EnvironmentEvent{Message: "environment teardown complete", State: "DONE", EnvironmentID: env.Id().String()})
			return
		case "ERROR":
			fallthrough
		case "STANDBY":
			fallthrough
		case "DEPLOYED":
			goErrorKillDestroy("transition STOP_ACTIVITY")
			return
		case "MIXED":
			continue
		case "":
			continue
		}
	}
}
