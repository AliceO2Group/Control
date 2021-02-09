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

package callable

import (
	"strconv"
	texttemplate "text/template"

	"github.com/AliceO2Group/Control/apricot"
	"github.com/AliceO2Group/Control/common/event"
	"github.com/AliceO2Group/Control/common/utils/uid"
	"github.com/AliceO2Group/Control/configuration/template"
	"github.com/AliceO2Group/Control/core/integration"
	"github.com/AliceO2Group/Control/core/task"
)

type Calls []*Call
type Hooks []Hook

type Hook interface {
	GetParentRole() interface{}
	GetParentRolePath() string
	GetName() string
	GetTraits() task.Traits
}

type Call struct {
	Func string
	Return string
	VarStack map[string]string
	Traits task.Traits
	parentRole ParentRole
}

func (s Hooks) FilterCalls() (calls Calls) {
	calls = make(Calls, 0)
	for _, v := range s {
		if c, ok := v.(*Call); ok {
			calls = append(calls, c)
		}
	}
	return
}

func (s Hooks) FilterTasks() (tasks task.Tasks) {
	tasks = make(task.Tasks, 0)
	for _, v := range s {
		if t, ok := v.(*task.Task); ok {
			tasks = append(tasks, t)
		}
	}
	return
}

func (s Calls) CallAll() map[*Call]error {
	errors := make(map[*Call]error)
	for _, v := range s {
		err := v.Call()
		if err != nil {
			errors[v] = err
		}
	}
	return errors
}


func NewCall(funcCall string, returnVar string, varStack map[string]string, parent ParentRole) (call *Call) {
	return &Call{
		Func:       funcCall,
		Return:     returnVar,
		VarStack:   varStack,
		Traits:     parent.GetTaskTraits(),
		parentRole: parent,
	}
}

func (c *Call) Call() error {
	output := "{{" + c.Func + "}}"
	returnVar := c.Return
	fields := template.Fields{
			template.WrapPointer(&output),
			template.WrapPointer(&returnVar),
		}
	c.VarStack["run_number"] = strconv.FormatUint(uint64(c.parentRole.GetCurrentRunNumber()), 10 )
	objStack := integration.PluginsInstance().ObjectStack(c)

	err := fields.Execute(apricot.Instance(), c.GetName(), c.VarStack, objStack, make(map[string]texttemplate.Template))
	if err != nil {
		return err
	}
	if len(returnVar) > 0 {
		c.parentRole.SetRuntimeVar(returnVar, output)
	}
	return nil
}

func (c *Call) GetParentRole() interface{} {
	return c.parentRole
}

func (c *Call) GetParentRolePath() string {
	return c.parentRole.GetPath()
}

func (c *Call) GetName() string {
	return c.parentRole.GetPath()
}

func (c *Call) GetTraits() task.Traits {
	return c.Traits
}

type ParentRole interface {
	GetPath() string
	GetTaskTraits() task.Traits
	GetEnvironmentId() uid.ID
	ConsolidatedVarStack() (varStack map[string]string, err error)
	SendEvent(event.Event)
	SetRuntimeVar(key string, value string)
	GetCurrentRunNumber() uint32
}