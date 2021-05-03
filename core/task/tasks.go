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

import (
	"fmt"

	"github.com/AliceO2Group/Control/core/controlcommands"
	"github.com/AliceO2Group/Control/core/task/channel"
)

type Tasks []*Task
type DeploymentMap map[*Task]*Descriptor

type Filter func(*Task) bool
var Filter_NIL Filter = func(*Task) bool {
	return true
}

// FIXME: make this structure thread-safe to fully replace big state lock in server.go

func (m Tasks) GetTaskIds() []string {
	taskIds := make([]string, len(m))
	for i, taskPtr := range m {
		taskIds[i] = taskPtr.taskId
	}
	return taskIds
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

func (m Tasks) GetMesosCommandTargets() (receivers []controlcommands.MesosCommandTarget, err error) {
	receivers = make([]controlcommands.MesosCommandTarget, 0)
	for _, task := range m {
		receivers = append(receivers, task.GetMesosCommandTarget())
	}
	return
}

func (m Tasks) BuildPropertyMaps(bindMap channel.BindMap) (propMapMap controlcommands.PropertyMapsMap, err error) {
	propMapMap = make(controlcommands.PropertyMapsMap)
	for _, task := range m {
		if !task.IsLocked() {
			return nil, fmt.Errorf("task %s is not locked, cannot send control commands", task.GetName())
		}
		receiver := task.GetMesosCommandTarget()

		var taskPropMap controlcommands.PropertyMap
		taskPropMap, err = task.BuildPropertyMap(bindMap)
		if err != nil {
			return nil, err
		}
		propMapMap[receiver] = taskPropMap
	}
	return
}