/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2018-2019 CERN and copyright holders of ALICE O².
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
	"errors"
	"fmt"

	"github.com/AliceO2Group/Control/common/event"
	"github.com/AliceO2Group/Control/common/utils"
	"github.com/AliceO2Group/Control/core/controlcommands"
	"github.com/AliceO2Group/Control/executor/executable"
	"github.com/AliceO2Group/Control/executor/executorcmd"
	mesos "github.com/mesos/mesos-go/api/v1/lib"
	"github.com/mesos/mesos-go/api/v1/lib/executor"
	"github.com/mesos/mesos-go/api/v1/lib/executor/calls"
	"github.com/sirupsen/logrus"
)

func makeSendStatusUpdateFunc(state *internalState, task mesos.TaskInfo) executable.SendStatusFunc {
	return func(mesosState mesos.TaskState, message string){
		status := newStatus(state, task.TaskID)
		status.State = &mesosState
		status.Message = utils.ProtoString(message)
		state.statusCh <- status
	}
}

func makeSendDeviceEventFunc(state *internalState) executable.SendDeviceEventFunc {
	return func(event event.DeviceEvent) {
		jsonEvent, err := json.Marshal(event)
		if err != nil {
			log.WithError(err).Warning("error marshaling event from task")
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

func handleOutgoingMessage(state *internalState, message []byte) {
	_, _ = state.cli.Send(context.TODO(), calls.NonStreaming(calls.Message(message)))
	log.WithFields(logrus.Fields{
		"event": string(message),
	}).
	Debug("event sent")
}

func handleStatusUpdate(state *internalState, status mesos.TaskStatus) {
	if status.State == nil {
		log.Warn("status with nil state received")
	} else if *status.State == mesos.TASK_FAILED { // failed task updates are sent separately with less priority
		state.failedTasks[status.TaskID] = status
		delete(state.activeTasks, status.TaskID)
	} else {
		switch *status.State {
		case mesos.TASK_DROPPED: fallthrough
		case mesos.TASK_FINISHED: fallthrough
		case mesos.TASK_GONE: fallthrough
		case mesos.TASK_KILLED: fallthrough
		case mesos.TASK_LOST:
			delete(state.activeTasks, status.TaskID)
		}
		err := update(state, status)
		if err != nil { // in case of failed update, we just print an error message
			log.WithFields(logrus.Fields{
				"task":  status.TaskID,
				"state": status.State.String(),
			}).
			Warn("failed to update task status")
		}
	}

}

// Handle incoming message event. This function is thread-safe with respect to state.
func handleMessageEvent(state *internalState, data []byte) (err error) {
	var incoming struct {
		Name            string       `json:"name"`
		TargetList      []struct{
			TaskId       mesos.TaskID
		}                            `json:"targetList"`
	}
	err = json.Unmarshal(data, &incoming)
	if err != nil {
		return
	}

	if len(incoming.TargetList) != 1 {
		err = fmt.Errorf("cannot apply ExecutorCommand with %d!=1 target taskIds", len(incoming.TargetList))
		return
	}

	log.WithField("name", incoming.Name).
		WithField("payload", string(data[:])).
		Debug("processing incoming MESSAGE")

	taskId := incoming.TargetList[0].TaskId

	switch incoming.Name {
	case "MesosCommand_TriggerHook":
		// Check whether the task exists and is active
		activeTask, ok := state.activeTasks[taskId]
		if !ok || activeTask == nil {
			err = fmt.Errorf("no active task %s", taskId.Value)
			log.WithFields(logrus.Fields{
					"name": incoming.Name,
					"message": string(data[:]),
					"error": err.Error(),
				}).
				Error("no task for incoming MESSAGE")
			return
		}

		// Asynchronous and thread-unsafe, but probably ok because a hook only fires
		// once per environment cycle
		go func() {
			var cmd *controlcommands.MesosCommand_TriggerHook
			err = json.Unmarshal(data, cmd)
			if err != nil {
				log.WithFields(logrus.Fields{
						"name": incoming.Name,
						"message": string(data[:]),
						"error": err.Error(),
					}).
					Error("cannot unmarshal incoming MESSAGE")
				return
			}

			response := controlcommands.NewMesosCommandResponse_TriggerHook(cmd, nil, taskId.String())
			err = activeTask.Launch()
			if err != nil {
				response.ErrorString = err.Error()
			}

			data, marshalError := json.Marshal(response)
			if marshalError != nil {
				log.WithFields(logrus.Fields{
						"commandName": response.GetCommandName(),
						"commandId": response.GetCommandId(),
						"error": response.Err().Error(),
						"marshalError": marshalError,
					}).
					Error("cannot marshal MesosCommandResponse for sending as MESSAGE")
				return
			}

			_, _ = state.cli.Send(context.TODO(), calls.NonStreaming(calls.Message(data)))
			log.WithFields(logrus.Fields{
					"commandName": response.GetCommandName(),
					"commandId": response.GetCommandId(),
					"error": response.Err().Error(),
				}).
				Debug("response sent")
		}()

	case "MesosCommand_Transition":
		// Check whether the task exists and is active
		activeTask, ok := state.activeTasks[taskId]
		if !ok || activeTask == nil {
			err = fmt.Errorf("no active task %s", taskId.Value)
			log.WithFields(logrus.Fields{
					"name": incoming.Name,
					"message": string(data[:]),
					"error": err.Error(),
				}).
				Error("no task for incoming MESSAGE")
			return
		}

		// Unmarshal and perform transition asynchronously.
		// This is not thread-safe but we don't expect the core to spam
		// transition requests with no regard for MESSAGEs back.
		// If we don't do this, we get a choke point (OCTRL-204)
		go func() {
			var cmd *executorcmd.ExecutorCommand_Transition
			cmd, err = activeTask.UnmarshalTransition(data)
			if err != nil {
				log.WithFields(logrus.Fields{
						"name": cmd.Name,
						"message": string(data[:]),
						"error": err.Error(),
					}).
					Error("cannot unmarshal incoming MESSAGE")
				return
			}

			if cmd.Event == "CONFIGURE" {
				log.WithFields(logrus.Fields{"map": cmd.Arguments, "taskId": taskId}).Debug("CONFIGURE pushing FairMQ properties")
			}

			response := activeTask.Transition(cmd)

			data, marshalError := json.Marshal(response)
			if marshalError != nil {
				log.WithFields(logrus.Fields{
						"commandName": response.GetCommandName(),
						"commandId": response.GetCommandId(),
						"error": response.Err().Error(),
						"marshalError": marshalError,
					}).
					Error("cannot marshal MesosCommandResponse for sending as MESSAGE")
				return
			}

			_, _ = state.cli.Send(context.TODO(), calls.NonStreaming(calls.Message(data)))
			log.WithFields(logrus.Fields{
					"commandName": response.GetCommandName(),
					"commandId": response.GetCommandId(),
					"error": response.Err().Error(),
					"state": response.CurrentState,
				}).
				Debug("response sent")
		}()


	default:
		err = errors.New(fmt.Sprintf("unrecognized controlcommand %s", incoming.Name))
	}
	return
}

// Attempts to launch a task described by a mesos.TaskInfo. This function is thread-safe with respect to state.
func handleLaunchEvent(state *internalState, taskInfo mesos.TaskInfo) {
	state.unackedTasks[taskInfo.TaskID] = taskInfo

	jsonTask, _ := json.MarshalIndent(taskInfo, "", "\t")
	log.WithField("payload", fmt.Sprintf("%s", jsonTask[:])).Trace("received task to launch")

	myTask := executable.NewTask(taskInfo,
		makeSendStatusUpdateFunc(state, taskInfo),
		makeSendDeviceEventFunc(state),
		makeSendMessageFunc(state))

	err := myTask.Launch()

	if err == nil {
		state.activeTasks[taskInfo.TaskID] = myTask
		log.Trace("task launching")
	} else {
		log.Error("task launch failed")
	}
}

// Attempts to kill a task. This function is thread-safe with respect to state.
func handleKillEvent(state *internalState, e *executor.Event_Kill) error {
	activeTask, ok := state.activeTasks[e.GetTaskID()]
	if !ok {
		return errors.New("invalid task ID")
	}
	delete(state.activeTasks, e.GetTaskID())

	go func() {
		_ = activeTask.Kill()
		activeTask = nil
	}()

	return nil
}
