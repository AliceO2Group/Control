/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2021 CERN and copyright holders of ALICE O².
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

type callRole struct {
	roleBase
	task.Traits
	FuncCall  string `yaml:"-,omitempty"`
	ReturnVar string `yaml:"-,omitempty"`
}

func (t *callRole) UnmarshalYAML(unmarshal func(interface{}) error) (err error) {
	aux := struct {
		Call struct {
			Func     string
			Return   string
			Trigger  *string
			Await    *string
			Timeout  *string
			Critical *bool
		}
	}{}

	type _callRole callRole
	role := _callRole{}

	err = unmarshal(&role)
	if err != nil {
		return
	}

	err = unmarshal(&aux)
	if err != nil {
		return
	}

	role.FuncCall = aux.Call.Func
	role.ReturnVar = aux.Call.Return

	// Set up basicTaskTraits defaults
	if aux.Call.Trigger != nil && len(*aux.Call.Trigger) > 0 { // hook
		role.Trigger = *aux.Call.Trigger
		if aux.Call.Timeout != nil && len(*aux.Call.Timeout) > 0 {
			role.Timeout = *aux.Call.Timeout
		} else {
			role.Timeout = (30 * time.Second).String()
		}
		if aux.Call.Await != nil && len(*aux.Call.Await) > 0 {
			role.Await = *aux.Call.Await
		} else {
			// if no await is specified, await := trigger
			role.Await = *aux.Call.Trigger
		}
	} else { // basic task
		if aux.Call.Timeout != nil && len(*aux.Call.Timeout) > 0 {
			role.Timeout = *aux.Call.Timeout
		} else {
			role.Timeout = "0s"
		}
	}

	if aux.Call.Critical != nil { // default for critical is always true
		role.Critical = *aux.Call.Critical
	} else {
		role.Critical = true
	}
	t.status.status = task.ACTIVE

	*t = callRole(role)
	return
}

func (t *callRole) MarshalYAML() (interface{}, error) {
	callRole := make(map[string]interface{})
	if t.Traits.Trigger != "" {
		callRole["trigger"] = t.Traits.Trigger
	}
	if t.Traits.Await != "" {
		callRole["await"] = t.Traits.Await
	}
	if t.Traits.Timeout != "" {
		callRole["timeout"] = t.Traits.Timeout
	}
	callRole["critical"] = t.Traits.Critical
	callRole["func"] = t.FuncCall
	callRole["return"] = t.ReturnVar

	auxRoleBase, err := t.roleBase.MarshalYAML()
	aux := auxRoleBase.(map[string]interface{})
	aux["call"] = callRole

	return aux, err
}

func (t *callRole) GlobFilter(g glob.Glob) (rs []Role) {
	if g.Match(t.GetPath()) {
		rs = []Role{t}
	}
	return
}

func (t *callRole) ProcessTemplates(workflowRepo repos.IRepo, _ LoadSubworkflowFunc, baseConfigStack map[string]string) (err error) {
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
			template.WrapPointer(&t.FuncCall),
			template.WrapPointer(&t.ReturnVar),
			template.WrapPointer(&t.Timeout),
			template.WrapPointer(&t.Trigger),
			template.WrapPointer(&t.Await),
		},
		template.STAGE5: append(append(
			WrapConstraints(t.Constraints),
			t.wrapBindAndConnectFields()...),
			template.WrapPointer(&t.Enabled)),
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

	return
}

func (t *callRole) UpdateStatus(s task.Status) {
	t.updateStatus(s)
}

func (t *callRole) UpdateState(s task.State) {
	t.updateState(s)
}

func (t *callRole) updateStatus(s task.Status) {
	if t.parent == nil {
		log.WithField("status", s.String()).Error("cannot update status with nil parent")
	}
	t.status.merge(s, t)
	t.SendEvent(&event.RoleEvent{Name: t.Name, Status: t.status.get().String(), RolePath: t.GetPath()})
	t.parent.updateStatus(s)
}

func (t *callRole) updateState(s task.State) {
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

func (t *callRole) copy() copyable {
	rCopy := callRole{
		roleBase:  *t.roleBase.copy().(*roleBase),
		FuncCall:  t.FuncCall,
		ReturnVar: t.ReturnVar,
		Traits:    t.Traits,
	}
	rCopy.status = SafeStatus{status: rCopy.GetStatus()}
	rCopy.state = SafeState{state: rCopy.GetState()}
	return &rCopy
}

func (t *callRole) GenerateTaskDescriptors() (ds task.Descriptors) {
	return nil
}

func (t *callRole) GetTasks() task.Tasks {
	return []*task.Task{}
}

func (t *callRole) GetHooksMapForTrigger(trigger string) (hooks callable.HooksMap) {
	if len(trigger) == 0 {
		return make(callable.HooksMap)
	}

	defer t.updateState(task.INVARIANT)

	// If a trigger is defined for this role &&
	//     If the input trigger is empty OR a positive match...
	if len(t.Trigger) > 0 {
		triggerName, triggerWeight := callable.ParseTriggerExpression(t.Trigger)
		if trigger == triggerName {
			return callable.HooksMap{
				triggerWeight: callable.Hooks{callable.NewCall(t.FuncCall, t.ReturnVar, t)},
			}
		}
	}
	return make(callable.HooksMap)
}

func (t *callRole) GetAllHooks() callable.Hooks {
	defer t.updateState(task.INVARIANT)

	// If a trigger is defined for this role
	if len(t.Trigger) > 0 {
		return []callable.Hook{
			callable.NewCall(t.FuncCall, t.ReturnVar, t),
		}
	}
	return callable.Hooks{}
}

func (t *callRole) GetTaskTraits() task.Traits {
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

func (t *callRole) GetTaskClasses() []string {
	return nil
}

func (*callRole) GetRoles() []Role {
	return nil
}

func (t *callRole) setParent(role Updatable) {
	t.parent = role
	t.Defaults.Wrap(role.GetDefaults())
	t.Vars.Wrap(role.GetVars())
	t.UserVars.Wrap(role.GetUserVars())
}
