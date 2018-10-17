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

// Package task defines the Task type and its Manager, with the purpose
// of handling the lifetime of O² Task objects.
// Each Task generally matches a running Mesos Task.
// All Tasks are kept in a roster in Manager, and the latter also takes
// care of resource acquisition and deployment.
package task

import (
	"github.com/AliceO2Group/Control/common"
		"github.com/AliceO2Group/Control/common/logger"
	"github.com/sirupsen/logrus"
	"github.com/pborman/uuid"
	"github.com/AliceO2Group/Control/core/controlcommands"
	"github.com/AliceO2Group/Control/common/controlmode"
)

var log = logger.New(logrus.StandardLogger(),"task")

type VarMap map[string]string

type parentRole interface {
	UpdateStatus(Status)
	UpdateState(State)
	GetPath() string
	GetTaskClass() string
	SetTask(*Task)
	GetEnvironmentId() uuid.Array
	//GetVars()
}

type Task struct {
	parent       parentRole
	className    string
	//configuration Descriptor
	name         string
	hostname     string
	agentId      string
	offerId      string
	taskId       string
	executorId   string

	bindPorts    map[string]uint64

	getTaskClass func() *TaskClass
	// ↑ to be filled in by NewTaskForMesosOffer in Manager
}

func (t Task) IsLocked() bool {
	return len(t.hostname) > 0 &&
		   len(t.agentId) > 0 &&
		   len(t.offerId) > 0 &&
		   len(t.taskId) > 0 &&
		   len(t.executorId) > 0 &&
		   t.parent != nil
}

func (t *Task) GetName() string {
	if t != nil {
		return t.name
	}
	return ""
}

func (t *Task) GetClassName() string {
	if t != nil {
		return t.className
	}
	return ""
}

// Returns a consolidated CommandInfo for this Task, based on Roles tree and
// TaskClass.
func (t Task) BuildTaskCommand() (cmd *common.TaskCommandInfo) {
	if class := t.getTaskClass(); class != nil {
		cmd = &common.TaskCommandInfo{}
		cmd.CommandInfo = *class.Command.Copy()
		if class.Control.Mode == controlmode.FAIRMQ {
			// FIXME read this from configuration
			cmd.Arguments = append(cmd.Arguments,
				"-S", "$CONTROL_OCCPLUGIN_ROOT/lib/",
				"-P", "OCC",
				"--id", t.GetTaskId(),
				"--color", "false")
		}
		cmd.ControlMode = class.Control.Mode
	} else {
		cmd = &common.TaskCommandInfo{}
	}
	return
}

func (t *Task) GetWantsCPU() float64 {
	if t != nil {
		if tt := t.getTaskClass(); tt != nil {
			return *tt.Wants.Cpu
		}
	}
	return -1
}

func (t *Task) GetWantsMemory() float64 {
	if t != nil {
		if tt := t.getTaskClass(); tt != nil {
			return *tt.Wants.Memory
		}
	}
	return -1
}

func (t *Task) GetWantsPorts() Ranges {
	if t != nil {
		if tt := t.getTaskClass(); tt != nil {
			wantsPorts := make(Ranges, len(tt.Wants.Ports))
			copy(wantsPorts, tt.Wants.Ports)
			return wantsPorts
		}
	}
	return nil
}

func (t Task) GetOfferId() string {
	return t.offerId
}

func (t Task) GetTaskId() string {
	return t.taskId
}

func (t Task) GetExecutorId() string {
	return t.executorId
}

func (t Task) GetAgentId() string {
	return t.agentId
}

func (t Task) GetHostname() string {
	return t.hostname
}

func (t Task) GetEnvironmentId() uuid.Array {
	if t.parent == nil {
		return uuid.NIL.Array()
	}
	return t.parent.GetEnvironmentId()
}

func (t Task) GetBindPorts() map[string]uint64 {
	return t.bindPorts
}

func (t Task) BuildPropertyMap() controlcommands.PropertyMap {
	//parentMap := t.parent.GetPropertyMap()
	//FIXME support properties
	propMap := make(controlcommands.PropertyMap)
	return propMap
}