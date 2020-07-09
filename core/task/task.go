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
	"errors"
	"fmt"
	"strings"
	texttemplate "text/template"

	"github.com/AliceO2Group/Control/common"
	"github.com/AliceO2Group/Control/common/controlmode"
	"github.com/AliceO2Group/Control/common/gera"
	"github.com/AliceO2Group/Control/common/logger"
	"github.com/AliceO2Group/Control/common/utils"
	"github.com/AliceO2Group/Control/core/controlcommands"
	"github.com/AliceO2Group/Control/core/task/channel"
	"github.com/AliceO2Group/Control/core/workflow/template"
	"github.com/mesos/mesos-go/api/v1/lib"
	"github.com/pborman/uuid"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
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
	ConsolidatedVarStack() (varStack map[string]string, err error)
	CollectInboundChannels() []channel.Inbound
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
	GetLocalBindMap() map[string]uint64
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

	localBindMap channel.BindMap

	status       Status
	state        State
	safeToStop   bool

	properties   gera.StringMap

	GetTaskClass func() *Class
	// ↑ to be filled in by NewTaskForMesosOffer in Manager

	commandInfo  *common.TaskCommandInfo
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

func (t *Task) GetTaskCommandInfo() *common.TaskCommandInfo {
	return t.commandInfo
}

func (t *Task) buildSpecialVarStack(role parentRole) map[string]string {
	varStack := make(map[string]string)
	varStack["task_name"]        = t.GetName()
	varStack["task_id"]          = t.GetTaskId()
	varStack["task_class_name"]  = t.GetClassName()
	varStack["task_hostname"]    = t.GetHostname()
	varStack["environment_id"]   = role.GetEnvironmentId().UUID().String()
	varStack["task_parent_role"] = role.GetPath()
	return varStack
}

// Returns a consolidated CommandInfo for this Task, based on Roles tree and
// Class.
func (t *Task) BuildTaskCommand(role parentRole) (err error) {
	if class := t.GetTaskClass(); class != nil {
		cmd := &common.TaskCommandInfo{}
		cmd.CommandInfo = *class.Command.Copy()

		// If it's a basic task, we parametrize its arguments
		// TODO: the task payload should be shipped on CONFIGURE and not on deployment,
		//       because this way we cannot reconfigure a basic task
		// FIXME: normally we should only allow parametrizing launch-time options such
		//        as the command value and argument for BASIC tasks, as they get
		//        unique classes.
		//        In order to support non-trivial QC workflows we temporarily allow
		//        parametrizing these values for all control modes.
		//        THIS BREAKS TASK CLASS REUSE! See OCTRL-227
		if class.Control.Mode == controlmode.BASIC ||
			class.Control.Mode == controlmode.DIRECT ||
			class.Control.Mode == controlmode.FAIRMQ {
			var varStack map[string]string

			// First we get the full varStack from the parent role, and
			// consolidate it.
			varStack, err = role.ConsolidatedVarStack()
			if err != nil {
				t.commandInfo = &common.TaskCommandInfo{}
				log.WithError(err).Error("cannot fetch variables stack for task command info")
				return
			}

			// We wrap the parent varStack around the class Defaults, ensuring
			// the class Defaults are overridden by anything else.
			varStack, err = gera.MakeStringMapWithMap(varStack).WrappedAndFlattened(class.Defaults)
			if err != nil {
				log.WithError(err).Error("cannot fetch task class defaults for task command info")
				return
			}

			// Finally we build the task-specific special values, and write them
			// into the varStack (overwriting anything).
			specialVarStack := t.buildSpecialVarStack(role)
			for k, v := range specialVarStack {
				varStack[k] = v
			}

			// Prepare the fields to be subject to templating
			fields := append(
				template.Fields{
					template.WrapPointer(cmd.Value),
					template.WrapPointer(cmd.User),
				},
				append(
					template.WrapSliceItems(cmd.Env),
					template.WrapSliceItems(cmd.Arguments)...
				)...
			)
			err = fields.Execute(t.name, varStack, nil, make(map[string]texttemplate.Template))
			if err != nil {
				t.commandInfo = &common.TaskCommandInfo{}
				log.WithError(err).Error("cannot resolve templates for task command info")
			}
		}

		if class.Control.Mode == controlmode.FAIRMQ {
			// FIXME read this from configuration
			// if the task class doesn't provide an id, we generate one ourselves
			if !utils.StringSliceContains(cmd.Arguments, "--id") {
				cmd.Arguments = append(cmd.Arguments, "--id", t.GetTaskId())
			}
			cmd.Arguments = append(cmd.Arguments,
				"-S", viper.GetString("fmqPluginSearchPath"),
				"-P", viper.GetString("fmqPlugin"),
				"--color", "false")
		}

		cmd.ControlMode = class.Control.Mode
		t.commandInfo = cmd
	} else {
		t.commandInfo = &common.TaskCommandInfo{}
		err = errors.New("cannot build task command info: task class not available")
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

func (t *Task) GetAgentId() string {
	return t.agentId
}

func (t *Task) GetHostname() string {
	return t.hostname
}

func (t *Task) GetEnvironmentId() uuid.Array {
	if t.parent == nil {
		return uuid.NIL.Array()
	}
	return t.parent.GetEnvironmentId()
}

func (t *Task) GetLocalBindMap() channel.BindMap {
	return t.localBindMap
}

func (t *Task) BuildPropertyMap(bindMap channel.BindMap) (propMap controlcommands.PropertyMap) {
	propMap = make(controlcommands.PropertyMap)
	if class := t.GetTaskClass(); class != nil {
		if class.Control.Mode != controlmode.BASIC { // if it's NOT a basic task, we template the props
			if t.parent == nil {
				return
			}

			// First we get the full varStack from the parent role, and
			// consolidate it.
			varStack, err := t.parent.ConsolidatedVarStack()
			if err != nil {
				log.WithError(err).Error("cannot fetch variables stack for property map")
				return
			}

			// We wrap the parent varStack around the class Defaults, ensuring
			// the class Defaults are overridden by anything else.
			varStack, err = gera.MakeStringMapWithMap(varStack).WrappedAndFlattened(class.Defaults)
			if err != nil {
				log.WithError(err).Error("cannot fetch task class defaults for property map")
				return
			}

			// Finally we build the task-specific special values, and write them
			// into the varStack (overwriting anything).
			specialVarStack := t.buildSpecialVarStack(t.parent)
			for k, v := range specialVarStack {
				varStack[k] = v
			}

			for k, v := range t.GetProperties() {
				propMap[k] = v
			}

			objStack := make(map[string]interface{})
			objStack["GetConfig"] = template.MakeGetConfigFunc(varStack)
			objStack["ToPtree"] = template.MakeToPtreeFunc(varStack, propMap)

			fields := template.WrapMapItems(propMap)

			err = fields.Execute(t.name, varStack, objStack, make(map[string]texttemplate.Template))
			if err != nil {
				log.WithError(err).Error("cannot resolve templates for property map")
				return
			}

			// Post-processing for the ToPtree mechanism.
			// The ToPtree function has no access to the keys of propMap, so we need
			// to do a second pass here.
			// For each run of ToPtree, a temporary __ptree__:<syntax>:<xid> key is created
			// and the value of the key that pointed to ToPtree is set to this key.
			// We need to clear both of these keys, and create a new one like
			// __ptree__:<syntax>:<key> with the plain payload.
			keysToDelete := make([]string, 0)
			for k, v := range propMap {

				if strings.HasPrefix(v, "__ptree__:") {
					keysToDelete = append(keysToDelete, k, v)
					splitValue := strings.Split(v, ":")
					syntax := splitValue[1]

					propMap[fmt.Sprintf("__ptree__:%s:%s", syntax, k)] = propMap[v]
				}
			}
			for _, k := range keysToDelete {
				delete(propMap, k)
			}
		}

		// For FAIRMQ tasks, we append FairMQ channel configuration
		if class.Control.Mode == controlmode.FAIRMQ ||
			class.Control.Mode == controlmode.DIRECT {
			for _, inbCh := range channel.MergeInbound(t.parent.CollectInboundChannels(), class.Bind) {
				endpoint, ok := t.localBindMap[inbCh.Name]
				if !ok {
					log.WithFields(logrus.Fields{
							"channelName": inbCh.Name,
							"taskName": t.name,
						}).
						Error("endpoint not allocated for inbound channel")
					continue
				}

				// We get the FairMQ-formatted propertyMap from the inbound channel spec
				chanProps := inbCh.ToFMQMap(endpoint)

				// And we copy it into the task's propertyMap
				for k, v := range chanProps {
					propMap[k] = v
				}
			}
			for _, outboundCh := range channel.MergeOutbound(t.parent.CollectOutboundChannels(), class.Connect) {
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

func (t *Task) GetMesosCommandTarget() controlcommands.MesosCommandTarget {
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
	if t == nil {
		log.Warn("attempted to get properties of nil task")
		return make(map[string]string)
	}
	propertiesMap, err := t.properties.Flattened()
	if err != nil {
		return make(map[string]string)
	}
	return propertiesMap
}
