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
	"strings"
	"sync"

	"github.com/AliceO2Group/Control/core/task"
	"github.com/AliceO2Group/Control/core/the"
	"github.com/AliceO2Group/Control/core/workflow"
	"github.com/pborman/uuid"
)

type Manager struct {
	mu      sync.RWMutex
	m       map[uuid.Array]*Environment
	taskman *task.Manager
}

func NewEnvManager(tm *task.Manager) *Manager {
	return &Manager{
		m:       make(map[uuid.Array]*Environment),
		taskman: tm,
	}
}

func (envs *Manager) CreateEnvironment(workflowPath string) (uuid.UUID, error) {
	envs.mu.Lock()
	defer envs.mu.Unlock()

	env, err := newEnvironment()
	if err != nil {
		return uuid.NIL, err
	}
	env.workflow, err = envs.loadWorkflow(workflowPath, env.wfAdapter)
	if err != nil {
		err = fmt.Errorf("cannot load workflow template: %s", err.Error())
		return env.id, err
	}

	envs.m[env.id.Array()] = env

	err = env.TryTransition(NewConfigureTransition(
		envs.taskman,
		nil, //roles,
		nil,
		true	))
	if err != nil {
		delete(envs.m, env.id.Array())
		return env.id, err
	}

	return env.id, err
}

func (envs *Manager) TeardownEnvironment(environmentId uuid.UUID) error {
	envs.mu.Lock()
	defer envs.mu.Unlock()

	env, err := envs.environment(environmentId)
	if err != nil {
		return err
	}

	if env.CurrentState() != "STANDBY" {
		return errors.New(fmt.Sprintf("cannot teardown environment in state %s", env.CurrentState()))
	}

	err = envs.taskman.ReleaseTasks(environmentId.Array(), env.Workflow().GetTasks())
	if err != nil {
		return err
	}

	delete(envs.m, environmentId.Array())
	return err
}

/*func (envs *Manager) Configuration(environmentId uuid.UUID) EnvironmentCfg {
	envs.mu.RLock()
	defer envs.mu.RUnlock()
	return envs.m[environmentId.Array()].cfg
}*/

func (envs *Manager) Ids() (keys []uuid.UUID) {
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

func (envs *Manager) Environment(environmentId uuid.UUID) (env *Environment, err error) {
	envs.mu.RLock()
	defer envs.mu.RUnlock()
	return envs.environment(environmentId)
}

func (envs *Manager) environment(environmentId uuid.UUID) (env *Environment, err error) {
	if len(environmentId.String()) == 0 { // invalid uuid
		return nil, fmt.Errorf("invalid uuid: %s", environmentId)
	}
	env, ok := envs.m[environmentId.Array()]
	if !ok {
		err = errors.New(fmt.Sprintf("no environment with id %s", environmentId))
	}
	return
}

func (envs *Manager) loadWorkflow(workflowPath string, parent workflow.Updatable) (root workflow.Role, err error) {
	if strings.Contains(workflowPath, "://") {
		return nil, errors.New("workflow loading from file not implemented yet")
	}
	return workflow.Load(the.ConfSvc().GetROSource(), workflowPath, parent)
}
