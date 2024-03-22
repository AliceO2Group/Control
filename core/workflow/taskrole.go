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
	"errors"
	"strings"
	texttemplate "text/template"
	"time"

	"github.com/AliceO2Group/Control/common/event"
	"github.com/AliceO2Group/Control/configuration/template"
	"github.com/AliceO2Group/Control/core/repos"
	"github.com/AliceO2Group/Control/core/task"
	"github.com/AliceO2Group/Control/core/the"
	"github.com/AliceO2Group/Control/core/workflow/callable"
	"github.com/gobwas/glob"
)

type taskRole struct {
	roleBase
	task.Traits
	Task          *task.Task `yaml:"-,omitempty"`
	LoadTaskClass string     `yaml:"-,omitempty"`
}

func (t *taskRole) UnmarshalYAML(unmarshal func(interface{}) error) (err error) {
	aux := struct {
		Task struct {
			Load     string
			Trigger  *string
			Await    *string
			Timeout  *string
			Critical *bool
		}
	}{}

	type _taskRole taskRole
	role := _taskRole{}

	err = unmarshal(&role)
	if err != nil {
		return
	}

	err = unmarshal(&aux)
	if err != nil {
		return
	}

	role.LoadTaskClass = aux.Task.Load

	// Set up basicTaskTraits defaults
	if aux.Task.Trigger != nil && len(*aux.Task.Trigger) > 0 { // hook
		role.Trigger = *aux.Task.Trigger
		if aux.Task.Timeout != nil && len(*aux.Task.Timeout) > 0 {
			role.Timeout = *aux.Task.Timeout
		} else {
			role.Timeout = (30 * time.Second).String()
		}
		if aux.Task.Await != nil && len(*aux.Task.Await) > 0 {
			role.Await = *aux.Task.Await
		} else {
			// if no await is specified, await := trigger
			role.Await = *aux.Task.Trigger
		}
	} else { // basic task
		if aux.Task.Timeout != nil && len(*aux.Task.Timeout) > 0 {
			role.Timeout = *aux.Task.Timeout
		} else {
			role.Timeout = "0s"
		}
	}

	if aux.Task.Critical != nil { // default for critical is always true
		role.Critical = *aux.Task.Critical
	} else {
		role.Critical = true
	}

	*t = taskRole(role)
	return
}

func (t *taskRole) MarshalYAML() (interface{}, error) {
	taskRole := make(map[string]interface{})
	if t.Traits.Trigger != "" {
		taskRole["trigger"] = t.Traits.Trigger
	}
	if t.Traits.Await != "" {
		taskRole["await"] = t.Traits.Await
	}
	if t.Traits.Timeout != "" {
		taskRole["timeout"] = t.Traits.Timeout
	}
	taskRole["critical"] = t.Traits.Critical
	taskRole["load"] = t.LoadTaskClass

	auxRoleBase, err := t.roleBase.MarshalYAML()
	aux := auxRoleBase.(map[string]interface{})
	aux["task"] = taskRole

	return aux, err
}

func (t *taskRole) GlobFilter(g glob.Glob) (rs []Role) {
	if g.Match(t.GetPath()) {
		rs = []Role{t}
	}
	return
}

func (t *taskRole) ProcessTemplates(workflowRepo repos.IRepo, _ LoadSubworkflowFunc, baseConfigStack map[string]string) (err error) {
	if t == nil {
		return errors.New("role tree error when processing templates")
	}

	templSequence := template.Sequence{
		template.STAGE0: template.Fields{
			template.WrapPointer(&t.Enabled),
		},
		template.STAGE1: template.WrapMapItems(t.Defaults.Raw()),
		template.STAGE2: template.WrapMapItems(t.Vars.Raw()),
		template.STAGE3: template.WrapMapItems(t.UserVars.Raw()),
		template.STAGE4: template.Fields{
			template.WrapPointer(&t.Name),
			template.WrapPointer(&t.LoadTaskClass),
			template.WrapPointer(&t.Timeout),
			template.WrapPointer(&t.Trigger),
			template.WrapPointer(&t.Await),
		},
		template.STAGE5: append(append(
			WrapConstraints(t.Constraints),
			t.wrapBindAndConnectFields()...)),
	}

	// FIXME: push cached templates here
	err = templSequence.Execute(
		the.ConfSvc(),
		t.GetPath(),
		template.VarStack{
			Locals:   t.Locals,
			Defaults: t.Defaults,
			Vars:     t.Vars,
			UserVars: t.UserVars,
		},
		t.makeBuildObjectStackFunc(),
		baseConfigStack,
		make(map[string]texttemplate.Template),
		workflowRepo,
		MakeDisabledRoleCallback(t),
	)
	if err != nil {
		var roleDisabledErrorType *template.RoleDisabledError
		if isRoleDisabled := errors.As(err, &roleDisabledErrorType); isRoleDisabled {
			log.WithField("partition", t.GetEnvironmentId().String()).Trace(err.Error())
			err = nil // we don't want a disabled role to be considered an error
		} else {
			return
		}
	}

	// After template processing we write the Locals to Vars in order to make them available to children
	for k, v := range t.Locals {
		t.Vars.Set(k, v)
	}

	t.Enabled = strings.TrimSpace(t.Enabled)

	if !t.IsEnabled() {
		return
	}

	t.resolveTaskClassIdentifier(workflowRepo)

	return
}

func (t *taskRole) resolveTaskClassIdentifier(repo repos.IRepo) {
	t.LoadTaskClass = repo.ResolveTaskClassIdentifier(t.LoadTaskClass)
}

func (t *taskRole) UpdateStatus(s task.Status) {
	t.updateStatus(s)
}

func (t *taskRole) UpdateState(s task.State) {
	t.updateState(s)
}

func (t *taskRole) updateStatus(s task.Status) {
	if t.parent == nil {
		log.WithField("status", s.String()).Error("cannot update status with nil parent")
	}
	t.status.merge(s, t)
	t.SendEvent(&event.RoleEvent{Name: t.Name, Status: t.status.get().String(), RolePath: t.GetPath()})
	t.parent.updateStatus(s)
}

func (t *taskRole) updateState(s task.State) {
	if t.parent == nil {
		log.WithField("state", s.String()).Error("cannot update state with nil parent")
	}
	log.WithField("role", t.Name).WithField("state", s.String()).Trace("updating state")
	t.state.merge(s, t)
	t.SendEvent(&event.RoleEvent{Name: t.Name, State: t.state.get().String(), RolePath: t.GetPath()})

	if t.Critical == true || s != task.ERROR {
		t.parent.updateState(s)
	}
}

func (t *taskRole) SetTask(taskPtr *task.Task) {
	t.Task = taskPtr
	// FIXME: when this is called, properties or vars should be pushed to the task
}

func (t *taskRole) copy() copyable {
	rCopy := taskRole{
		roleBase:      *t.roleBase.copy().(*roleBase),
		Task:          nil,
		LoadTaskClass: t.LoadTaskClass,
		Traits:        t.Traits,
	}
	rCopy.status = SafeStatus{status: task.INACTIVE}
	rCopy.state = SafeState{state: task.STANDBY}
	return &rCopy
}

func (t *taskRole) GenerateTaskDescriptors() (ds task.Descriptors) {
	if t == nil {
		return nil
	}
	ds = make(task.Descriptors, 0)
	taskPtr := t.GetTask()
	if taskPtr != nil { // If we already have a running task
		return
	}

	ds = task.Descriptors{{
		TaskRole:        t,
		TaskClassName:   t.LoadTaskClass,
		RoleConstraints: t.getConstraints(),
		RoleConnect:     t.CollectOutboundChannels(),
		RoleBind:        t.CollectInboundChannels(),
	}}
	return
}

func (t *taskRole) GetTasks() task.Tasks {
	if ttask := t.GetTask(); ttask == nil {
		return []*task.Task{}
	}
	return []*task.Task{t.GetTask()}
}

func (t *taskRole) GetHooksMapForTrigger(trigger string) (hooks callable.HooksMap) {
	if ttask := t.GetTask(); ttask == nil {
		return make(callable.HooksMap)
	}
	if len(trigger) == 0 {
		return make(callable.HooksMap)
	}

	// If a trigger is defined for this role &&
	//     If the input trigger is a positive match...
	if len(t.Trigger) > 0 {
		triggerName, triggerWeight := callable.ParseTriggerExpression(t.Trigger)
		if trigger == triggerName {
			return callable.HooksMap{
				triggerWeight: callable.Hooks{t.GetTask()},
			}
		}
	}
	return make(callable.HooksMap)
}

func (t *taskRole) GetAllHooks() callable.Hooks {
	if ttask := t.GetTask(); ttask == nil {
		return []callable.Hook{}
	}

	// If a trigger is defined for this role
	if len(t.Trigger) > 0 {
		return []callable.Hook{t.GetTask()}
	}
	return []callable.Hook{}
}

func (t *taskRole) GetTask() *task.Task {
	if t == nil {
		return nil
	}
	return t.Task.GetTask()
}

func (t *taskRole) GetTaskClass() string {
	if t == nil {
		return ""
	}
	return t.LoadTaskClass
}

func (t *taskRole) GetTaskTraits() task.Traits {
	if t == nil {
		return task.Traits{
			Trigger:  "",
			Await:    "",
			Timeout:  "0s",
			Critical: false,
		}
	}
	return t.Traits
}

func (t *taskRole) GetTaskClasses() []string {
	if t == nil {
		return nil
	}
	return []string{t.LoadTaskClass}
}

func (*taskRole) GetRoles() []Role {
	return nil
}

//FIXME: figure out if nested doTransition calls are even desirable
// Intead of this stuff, we could have a similar method which does not perform a transition,
// but just builds the mesoscommand_transition and sends it.
// When calling workflow.doTransition it would still appear that we block until we return,
// but we'd have a first passage down the tree to acquire the list of Tasks and then taskman
// to build the MC and enqueue it
// It's critical to have a method which returns all tasks for a role

/*func (t *taskRole) doTransition(transition Transition) (task.Status, task.State) {
	if t == nil || t.Task == nil {
		return task.UNDEFINED, task.MIXED
	}
	if t.GetStatus() != task.ACTIVE {
		return t.GetStatus(), task.MIXED
	}
	newState, err := t.Task.DoTransition(transition)
	if err != nil {
		log.WithError(err).Error("task transition error")
	}

	return t.GetStatus(), newState
}*/

func (t *taskRole) setParent(role Updatable) {
	t.parent = role
	t.Defaults.Wrap(role.GetDefaults())
	t.Vars.Wrap(role.GetVars())
	t.UserVars.Wrap(role.GetUserVars())
}
