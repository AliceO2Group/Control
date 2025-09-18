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
	"strconv"
	"strings"
	"sync"
	texttemplate "text/template"
	"time"

	"github.com/AliceO2Group/Control/common"
	"github.com/AliceO2Group/Control/common/controlmode"
	"github.com/AliceO2Group/Control/common/event"
	"github.com/AliceO2Group/Control/common/event/topic"
	"github.com/AliceO2Group/Control/common/gera"
	"github.com/AliceO2Group/Control/common/logger"
	evpb "github.com/AliceO2Group/Control/common/protos"
	"github.com/AliceO2Group/Control/common/utils"
	"github.com/AliceO2Group/Control/common/utils/uid"
	"github.com/AliceO2Group/Control/configuration/template"
	"github.com/AliceO2Group/Control/core/controlcommands"
	"github.com/AliceO2Group/Control/core/task/channel"
	"github.com/AliceO2Group/Control/core/task/sm"
	"github.com/AliceO2Group/Control/core/task/taskclass"
	"github.com/AliceO2Group/Control/core/task/taskclass/port"
	"github.com/AliceO2Group/Control/core/the"
	mesos "github.com/mesos/mesos-go/api/v1/lib"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var log = logger.New(logrus.StandardLogger(), "task")

type parentRole interface {
	UpdateStatus(Status)
	UpdateState(sm.State)
	GetPath() string
	GetTaskClass() string
	GetTaskTraits() Traits
	SetTask(*Task)
	GetEnvironmentId() uid.ID
	CollectOutboundChannels() []channel.Outbound
	GetDefaults() gera.Map[string, string]
	GetVars() gera.Map[string, string]
	GetUserVars() gera.Map[string, string]
	ConsolidatedVarStack() (varStack map[string]string, err error)
	CollectInboundChannels() []channel.Inbound
	SendEvent(event.Event)
	GetName() string
}

type Traits struct {
	Trigger  string
	Await    string
	Timeout  string
	Critical bool
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
	GetEnvironmentId() uid.ID
	GetLocalBindMap() map[string]uint64
	BuildPropertyMap(bindMap channel.BindMap) controlcommands.PropertyMap
	GetMesosCommandTarget() controlcommands.MesosCommandTarget
}*/

type Task struct {
	mu        sync.RWMutex
	parent    parentRole
	className string
	// configuration Descriptor
	name       string
	hostname   string
	agentId    string
	offerId    string
	taskId     string
	executorId string

	localBindMap channel.BindMap

	status     Status
	state      sm.State
	safeToStop bool

	properties gera.Map[string, string]

	GetTaskClass func() *taskclass.Class
	// ↑ to be filled in by NewTaskForMesosOffer in Manager

	commandInfo *common.TaskCommandInfo
	pid         string
}

func (t *Task) IsSafeToStop() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if t.GetControlMode() != controlmode.BASIC {
		return t.state == sm.RUNNING
	}
	return t.state == sm.RUNNING && t.safeToStop
}

func (t *Task) SetSafeToStop(done bool) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.safeToStop = done
}

func (t *Task) GetState() sm.State {
	t.mu.Lock()
	defer t.mu.Unlock()

	return t.state
}

func (t *Task) GetParentRole() interface{} {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.parent
}

func (t *Task) getParentRolePath() string {
	if t.parent == nil {
		return ""
	}
	return t.parent.GetPath()
}

func (t *Task) GetParentRolePath() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.getParentRolePath()
}

func (t *Task) IsLocked() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.isLocked()
}

func (t *Task) isLocked() bool {
	return len(t.hostname) > 0 &&
		len(t.agentId) > 0 &&
		len(t.offerId) > 0 &&
		len(t.taskId) > 0 &&
		len(t.executorId) > 0 &&
		t.parent != nil
}

func (t *Task) IsClaimable() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return !t.isLocked() && t.status == ACTIVE && t.state == sm.STANDBY
}

func (t *Task) GetName() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if t != nil {
		return t.name
	}
	return ""
}

func (t *Task) GetClassName() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if t != nil {
		return t.className
	}
	return ""
}

func (t *Task) GetTaskCommandInfo() *common.TaskCommandInfo {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.commandInfo
}

func (t *Task) buildSpecialVarStack(role parentRole) map[string]string {
	varStack := make(map[string]string)
	varStack["task_name"] = t.GetName()
	varStack["task_id"] = t.GetTaskId()
	varStack["task_class_name"] = t.GetClassName()
	varStack["task_hostname"] = t.GetHostname()
	varStack["environment_id"] = role.GetEnvironmentId().String()
	varStack["task_parent_role"] = role.GetPath()
	return varStack
}

func (t *Task) GetControlMode() controlmode.ControlMode {
	if class := t.GetTaskClass(); class != nil {
		// If it's a BASIC task but its parent role uses it as a HOOK,
		// we modify the actual control mode of the task.
		// The class itself can never be HOOK, only BASIC
		if class.Control.Mode == controlmode.BASIC && t.GetParent() != nil {
			traits := t.GetParent().GetTaskTraits()
			if len(traits.Trigger) > 0 {
				return controlmode.HOOK
			}
		}
		return class.Control.Mode
	}
	return controlmode.DIRECT
}

func getTraits(role parentRole) Traits {
	if role != nil {
		return role.GetTaskTraits()
	}
	return Traits{}
}

func (t *Task) GetTraits() Traits {
	if class := t.GetTaskClass(); class != nil {
		parent := t.GetParent()
		if parent != nil {
			return getTraits(parent)
		}
	}
	return Traits{}
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
			class.Control.Mode == controlmode.HOOK ||
			class.Control.Mode == controlmode.DIRECT ||
			class.Control.Mode == controlmode.FAIRMQ {
			var varStack map[string]string

			// First we get the full varStack from the parent role, and
			// consolidate it.
			varStack, err = role.ConsolidatedVarStack()
			if err != nil {
				t.commandInfo = &common.TaskCommandInfo{}
				log.WithError(err).
					WithField("partition", role.GetEnvironmentId().String()).
					Error("cannot fetch variables stack for task command info")
				return
			}

			detector, ok := varStack["detector"]
			if !ok {
				detector = ""
			}

			// We first build the task-specific special values, and write them into the varStack, overwriting anything
			// coming from this task's parent role. However, these values can still be overwritten by subsequent
			// Defaults/Vars processing within the task template.
			specialVarStack := t.buildSpecialVarStack(role)
			for k, v := range specialVarStack {
				varStack[k] = v
			}

			// We make a copy of the Defaults/Vars maps from the taskClass, this is necessary because
			// entries may be templated, so they may resolve to different values in each task.
			localDefaults := class.Defaults.RawCopy()

			// We resolve any template expressions in Defaults
			defaultFields := template.WrapMapItems(localDefaults)

			err = defaultFields.Execute(the.ConfSvc(), t.name, varStack, nil, nil, make(map[string]texttemplate.Template), nil)
			if err != nil {
				t.commandInfo = &common.TaskCommandInfo{}
				log.WithError(err).
					WithField("partition", role.GetEnvironmentId().String()).
					WithField("detector", detector).
					Error("cannot resolve templates for task defaults")
				return fmt.Errorf("cannot resolve templates for task defaults: %w", err)
			}

			varStack, err = gera.MakeMapWithMap(varStack).WrappedAndFlattened(gera.MakeMapWithMap(localDefaults))
			if err != nil {
				log.WithError(err).
					WithField("partition", role.GetEnvironmentId().String()).
					WithField("detector", detector).
					Error("cannot fetch task class defaults for task command info")
				return fmt.Errorf("cannot fetch task class defaults for task command info: %w", err)
			}

			localVars := class.Vars.RawCopy()

			// We resolve any template expressions in Vars
			varFields := template.WrapMapItems(localVars)

			err = varFields.Execute(the.ConfSvc(), t.name, varStack, nil, nil, make(map[string]texttemplate.Template), nil)
			if err != nil {
				t.commandInfo = &common.TaskCommandInfo{}
				log.WithError(err).
					WithField("partition", role.GetEnvironmentId().String()).
					WithField("detector", detector).
					Error("cannot resolve templates for task vars")
				return fmt.Errorf("cannot resolve templates for task vars: %w", err)
			}

			// We wrap the parent varStack around the task's already processed Defaults,
			// ensuring that any taskclass Defaults are overridden by anything else.
			varStack, err = gera.MakeMapWithMap(varStack).WrappedAndFlattened(gera.MakeMapWithMap(localVars))
			if err != nil {
				log.WithError(err).
					WithField("partition", role.GetEnvironmentId().String()).
					WithField("detector", detector).
					Error("cannot fetch task class vars for task command info")
				return fmt.Errorf("cannot fetch task class vars for task command info: %w", err)
			}

			// Prepare the fields to be subject to templating
			fields := append(
				template.Fields{
					template.WrapPointer(cmd.Value),
					template.WrapPointer(cmd.User),
				},
				append(
					template.WrapSliceItems(cmd.Env),
					template.WrapSliceItems(cmd.Arguments)...,
				)...,
			)
			if cmd.Stdout != nil { // we only template it if it's defined
				fields = append(fields, template.WrapPointer(cmd.Stdout))
			}
			if cmd.Stderr != nil { // we only template it if it's defined
				fields = append(fields, template.WrapPointer(cmd.Stderr))
			}
			err = fields.Execute(the.ConfSvc(), t.name, varStack, nil, nil, make(map[string]texttemplate.Template), nil)
			if err != nil {
				t.commandInfo = &common.TaskCommandInfo{}
				log.WithError(err).
					WithField("partition", role.GetEnvironmentId().String()).
					WithField("detector", detector).
					Error("cannot resolve templates for task command info fields")
				return fmt.Errorf("cannot resolve templates for task command info fields: %w", err)
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

		cmd.ControlMode = t.GetControlMode() // This might change BASIC->HOOK

		// If it's a HOOK, we must pass the Timeout to the TCI for
		// executor-side timeout enforcement
		if cmd.ControlMode == controlmode.HOOK || cmd.ControlMode == controlmode.BASIC {
			traits := t.GetParent().GetTaskTraits()
			cmd.Timeout, err = time.ParseDuration(traits.Timeout)
		}

		if cmd.Stdout != nil {
			*cmd.Stdout = strings.TrimSpace(strings.ToLower(*cmd.Stdout))
			if !utils.StringSliceContains([]string{"stdout", "all", "none"}, *cmd.Stdout) {
				err = fmt.Errorf("bad stdout forwarding expression %s, allowed values: %s",
					*cmd.Stdout, "none, stdout, all")
				t.commandInfo = &common.TaskCommandInfo{}
				return
			}
		} else { // stdout not defined
			none := "none"
			cmd.Stdout = &none
		}

		if cmd.Stderr != nil {
			*cmd.Stderr = strings.TrimSpace(strings.ToLower(*cmd.Stderr))
			if !utils.StringSliceContains([]string{"stdout", "all", "none"}, *cmd.Stderr) {
				err = fmt.Errorf("bad stderr forwarding expression %s, allowed values: %s",
					*cmd.Stderr, "none, stdout, all")
				t.commandInfo = &common.TaskCommandInfo{}
				return
			}
		} else { // stderr not defined
			none := "none"
			cmd.Stderr = &none
		}

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

func (t *Task) GetWantsPorts() port.Ranges {
	if t != nil {
		if tt := t.GetTaskClass(); tt != nil {
			wantsPorts := make(port.Ranges, len(tt.Wants.Ports))
			copy(wantsPorts, tt.Wants.Ports)
			return wantsPorts
		}
	}
	return nil
}

func (t *Task) GetOfferId() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.offerId
}

func (t *Task) GetTaskId() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.taskId
}

func (t *Task) GetExecutorId() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.executorId
}

func (t *Task) GetAgentId() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.agentId
}

func (t *Task) GetHostname() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.hostname
}

func (t *Task) GetEnvironmentId() uid.ID {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.parent == nil {
		return uid.NilID()
	}
	return t.parent.GetEnvironmentId()
}

func traitsToPbTraits(traits Traits) *evpb.Traits {
	return &evpb.Traits{
		Trigger:  traits.Trigger,
		Await:    traits.Await,
		Timeout:  traits.Timeout,
		Critical: traits.Critical,
	}
}

func (t *Task) SendEvent(ev event.Event) {
	if t == nil {
		return
	}
	t.mu.RLock()
	defer t.mu.RUnlock()

	outgoingEvent := &evpb.Ev_TaskEvent{
		Name:      t.name,
		Taskid:    t.taskId,
		State:     t.state.String(),
		Status:    t.status.String(),
		Hostname:  t.hostname,
		ClassName: t.className,
		Path:      t.getParentRolePath(),
		Traits:    traitsToPbTraits(getTraits(t.parent)),
	}

	if t.parent == nil {
		the.EventWriterWithTopic(topic.Task).WriteEvent(outgoingEvent)
		return
	}

	outgoingEvent.EnvironmentId = t.parent.GetEnvironmentId().String()

	taskEvent, ok := ev.(*event.TaskEvent)
	if ok {
		if len(taskEvent.State) != 0 {
			outgoingEvent.State = taskEvent.State
		}
		if len(taskEvent.Status) != 0 {
			outgoingEvent.Status = taskEvent.Status
		}
	}
	the.EventWriterWithTopic(topic.Task).WriteEvent(outgoingEvent)

	t.parent.SendEvent(ev)
}

func (t *Task) GetLocalBindMap() channel.BindMap {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.localBindMap
}

func fillCommonProperties(t *Task, propMap controlcommands.PropertyMap) {
	envId := t.GetEnvironmentId()
	propMap["environment_id"] = envId.String()
}

func (t *Task) BuildPropertyMap(bindMap channel.BindMap) (propMap controlcommands.PropertyMap, err error) {
	propMap = make(controlcommands.PropertyMap)
	if class := t.GetTaskClass(); class != nil {
		if class.Control.Mode != controlmode.BASIC { // if it's NOT a basic task or hook, we template the props
			if t.GetParent() == nil {
				err = fmt.Errorf("cannot build property map for parentless task %s (id %s)", t.name, t.taskId)
				return
			}

			// The baseline prop list that we then complete or overwrite as needed
			fillCommonProperties(t, propMap)

			// First we get the full varStack from the parent role, and
			// consolidate it.
			var varStack map[string]string
			varStack, err = t.GetParent().ConsolidatedVarStack()
			if err != nil {
				err = fmt.Errorf("cannot fetch variables stack for property map: %w", err)
				return
			}

			detector, ok := varStack["detector"]
			if !ok {
				detector = ""
			}

			// We wrap the parent varStack around the class Defaults+Vars, ensuring
			// the class Defaults+Vars are overridden by anything else.
			classStack := gera.MakeMapWithMap(class.Vars.Raw()).Wrap(class.Defaults)
			if err != nil {
				err = fmt.Errorf("cannot fetch task class defaults for property map: %w", err)
				return
			}
			varStack, err = gera.MakeMapWithMap(varStack).WrappedAndFlattened(classStack)
			if err != nil {
				err = fmt.Errorf("cannot fetch task class vars for property map: %w", err)
				return
			}

			// Finally we build the task-specific special values, and write them
			// into the varStack (overwriting anything).
			specialVarStack := t.buildSpecialVarStack(t.GetParent())
			for k, v := range specialVarStack {
				varStack[k] = v
			}

			for k, v := range t.GetProperties() {
				propMap[k] = v
			}

			// Push orbit-reset-time if pdp_override_run_start_time set
			pdpOverrideRunStartTime, ok := varStack["pdp_override_run_start_time"]
			if ok {
				propMap["orbit-reset-time"] = pdpOverrideRunStartTime
			}

			// For FAIRMQ tasks, we append FairMQ channel configuration
			if class.Control.Mode == controlmode.FAIRMQ ||
				class.Control.Mode == controlmode.DIRECT {
				for _, inbCh := range channel.MergeInbound(t.GetParent().CollectInboundChannels(), class.Bind) {
					// We get the FairMQ-formatted propertyMap from the inbound channel spec
					var chanProps controlcommands.PropertyMap
					chanProps, err = inbCh.ToFMQMap(t.localBindMap)
					if err != nil {
						continue
					}

					// And we copy it into the task's propertyMap
					for k, v := range chanProps {
						propMap[k] = v
					}
				}
				for _, outboundCh := range channel.MergeOutbound(t.GetParent().CollectOutboundChannels(), class.Connect) {
					// We get the FairMQ-formatted propertyMap from the outbound channel spec
					var chanProps controlcommands.PropertyMap
					chanProps, err = outboundCh.ToFMQMap(bindMap)
					if err != nil {
						return nil, fmt.Errorf("task %s channel generation failed: %w", t.GetName(), err)
					}

					// And if valid, we copy it into the task's propertyMap
					if len(chanProps) > 0 {
						for k, v := range chanProps {
							propMap[k] = v
						}
					}
				}
			} // end append FairMQ configuration

			objStack := make(map[string]interface{})
			objStack["ToPtree"] = template.MakeToPtreeFunc(varStack, propMap)

			fields := template.WrapMapItems(propMap)

			err = fields.Execute(the.ConfSvc(), t.name, varStack, objStack, nil, make(map[string]texttemplate.Template), nil)
			if err != nil {
				log.WithError(err).
					WithField("partition", t.GetParent().GetEnvironmentId().String()).
					WithField("detector", detector).
					Error("cannot resolve templates for property map")
				return
			}

			// Post-processing for the ToPtree mechanism.
			// The ToPtree function has no access to the keys of propMap, so we need
			// to do a second pass here.
			// For each run of ToPtree, a temporary __ptree__:<syntax>:<uid> key is created
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
	}
	return propMap, err
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
	t.mu.RLock()
	defer t.mu.RUnlock()

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

func (t *Task) setTaskPID(pid int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t == nil {
		return
	}
	t.pid = strconv.Itoa(pid)
}

func (t *Task) GetTaskPID() string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if t == nil {
		return ""
	}
	return t.pid
}

func (t *Task) GetParent() parentRole {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.parent
}

func (t *Task) SetParent(parent parentRole) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.parent = parent
}

func (t *Task) GetTask() *Task {
	if t == nil {
		return nil
	}
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t
}
