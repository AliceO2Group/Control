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
	"strconv"
	"github.com/AliceO2Group/Control/core/task"
)

type aggregator struct {
	Roles       controllableRoles      `json:"roles,omitempty"`
}

func (r* aggregator) copy() copyable {
	rCopy := aggregator{
		Roles: make(controllableRoles, len(r.Roles)),
	}
	for i, childRole := range r.Roles {
		rCopy.Roles[i] = childRole.copy().(controllableRole)
	}
	return &rCopy
}

type controllableRoles []controllableRole

type _roleUnion struct {
	For *struct{}
	Task *struct{}
	Roles *struct{}
}

func (r *controllableRoles) UnmarshalYAML(unmarshal func(interface{}) error) (err error) {
	roleUnionSlice := make([]_roleUnion, 0)
	err = unmarshal(&roleUnionSlice)
	if err != nil {
		return
	}
	roles := make(controllableRoles, len(roleUnionSlice))
	for i, v := range roleUnionSlice {
		switch {
		case v.For != nil:
			roles[i] = new(iteratorRole)
		case v.Roles != nil && v.Task == nil:
			roles[i] = new(aggregatorRole)
		case v.Task != nil && v.Roles == nil:
			roles[i] = new(taskRole)
		default:
			err = errors.New("invalid child role at index " + strconv.Itoa(i))
			return
		}
	}
	err = unmarshal(&roles)
	if err != nil {
		return
	}
	*r = roles
	return
}

func (r *aggregator) GenerateTaskDescriptors() (ds task.Descriptors) {
	if r == nil {
		return nil
	}

	ds = make(task.Descriptors, 0)
	for _, role := range r.GetRoles() {
		ds = append(ds, role.GenerateTaskDescriptors()...)
	}
	return
}

func (r *aggregator) GetTasks() (tasks []*task.Task) {
	if r == nil {
		return nil
	}

	tasks = make([]*task.Task, 0)
	for _, role := range r.GetRoles() {
		tasks = append(tasks, role.GetTasks()...)
	}
	return
}

func (r *aggregator) GetRoles() []Role {
	if r == nil {
		return nil
	}
	roles := make([]Role, 0)
	for _, v := range r.Roles {
		if iter, ok := v.(*iteratorRole); ok {
			roles = append(roles, iter.GetRoles()...)
			continue
		}
		roles = append(roles, v)
	}
	return roles
}

/*func (r *aggregator) doTransition(transition Transition) (status task.Status, state task.State) {
	if r == nil || len(r.Roles) == 0 {
		status = task.UNDEFINED
		state  = task.MIXED
		return
	}

	// parallel for
	var wg sync.WaitGroup
	wg.Add(len(r.Roles))

	statuses := make([]task.Status, len(r.Roles))
	states := make([]task.State, len(r.Roles))

	for i := 0; i < len(r.Roles); i++ {
		go func(i int) {
			defer wg.Done()
			statuses[i], states[i] = r.Roles[i].doTransition(transition)
		}(i)
	}
	wg.Wait()

	status = statuses[0]
	state = states[0]
	for i := 1; i < len(r.Roles); i++ {
		status = status.X(statuses[i])
		state = state.X(states[i])
	}
	return
}*/
