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

	"github.com/AliceO2Group/Control/core/task"
	"github.com/gobwas/glob"
)

type taskRole struct {
	roleBase
	Task          *task.Task `yaml:"-,omitempty"`
	LoadTaskClass string     `yaml:"-,omitempty"`
}

func (t *taskRole) UnmarshalYAML(unmarshal func(interface{}) error) (err error) {
	aux := struct{
		Task struct{
			Load string
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
	*t = taskRole(role)
	return
}

func (t *taskRole) GlobFilter(g glob.Glob) (rs []Role) {
	if g.Match(t.GetPath()) {
		rs = []Role{t}
	}
	return
}

func (t *taskRole) ProcessTemplates(repoPath string) (err error) {
	if t == nil {
		return errors.New("role tree error when processing templates")
	}

	t.resolveTaskIdentifier(repoPath)
	t.resolveOutboundChannelTargets()

	return
}

func (t *taskRole) resolveTaskIdentifier(repoPath string) {
	if !strings.Contains(t.LoadTaskClass, "/") {
		t.LoadTaskClass = repoPath + t.LoadTaskClass
	}
}

func (t* taskRole) UpdateStatus(s task.Status) {
	t.updateStatus(s)
}

func (t* taskRole) UpdateState(s task.State) {
	t.updateState(s)
}

func (t *taskRole) updateStatus(s task.Status) {
	if t.parent == nil {
		log.WithField("status", s.String()).Error("cannot update status with nil parent")
	}
	t.status.merge(s, t)
	t.parent.updateStatus(s)
}

func (t *taskRole) updateState(s task.State) {
	if t.parent == nil {
		log.WithField("state", s.String()).Error("cannot update state with nil parent")
	}
	log.WithField("role", t.Name).WithField("state", s.String()).Debug("updating state")
	t.state.merge(s, t)
	t.parent.updateState(s)
}

func (t *taskRole) SetTask(taskPtr *task.Task) {
	t.Task = taskPtr
}

func (t *taskRole) copy() copyable {
	rCopy := taskRole{
		roleBase:      *t.roleBase.copy().(*roleBase),
		Task:          nil,
		LoadTaskClass: t.LoadTaskClass,
	}
	rCopy.status = SafeStatus{status:task.INACTIVE}
	rCopy.state  = SafeState{state:task.STANDBY}
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
		TaskRole: t,
		TaskClassName: t.LoadTaskClass,
		RoleConstraints: t.getConstraints(),
	}}
	return
}

func (t *taskRole) GetTasks() []*task.Task {
	return []*task.Task{t.GetTask()}
}

func (t *taskRole) GetTask() *task.Task {
	if t == nil {
		return nil
	}
	return t.Task
}

func (t* taskRole) GetTaskClass() string {
	if t == nil {
		return ""
	}
	return t.LoadTaskClass
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
}

func (t *taskRole) GetVars() task.VarMap {
	panic("niy")
}