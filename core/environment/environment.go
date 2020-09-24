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
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/AliceO2Group/Control/common/event"
	"github.com/AliceO2Group/Control/common/gera"
	"github.com/AliceO2Group/Control/common/logger"
	"github.com/AliceO2Group/Control/core/task"
	"github.com/AliceO2Group/Control/core/the"
	"github.com/AliceO2Group/Control/core/workflow"
	"github.com/gobwas/glob"
	"github.com/looplab/fsm"
	"github.com/pborman/uuid"
	"github.com/sirupsen/logrus"
)

var log = logger.New(logrus.StandardLogger(),"env")


type Environment struct {
	Mu               sync.RWMutex
	Sm               *fsm.FSM
	name             string
	id               uuid.UUID
	ts               time.Time
	workflow         workflow.Role
	wfAdapter        *workflow.ParentAdapter
	currentRunNumber uint32
	hookHandlerF     func(hooks task.Tasks) error
	incomingEvents   chan event.DeviceEvent

	GlobalDefaults gera.StringMap // From Consul
	GlobalVars     gera.StringMap // From Consul
	UserVars       gera.StringMap // From user input
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
	envId := uuid.NewUUID()
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
	}

	// Make the KVs accessible to the workflow via ParentAdapter
    env.wfAdapter = workflow.NewParentAdapter(
    	func() uuid.Array { return env.Id().Array() },
		func() gera.StringMap { return env.GlobalDefaults },
		func() gera.StringMap { return env.GlobalVars },
		func() gera.StringMap { return env.UserVars },
    	)
	env.Sm = fsm.NewFSM(
		"STANDBY",
		fsm.Events{
			{Name: "CONFIGURE",      Src: []string{"STANDBY"},                   Dst: "CONFIGURED"},
			{Name: "RESET",          Src: []string{"CONFIGURED"},                Dst: "STANDBY"},
			{Name: "START_ACTIVITY", Src: []string{"CONFIGURED"},                Dst: "RUNNING"},
			{Name: "STOP_ACTIVITY",  Src: []string{"RUNNING"},                   Dst: "CONFIGURED"},
			{Name: "EXIT",           Src: []string{"CONFIGURED", "STANDBY"},     Dst: "DONE"},
			{Name: "GO_ERROR",       Src: []string{"CONFIGURED", "RUNNING"},     Dst: "ERROR"},
			{Name: "RECOVER",        Src: []string{"ERROR"},                     Dst: "STANDBY"},
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
			},
		},
	)
	return
}

func (env *Environment) handleHooks(workflow workflow.Role, trigger string) (err error) {
	hooksToTrigger := workflow.GetHooksForTrigger(trigger)
	if len(hooksToTrigger) == 0 {
		return nil
	}

	err = env.hookHandlerF(hooksToTrigger)
	if err != nil {
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
	failedHooksById := make(map[string]*task.Task)

	for {
		select {
		case e := <- env.incomingEvents:
			switch evt := e.(type) {
			case *event.BasicTaskTerminated:
				tid := evt.GetOrigin().TaskId
				thisHook := hooksToTrigger.GetByTaskId(tid.Value)
				if thisHook == nil {
					continue
				}

				hookTimers[thisHook].Stop()
				delete(hookTimers, thisHook)
				if evt.ExitCode != 0 || !evt.VoluntaryTermination {
					failedHooksById[thisHook.GetTaskId()] = thisHook
					log.WithField("task", thisHook.GetName()).
						WithFields(logrus.Fields{
							"exitCode": evt.ExitCode,
							"stdout": evt.Stdout,
							"stderr": evt.Stderr,
							"partition": env.Id().String(),
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
		case thisHook := <- timeoutCh:
			log.WithField("partition", env.Id().String()).
				WithField("task", thisHook.GetName()).Warn("hook response timed out")
			delete(hookTimers, thisHook)
			failedHooksById[thisHook.GetTaskId()] = thisHook
		}

		if len(hookTimers) == 0 {
			break
		}
	}

	if len(hooksToTrigger) == len(successfulHooks) {
		err = nil
		return
	}

	// We only report non-nil error if at least one CRITICAL HOOK failed
	failedCriticalHookNames := make([]string, 0)
	for _, thisHook := range failedHooksById {
		if thisHook.GetTraits().Critical {
			failedCriticalHookNames = append(failedCriticalHookNames, thisHook.GetParentRolePath())
		}
	}
	sort.Strings(failedCriticalHookNames)

	if len(failedCriticalHookNames) > 0 {
		err = fmt.Errorf("%d critical hooks failed: %s",
			len(failedCriticalHookNames),
			strings.Join(failedCriticalHookNames, ", "))
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

func (env *Environment) Id() uuid.UUID {
	if env == nil {
		return uuid.NIL
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