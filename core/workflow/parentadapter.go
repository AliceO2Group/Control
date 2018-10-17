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

package workflow

import (
	"sync"

	"github.com/AliceO2Group/Control/core/task"
	"github.com/pborman/uuid"
)

type GetEnvIdFunc func() uuid.Array

type ParentAdapter struct {
	mu sync.Mutex
	getEnvIdFunc GetEnvIdFunc
	stateSubscriptions map[string]chan task.State
	statusSubscriptions map[string]chan task.Status
}

func NewParentAdapter(getEnvId GetEnvIdFunc) *ParentAdapter {
	return &ParentAdapter{
		getEnvIdFunc: getEnvId,
		stateSubscriptions: make(map[string]chan task.State),
		statusSubscriptions: make(map[string]chan task.Status, 0),
	}
}

func (p *ParentAdapter) SubscribeToStateChange(subscriptionId string, c chan task.State) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.stateSubscriptions[subscriptionId] = c
}

func (p *ParentAdapter) UnsubscribeFromStateChange(subscriptionId string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.stateSubscriptions, subscriptionId)
}

func (p *ParentAdapter) SubscribeToStatusChange(subscriptionId string, c chan task.Status) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.statusSubscriptions[subscriptionId] = c
}

func (p *ParentAdapter) UnsubscribeFromStatusChange(subscriptionId string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.statusSubscriptions, subscriptionId)
}

func (p *ParentAdapter) updateState(s task.State) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, ch := range p.stateSubscriptions {
		select {
		case ch <- s:
		default:
		}
	}
}

func (p *ParentAdapter) updateStatus(s task.Status) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, ch := range p.statusSubscriptions {
		select {
		case ch <- s:
		default:
		}
	}
}

func (p *ParentAdapter) GetEnvironmentId() uuid.Array {
	return p.getEnvIdFunc()
}

func (*ParentAdapter) GetPath() string {
	return ""
}
