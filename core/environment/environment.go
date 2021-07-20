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
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/AliceO2Group/Control/common/event"
	"github.com/AliceO2Group/Control/common/gera"
	"github.com/AliceO2Group/Control/common/logger"
	"github.com/AliceO2Group/Control/common/utils/uid"
	"github.com/AliceO2Group/Control/core/task"
	"github.com/AliceO2Group/Control/core/the"
	"github.com/AliceO2Group/Control/core/workflow"
	"github.com/AliceO2Group/Control/core/workflow/callable"
	"github.com/gobwas/glob"
	"github.com/looplab/fsm"
	"github.com/pborman/uuid"
	"github.com/sirupsen/logrus"
)

var log = logger.New(logrus.StandardLogger(),"env")


type Environment struct {
	Mu               sync.RWMutex
	once             sync.Once
	Sm               *fsm.FSM
	name             string
	id               uid.ID
	ts               time.Time
	workflow         workflow.Role
	wfAdapter        *workflow.ParentAdapter
	currentRunNumber uint32
	hookHandlerF     func(hooks task.Tasks) error
	incomingEvents   chan event.DeviceEvent

	GlobalDefaults gera.StringMap // From Consul
	GlobalVars     gera.StringMap // From Consul
	UserVars       gera.StringMap // From user input
	stateChangedCh chan *event.TasksStateChangedEvent
	unsubscribe    chan struct{}
	eventStream    Subscription
	Public         bool // From workflow or user

	callsPendingAwait map[string /*await expression*/]callable.Calls
}

func (env *Environment) NotifyEvent(e event.DeviceEvent) {
	if e != nil && env.incomingEvents != nil {
		select {
		case env.incomingEvents <- e:
		default:
		}
	}
}

func newEnvironment(userVars map[string]string) (env *Environment, err error) {
	envId := uid.New()
	env = &Environment{
		id: envId,
		workflow: nil,
		ts:  time.Now(),
		incomingEvents: make(chan event.DeviceEvent),
		// Every Environment instantiation performs a ConfSvc query for defaults and vars
		// these key-values stay frozen throughout the lifetime of the environment
		GlobalDefaults: gera.MakeStringMapWithMap(the.ConfSvc().GetDefaults()),
		GlobalVars:     gera.MakeStringMapWithMap(the.ConfSvc().GetVars()),
		UserVars:       gera.MakeStringMapWithMap(userVars),
		stateChangedCh: make(chan *event.TasksStateChangedEvent),

		callsPendingAwait: make(map[string]callable.Calls),
	}

	// Make the KVs accessible to the workflow via ParentAdapter
    env.wfAdapter = workflow.NewParentAdapter(
        func() uid.ID { return env.Id() },
        func() uint32 { return env.GetCurrentRunNumber() },
		func() gera.StringMap { return env.GlobalDefaults },
		func() gera.StringMap { return env.GlobalVars },
		func() gera.StringMap { return env.UserVars },
		func(ev event.Event) { 
			if env.eventStream != nil{
				env.eventStream.Send(ev)
			}},
    	)
	env.Sm = fsm.NewFSM(
		"STANDBY",
		fsm.Events{
			{Name: "DEPLOY",         Src: []string{"STANDBY"},                   Dst: "DEPLOYED"},
			{Name: "CONFIGURE",      Src: []string{"DEPLOYED"},                  Dst: "CONFIGURED"},
			{Name: "RESET",          Src: []string{"CONFIGURED"},                Dst: "DEPLOYED"},
			{Name: "START_ACTIVITY", Src: []string{"CONFIGURED"},                Dst: "RUNNING"},
			{Name: "STOP_ACTIVITY",  Src: []string{"RUNNING"},                   Dst: "CONFIGURED"},
			{Name: "EXIT",           Src: []string{"CONFIGURED", "DEPLOYED", "STANDBY"},     Dst: "DONE"},
			{Name: "GO_ERROR",       Src: []string{"CONFIGURED", "DEPLOYED", "RUNNING"},     Dst: "ERROR"},
			{Name: "RECOVER",        Src: []string{"ERROR"},                     Dst: "DEPLOYED"},
		},
		fsm.Callbacks{
			"before_event": func(e *fsm.Event) {
				errHooks := env.handleHooks(env.Workflow(), fmt.Sprintf("before_%s", e.Event))
				if errHooks != nil {
					e.Cancel(errHooks)
				}
			},
			"leave_state": func(e *fsm.Event) {
				errHooks := env.handleHooks(env.Workflow(), fmt.Sprintf("leave_%s", e.Src))
				if errHooks != nil {
					e.Cancel(errHooks)
					return
				}

				env.handlerFunc()(e)
			},
			"enter_state": func(e *fsm.Event) {
				errHooks := env.handleHooks(env.Workflow(), fmt.Sprintf("enter_%s", e.Dst))
				if errHooks != nil {
					e.Cancel(errHooks)
					return
				}

				log.WithFields(logrus.Fields{
					"event":     e.Event,
					"src":       e.Src,
					"dst":       e.Dst,
					"partition": envId,
				}).Debug("environment.sm entering state")
			},
			"after_event": func(e *fsm.Event) {
				errHooks := env.handleHooks(env.Workflow(), fmt.Sprintf("after_%s", e.Event))
				if errHooks != nil {
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
			},
		},
	)
	return
}

func (env *Environment) handleHooks(workflow workflow.Role, trigger string) (err error) {
	// First we start any tasks
	allHooks := workflow.GetHooksForTrigger(trigger)
	callsToStart := allHooks.FilterCalls()
	if len(callsToStart) != 0 {
		// Before we run anything asynchronously we must associate each call we're about
		// to start with its corresponding await expression
		for _, call := range callsToStart {
			awaitExpr := call.GetTraits().Await
			if _, ok := env.callsPendingAwait[awaitExpr]; !ok || len(env.callsPendingAwait[awaitExpr]) == 0 {
				env.callsPendingAwait[awaitExpr] = make(callable.Calls, 0)
			}
			env.callsPendingAwait[awaitExpr] = append(env.callsPendingAwait[awaitExpr], call)
		}
		callsToStart.StartAll()
	}

	// Then we take care of any pending hooks, including from the current trigger
	// TODO: this should be further refined by adding priority/weight
	pendingCalls, ok := env.callsPendingAwait[trigger]
	callErrors := make(map[*callable.Call]error)
	if ok && len(pendingCalls) != 0 { // there are hooks to take care of
		callErrors = pendingCalls.AwaitAll()
	}

	// Tasks are handled separately for now, and they cannot have trigger!=await
	hooksToTrigger := allHooks.FilterTasks()
	taskErrors := env.runTasksAsHooks(hooksToTrigger)

	allErrors := make(map[callable.Hook]error)
	criticalFailures := make([]error, 0)

	// We merge hook call errors and hook task errors into a single map for
	// critical trait processing
	for hook, err := range callErrors {
		allErrors[hook] = err
	}
	for hook, err := range taskErrors {
		allErrors[hook] = err
	}

	for hook, err := range allErrors {
		if hook == nil || err == nil {
			continue
		}

		// If the hook call or task is critical: true
		if hook.GetTraits().Critical {
			log.Errorf("critical hook failed: %s", err)
			criticalFailures = append(criticalFailures, err)
		}
	}

	if len(criticalFailures) != 0 {
		return fmt.Errorf("one or more critical hooks failed")
	}
	return nil
}

// runTasksAsHooks returns a map of failed hook tasks and their respective error values.
// The returned map includes both critical and non-critical failures, and it's up to the caller
// to further filter as needed.
func (env *Environment) runTasksAsHooks(hooksToTrigger task.Tasks) (errorMap map[*task.Task]error) {
	errorMap = make(map[*task.Task]error)

	if len(hooksToTrigger) == 0 {
		return
	}

	err := env.hookHandlerF(hooksToTrigger)
	if err != nil {
		for _, h := range hooksToTrigger {
			errorMap[h] = err
		}
		return
	}

	timeoutCh := make(chan *task.Task)
	hookTimers := make(map[*task.Task]*time.Timer)

	for _, hook := range hooksToTrigger {
		timeout, _ := time.ParseDuration(hook.GetTraits().Timeout)
		hookTimers[hook] = time.AfterFunc(timeout,
			func() {
				thisHook := hook
				timeoutCh <- thisHook
			})
	}

	successfulHooks := make(task.Tasks, 0)

	for {
		select {
		case e := <-env.incomingEvents:
			switch evt := e.(type) {
			case *event.BasicTaskTerminated:
				tid := evt.GetOrigin().TaskId
				thisHook := hooksToTrigger.GetByTaskId(tid.Value)
				if thisHook == nil {
					continue
				}

				hookTimers[thisHook].Stop()
				delete(hookTimers, thisHook)

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
						WithField("task", thisHook.GetName()).Trace("hook completed")
				}

			default:
				continue
			}
		case thisHook := <-timeoutCh:
			log.WithField("partition", env.Id().String()).
				WithField("task", thisHook.GetName()).Warn("hook response timed out")
			delete(hookTimers, thisHook)
			errorMap[thisHook] = fmt.Errorf("hook task %s timed out after %s",
				thisHook.GetName(), thisHook.GetTraits().Timeout)
		}

		if len(hookTimers) == 0 {
			break
		}
	}

	if len(hooksToTrigger) == len(successfulHooks) {
		return make(map[*task.Task]error)
	}

	return
}

func (env *Environment) TryTransition(t Transition) (err error) {
	err = t.check()
	if err != nil {
		return
	}
	err = env.Sm.Event(t.eventName(), t)
	return
}

func (env *Environment) handlerFunc() func(e *fsm.Event) {
	if env == nil {
		return nil
	}
	return func(e *fsm.Event) {
		if e.Err != nil {	// If the event was already cancelled
			return
		}
		log.WithFields(logrus.Fields{
			"event":     e.Event,
			"src":       e.Src,
			"dst":       e.Dst,
			"partition": env.id.String(),
		}).Debug("environment.sm starting transition")

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
		return time.Unix(0,0)
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

	if env.Sm.Current() != "RUNNING" {
		return 0
	}
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
		notify := make(chan task.State)
		subscriptionId := uuid.NewUUID().String()
		env.wfAdapter.SubscribeToStateChange(subscriptionId, notify)
		defer env.wfAdapter.UnsubscribeFromStateChange(subscriptionId)
		env.unsubscribe = make(chan struct{})

		wfState := wf.GetState()
		if wfState != task.ERROR {
			WORKFLOW_STATE_LOOP:
			for {
				select {
				case wfState = <-notify:
					if wfState == task.ERROR {
						env.setState(wfState.String())
						toStop := env.Workflow().GetTasks().Filtered(func(t *task.Task) bool {
							t.SetSafeToStop(true)
							return t.IsSafeToStop()
						})
						if len(toStop) > 0 {
							taskmanMessage := task.NewTransitionTaskMessage(
							toStop,
							task.RUNNING.String(),
							task.STOP.String(),
							task.CONFIGURED.String(),
							nil,
							env.Id(),
						)
						taskman.MessageChannel <- taskmanMessage
						<-env.stateChangedCh
						}
						break WORKFLOW_STATE_LOOP
					}
					if wfState == task.DONE {
						break WORKFLOW_STATE_LOOP
					}
				case <- env.unsubscribe:
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

func (env *Environment) GetFLPs() []string {
	if env == nil {
		return nil
	}
	env.Mu.RLock()
	defer env.Mu.RUnlock()
	varStack, _ := gera.FlattenStack(
			env.GlobalDefaults,
			env.GlobalVars,
			env.UserVars,
		)
	stringFLPs := varStack["hosts"]
	stringFLPs = strings.TrimPrefix(stringFLPs, "[")
	stringFLPs = strings.TrimSuffix(stringFLPs, "]")
	stringFLPs = strings.Replace(stringFLPs,`"`,"",-1)
	return strings.Split(stringFLPs,",")
}

func (env *Environment) GetRunType() string {
	if env == nil {
		return ""
	}
	env.Mu.RLock()
	defer env.Mu.RUnlock()
	varStack, _ := gera.FlattenStack(
		env.workflow.GetDefaults(),
		env.workflow.GetVars(),
		env.workflow.GetUserVars(),
		)
	if runtype, ok := varStack["run_type"]; ok {
		return runtype

	}
	return ""
}
