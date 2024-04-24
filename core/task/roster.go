/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2020 CERN and copyright holders of ALICE O².
 * Author: Miltiadis Alexis <miltiadis.alexis@cern.ch>
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
	"sync"

	"github.com/AliceO2Group/Control/core/controlcommands"
	"github.com/AliceO2Group/Control/core/task/channel"
)

type roster struct {
	mu    sync.RWMutex
	tasks Tasks
}

func (m *roster) getTaskIds() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.tasks.GetTaskIds()
}

func (m *roster) getByTaskId(id string) *Task {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.tasks.GetByTaskId(id)
}

func (m *roster) contains(filter Filter) (has bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.tasks.Contains(filter)
}

func (m *roster) filteredForClass(className string) (tasks Tasks) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.tasks.FilteredForClass(className)
}

func (m *roster) filtered(filter Filter) (tasks Tasks) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.tasks.Filtered(filter)
}

func (m *roster) grouped(grouping Grouping) (tasksMap map[string]Tasks) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.tasks.Grouped(grouping)
}

func (m *roster) getMesosCommandTargets() (receivers []controlcommands.MesosCommandTarget, err error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.tasks.GetMesosCommandTargets()
}

func (m *roster) buildPropertyMaps(bindMap channel.BindMap) (propMapMap controlcommands.PropertyMapsMap, err error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.tasks.BuildPropertyMaps(bindMap)
}

func (m *roster) getTasks() Tasks {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tasks:= make(Tasks, len(m.tasks))
	copy(tasks, m.tasks)

	return tasks
}

func (m *roster) updateTasks(tasks Tasks) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.tasks = tasks
}

func (m *roster) append(task *Task) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.tasks = append(m.tasks, task)
}

func newRoster() *roster {
	roster := &roster{
		tasks: make(Tasks, 0),
	}
	return roster
}
