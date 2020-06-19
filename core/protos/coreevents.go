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

package pb

import (
	"encoding/json"
	"time"
	
	"github.com/mesos/mesos-go/api/v1/lib"
)

func NewEnvironmentStateEvent(cmdb []byte) *Event {
	var te Ev_EnvironmentStateChanged
	var tEvnt Event_EnvironmentStateChanged
	err := json.Unmarshal(cmdb, &te)
	if err != nil {
		return nil
	}

	tEvnt.EnvironmentStateChanged=&te
	return WrapEvent(&tEvnt)
}

func NewEnvironmentCreatedEvent(ei *EnvironmentInfo) *Event {
	var ec Event_EnvironmentCreated
	ec.EnvironmentCreated = &Ev_EnvironmentCreated{	
		Environment: ei,
	}
	return WrapEvent(&ec)
}

func NewEnvironmentDestroyedEvent(ctrl *CleanupTasksReply, envid string) *Event {
	var de Event_EnvironmentDestroyed
	de.EnvironmentDestroyed = &Ev_EnvironmentDestroyed{
		CleanupTasksReply: ctrl,
		Environmentid: envid,
	}
	return WrapEvent(&de)
}

func NewEnvironmentErrorEvent() *Event {
	var ee Event_EnvironmentError
	ee.EnvironmentError = &Ev_EnvironmentError{}
	return WrapEvent(&ee)
}

func NewEventTaskState(taskid,state string) *Event {
	var sc Ev_TaskStateChanged
	var stCh Event_TaskStateChanged
	sc.Taskid = taskid
	sc.State = state
	stCh.TaskStateChanged = &sc
	return WrapEvent(&stCh)
}

func NewEventTaskStatus(status *mesos.TaskStatus) *Event {

	taskId := status.GetTaskID().Value
	var statuschanged Ev_TaskStatusChanged
	var etSt Event_TaskStatusChanged
	switch st := status.GetState(); st {
	case mesos.TASK_RUNNING:
		// wrap to create environment ACTIVE 
		statuschanged.Taskid = taskId
		statuschanged.Status = "ACTIVE"
	case mesos.TASK_DROPPED, mesos.TASK_LOST, mesos.TASK_KILLED, mesos.TASK_FAILED, mesos.TASK_ERROR:
		statuschanged.Taskid = taskId
		statuschanged.Status = "INACTIVE"
	case mesos.TASK_FINISHED:
		statuschanged.Taskid = taskId
	}

	etSt.TaskStatusChanged = &statuschanged
	
	return WrapEvent(&etSt)
}

func NewEventTaskLaunch(taskid string) *Event {
	var ltask Ev_TaskLaunched
	var eltask Event_Tasklaunched

	ltask.Taskid = taskid
	eltask.Tasklaunched = &ltask
	
	return WrapEvent(&eltask)
}

func NewEventMesosTaskCreated(resourcesRequest,executorResources string) *Event {
	var mtask Ev_MesosTaskCreated
	var emtask Event_MesosTaskcreated

	mtask.ExecutorResources = executorResources
	mtask.Taskresources = resourcesRequest
	emtask.MesosTaskcreated = &mtask
	
	return WrapEvent(&emtask)
}

func NewKillTasksEvent() *Event {
	var ke Event_KilltasksMesos
	ke.KilltasksMesos = &Ev_KillTasksMesos{}
	return WrapEvent(&ke)
}

func WrapEvent(ce isEvent_Payload) *Event {
	return &Event{
		Timestamp:time.Now().Format(time.RFC3339),
		Payload: ce,
	}
}