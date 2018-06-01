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
	"errors"
	"fmt"
	"sync"
	"github.com/pborman/uuid"
)

type EnvManager struct {
	mu      sync.RWMutex
	m       map[uuid.Array]*Environment
	roleman *RoleManager
}

func NewEnvManager(rm *RoleManager) *EnvManager {
	return &EnvManager{
		m: make(map[uuid.Array]*Environment),
		roleman: rm,
	}
}

func (envs *EnvManager) CreateEnvironment(roles []string) (uuid.UUID, error) {
	envs.mu.Lock()
	defer envs.mu.Unlock()

	env, err := newEnvironment()
	if err != nil {
		return uuid.NIL, err
	}

	err = env.TryTransition(NewConfigureTransition(
		envs.roleman,
		roles,
		nil,
		true	))
	if err != nil {
		return env.id, err
	}

	envs.m[env.id.Array()] = env
	return env.id, err
}

func (envs *EnvManager) TeardownEnvironment(environmentId uuid.UUID) error {
	envs.mu.Lock()
	defer envs.mu.Unlock()

	//TODO implement

	return nil
}

func (envs *EnvManager) Configuration(environmentId uuid.UUID) EnvironmentCfg {
	envs.mu.RLock()
	defer envs.mu.RUnlock()
	return envs.m[environmentId.Array()].cfg
}

func (envs *EnvManager) Ids() (keys []uuid.UUID) {
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

func (envs *EnvManager) Environment(environmentId uuid.UUID) (env *Environment, err error) {
	envs.mu.RLock()
	defer envs.mu.RUnlock()
	env, ok := envs.m[environmentId.Array()]
	if !ok {
		err = errors.New(fmt.Sprintf("no environment with id %s", environmentId))
	}
	return
}
