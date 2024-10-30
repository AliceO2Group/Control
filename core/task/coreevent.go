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
	"github.com/AliceO2Group/Control/common/utils/uid"
	"github.com/AliceO2Group/Control/core/controlcommands"
	"github.com/AliceO2Group/Control/core/task/taskop"
	"github.com/mesos/mesos-go/api/v1/lib"
)

type TaskmanMessage struct {
	MessageType taskop.MessageType `json:"_messageType"`

	environmentMessage
	transitionTasksMessage
	updateTaskMessage
	// killTasksMessage
}

func newTaskmanMessage(mt taskop.MessageType) (t *TaskmanMessage) {
	t = &TaskmanMessage{
		MessageType: mt,
	}
	return t
}

func (tm *TaskmanMessage) GetMessageType() taskop.MessageType {
	return tm.MessageType
}

type environmentMessage struct {
	envId       uid.ID
	tasks       Tasks
	descriptors Descriptors
	runNumber   string
	errSt       string
}

func (em *environmentMessage) GetEnvironmentId() (envid uid.ID) {
	if em == nil {
		return
	}
	return em.envId
}

func (em *environmentMessage) GetTasks() Tasks {
	if em == nil {
		return nil
	}
	return em.tasks
}

func (em *environmentMessage) GetDescriptors() Descriptors {
	if em == nil {
		return nil
	}
	return em.descriptors
}

func (em *environmentMessage) GetRunNumber() string {
	if em == nil {
		return ""
	}
	return em.runNumber
}

func (em *environmentMessage) GetError() string {
	if em == nil {
		return ""
	}
	return em.errSt
}

func NewEnvironmentMessage(mt taskop.MessageType, envId uid.ID, tasks Tasks, desc Descriptors) (t *TaskmanMessage) {
	t = newTaskmanMessage(mt)
	t.environmentMessage = environmentMessage{
		envId:       envId,
		tasks:       tasks,
		descriptors: desc,
	}
	return t
}

type transitionTasksMessage struct {
	src        string
	event      string
	dest       string
	commonArgs controlcommands.PropertyMap
}

func (trm *transitionTasksMessage) GetSource() string {
	if trm == nil {
		return ""
	}
	return trm.src
}

func (trm *transitionTasksMessage) GetEvent() string {
	if trm == nil {
		return ""
	}
	return trm.event
}

func (trm *transitionTasksMessage) GetDestination() string {
	if trm == nil {
		return ""
	}
	return trm.dest
}

func (trm *transitionTasksMessage) GetArguments() controlcommands.PropertyMap {
	if trm == nil {
		return nil
	}
	return trm.commonArgs
}

func NewTransitionTaskMessage(tasks Tasks, src, transitionEvent, dest string, cargs controlcommands.PropertyMap, envID uid.ID) (t *TaskmanMessage) {
	t = newTaskmanMessage(taskop.TransitionTasks)
	t.transitionTasksMessage = transitionTasksMessage{
		src:        src,
		event:      transitionEvent,
		dest:       dest,
		commonArgs: cargs,
	}
	t.environmentMessage = environmentMessage{
		tasks:     tasks,
		envId:     envID,
		runNumber: cargs["runNumber"],
	}
	return t
}

type updateTaskMessage struct {
	taskId string
	state  string
	status mesos.TaskStatus
}

func NewTaskStatusMessage(mesosStatus mesos.TaskStatus) (t *TaskmanMessage) {
	t = newTaskmanMessage(taskop.TaskStatusMessage)
	t.updateTaskMessage = updateTaskMessage{
		status: mesosStatus,
	}
	return t
}

func NewTaskStateMessage(taskid, state string) (t *TaskmanMessage) {
	t = newTaskmanMessage(taskop.TaskStateMessage)
	t.updateTaskMessage = updateTaskMessage{
		taskId: taskid,
		state:  state,
	}
	return t
}
