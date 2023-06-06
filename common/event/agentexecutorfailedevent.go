/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2023 CERN and copyright holders of ALICE O².
 * Author: Teo Mrnjavac <teo.m@cern.ch>
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

package event

import mesos "github.com/mesos/mesos-go/api/v1/lib"

type ExecutorFailedEvent struct {
	eventBase
	ExecutorId mesos.ExecutorID `json:"executorId"`
}

func NewExecutorFailedEvent(executorId *mesos.ExecutorID) *ExecutorFailedEvent {
	return &ExecutorFailedEvent{
		eventBase:  *newDeviceEventBase("ExecutorFailedEvent", nil),
		ExecutorId: *executorId,
	}
}

func (e *ExecutorFailedEvent) GetName() string {
	return e.ExecutorId.GetValue()
}

func (e *ExecutorFailedEvent) GetId() mesos.ExecutorID {
	return e.ExecutorId
}

type AgentFailedEvent struct {
	eventBase
	AgentId mesos.AgentID `json:"agentId"`
}

func NewAgentFailedEvent(agentId *mesos.AgentID) *AgentFailedEvent {
	return &AgentFailedEvent{
		eventBase: *newDeviceEventBase("AgentFailedEvent", nil),
		AgentId:   *agentId,
	}
}

func (e *AgentFailedEvent) GetName() string {
	return e.AgentId.GetValue()
}

func (e *AgentFailedEvent) GetId() mesos.AgentID {
	return e.AgentId
}
