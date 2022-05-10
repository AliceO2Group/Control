/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2021-2022 CERN and copyright holders of ALICE O².
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

import "github.com/AliceO2Group/Control/core/task"

type Hook interface {
	GetParentRole() interface{}
	GetParentRolePath() string
	GetName() string
	GetTraits() task.Traits
}

type Hooks []Hook

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
