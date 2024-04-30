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
	event2 "github.com/AliceO2Group/Control/core/integration/odc/event"
	"github.com/AliceO2Group/Control/core/task"
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

var (
	instance *Manager
)

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
								Error("cannot find environment for incoming executor failed event")
						}
						log.WithPrefix("scheduler").
							WithField("partition", envId.String()).
							WithField("agentId", typedEvent.GetId().Value).
							WithField("envState", env.CurrentState()).
							Debug("received executor failed event")
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
						var releaseCriticalTask = false
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
					if ok {
						thisEnvCh <- typedEvent
					} else {
						// If there is no pending environment transition, it means that the changed task did so
						// unexpectedly. In that case, the environment should transition only if the task
						// is critical.
						var changeCriticalTask = false
						for _, v := range typedEvent.GetTaskIds() {
							if tm.GetTask(v) != nil {
								if tm.GetTask(v).GetTraits().Critical == true {
									changeCriticalTask = true
								}
							}
						}
						if changeCriticalTask {
							thisEnvCh <- typedEvent
						}
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
		for det, _ := range envDetectors {
			response[det] = struct{}{}
		}
	}
	return response
}

func (envs *Manager) CreateEnvironment(workflowPath string, userVars map[string]string, public bool, newId uid.ID) (uid.ID, error) {
	// Before we load the workflow, we get the list of currently active detectors. This query must be performed before
	// loading the workflow in order to compare the currently used detectors with the detectors required by the newly
	// created environment.
	alreadyActiveDetectors := envs.GetActiveDetectors()

	the.EventWriterWithTopic(topic.Environment).WriteEvent(&evpb.Ev_EnvironmentEvent{
		EnvironmentId:  newId.String(),
		State:          "PENDING",
		Transition:     "CREATE",
		TransitionStep: "before_CREATE",
		Message:        "instantiating",
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
	if err != nil {
		log.WithError(err).
			WithField("partition", gotEnvId.String()).
			Logf(logrus.FatalLevel, "environment creation failed")
		return gotEnvId, err
	}

	log.WithFields(logrus.Fields{
		"workflow":  workflowPath,
		"partition": gotEnvId.String(),
	}).Info("creating new environment")

	the.EventWriterWithTopic(topic.Environment).WriteEvent(&evpb.Ev_EnvironmentEvent{
		EnvironmentId:  newId.String(),
		State:          "PENDING",
		Transition:     "CREATE",
		TransitionStep: "before_CREATE",
		Message:        "running hooks",
	})

	env.hookHandlerF = func(hooks task.Tasks) error {
		return envs.taskman.TriggerHooks(gotEnvId, hooks)
	}

	// Ensure the environment_id is available to all
	env.UserVars.Set("environment_id", env.id.String())

	// in case of err==nil, env will be false unless user
	// set it to True which will be overwritten in server.go
	env.Public, env.Description, err = parseWorkflowPublicInfo(workflowPath)
	if err != nil {
		log.WithField("public info", env.Public).
			WithField("partition", env.Id().String()).
			WithError(err).
			Warn("parse workflow public info failed.")
		return gotEnvId, fmt.Errorf("workflow public info parsing failed: %w", err)
	}

	the.EventWriterWithTopic(topic.Environment).WriteEvent(&evpb.Ev_EnvironmentEvent{
		EnvironmentId:  newId.String(),
		State:          "PENDING",
		Transition:     "CREATE",
		TransitionStep: "CREATE",
		Message:        "loading workflow",
	})

	// We load the workflow (includes template processing)
	env.workflow, err = envs.loadWorkflow(workflowPath, env.wfAdapter, workflowUserVars, env.BaseConfigStack)
	if err != nil {
		err = fmt.Errorf("cannot load workflow template: %w", err)

		return env.id, err
	}

	// Ensure we provide a very defaulty `detectors` variable
	detectors, err := the.ConfSvc().GetDetectorsForHosts(env.GetFLPs())
	if err != nil {
		err = fmt.Errorf("cannot acquire detectors in loaded workflow template: %w", err)

		return env.id, err
	}
	detectorsStr, err := SliceToJSONSlice(detectors)
	if err != nil {
		err = fmt.Errorf("cannot process detectors in loaded workflow template: %w", err)

		return env.id, err
	}
	env.GlobalDefaults.Set("detectors", detectorsStr)

	log.WithFields(logrus.Fields{
		"partition": gotEnvId.String(),
	}).Infof("detectors in environment: %s", strings.Join(detectors, " "))

	// env.GetActiveDetectors() is valid starting now, so we can check for detector exclusion
	neededDetectors := env.GetActiveDetectors()
	for det, _ := range neededDetectors {
		if _, contains := alreadyActiveDetectors[det]; contains {
			// required detector det is already active in some other environment
			return env.id, fmt.Errorf("detector %s is already in use", det.String())
		}
	}

	cvs, _ := env.Workflow().ConsolidatedVarStack()
	the.EventWriterWithTopic(topic.Environment).WriteEvent(&evpb.Ev_EnvironmentEvent{
		EnvironmentId:  newId.String(),
		State:          env.CurrentState(),
		Transition:     "CREATE",
		TransitionStep: "after_CREATE",
		Message:        "workflow loaded",
		Vars:           cvs, // we push the full var stack of the root role in the workflow loaded event
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
		nil, //roles,
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

		return env.id, err
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

	if env.CurrentState() != "STANDBY" && env.CurrentState() != "DEPLOYED" && !force {
		return errors.New(fmt.Sprintf("cannot teardown environment in state %s", env.CurrentState()))
	}

	env.Mu.Lock()
	env.currentTransition = "DESTROY"
	env.Mu.Unlock()

	err = env.handleHooks(env.Workflow(), "leave_"+env.CurrentState())
	if err != nil {
		log.WithFields(logrus.Fields{
			"partition": environmentId.String(),
		}).Error(fmt.Errorf("could not handle hooks for the trigger leave_%s, error: %w", env.CurrentState(), err))
	}

	if env.CurrentState() == "RUNNING" {
		endTime, ok := env.workflow.GetUserVars().Get("run_end_time_ms")
		if ok && endTime == "" {
			runEndTime := strconv.FormatInt(time.Now().UnixMilli(), 10)
			env.workflow.SetRuntimeVar("run_end_time_ms", runEndTime)
		} else {
			log.WithField("partition", environmentId.String()).
				Debug("O2 End time already set before DESTROY")
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
		return err
	}

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
		return err
	}

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
							err = env.TryTransition(NewGoErrorTransition(envs.taskman))
							if err != nil {
								log.WithPrefix("scheduler").
									WithField("partition", envId.String()).
									WithError(err).
									Error("environment GO_ERROR transition failed after ODC_PARTITION_STATE_CHANGE ERROR event")
							}
							env.setState("ERROR")
						}
					}()
				}
			}
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
								WithError(err).
								Error("cannot find environment for DeviceEvent")
						}
						if env != nil {
							env.NotifyEvent(evt)
						}
					}
				} else {
					log.WithPrefix("scheduler").
						WithField("partition", envId.String()).
						Error("DeviceEvent BASIC_TASK_TERMINATED received for task with no parent role")
				}
			} else {
				log.WithPrefix("scheduler").
					WithField("partition", envId.String()).
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
				Debug("cannot find task for DeviceEvent END_OF_STREAM")
			return
		}
		env, err := envs.environment(t.GetEnvironmentId())
		if err != nil {
			log.WithPrefix("scheduler").
				WithField("partition", envId.String()).
				WithField("taskId", taskId.Value).
				WithError(err).
				Error("cannot find environment for DeviceEvent")
		} else {
			log.WithPrefix("scheduler").
				WithField("partition", envId.String()).
				WithField("taskId", taskId.Value).
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
				WithError(err).
				Error("cannot find environment for DeviceEvent")
		} else {
			log.WithPrefix("scheduler").
				WithField("partition", envId.String()).
				WithField("taskId", taskId.Value).
				WithField("taskRole", t.GetParentRolePath()).
				WithField("envState", env.CurrentState()).
				Debug("received TASK_INTERNAL_ERROR event from task, trying to stop the run")
			if env.CurrentState() == "RUNNING" {
				go func() {
					t.GetParent().UpdateState(task.ERROR)
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

	env, err := newEnvironment(envUserVars, newId)
	newEnvId := uid.NilID()
	if err == nil && env != nil {
		newEnvId = env.Id()
	} else {
		log.WithError(err).Logf(logrus.FatalLevel, "environment creation failed")
		env.sendEnvironmentEvent(&event.EnvironmentEvent{EnvironmentID: newEnvId.String(), Error: err})
		return
	}

	log.WithFields(logrus.Fields{
		"workflow":  workflowPath,
		"partition": newEnvId.String(),
	}).Info("creating new automatic environment")

	env.addSubscription(sub)
	defer env.closeStream()

	env.hookHandlerF = func(hooks task.Tasks) error {
		return envs.taskman.TriggerHooks(newEnvId, hooks)
	}

	// Ensure the environment_id is available to all
	env.UserVars.Set("environment_id", env.id.String())

	// in case of err==nil, env will be false unless user
	// set it to True which will be overwriten in server.go
	env.Public, env.Description, err = parseWorkflowPublicInfo(workflowPath)
	if err != nil {
		log.WithField("public info", env.Public).
			WithField("environment", env.Id().String()).
			WithError(err).
			Warn("parse workflow public info failed.")
	}

	env.workflow, err = envs.loadWorkflow(workflowPath, env.wfAdapter, workflowUserVars, env.BaseConfigStack)
	if err != nil {
		err = fmt.Errorf("cannot load workflow template: %w", err)
		env.sendEnvironmentEvent(&event.EnvironmentEvent{EnvironmentID: env.Id().String(), Error: err})
		return
	}

	env.Public, env.Description, _ = parseWorkflowPublicInfo(workflowPath)

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
		nil, //roles,
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
			WithField("environment", env.Id().String()).
			WithError(err).
			Warnf("auto-transitioning environment failed %s, cleanup in progress", op)

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

		env.sendEnvironmentEvent(&event.EnvironmentEvent{Message: "environment teardown complete", EnvironmentID: env.Id().String()})
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

			env.sendEnvironmentEvent(&event.EnvironmentEvent{Message: "environment teardown complete", EnvironmentID: env.Id().String()})
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
