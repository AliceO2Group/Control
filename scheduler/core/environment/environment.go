/*
 * === This file is part of octl <https://github.com/teo/octl> ===
 *
 * Copyright 2017 CERN and copyright holders of ALICE O².
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
	"fmt"
	"github.com/sirupsen/logrus"
	"gitlab.cern.ch/tmrnjava/test-scheduler/scheduler/logger"
)

var log = logger.New(logrus.StandardLogger(),"env")

type Environment struct {
	Mu  sync.RWMutex
	Sm  *fsm.FSM
	id  uuid.UUID
	cfg Configuration
}

func (env *Environment) Id() uuid.UUID {
	return env.id
}

func (env *Environment) Configuration() Configuration {
	return env.cfg
}

type Environments struct {
	mu sync.RWMutex
	m  map[uuid.Array]*Environment
}

func NewEnvironments() *Environments {
	return &Environments {
		m: make(map[uuid.Array]*Environment),
	}
}

func (envs *Environments) CreateNew(configuration Configuration) (uuid.UUID, error) {
	envs.mu.Lock()
	defer envs.mu.Unlock()

	envId := uuid.NewUUID()

	env := &Environment{
		id: envId,
		Sm: fsm.NewFSM(
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
				"before_event": func(e *fsm.Event) {
					log.WithFields(logrus.Fields{
						"event":			e.Event,
						"src":				e.Src,
						"dst":				e.Dst,
						"environmentId": 	envId,
					}).Debug("environment.sm starting transition")
				},
				"enter_state": func(e *fsm.Event) {
					log.WithFields(logrus.Fields{
						"event":			e.Event,
						"src":				e.Src,
						"dst":				e.Dst,
						"environmentId": 	envId,
					}).Debug("environment.sm entering state")
				},
				"leave_ENV_STANDBY": func(e *fsm.Event) {
					if e.Event == "CONFIGURE" {
						e.Async() //transition frozen until the corresponding fsm.Transition call
					}
				},
			},
		),
		cfg: configuration,
	}

	envs.m[env.id.Array()] = env
	return env.id, nil
}

func (envs *Environments) Teardown(environmentId uuid.UUID) error {
	envs.mu.Lock()
	defer envs.mu.Unlock()

	//TODO implement

	return nil
}

func (envs *Environments) Configuration(environmentId uuid.UUID) Configuration {
	envs.mu.RLock()
	defer envs.mu.RUnlock()
	return envs.m[environmentId.Array()].cfg
}

func (envs *Environments) Ids() (keys []uuid.UUID) {
	envs.mu.RLock()
	defer envs.mu.RUnlock()
	keys = make([]uuid.UUID, len(envs.m))
	i := 0
	for k := range envs.m {
		keys[i] = k.UUID()
		i++
	}
	return
}

func (envs *Environments) Environment(environmentId uuid.UUID) (env *Environment, err error) {
	env, ok := envs.m[environmentId.Array()]
	if !ok {
		err = errors.New(fmt.Sprintf("no environment with id %s", environmentId))
	}
	return
}

// operation: move a process from one environment to another
// requirement: incremental generator of run numbers, every new activity from any env should get
// the next integer, presumably from a db