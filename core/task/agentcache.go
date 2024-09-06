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

package task

import (
	"github.com/mesos/mesos-go/api/v1/lib"
	"sync"
	"github.com/AliceO2Group/Control/core/task/constraint"
)

type AgentCache struct {
	mu sync.RWMutex
	store map[mesos.AgentID]AgentCacheInfo
}

type AgentCacheInfo struct{
	AgentId    mesos.AgentID
	Attributes constraint.Attributes
	Hostname   string
}

func (ac *AgentCache) Update(agents ...AgentCacheInfo) {
	if ac == nil {
		return
	}
	ac.mu.Lock()
	defer ac.mu.Unlock()

	if ac.store == nil {
		ac.store = make(map[mesos.AgentID]AgentCacheInfo)
	}

	for _, agent := range agents {
		ac.store[agent.AgentId] = agent
	}
}

func (ac *AgentCache) Get(id mesos.AgentID) (agent *AgentCacheInfo) {
	if ac == nil || ac.store == nil {
		return
	}
	ac.mu.RLock()
	defer ac.mu.RUnlock()

	aci, ok := ac.store[id]
	if ok {
		agent = &aci
	}
	return
}

func (ac *AgentCache) Count() (count int) {
	if ac == nil || ac.store == nil {
		return 0
	}
  ac.mu.RLock()
  defer ac.mu.RUnlock()
	return len(ac.store)
}
