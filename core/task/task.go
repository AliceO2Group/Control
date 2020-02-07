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
	"github.com/AliceO2Group/Control/common/controlmode"
	"github.com/AliceO2Group/Control/common/gera"
	"github.com/AliceO2Group/Control/common/logger"
	"github.com/AliceO2Group/Control/core/controlcommands"
	"github.com/AliceO2Group/Control/core/task/channel"
	"github.com/mesos/mesos-go/api/v1/lib"
	"github.com/pborman/uuid"
	"github.com/sirupsen/logrus"
)

var log = logger.New(logrus.StandardLogger(),"task")

type parentRole interface {
	UpdateStatus(Status)
	UpdateState(State)
	GetPath() string
	GetTaskClass() string
	SetTask(*Task)
	GetEnvironmentId() uuid.Array
	CollectOutboundChannels() []channel.Outbound
	GetDefaults() gera.StringMap
	GetVars() gera.StringMap
	GetUserVars() gera.StringMap
}

/*
type Task interface {
	GetParentRole() interface{}
	GetParentRolePath() string
	IsLocked() bool
	GetName() string
	GetClassName() string
	BuildTaskCommand() *common.TaskCommandInfo
	GetWantsCPU() float64
	GetWantsMemory() float64
	GetWantsPorts() Ranges
	GetOfferId() string
	GetTaskId() string
	GetExecutorId() string
	GetAgentId() string
	GetHostname() string
	GetEnvironmentId() uuid.Array
	GetBindPorts() map[string]uint64
	BuildPropertyMap(bindMap channel.BindMap) controlcommands.PropertyMap
	GetMesosCommandTarget() controlcommands.MesosCommandTarget
}*/


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

	status       Status
	state        State
	safeToStop   bool

	properties   gera.StringMap

	GetTaskClass func() *TaskClass
	// ↑ to be filled in by NewTaskForMesosOffer in Manager
}

func (t *Task) IsSafeToStop() bool {
	if t.GetTaskClass().Control.Mode != controlmode.BASIC {
		return t.state == RUNNING
	}
	return t.state == RUNNING && t.safeToStop
}

func (t *Task) SetSafeToStop(done bool) {
	t.safeToStop = done
}

func (t *Task) GetParentRole() interface{} {
	return t.parent
}

func (t *Task) GetParentRolePath() string {
	return t.parent.GetPath()
}

func (t *Task) IsLocked() bool {
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
	if class := t.GetTaskClass(); class != nil {
		cmd = &common.TaskCommandInfo{}
		cmd.CommandInfo = *class.Command.Copy()
		if class.Control.Mode == controlmode.FAIRMQ {
			// FIXME read this from configuration
			contains := func(s []string, str string) bool {
				for _, a := range s {
					if a == str {
						return true
					}
				}
				return false
			}
			// if the task class doesn't provide an id, we generate one ourselves
			if !contains(cmd.Arguments, "--id") {
				cmd.Arguments = append(cmd.Arguments, "--id", t.GetTaskId())
			}
			cmd.Arguments = append(cmd.Arguments,
				"-S", "$CONTROL_OCCPLUGIN_ROOT/lib/",
				"-P", "OCC",
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
		if tt := t.GetTaskClass(); tt != nil {
			return *tt.Wants.Cpu
		}
	}
	return -1
}

func (t *Task) GetWantsMemory() float64 {
	if t != nil {
		if tt := t.GetTaskClass(); tt != nil {
			return *tt.Wants.Memory
		}
	}
	return -1
}

func (t *Task) GetWantsPorts() Ranges {
	if t != nil {
		if tt := t.GetTaskClass(); tt != nil {
			wantsPorts := make(Ranges, len(tt.Wants.Ports))
			copy(wantsPorts, tt.Wants.Ports)
			return wantsPorts
		}
	}
	return nil
}

func (t *Task) GetOfferId() string {
	return t.offerId
}

func (t *Task) GetTaskId() string {
	return t.taskId
}

func (t *Task) GetExecutorId() string {
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

func (t Task) BuildPropertyMap(bindMap channel.BindMap) controlcommands.PropertyMap {
	//parentMap := t.parent.GetPropertyMap()
	//FIXME support parent properties

	propMap := make(controlcommands.PropertyMap)
	if class := t.GetTaskClass(); class != nil {
		if class.Control.Mode == controlmode.FAIRMQ {
			for _, inbCh := range class.Bind {
				port, ok := t.bindPorts[inbCh.Name]
				if !ok {
					log.WithFields(logrus.Fields{
							"channelName": inbCh.Name,
							"taskName": t.name,
						}).
						Error("port not allocated for inbound channel")
					continue
				}

				// We get the FairMQ-formatted propertyMap from the inbound channel spec
				chanProps := inbCh.ToFMQMap(port)

				// And we copy it into the task's propertyMap
				for k, v := range chanProps {
					propMap[k] = v
				}
			}

			for _, outboundCh := range t.parent.CollectOutboundChannels() {
				// We get the FairMQ-formatted propertyMap from the outbound channel spec
				chanProps := outboundCh.ToFMQMap(bindMap)

				// And if valid, we copy it into the task's propertyMap
				if len(chanProps) > 0 {
					for k, v := range chanProps {
						propMap[k] = v
					}
				}
			}
		}
	}

	return propMap
}

func (t Task) GetMesosCommandTarget() controlcommands.MesosCommandTarget {
	return controlcommands.MesosCommandTarget{
		AgentId: mesos.AgentID{
			Value: t.GetAgentId(),
		},
		ExecutorId: mesos.ExecutorID{
			Value: t.GetExecutorId(),
		},
		TaskId: mesos.TaskID{
			Value: t.GetTaskId(),
		},
	}
}

func (t *Task) GetProperties() map[string]string {
	propertiesMap, err := t.properties.Flattened()
	if err != nil {
		return make(map[string]string)
	}
	return propertiesMap
	// FIXME: this should merge TaskClass properties and properties acquired from the workflow
}