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

package event

import (
	"github.com/AliceO2Group/Control/executor/protos"
	"github.com/mesos/mesos-go/api/v1/lib"
)

type DeviceEventOrigin struct {
	// FIXME: replace these Mesos-tainted string wrappers with plain strings
	AgentId      mesos.AgentID      `json:"agentId"`
	ExecutorId   mesos.ExecutorID   `json:"executorId"`
	TaskId       mesos.TaskID       `json:"taskId"`
}

type DeviceEvent interface {
	Event
	GetOrigin() DeviceEventOrigin
	GetType() pb.DeviceEventType
}

type DeviceEventBase struct {
	eventBase
	Type        pb.DeviceEventType       `json:"type"`
	Origin      DeviceEventOrigin        `json:"origin"`
}

func (b *DeviceEventBase) GetOrigin() DeviceEventOrigin {
	if b == nil {
		return DeviceEventOrigin{}
	}
	return b.Origin
}

func (b *DeviceEventBase) GetType() pb.DeviceEventType {
	if b == nil {
		return pb.DeviceEventType_NULL_DEVICE_EVENT
	}
	return b.Type
}

func NewDeviceEvent(origin DeviceEventOrigin, t pb.DeviceEventType) (de DeviceEvent) {
	switch t {
	case pb.DeviceEventType_END_OF_STREAM:
		de = &EndOfStream{
			DeviceEventBase: DeviceEventBase{
				eventBase: *newDeviceEventBase("DeviceEvent", nil),
				Type:   t,
				Origin: origin,
			},
		}
	case pb.DeviceEventType_BASIC_TASK_TERMINATED:
		de = &BasicTaskTerminated{
			DeviceEventBase: DeviceEventBase{
				eventBase: *newDeviceEventBase("DeviceEvent", nil),
				Type:   t,
				Origin: origin,
			},
		}
	case pb.DeviceEventType_NULL_DEVICE_EVENT:
		de = nil
	}
	return de
}

type EndOfStream struct {
	DeviceEventBase
}

func (e *EndOfStream) GetName() string {
	return "END_OF_STREAM"
}

type BasicTaskTerminated struct {
	DeviceEventBase
	ExitCode int                    `json:"exitCode"`
	Stdout string                   `json:"stdout"`
	Stderr string                   `json:"stderr"`
	VoluntaryTermination bool       `json:"voluntaryTermination"`
	FinalMesosState mesos.TaskState `json:"finalMesosState"`
}

func (e *BasicTaskTerminated) GetName() string {
	return "BASIC_TASK_TERMINATED"
}