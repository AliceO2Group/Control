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

package task

type Tasks []*Task
type DeploymentMap map[*Task]*Descriptor

type Filter func(*Task) bool
var Filter_NIL Filter = func(*Task) bool {
	return true
}

func (m Tasks) GetByTaskId(id string) *Task {
	for _, taskPtr := range m {
		if taskPtr != nil && taskPtr.taskId == id {
			return taskPtr
		}
	}
	return nil
}

func (m Tasks) Contains(filter Filter) (has bool) {
	if m == nil {
		return
	}
	for _, taskPtr := range m {
		has = filter(taskPtr)
		if has {
			return
		}
	}
	return
}

func (m Tasks) FilteredForClass(className string) (tasks Tasks) {
	return m.Filtered(func(task *Task) bool {
		if task == nil {
			return false
		}
		return task.className == className
	})
}

func (m Tasks) Filtered(filter Filter) (tasks Tasks) {
	if m == nil {
		return
	}
	tasks = make(Tasks, 0)
	for _, taskPtr := range m {
		if filter(taskPtr) {
			tasks = append(tasks, taskPtr)
		}
	}
	return tasks
}
