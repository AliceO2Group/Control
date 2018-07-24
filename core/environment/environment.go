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
	"sync"
	"github.com/pborman/uuid"
	"github.com/looplab/fsm"
	"errors"
	"github.com/sirupsen/logrus"
	"github.com/AliceO2Group/Control/common/logger"
	"time"
)

var log = logger.New(logrus.StandardLogger(),"env")


type Environment struct {
	Mu    sync.RWMutex
	Sm    *fsm.FSM
	id    uuid.UUID
	cfg   EnvironmentCfg
	ts    time.Time
	roles []string
}


func newEnvironment() (env *Environment, err error) {
	envId := uuid.NewUUID()
	env = &Environment{
		id: envId,
		roles: []string{},
		ts:  time.Now(),
	}
	env.Sm = fsm.NewFSM(
		"ENV_STANDBY",
		fsm.Events{
			{Name: "CONFIGURE",      Src: []string{"ENV_STANDBY", "CONFIGURED"}, Dst: "CONFIGURED"},
			{Name: "START_ACTIVITY", Src: []string{"CONFIGURED"},                Dst: "RUNNING"},
			{Name: "STOP_ACTIVITY",  Src: []string{"RUNNING"},                   Dst: "CONFIGURED"},
			{Name: "EXIT",           Src: []string{"CONFIGURED", "ENV_STANDBY"}, Dst: "ENV_DONE"},
			{Name: "GO_ERROR",       Src: []string{"CONFIGURED", "RUNNING"},     Dst: "ERROR"},
			{Name: "RESET",          Src: []string{"ERROR"},                     Dst: "ENV_STANDBY"},
		},
		fsm.Callbacks{
			"enter_state": func(e *fsm.Event) {
				log.WithFields(logrus.Fields{
					"event":			e.Event,
					"src":				e.Src,
					"dst":				e.Dst,
					"environmentId": 	envId,
				}).Debug("environment.sm entering state")
			},
			"before_event": env.handlerFunc(),
		},
	)
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
		log.WithFields(logrus.Fields{
			"event":			e.Event,
			"src":				e.Src,
			"dst":				e.Dst,
			"environmentId": 	env.id.String(),
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

func (env *Environment) Configuration() EnvironmentCfg {
	if env == nil {
		return EnvironmentCfg{}
	}
	env.Mu.RLock()
	defer env.Mu.RUnlock()
	return env.cfg
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

func (env *Environment) Roles() []string {
	if env == nil {
		return nil
	}
	env.Mu.RLock()
	defer env.Mu.RUnlock()
	return env.roles
}