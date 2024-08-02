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

// Package environment defines Environment, environment.Manager and
// other types and methods related to handling O² environments.
package environment

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/AliceO2Group/Control/common/event"
	"github.com/AliceO2Group/Control/common/event/topic"
	"github.com/AliceO2Group/Control/common/gera"
	"github.com/AliceO2Group/Control/common/logger"
	"github.com/AliceO2Group/Control/common/logger/infologger"
	pb "github.com/AliceO2Group/Control/common/protos"
	"github.com/AliceO2Group/Control/common/runtype"
	"github.com/AliceO2Group/Control/common/system"
	"github.com/AliceO2Group/Control/common/utils"
	"github.com/AliceO2Group/Control/common/utils/uid"
	"github.com/AliceO2Group/Control/core/task"
	"github.com/AliceO2Group/Control/core/task/sm"
	"github.com/AliceO2Group/Control/core/the"
	"github.com/AliceO2Group/Control/core/workflow"
	"github.com/AliceO2Group/Control/core/workflow/callable"
	"github.com/gobwas/glob"
	"github.com/looplab/fsm"
	"github.com/pborman/uuid"
	"github.com/sirupsen/logrus"
)

var log = logger.New(logrus.StandardLogger(), "env")

type Environment struct {
	Mu               sync.RWMutex
	once             sync.Once
	transitionMutex  sync.RWMutex
	Sm               *fsm.FSM
	name             string
	id               uid.ID
	ts               time.Time
	workflow         workflow.Role
	wfAdapter        *workflow.ParentAdapter
	currentRunNumber uint32
	hookHandlerF     func(hooks task.Tasks) error
	incomingEvents   chan event.DeviceEvent

	GlobalDefaults  gera.StringMap    // From Consul
	GlobalVars      gera.StringMap    // From Consul
	UserVars        gera.StringMap    // From user input
	BaseConfigStack map[string]string // Exclusively from Consul, already flattened for performance

	stateChangedCh chan *event.TasksStateChangedEvent
	unsubscribe    chan struct{}
	eventStream    Subscription
	Public         bool   // From workflow or user
	Description    string // From workflow

	callsPendingAwait map[string] /*await expression, trigger only*/ callable.CallsMap
	currentTransition string

	autoStopTimer *time.Timer
}

func (env *Environment) NotifyEvent(e event.DeviceEvent) {
	if e != nil && env.incomingEvents != nil {
		select {
		case env.incomingEvents <- e:
		default:
		}
	}
}

func newEnvironment(userVars map[string]string, newId uid.ID) (env *Environment, err error) {
	envId := newId
	env = &Environment{
		id:             envId,
		workflow:       nil,
		ts:             time.Now(),
		incomingEvents: make(chan event.DeviceEvent),
		// Every Environment instantiation performs a ConfSvc query for defaults and vars
		// these key-values stay frozen throughout the lifetime of the environment
		GlobalDefaults: gera.MakeStringMapWithMap(the.ConfSvc().GetDefaults()),
		GlobalVars:     gera.MakeStringMapWithMap(the.ConfSvc().GetVars()),
		UserVars:       gera.MakeStringMapWithMap(userVars),
		stateChangedCh: make(chan *event.TasksStateChangedEvent),

		callsPendingAwait: make(map[string]callable.CallsMap),
	}

	// Make the KVs accessible to the workflow via ParentAdapter
	env.wfAdapter = workflow.NewParentAdapter(
		func() uid.ID { return env.Id() },
		func() uint32 { return env.GetCurrentRunNumber() },
		func() gera.StringMap { return env.GlobalDefaults },
		func() gera.StringMap { return env.GlobalVars },
		func() gera.StringMap { return env.UserVars },
		func(ev event.Event) {
			env.Mu.Lock()
			defer env.Mu.Unlock()
			if env.eventStream != nil {
				env.eventStream.Send(ev)
			}
		},
	)
	env.GlobalVars.Set("__fmq_cleanup_count", "0") // initialize to 0 the number of START transitions

	env.BaseConfigStack, err = gera.MakeStringMapWithMap(env.GlobalVars.Raw()).
		WrappedAndFlattened(gera.MakeStringMapWithMap(env.GlobalDefaults.Raw())) // prepare the base config stack
	if err != nil {
		return nil, err
	}

	// We start with STANDBY, which will not be preceded with enter_STANDBY, thus we set the value here.
	enterStateTimeMs := strconv.FormatInt(time.Now().UnixMilli(), 10)
	env.UserVars.Set("enter_state_time_ms", enterStateTimeMs)

	env.Sm = fsm.NewFSM(
		"STANDBY",
		fsm.Events{
			{Name: "DEPLOY", Src: []string{"STANDBY"}, Dst: "DEPLOYED"},
			{Name: "CONFIGURE", Src: []string{"DEPLOYED"}, Dst: "CONFIGURED"},
			{Name: "RESET", Src: []string{"CONFIGURED"}, Dst: "DEPLOYED"},
			{Name: "START_ACTIVITY", Src: []string{"CONFIGURED"}, Dst: "RUNNING"},
			{Name: "STOP_ACTIVITY", Src: []string{"RUNNING"}, Dst: "CONFIGURED"},
			{Name: "EXIT", Src: []string{"CONFIGURED", "DEPLOYED", "STANDBY"}, Dst: "DONE"},
			{Name: "GO_ERROR", Src: []string{"STANDBY", "CONFIGURED", "DEPLOYED", "RUNNING"}, Dst: "ERROR"},
			{Name: "RECOVER", Src: []string{"ERROR"}, Dst: "DEPLOYED"},
		},
		fsm.Callbacks{
			"before_event": func(_ context.Context, e *fsm.Event) {

				env.Mu.Lock()
				env.currentTransition = e.Event
				env.Mu.Unlock()

				trigger := fmt.Sprintf("before_%s", e.Event)

				the.EventWriterWithTopic(topic.Environment).WriteEvent(&pb.Ev_EnvironmentEvent{
					EnvironmentId:   env.id.String(),
					State:           env.Sm.Current(),
					RunNumber:       env.GetCurrentRunNumber(),
					Transition:      e.Event,
					TransitionStep:  trigger,
					Message:         "transition step starting",
					LastRequestUser: env.GetLastRequestUser(),
				})

				// first, we execute hooks which should be executed before an event officially starts
				errHooks := env.handleHooksWithNegativeWeights(env.Workflow(), trigger)
				if errHooks != nil {
					e.Cancel(errHooks)
					the.EventWriterWithTopic(topic.Environment).WriteEvent(&pb.Ev_EnvironmentEvent{
						EnvironmentId:  env.id.String(),
						State:          env.Sm.Current(),
						RunNumber:      env.GetCurrentRunNumber(),
						Error:          errHooks.Error(),
						Transition:     e.Event,
						TransitionStep: trigger,
						Message:        "transition step finished",
					})
					return
				}

				// If the event is START_ACTIVITY, we set up and update variables relevant to plugins early on.
				// This used to be done inside the transition_startactivity, but then the new RN isn't available to the
				// before_START_ACTIVITY hooks. By setting it up here, we ensure the run number is available especially
				// to plugin hooks.
				if e.Event == "START_ACTIVITY" {
					runNumber, rnErr := the.ConfSvc().NewRunNumber()
					if rnErr != nil {
						e.Cancel(rnErr)
						return
					}
					env.currentRunNumber = runNumber
					rnString := strconv.FormatUint(uint64(runNumber), 10)
					env.workflow.GetVars().Set("run_number", rnString)
					env.workflow.GetVars().Set("runNumber", rnString)

					runStartTime := time.Now()
					runStartTimeStr := strconv.FormatInt(runStartTime.UnixMilli(), 10)
					env.workflow.SetRuntimeVar("run_start_time_ms", runStartTimeStr)
					env.workflow.SetRuntimeVar("run_start_completion_time_ms", "") // we delete previous EOSOR
					env.workflow.SetRuntimeVar("run_end_time_ms", "")              // we delete previous SOEOR
					env.workflow.SetRuntimeVar("run_end_completion_time_ms", "")   // we delete previous EOEOR

					the.EventWriterWithTopic(topic.Run).WriteEventWithTimestamp(&pb.Ev_RunEvent{
						EnvironmentId:    envId.String(),
						RunNumber:        runNumber,
						State:            env.Sm.Current(),
						Error:            "",
						Transition:       e.Event,
						TransitionStatus: pb.OpStatus_STARTED,
						Vars:             nil,
						LastRequestUser:  env.GetLastRequestUser(),
					}, runStartTime)

					cleanupCount := 0
					cleanupCountS, ok := env.GlobalVars.Get("__fmq_cleanup_count")
					if ok && len(cleanupCountS) > 0 {
						var parseErr error
						cleanupCount, parseErr = strconv.Atoi(cleanupCountS)
						if parseErr != nil {
							cleanupCount = 1 // something was there, even though non-parsable, so we signal to clean up
						}
					}
					env.GlobalVars.Set("__fmq_cleanup_count", strconv.Itoa(cleanupCount)) // number of times the START transition has run for this env

					if err == nil {
						lhcPeriod, ok := env.BaseConfigStack["lhc_period"]
						if ok {
							env.workflow.GetVars().Set("lhc_period", lhcPeriod)
						}
						nHbfPerTf, ok := env.BaseConfigStack["pdp_n_hbf_per_tf"]
						if ok {
							env.workflow.GetVars().Set("pdp_n_hbf_per_tf", nHbfPerTf)
						}
					} else {
						log.Error("cannot access AliECS workflow configuration defaults")
					}
				} else if e.Event == "STOP_ACTIVITY" {
					endTime, ok := env.workflow.GetUserVars().Get("run_end_time_ms")
					if ok && endTime == "" {
						runEndTime := time.Now()
						runEndTimeStr := strconv.FormatInt(runEndTime.UnixMilli(), 10)
						env.workflow.SetRuntimeVar("run_end_time_ms", runEndTimeStr)

						the.EventWriterWithTopic(topic.Run).WriteEventWithTimestamp(&pb.Ev_RunEvent{
							EnvironmentId:    envId.String(),
							RunNumber:        env.GetCurrentRunNumber(),
							State:            env.Sm.Current(),
							Error:            "",
							Transition:       e.Event,
							TransitionStatus: pb.OpStatus_STARTED,
							Vars:             nil,
							LastRequestUser:  env.GetLastRequestUser(),
						}, runEndTime)

					} else {
						log.WithField("partition", envId.String()).
							Debug("O2 End time already set before before_STOP_ACTIVITY")
					}
				} else if e.Event == "GO_ERROR" {
					endTime, ok := env.workflow.GetUserVars().Get("run_end_time_ms")
					if ok && endTime == "" {
						runEndTime := time.Now()
						runEndTimeStr := strconv.FormatInt(runEndTime.UnixMilli(), 10)
						env.workflow.SetRuntimeVar("run_end_time_ms", runEndTimeStr)

						the.EventWriterWithTopic(topic.Run).WriteEventWithTimestamp(&pb.Ev_RunEvent{
							EnvironmentId:    envId.String(),
							RunNumber:        env.GetCurrentRunNumber(),
							State:            env.Sm.Current(),
							Error:            "",
							Transition:       e.Event,
							TransitionStatus: pb.OpStatus_STARTED,
							Vars:             nil,
							LastRequestUser:  env.GetLastRequestUser(),
						}, runEndTime)

					} else {
						log.WithField("partition", envId.String()).
							Debug("O2 End time already set before before_GO_ERROR")
					}
				}

				if rn := env.GetCurrentRunNumber(); rn != 0 {
					log.WithField("partition", envId).
						WithField("run", rn).
						Infof("%s transition starting",
							e.Event,
						)
				} else {
					log.WithField("partition", envId).
						Infof("%s transition starting",
							e.Event,
						)
				}

				errHooks = env.handleHooksWithPositiveWeights(env.Workflow(), trigger)
				if errHooks != nil {
					e.Cancel(errHooks)
				}

				errorMsg := ""
				if e.Err != nil {
					errorMsg = e.Err.Error()
				}

				the.EventWriterWithTopic(topic.Environment).WriteEvent(&pb.Ev_EnvironmentEvent{
					EnvironmentId:   env.id.String(),
					State:           env.Sm.Current(),
					RunNumber:       env.currentRunNumber,
					Error:           errorMsg,
					Message:         "transition step finished",
					Transition:      e.Event,
					TransitionStep:  trigger,
					LastRequestUser: env.GetLastRequestUser(),
				})
			},
			"leave_state": func(_ context.Context, e *fsm.Event) {
				trigger := fmt.Sprintf("leave_%s", e.Src)

				the.EventWriterWithTopic(topic.Environment).WriteEvent(&pb.Ev_EnvironmentEvent{
					EnvironmentId:   env.id.String(),
					State:           env.Sm.Current(),
					RunNumber:       env.currentRunNumber,
					Transition:      e.Event,
					TransitionStep:  trigger,
					Message:         "transition step starting",
					LastRequestUser: env.GetLastRequestUser(),
				})

				errHooks := env.handleHooksWithNegativeWeights(env.Workflow(), trigger)
				// fixme: in principle we should not need it anymore, since both STOP_ACTIVITY and GO_ERROR set EOR
				// We might leave RUNNING not only through STOP_ACTIVITY. In such cases we also need a run stop time.
				if e.Src == "RUNNING" {
					endTime, ok := env.workflow.GetUserVars().Get("run_end_time_ms")
					if ok && endTime == "" {
						runEndTime := strconv.FormatInt(time.Now().UnixMilli(), 10)
						env.workflow.SetRuntimeVar("run_end_time_ms", runEndTime)
					} else {
						log.WithField("partition", envId.String()).
							Debug("O2 End time already set before leave_RUNNING")
					}
				}
				if errHooks != nil {
					e.Cancel(errHooks)
					the.EventWriterWithTopic(topic.Environment).WriteEvent(&pb.Ev_EnvironmentEvent{
						EnvironmentId:  env.id.String(),
						State:          env.Sm.Current(),
						RunNumber:      env.GetCurrentRunNumber(),
						Error:          errHooks.Error(),
						Transition:     e.Event,
						TransitionStep: trigger,
						Message:        "transition step finished",
					})
					return
				}

				errHooks = env.handleHooksWithPositiveWeights(env.Workflow(), trigger)
				if errHooks != nil {
					e.Cancel(errHooks)
				}

				errorMsg := ""
				if e.Err != nil {
					errorMsg = e.Err.Error()
				}

				the.EventWriterWithTopic(topic.Environment).WriteEvent(&pb.Ev_EnvironmentEvent{
					EnvironmentId:   env.id.String(),
					State:           env.Sm.Current(),
					RunNumber:       env.currentRunNumber,
					Error:           errorMsg,
					Message:         "transition step finished",
					Transition:      e.Event,
					TransitionStep:  trigger,
					LastRequestUser: env.GetLastRequestUser(),
				})

				if e.Err != nil {
					return
				}

				the.EventWriterWithTopic(topic.Environment).WriteEvent(&pb.Ev_EnvironmentEvent{
					EnvironmentId:   env.id.String(),
					State:           env.Sm.Current(),
					RunNumber:       env.currentRunNumber,
					Message:         "transition step starting",
					Transition:      e.Event,
					TransitionStep:  fmt.Sprintf("tasks_%s", e.Event),
					LastRequestUser: env.GetLastRequestUser(),
				})

				env.handlerFunc()(e)

				if e.Err != nil {
					errorMsg = e.Err.Error()
				}

				the.EventWriterWithTopic(topic.Environment).WriteEvent(&pb.Ev_EnvironmentEvent{
					EnvironmentId:   env.id.String(),
					State:           e.Dst, // exceptionally we take the destination state here instead of the current, because the tasks have transitioned
					RunNumber:       env.currentRunNumber,
					Error:           errorMsg,
					Message:         "transition step finished",
					Transition:      e.Event,
					TransitionStep:  fmt.Sprintf("tasks_%s", e.Event),
					LastRequestUser: env.GetLastRequestUser(),
				})
			},
			"enter_state": func(_ context.Context, e *fsm.Event) {
				trigger := fmt.Sprintf("enter_%s", e.Dst)

				the.EventWriterWithTopic(topic.Environment).WriteEvent(&pb.Ev_EnvironmentEvent{
					EnvironmentId:   env.id.String(),
					State:           env.Sm.Current(),
					RunNumber:       env.currentRunNumber,
					Transition:      e.Event,
					TransitionStep:  trigger,
					Message:         "transition step starting",
					LastRequestUser: env.GetLastRequestUser(),
				})

				errHooks := env.handleHooksWithNegativeWeights(env.Workflow(), trigger)

				enterStateTimeMs = strconv.FormatInt(time.Now().UnixMilli(), 10)
				env.workflow.SetRuntimeVar("enter_state_time_ms", enterStateTimeMs)

				errHooks = errors.Join(errHooks, env.handleHooksWithPositiveWeights(env.Workflow(), trigger))
				if errHooks != nil {
					// at enter_<state> it will not cancel the transition but only set the error
					e.Cancel(errHooks)
				}

				errorMsg := ""
				if e.Err != nil {
					errorMsg = e.Err.Error()
				}

				the.EventWriterWithTopic(topic.Environment).WriteEvent(&pb.Ev_EnvironmentEvent{
					EnvironmentId:   env.id.String(),
					State:           env.Sm.Current(),
					RunNumber:       env.currentRunNumber,
					Error:           errorMsg,
					Message:         "transition step finished",
					Transition:      e.Event,
					TransitionStep:  trigger,
					LastRequestUser: env.GetLastRequestUser(),
				})

				if e.Err != nil {
					return
				}

				log.WithFields(logrus.Fields{
					"event":     e.Event,
					"src":       e.Src,
					"dst":       e.Dst,
					"partition": envId,
				}).Debug("environment.sm entering state")
			},
			"after_event": func(_ context.Context, e *fsm.Event) {
				defer func() {
					env.Mu.Lock()
					env.currentTransition = ""
					env.Mu.Unlock()
				}()

				trigger := fmt.Sprintf("after_%s", e.Event)

				the.EventWriterWithTopic(topic.Environment).WriteEvent(&pb.Ev_EnvironmentEvent{
					EnvironmentId:   env.id.String(),
					State:           env.Sm.Current(),
					RunNumber:       env.currentRunNumber,
					Transition:      e.Event,
					TransitionStep:  trigger,
					Message:         "transition step starting",
					LastRequestUser: env.GetLastRequestUser(),
				})

				errHooks := env.handleHooksWithNegativeWeights(env.Workflow(), trigger)
				if errHooks != nil {
					// at after_<event> it will not cancel the transition but only set the error
					e.Cancel(errHooks)
				}

				if rn := env.GetCurrentRunNumber(); rn != 0 {
					log.WithField("partition", envId).
						WithField("run", rn).
						Infof("%s transition complete",
							e.Event,
						)
				} else {
					log.WithField("partition", envId).
						Infof("%s transition complete",
							e.Event,
						)
				}

				if e.Event == "START_ACTIVITY" {
					// If START_ACTIVITY is complete, we increment the FairMQ cleanup counter
					cleanupCount := 0
					cleanupCountS, ok := env.GlobalVars.Get("__fmq_cleanup_count")
					if ok && len(cleanupCountS) > 0 {
						cleanupCount, _ = strconv.Atoi(cleanupCountS)
					}
					cleanupCount++
					env.GlobalVars.Set("__fmq_cleanup_count", strconv.Itoa(cleanupCount))

					// Register auto stop transition (if enabled)
					scheduled, expected := env.scheduleAutoStopTransition()
					if scheduled {
						log.WithField("partition", env.id).
							WithField("run", env.currentRunNumber).
							Infof("auto stop transition scheduled, expected execution at %s", expected)
					}

					runStartCompletionTime := time.Now()
					runStartCompletionTimeStr := strconv.FormatInt(runStartCompletionTime.UnixMilli(), 10)
					env.workflow.SetRuntimeVar("run_start_completion_time_ms", runStartCompletionTimeStr)

					runEvent := &pb.Ev_RunEvent{
						EnvironmentId:    envId.String(),
						RunNumber:        env.GetCurrentRunNumber(),
						State:            env.Sm.Current(),
						Error:            "",
						Transition:       e.Event,
						TransitionStatus: pb.OpStatus_DONE_OK,
						Vars:             nil,
						LastRequestUser:  env.GetLastRequestUser(),
					}
					if e.Err != nil {
						runEvent.Error = e.Err.Error()
						runEvent.TransitionStatus = pb.OpStatus_DONE_ERROR
					}

					the.EventWriterWithTopic(topic.Run).WriteEventWithTimestamp(runEvent, runStartCompletionTime)

				} else if e.Event == "STOP_ACTIVITY" {
					runEndCompletionTime := time.Now()
					runEndCompletionTimeStr := strconv.FormatInt(runEndCompletionTime.UnixMilli(), 10)
					env.workflow.SetRuntimeVar("run_end_completion_time_ms", runEndCompletionTimeStr)

					runEvent := &pb.Ev_RunEvent{
						EnvironmentId:    envId.String(),
						RunNumber:        env.GetCurrentRunNumber(),
						State:            env.Sm.Current(),
						Error:            "",
						Transition:       e.Event,
						TransitionStatus: pb.OpStatus_DONE_OK,
						Vars:             nil,
						LastRequestUser:  env.GetLastRequestUser(),
					}
					if e.Err != nil {
						runEvent.Error = e.Err.Error()
						runEvent.TransitionStatus = pb.OpStatus_DONE_ERROR
					}

					the.EventWriterWithTopic(topic.Run).WriteEventWithTimestamp(runEvent, runEndCompletionTime)

					// Ensure the auto stop timer is stopped (important for stop transitions NOT triggered by the timer itself)
					env.invalidateAutoStopTransition()
				} else if e.Event == "GO_ERROR" {
					endCompletionTime, ok := env.workflow.GetUserVars().Get("run_end_completion_time_ms")
					if ok && endCompletionTime == "" {
						runEndCompletionTime := time.Now()
						runEndCompletionTimeStr := strconv.FormatInt(runEndCompletionTime.UnixMilli(), 10)
						env.workflow.SetRuntimeVar("run_end_completion_time_ms", runEndCompletionTimeStr)

						the.EventWriterWithTopic(topic.Run).WriteEventWithTimestamp(&pb.Ev_RunEvent{
							EnvironmentId:    envId.String(),
							RunNumber:        env.GetCurrentRunNumber(),
							State:            env.Sm.Current(),
							Error:            "",
							Transition:       e.Event,
							TransitionStatus: pb.OpStatus_DONE_OK,
							Vars:             nil,
							LastRequestUser:  env.GetLastRequestUser(),
						}, runEndCompletionTime)

					} else {
						log.WithField("partition", envId.String()).
							Debug("O2 End Completion time already set before after_GO_ERROR")
					}
				}

				errHooks = errors.Join(errHooks, env.handleHooksWithPositiveWeights(env.Workflow(), trigger))
				if errHooks != nil {
					e.Cancel(errHooks)
				}

				errorMsg := ""
				if e.Err != nil {
					errorMsg = e.Err.Error()
				}

				if e.Event == "STOP_ACTIVITY" {
					// If the event is STOP_ACTIVITY, we remove the active run number after all hooks are done.
					env.workflow.GetVars().Set("last_run_number", strconv.Itoa(int(env.currentRunNumber)))
					env.currentRunNumber = 0
					env.workflow.GetVars().Del("run_number")
					env.workflow.GetVars().Del("runNumber")
				}

				// publish transition step complete event
				the.EventWriterWithTopic(topic.Environment).WriteEvent(&pb.Ev_EnvironmentEvent{
					EnvironmentId:   env.id.String(),
					State:           env.Sm.Current(),
					RunNumber:       env.currentRunNumber,
					Error:           errorMsg,
					Message:         "transition step finished",
					Transition:      e.Event,
					TransitionStep:  trigger,
					LastRequestUser: env.GetLastRequestUser(),
				})
			},
		},
	)
	return
}

func (env *Environment) handleHooks(workflow workflow.Role, trigger string, weightPredicate func(callable.HookWeight) bool) (err error) {

	// Starting point: get all hooks to be started for the current trigger
	hooksMapForTrigger := workflow.GetHooksMapForTrigger(trigger)
	callsMapForAwait := env.callsPendingAwait[trigger]

	allWeightsSet := make(callable.HooksMap)
	for k, _ := range hooksMapForTrigger {
		allWeightsSet[k] = callable.Hooks{}
	}
	for k, _ := range callsMapForAwait {
		allWeightsSet[k] = callable.Hooks{}
	}
	allWeights := allWeightsSet.GetWeights()

	filteredWeights := make([]callable.HookWeight, 0)
	for _, weight := range allWeights {
		if weightPredicate(weight) {
			filteredWeights = append(filteredWeights, weight)
		}
	}

	// Prepare structures to accumulate errors
	allErrors := make(map[callable.Hook]error)
	criticalFailures := make([]error, 0)

	// FOR EACH weight within the current state machine trigger moment
	// 4 phases: start calls, await calls, execute task hooks, error handling
	for _, weight := range filteredWeights {
		hooksForWeight, thereAreHooksToStartForTheCurrentTriggerAndWeight := hooksMapForTrigger[weight]

		// PHASE 1: start asynchronously any call hooks and add them to the pending await map

		if thereAreHooksToStartForTheCurrentTriggerAndWeight {
			// Hooks can be call hooks or task hooks, we do the calls first
			callsToStart := hooksForWeight.FilterCalls()
			if len(callsToStart) != 0 {
				// Before we run anything asynchronously we must associate each call we're about
				// to start with its corresponding await expression
				for _, call := range callsToStart {
					awaitExpr := call.GetTraits().Await

					awaitName, awaitWeight := callable.ParseTriggerExpression(awaitExpr)

					// If the callsPendingAwait map has no pending calls list for the given await expression
					// (await name + await weight), we make sure the per-name map and per-weight slice are
					// created before we add any pending awaits.
					if _, ok := env.callsPendingAwait[awaitName]; !ok || len(env.callsPendingAwait[awaitName]) == 0 {
						env.callsPendingAwait[awaitName] = make(callable.CallsMap)
					}
					if _, ok := env.callsPendingAwait[awaitName][awaitWeight]; !ok || len(env.callsPendingAwait[awaitName][awaitWeight]) == 0 {
						env.callsPendingAwait[awaitName][awaitWeight] = make(callable.Calls, 0)
					}
					env.callsPendingAwait[awaitName][awaitWeight] = append(
						env.callsPendingAwait[awaitName][awaitWeight], call)
				}
				callsToStart.StartAll() // returns immediately (async)
			}
		}

		// PHASE 2: collect any calls awaiting termination

		// We take care of any pending hooks whose await expression corresponds to the current trigger,
		// including any calls that have just been started (for which trigger == call.Trigger == call.Await).
		callErrors := make(map[*callable.Call]error)
		if _, ok := env.callsPendingAwait[trigger]; ok {
			pendingCalls, ok := env.callsPendingAwait[trigger][weight]
			if ok && len(pendingCalls) != 0 { // meaning there are hook calls to take care of
				// AwaitAll blocks with no global timeout - it is up to the specific called function to implement
				// a timeout internally.
				// The Call instance pushes to the call's varStack some special values including the timeout
				// (provided by the workflow template). At that point the integration plugin must acquire the
				// timeout value and use the Context mechanism or some other approach to ensure the timeouts are
				// respected.

				callErrors = pendingCalls.AwaitAll()
				delete(env.callsPendingAwait[trigger], weight)
			}
		}

		// PHASE 3: start and finish any task hooks (synchronous!)

		taskErrors := make(map[*task.Task]error)
		if thereAreHooksToStartForTheCurrentTriggerAndWeight {

			// Tasks are handled separately for now, and they must have trigger==await
			hookTasksToTrigger := hooksForWeight.FilterTasks()
			taskErrors = env.runTasksAsHooks(hookTasksToTrigger) // blocking call, timeouts in executor
		}

		// PHASE 4: collect any errors

		thereAreCriticalErrors := false
		// We merge hook call errors and hook task errors into a single map for
		// critical trait processing
		for hook, err := range callErrors {
			allErrors[hook] = err
			if hook.GetTraits().Critical {
				thereAreCriticalErrors = true
			}
		}
		for hook, err := range taskErrors {
			allErrors[hook] = err
			if hook.GetTraits().Critical {
				thereAreCriticalErrors = true
			}
		}

		if thereAreCriticalErrors {
			break
			// if at least one critical error occurred, we stop processing hooks for the current trigger beyond the
			// current weight step
		}
	}

	for hook, err := range allErrors {
		if hook == nil || err == nil {
			continue
		}

		// If the hook call or task is critical: true
		if hook.GetTraits().Critical {
			log.WithField("partition", env.Id().String()).
				Logf(logrus.FatalLevel, "critical hook failed: %s", err) // Must use Logf(FatalLevel) instead of
			// Fatalf because the latter calls Exit
			criticalFailures = append(criticalFailures, err)
		} else {
			log.WithField("level", infologger.IL_Devel).
				WithField("partition", env.Id().String()).
				Debugf("non-critical hook failed: %s", err)
		}
	}

	if len(criticalFailures) != 0 {
		if len(criticalFailures) > 3 {
			return fmt.Errorf("%d critical hooks failed at trigger %s (see InfoLogger for details)", len(criticalFailures), trigger)
		} else if len(criticalFailures) > 1 { // 2-3 failed hooks
			consolidated := make([]string, len(criticalFailures))
			for i, cf := range criticalFailures {
				consolidated[i] = cf.Error()
			}
			consolidatedS := strings.Join(consolidated, "; ")

			return fmt.Errorf("%d critical hooks failed at trigger %s: %s", len(criticalFailures), trigger, consolidatedS)
		} else { // 1 hook failed
			return fmt.Errorf("critical hook failed at trigger %s: %s", trigger, criticalFailures[0])
		}
	}
	return nil
}

func (env *Environment) handleAllHooks(workflow workflow.Role, trigger string) (err error) {
	log.WithField("partition", env.id).Debugf("begin handling hooks for trigger %s", trigger)
	defer utils.TimeTrack(time.Now(), fmt.Sprintf("finished handling hooks for trigger %s", trigger), log.WithPrefix("env").WithField("partition", env.id))
	return env.handleHooks(workflow, trigger, func(w callable.HookWeight) bool { return true })
}

func (env *Environment) handleHooksWithNegativeWeights(workflow workflow.Role, trigger string) (err error) {
	log.WithField("partition", env.id).Debugf("begin handling hooks with negative weights for trigger %s", trigger)
	defer utils.TimeTrack(time.Now(), fmt.Sprintf("finished handling hooks with negative weights for trigger %s", trigger), log.WithPrefix("env").WithField("partition", env.id))
	return env.handleHooks(workflow, trigger, func(w callable.HookWeight) bool { return w < 0 })
}

// "positive" include 0
func (env *Environment) handleHooksWithPositiveWeights(workflow workflow.Role, trigger string) (err error) {
	log.WithField("partition", env.id).Debugf("begin handling hooks with positive weights for trigger %s", trigger)
	defer utils.TimeTrack(time.Now(), fmt.Sprintf("finished handling hooks with positive weights for trigger %s", trigger), log.WithPrefix("env").WithField("partition", env.id))
	return env.handleHooks(workflow, trigger, func(w callable.HookWeight) bool { return w >= 0 })
}

// runTasksAsHooks returns a map of failed hook tasks and their respective error values.
// The returned map includes both critical and non-critical failures, and it's up to the caller
// to further filter as needed.
func (env *Environment) runTasksAsHooks(hooksToTrigger task.Tasks) (errorMap map[*task.Task]error) {
	errorMap = make(map[*task.Task]error)

	if len(hooksToTrigger) == 0 {
		return
	}

	timeoutCh := make(chan string)
	hookTimers := make(map[string]*time.Timer)

	for _, hook := range hooksToTrigger {
		timeout, _ := time.ParseDuration(hook.GetTraits().Timeout)
		log.WithField("partition", env.Id().String()).
			WithField("task", hook.GetName()).
			WithField("taskId", hook.GetTaskId()).
			WithField("command", hook.GetTaskCommandInfo().GetValue()).
			WithField("args", hook.GetTaskCommandInfo().GetArguments()).
			WithField("failedHost", hook.GetHostname()).
			WithField("timeout", timeout.Seconds()).
			Trace("setting timer for hook before triggering")

		tid := hook.GetTaskId()
		hookTimers[tid] = time.AfterFunc(timeout,
			func() {
				timeoutCh <- tid
			})
	}

	doneCh := make(chan struct{})

	go func() {
		successfulHooks := make(task.Tasks, 0)

		for {
			select {
			case tid := <-timeoutCh:
				log.WithField("taskId", tid).Debug("incoming hook timeout")
				thisHook := hooksToTrigger.GetByTaskId(tid)
				if thisHook != nil {
					if _, hasTimer := hookTimers[tid]; !hasTimer {
						log.WithField("partition", env.Id().String()).
							WithField("task", thisHook.GetName()).
							WithField("taskId", thisHook.GetTaskId()).
							WithField("command", thisHook.GetTaskCommandInfo().GetValue()).
							WithField("args", thisHook.GetTaskCommandInfo().GetArguments()).
							WithField("failedHost", thisHook.GetHostname()).
							WithField("level", infologger.IL_Devel).
							Warn("timeout for hook but no timer in timers map")
					} else {
						log.WithField("partition", env.Id().String()).
							WithField("task", thisHook.GetName()).
							WithField("taskId", thisHook.GetTaskId()).
							WithField("command", thisHook.GetTaskCommandInfo().GetValue()).
							WithField("args", thisHook.GetTaskCommandInfo().GetArguments()).
							WithField("failedHost", thisHook.GetHostname()).
							WithField("level", infologger.IL_Devel).
							Warn("hook response timed out")
						delete(hookTimers, tid)
						errorMap[thisHook] = fmt.Errorf("hook task %s timed out after %s",
							thisHook.GetName(), thisHook.GetTraits().Timeout)
					}
				}

			case e := <-env.incomingEvents:
				if evt, ok := e.(*event.BasicTaskTerminated); ok {
					tid := evt.GetOrigin().TaskId.Value
					thisHook := hooksToTrigger.GetByTaskId(tid)
					if thisHook == nil {
						continue
					}

					hookTimers[tid].Stop()
					delete(hookTimers, tid)

					if evt.ExitCode != 0 {
						errorMap[thisHook] = fmt.Errorf("hook task %s finished with non-zero exit code %d (status %s)",
							thisHook.GetName(), evt.ExitCode, evt.FinalMesosState)

						log.WithField("task", thisHook.GetName()).
							WithFields(logrus.Fields{
								"exitCode":        evt.ExitCode,
								"stdout":          evt.Stdout,
								"stderr":          evt.Stderr,
								"partition":       env.Id().String(),
								"finalMesosState": evt.FinalMesosState.String(),
							}).
							Warn("hook failed")
					} else if !evt.VoluntaryTermination {
						errorMap[thisHook] = fmt.Errorf("hook task %s involuntary termination with exit code %d (status %s)",
							thisHook.GetName(), evt.ExitCode, evt.FinalMesosState)

						log.WithField("task", thisHook.GetName()).
							WithFields(logrus.Fields{
								"exitCode":        evt.ExitCode,
								"stdout":          evt.Stdout,
								"stderr":          evt.Stderr,
								"partition":       env.Id().String(),
								"finalMesosState": evt.FinalMesosState.String(),
							}).
							Warn("hook failed")
					} else {
						successfulHooks = append(successfulHooks, thisHook)
						log.WithField("partition", env.Id().String()).
							WithField("taskId", tid).
							WithField("task", thisHook.GetName()).
							Debug("hook completed")
					}
				}
			}

			if len(hookTimers) == 0 {
				break
			} else {
				keys := make([]string, 0)
				for k, _ := range hookTimers {
					keys = append(keys, k)
				}
				log.WithField("taskIds", strings.Join(keys, ",")).
					WithField("successfulHooks", len(successfulHooks)).
					WithField("level", infologger.IL_Devel).
					WithField("partition", env.Id().String()).
					Debugf("hook timeout timers still left: %d, next cycle", len(hookTimers))
			}
		}

		log.WithField("level", infologger.IL_Devel).
			WithField("partition", env.Id().String()).
			Debugf("hooks to trigger: %d, successful: %d", len(hooksToTrigger), len(successfulHooks))

		if len(hooksToTrigger) == len(successfulHooks) {
			errorMap = make(map[*task.Task]error)
		}
		doneCh <- struct{}{}
	}()

	err := env.hookHandlerF(hooksToTrigger)
	if err != nil {
		for _, h := range hooksToTrigger {
			errorMap[h] = err
			timer, ok := hookTimers[h.GetTaskId()]
			if ok {
				timer.Stop()
				delete(hookTimers, h.GetTaskId())
			}
		}
		return
	}

	<-doneCh

	return
}

func (env *Environment) TryTransition(t Transition) (err error) {
	if !env.transitionMutex.TryLock() {
		log.WithField("partition", env.id.String()).
			Warnf("environment transition attempt delayed: transition '%s' in progress. waiting for completion or failure", env.currentTransition)
		env.transitionMutex.Lock()
	}
	defer env.transitionMutex.Unlock()

	the.EventWriterWithTopic(topic.Environment).WriteEvent(&pb.Ev_EnvironmentEvent{
		EnvironmentId:   env.id.String(),
		State:           env.Sm.Current(),
		RunNumber:       env.currentRunNumber,
		Message:         "transition starting",
		Transition:      t.eventName(),
		LastRequestUser: env.GetLastRequestUser(),
	})

	err = t.check()
	if err != nil {
		the.EventWriterWithTopic(topic.Environment).WriteEvent(&pb.Ev_EnvironmentEvent{
			EnvironmentId:   env.id.String(),
			State:           env.Sm.Current(),
			RunNumber:       env.currentRunNumber,
			Error:           err.Error(),
			Message:         "transition impossible",
			Transition:      t.eventName(),
			LastRequestUser: env.GetLastRequestUser(),
		})
		return
	}
	err = env.Sm.Event(context.Background(), t.eventName(), t)

	if err != nil {
		the.EventWriterWithTopic(topic.Environment).WriteEvent(&pb.Ev_EnvironmentEvent{
			EnvironmentId:   env.id.String(),
			State:           env.Sm.Current(),
			RunNumber:       env.currentRunNumber,
			Error:           err.Error(),
			Message:         "transition error",
			Transition:      t.eventName(),
			LastRequestUser: env.GetLastRequestUser(),
		})
	} else {
		the.EventWriterWithTopic(topic.Environment).WriteEvent(&pb.Ev_EnvironmentEvent{
			EnvironmentId:   env.id.String(),
			State:           env.Sm.Current(),
			RunNumber:       env.currentRunNumber,
			Message:         "transition completed successfully",
			Transition:      t.eventName(),
			LastRequestUser: env.GetLastRequestUser(),
		})
	}
	return
}

func (env *Environment) handlerFunc() func(e *fsm.Event) {
	if env == nil {
		return nil
	}
	return func(e *fsm.Event) {
		if e.Err != nil { // If the event was already cancelled
			return
		}
		log.WithFields(logrus.Fields{
			"event":     e.Event,
			"src":       e.Src,
			"dst":       e.Dst,
			"partition": env.id.String(),
		}).Debug("environment.sm starting transition")

		if len(e.Args) == 0 {
			e.Cancel(errors.New("transition missing in FSM event"))
			return
		}
		transition, ok := e.Args[0].(Transition)
		if !ok {
			e.Cancel(errors.New("transition wrapping error"))
			return
		}

		if transition.eventName() == e.Event {
			transErr := transition.do(env)
			if transErr != nil {
				e.Cancel(transErr)
			}
		}
	}
}

// Accessors

func (env *Environment) Id() uid.ID {
	if env == nil {
		return uid.NilID()
	}
	env.Mu.RLock()
	defer env.Mu.RUnlock()
	return env.id
}

func (env *Environment) CreatedWhen() time.Time {
	if env == nil {
		return time.Unix(0, 0)
	}
	env.Mu.RLock()
	defer env.Mu.RUnlock()
	return env.ts
}

func (env *Environment) CurrentState() string {
	if env == nil {
		return ""
	}
	env.Mu.RLock()
	defer env.Mu.RUnlock()
	return env.Sm.Current()
}

func (env *Environment) CurrentTransition() string {
	if env == nil {
		return ""
	}
	env.Mu.RLock()
	defer env.Mu.RUnlock()
	return env.currentTransition
}

func (env *Environment) SetLastRequestUser(lastRequestUser *pb.User) {
	if env == nil {
		return
	}
	lastRequestUserJ, err := json.Marshal(lastRequestUser)
	if err == nil {
		env.UserVars.Set("last_request_user", string(lastRequestUserJ[:]))
	}
}

func (env *Environment) GetLastRequestUser() *pb.User {
	if env == nil {
		return nil
	}
	lastRequestUser := &pb.User{}
	lastRequestUserJ, ok := env.UserVars.Get("last_request_user")
	if ok {
		_ = json.Unmarshal([]byte(lastRequestUserJ), lastRequestUser)
	}
	return lastRequestUser
}

func (env *Environment) IsSafeToStop() bool {
	tasks := env.Workflow().GetTasks()
	for _, t := range tasks {
		if !t.IsSafeToStop() {
			return false
		}
	}
	return true
}

func (env *Environment) Workflow() workflow.Role {
	if env == nil {
		return nil
	}
	env.Mu.RLock()
	defer env.Mu.RUnlock()
	return env.workflow
}

func (env *Environment) QueryRoles(pathSpec string) (rs []workflow.Role) {
	env.Mu.RLock()
	defer env.Mu.RUnlock()
	g := glob.MustCompile(pathSpec, workflow.PATH_SEPARATOR_RUNE)
	rs = env.workflow.GlobFilter(g)
	return
}

func (env *Environment) GetPath() string {
	return ""
}

func (env *Environment) GetCurrentRunNumber() (rn uint32) {
	env.Mu.RLock()
	defer env.Mu.RUnlock()

	return env.currentRunNumber
}

// setState will move environment to a given state from current state.
// The call does not trigger any callbacks, if defined.
func (env *Environment) setState(state string) {
	if env == nil {
		return
	}
	env.Mu.Lock()
	defer env.Mu.Unlock()
	env.Sm.SetState(state)
}

func (env *Environment) subscribeToWfState(taskman *task.Manager) {
	go func() {
		wf := env.Workflow()
		notify := make(chan sm.State)
		subscriptionId := uuid.NewUUID().String()
		env.wfAdapter.SubscribeToStateChange(subscriptionId, notify)
		defer env.wfAdapter.UnsubscribeFromStateChange(subscriptionId)
		env.unsubscribe = make(chan struct{})

		wfState := wf.GetState()
		if wfState != sm.ERROR {
			handlingError := false
		WORKFLOW_STATE_LOOP:
			for {
				select {
				case wfState = <-notify:
					if wfState == sm.ERROR {
						if !handlingError {
							handlingError = true

							time.AfterFunc(500*time.Millisecond, func() { // wait 0.5s for any other tasks to go to ERROR/INACTIVE
								log.WithField("partition", env.id).
									WithField("level", infologger.IL_Ops).
									Warn("one of the critical tasks went into ERROR state, transitioning the environment into ERROR")
								err := env.TryTransition(NewGoErrorTransition(taskman))
								if err != nil {
									if env.Sm.Current() == "ERROR" {
										log.WithField("partition", env.id).
											WithField("level", infologger.IL_Devel).
											Info("skipped requested transition to ERROR: environment already in ERROR state")
									} else {
										log.WithField("partition", env.id).
											WithError(err).
											WithField("level", infologger.IL_Devel).
											Warn("could not transition gently to ERROR, forcing it")
										env.setState(wfState.String())
									}
								}
								toStop := env.Workflow().GetTasks().Filtered(func(t *task.Task) bool {
									t.SetSafeToStop(true)
									return t.IsSafeToStop()
								})
								if len(toStop) > 0 {
									taskmanMessage := task.NewTransitionTaskMessage(
										toStop,
										sm.RUNNING.String(),
										sm.STOP.String(),
										sm.CONFIGURED.String(),
										nil,
										env.Id(),
									)
									taskman.MessageChannel <- taskmanMessage
									<-env.stateChangedCh
								}
							})
							break WORKFLOW_STATE_LOOP
						}
					}
					if wfState == sm.DONE {
						break WORKFLOW_STATE_LOOP
					}
				case <-env.unsubscribe:
					break WORKFLOW_STATE_LOOP
				}
			}
		}
	}()
}

func (env *Environment) unsubscribeFromWfState() {
	// Use select to unblock in case the above goroutine
	// exits due to an ERROR state. If that's the case
	// we close the channel.
	env.once.Do(func() {
		select {
		case env.unsubscribe <- struct{}{}:
		default:
			if env.unsubscribe != nil {
				close(env.unsubscribe)
			}
		}
	})
}

func (env *Environment) addSubscription(sub Subscription) {
	env.Mu.Lock()
	env.eventStream = sub
	env.Mu.Unlock()
}

func (env *Environment) sendEnvironmentEvent(ev event.Event) {
	env.Mu.Lock()
	if env.eventStream != nil {
		env.eventStream.Send(ev)
	}
	env.Mu.Unlock()
}

func (env *Environment) closeStream() {
	env.Mu.Lock()
	if env.eventStream != nil {
		env.eventStream.Unsubscribe()
		env.eventStream = nil
	}
	env.Mu.Unlock()
}

func (env *Environment) GetKV(path, key string) string {
	if env == nil {
		return ""
	}
	if len(path) == 0 { // empty path provided, we default to root item of current env workflow
		path = env.workflow.GetName()
	}
	rolesForPath := env.QueryRoles(path)
	if len(rolesForPath) == 0 {
		return ""
	}
	env.Mu.RLock()
	defer env.Mu.RUnlock()
	role := rolesForPath[0]
	varStack, err := role.ConsolidatedVarStack()
	if err != nil {
		return ""
	}
	payload := varStack[key]
	return payload
}

func (env *Environment) GetActiveDetectors() (response system.IDMap) {
	if env == nil || env.workflow == nil {
		return nil
	}
	response = make(system.IDMap)

	payload := env.GetKV("", "detectors")
	slice, err := JSONSliceToSlice(payload)
	if err != nil {
		return
	}
	for _, det := range slice {
		sid, err := system.IDString(det) // generated by enumer
		if err != nil {
			continue
		}
		response[sid] = struct{}{}
	}
	return
}

func (env *Environment) GetFLPs() []string {
	if env == nil || env.workflow == nil {
		return nil
	}
	payload := env.GetKV("", "hosts")
	response, err := JSONSliceToSlice(payload)
	if err != nil {
		return []string{}
	}
	return response
}

func (env *Environment) GetAllHosts() []string {
	if env == nil || env.workflow == nil {
		return nil
	}

	tasks := env.workflow.GetTasks()
	hostSet := make(map[string]struct{})
	for _, t := range tasks {
		hostSet[t.GetHostname()] = struct{}{}
	}

	out := make([]string, len(hostSet))
	i := 0
	for hostname, _ := range hostSet {
		out[i] = hostname
		i++
	}
	sort.Strings(out)
	return out
}

func (env *Environment) GetRunType() runtype.RunType {
	if env == nil || env.workflow == nil {
		return runtype.NONE
	}
	rtString := env.GetKV("", "run_type")
	rt, err := runtype.RunTypeString(rtString)
	if err != nil {
		log.WithField("partition", env.id).
			WithField("level", infologger.IL_Support).
			WithError(err).
			Warnf("invalid run type %s", rtString)
		return runtype.NONE
	}
	return rt
}

func (env *Environment) GetVarsAsString() string {
	if env == nil {
		return ""
	}
	path := env.workflow.GetName()
	rolesForPath := env.QueryRoles(path)
	if len(rolesForPath) == 0 {
		return ""
	}
	env.Mu.RLock()
	defer env.Mu.RUnlock()
	role := rolesForPath[0]
	varStack, err := role.ConsolidatedVarStack()
	if err != nil {
		return ""
	}
	for _, r := range role.GetRoles() {
		if r.GetName() == "odc" {
			epnWorkflowCmdResult, ok := r.GetVars().Get("odc_script")
			if !ok {
				epnWorkflowCmdResult = ""
			}
			varStack["odc_script"] = epnWorkflowCmdResult
			return sortMapToString(varStack)
		}
	}
	return sortMapToString(varStack)
}

// return true if the auto stop transition has been scheduled, false otherwise
// if true, the expected stop time is also returned
func (env *Environment) scheduleAutoStopTransition() (scheduled bool, expected time.Time) {
	autoStopEnabled := env.GetKV("", "auto_stop_enabled")
	if autoStopEnabled == "true" {
		autoStopTimeout := env.GetKV("", "auto_stop_timeout")
		if autoStopTimeout != "" {
			// if auto stop is enabled, parse the timeout, start a timer
			// and start a go routine that will try a STOP transition after the timeout
			autoStopDuration, err := time.ParseDuration(autoStopTimeout)
			if err != nil {
				log.WithField("partition", env.id).
					WithField("run", env.currentRunNumber).
					Errorf("Auto stop duration string parsing failed: %s", err.Error())
				return
			}

			env.autoStopTimer = time.NewTimer(autoStopDuration)
			go func() {
				select {
				case <-env.autoStopTimer.C:
					log.WithField("partition", env.id).
						WithField("run", env.currentRunNumber).
						Infof("Executing scheduled auto stop transition following expiration of %s", autoStopDuration)
					err = env.TryTransition(NewStopActivityTransition(ManagerInstance().taskman))
					if err != nil {
						log.WithField("partition", env.id).
							WithField("run", env.currentRunNumber).
							Errorf("Scheduled auto stop transition failed: %s, Transitioning into ERROR", err.Error())
						err = env.TryTransition(NewGoErrorTransition(ManagerInstance().taskman))
						if err != nil {
							log.WithField("partition", env.id).
								WithField("run", env.currentRunNumber).
								Errorf("Forced transition to ERROR failed: %s", err.Error())
							env.setState("ERROR")
						}
						return
					}
				}
			}()

			// if registered
			scheduled = true
			expected = time.Now().Add(autoStopDuration)
			return
		}
	}
	return
}

func (env *Environment) invalidateAutoStopTransition() {
	// Only try to stop an initialized timer
	if env.autoStopTimer != nil {
		env.autoStopTimer.Stop()
	}
}
