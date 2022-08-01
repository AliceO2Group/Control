/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2018-2022 CERN and copyright holders of ALICE O².
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

package executor

import (
	"context"
	"encoding/json"

	"github.com/AliceO2Group/Control/common/event"
	"github.com/AliceO2Group/Control/common/utils"
	"github.com/AliceO2Group/Control/common/utils/uid"
	"github.com/AliceO2Group/Control/executor/executable"
	mesos "github.com/mesos/mesos-go/api/v1/lib"
	"github.com/mesos/mesos-go/api/v1/lib/executor/calls"
	"github.com/sirupsen/logrus"
)

func makeSendStatusUpdateFunc(state *internalState, task mesos.TaskInfo) executable.SendStatusFunc {
	return func(envId uid.ID, mesosState mesos.TaskState, message string) {
		status := newStatus(envId, state, task.TaskID)
		status.State = &mesosState
		status.Message = utils.ProtoString(message)
		state.statusCh <- status
	}
}

func makeSendDeviceEventFunc(state *internalState) executable.SendDeviceEventFunc {
	return func(envId uid.ID, event event.DeviceEvent) {
		jsonEvent, err := json.Marshal(event)
		if err != nil {
			log.WithError(err).
				Warning("error marshaling event from task")
			return
		}
		state.messageCh <- jsonEvent
	}
}

func makeSendMessageFunc(state *internalState) executable.SendMessageFunc {
	return func(message []byte) {
		// to send task events using state.
		state.messageCh <- message
	}
}

func sendOutgoingMessage(state *internalState, message []byte) {
	_, _ = state.cli.Send(context.TODO(), calls.NonStreaming(calls.Message(message)))
}

func performStatusUpdate(state *internalState, status mesos.TaskStatus) {
	if status.State == nil {
		log.Warn("status with nil state received")
	} else if *status.State == mesos.TASK_FAILED { // failed task updates are sent separately with less priority
		state.activeTasksMu.Lock()
		state.failedTasks[status.TaskID] = status
		delete(state.activeTasks, status.TaskID)
		state.activeTasksMu.Unlock()
	} else {
		switch *status.State {
		case mesos.TASK_DROPPED:
			fallthrough
		case mesos.TASK_FINISHED:
			fallthrough
		case mesos.TASK_GONE:
			fallthrough
		case mesos.TASK_KILLED:
			fallthrough
		case mesos.TASK_LOST:
			state.activeTasksMu.Lock()
			delete(state.activeTasks, status.TaskID)
			state.activeTasksMu.Unlock()
		}
		err := update(state, status)
		if err != nil { // in case of failed update, we just print an error message
			log.WithFields(logrus.Fields{
				"task":  status.TaskID,
				"state": status.State.String(),
			}).
				Warn("executor failed to send task status update")
		}
	}
}
