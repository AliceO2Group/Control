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
	"time"
	
	"github.com/AliceO2Group/Control/common/utils/uid"
)

func NewEnvironmentStateEvent(envId uid.ID, state string, rn uint32) *Event {
	var te Ev_EnvironmentStateChanged
	var tEvnt Event_EnvironmentStateChanged
	te.State = state
	te.Environmentid = envId.String()
	te.CurrentRunNumber = rn

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

func NewEnvironmentErrorEvent(errSt string, close bool) *Event {
	var ee Event_EnvironmentError
	ee.EnvironmentError = &Ev_EnvironmentError{
		Error: errSt,
		Closestream: close,
	}
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

func NewEventTaskStatus(taskid,status string) *Event {
	var etSt Event_TaskStatusChanged

	etSt.TaskStatusChanged = &Ev_TaskStatusChanged{
		Taskid: taskid,
		Status: status,
	}
	
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

func NewKillTasksEvent(killed []*ShortTaskInfo) *Event {
	var ke Event_Taskskilled
	ke.Taskskilled = &Ev_TasksKilled{
		KilledTasks: killed,
	}
	return WrapEvent(&ke)
}

func WrapEvent(ce isEvent_Payload) *Event {
	return &Event{
		Timestamp:time.Now().Format(time.RFC3339),
		Payload: ce,
	}
}