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
	"errors"
	"fmt"
	"time"

	"github.com/AliceO2Group/Control/common/logger/infologger"
	"github.com/AliceO2Group/Control/common/utils"
	"github.com/AliceO2Group/Control/core/controlcommands"
	"github.com/AliceO2Group/Control/executor/executable"
	"github.com/AliceO2Group/Control/executor/executorcmd"
	"github.com/AliceO2Group/Control/executor/executorutil"
	mesos "github.com/mesos/mesos-go/api/v1/lib"
	"github.com/mesos/mesos-go/api/v1/lib/executor"
	"github.com/mesos/mesos-go/api/v1/lib/executor/calls"
	"github.com/sirupsen/logrus"
)

// Handle incoming message event. This function is thread-safe with respect to state.
func handleMessageEvent(state *internalState, data []byte) (err error) {
	var incoming struct {
		Name       string `json:"name"`
		TargetList []struct {
			TaskId mesos.TaskID
		} `json:"targetList"`
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
		Trace("processing incoming MESSAGE")

	taskId := incoming.TargetList[0].TaskId

	switch incoming.Name {
	case "MesosCommand_TriggerHook":
		// Check whether the task exists and is active
		state.activeTasksMu.RLock()
		activeTask, ok := state.activeTasks[taskId]
		state.activeTasksMu.RUnlock()
		if !ok || activeTask == nil {
			err = fmt.Errorf("no active task %s", taskId.Value)
			log.WithFields(logrus.Fields{
				"name":    incoming.Name,
				"message": string(data[:]),
				"error":   err.Error(),
			}).
				Error("no task for incoming MESSAGE")
			return
		}

		// Asynchronous and thread-unsafe, but probably ok because a hook only fires
		// once per environment cycle
		go func() {
			var cmd = new(controlcommands.MesosCommand_TriggerHook)
			err = json.Unmarshal(data, cmd)
			if err != nil {
				log.WithFields(logrus.Fields{
					"name":    incoming.Name,
					"message": string(data[:]),
					"error":   err.Error(),
				}).
					Error("cannot unmarshal incoming MESSAGE")
				return
			}

			response := controlcommands.NewMesosCommandResponse_TriggerHook(cmd, nil, taskId.Value)
			hookTask, ok := activeTask.(*executable.HookTask)
			if !ok {
				log.WithFields(logrus.Fields{
					"name":    incoming.Name,
					"message": string(data[:]),
					"error":   "type assertion error",
				}).
					Warning("received TriggerHook for non-hook task")
				return
			}

			err = hookTask.Trigger()
			if err != nil {
				response.ErrorString = err.Error()
			}

			jsonData, marshalError := json.Marshal(response)
			if marshalError != nil {
				if response.Err() != nil {
					log.WithFields(logrus.Fields{
						"commandName":  response.GetCommandName(),
						"commandId":    response.GetCommandId(),
						"error":        response.Err().Error(),
						"marshalError": marshalError,
					}).
						Error("cannot marshal MesosCommandResponse for sending as MESSAGE")
				} else {
					log.WithFields(logrus.Fields{
						"commandName":  response.GetCommandName(),
						"commandId":    response.GetCommandId(),
						"marshalError": marshalError,
					}).
						Error("cannot marshal MesosCommandResponse for sending as MESSAGE")
				}
				return
			}

			_, _ = state.cli.Send(context.TODO(), calls.NonStreaming(calls.Message(jsonData)))
			if response.Err() != nil {
				log.WithFields(logrus.Fields{
					"commandName": response.GetCommandName(),
					"commandId":   response.GetCommandId(),
					"taskId":      response.TaskId,
					"error":       response.Err().Error(),
				}).
					Trace("response sent")
			} else {
				log.WithFields(logrus.Fields{
					"commandName": response.GetCommandName(),
					"commandId":   response.GetCommandId(),
					"taskId":      response.TaskId,
				}).
					Trace("response sent")
			}
		}()

	case "MesosCommand_Transition":
		// Check whether the task exists and is active
		state.activeTasksMu.RLock()
		activeTask, ok := state.activeTasks[taskId]
		state.activeTasksMu.RUnlock()
		if !ok || activeTask == nil {
			err = fmt.Errorf("no active task %s", taskId.Value)
			log.WithFields(logrus.Fields{
				"name":    incoming.Name,
				"message": string(data[:]),
				"error":   err.Error(),
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
				fields := logrus.Fields{
					"error": err.Error(),
				}
				if cmd != nil {
					fields["name"] = cmd.Name
				}
				if len(data) > 0 {
					fields["message"] = string(data[:])
				}

				log.WithFields(fields).
					Error("cannot unmarshal incoming MESSAGE")
				return
			}

			response := activeTask.Transition(cmd)

			jsonData, marshalError := json.Marshal(response)
			if marshalError != nil {
				if response.Err() != nil {
					log.WithFields(logrus.Fields{
						"commandName":  response.GetCommandName(),
						"commandId":    response.GetCommandId(),
						"error":        response.Err().Error(),
						"marshalError": marshalError,
					}).
						Error("cannot marshal MesosCommandResponse for sending as MESSAGE")
				} else {
					log.WithFields(logrus.Fields{
						"commandName":  response.GetCommandName(),
						"commandId":    response.GetCommandId(),
						"marshalError": marshalError,
					}).
						Error("cannot marshal MesosCommandResponse for sending as MESSAGE")
				}
				return
			}

			_, _ = state.cli.Send(context.TODO(), calls.NonStreaming(calls.Message(jsonData)))
			if response.Err() != nil {
				log.WithFields(logrus.Fields{
					"commandName": response.GetCommandName(),
					"commandId":   response.GetCommandId(),
					"error":       response.Err().Error(),
					"state":       response.CurrentState,
				}).
					Trace("response sent")
			} else {
				log.WithFields(logrus.Fields{
					"commandName": response.GetCommandName(),
					"commandId":   response.GetCommandId(),
					"state":       response.CurrentState,
				}).
					Trace("response sent")
			}
		}()

	default:
		err = errors.New(fmt.Sprintf("unrecognized controlcommand %s", incoming.Name))
	}
	return
}

// Attempts to launch a task described by a mesos.TaskInfo. This function is thread-safe with respect to state.
func handleLaunchEvent(state *internalState, taskInfo mesos.TaskInfo) error {
	// Before we do anything else, we try to get an environment ID for log messages
	envId := executorutil.GetEnvironmentIdFromLabelerType(&taskInfo)
	detector := executorutil.GetValueFromLabelerType(&taskInfo, "detector")

	log.WithFields(logrus.Fields{
		"taskId":    taskInfo.TaskID.GetValue(),
		"taskName":  taskInfo.Name,
		"hostname":  state.agent.GetHostname(),
		"level":     infologger.IL_Devel,
		"partition": envId.String(),
		"detector":  detector,
	}).Debug("executor.handleLaunchEvent begin")

	defer utils.TimeTrack(time.Now(),
		"executor.handleLaunchEvent",
		log.WithFields(logrus.Fields{
			"taskId":    taskInfo.TaskID.GetValue(),
			"taskName":  taskInfo.Name,
			"hostname":  state.agent.GetHostname(),
			"level":     infologger.IL_Devel,
			"partition": envId.String(),
			"detector":  detector,
		}))

	state.unackedTasks[taskInfo.TaskID] = taskInfo

	jsonTask, _ := json.MarshalIndent(taskInfo, "", "\t")
	log.WithField("payload", fmt.Sprintf("%s", jsonTask[:])).
		WithField("partition", envId.String()).
		WithField("detector", detector).
		Trace("received task to launch")

	myTask := executable.NewTask(taskInfo,
		makeSendStatusUpdateFunc(state, taskInfo),
		makeSendDeviceEventFunc(state),
		makeSendMessageFunc(state))

	err := myTask.Launch()

	if err == nil {
		state.activeTasksMu.Lock()
		state.activeTasks[taskInfo.TaskID] = myTask
		state.activeTasksMu.Unlock()
		log.WithFields(logrus.Fields{
			"taskId":    taskInfo.TaskID.GetValue(),
			"taskName":  taskInfo.Name,
			"hostname":  state.agent.GetHostname(),
			"level":     infologger.IL_Devel,
			"partition": envId.String(),
			"detector":  detector,
		}).Debug("task launching")
		return nil
	} else {
		// If Launch returned non-nil error, it should already have sent back a status update
		log.WithError(err).
			WithFields(logrus.Fields{
				"taskId":    taskInfo.TaskID.GetValue(),
				"taskName":  taskInfo.Name,
				"hostname":  state.agent.GetHostname(),
				"level":     infologger.IL_Devel,
				"partition": envId.String(),
				"detector":  detector,
			}).
			Error("task launch failed")
		return err
	}
}

// Attempts to kill a task. This function is thread-safe with respect to state.
func handleKillEvent(state *internalState, e *executor.Event_Kill) error {
	state.activeTasksMu.RLock()
	activeTask, ok := state.activeTasks[e.GetTaskID()]
	state.activeTasksMu.RUnlock()
	if !ok {
		return errors.New("invalid task ID")
	}

	go func() {
		_ = activeTask.Kill()
		activeTask = nil
		if ht, ok := activeTask.(*executable.HookTask); ok {
			// if it's a hook, it might be a DESTROY hook and therefore run after Kill
			// so we give it timeout seconds to stop, and in any case no more than 10s
			timeout := 10 * time.Second
			if ht.Tci.Timeout != 0 && ht.Tci.Timeout < timeout {
				timeout = ht.Tci.Timeout
			}

			// this timeout is necessary so any incoming trigger commands can still use
			// state.activeTasks after the task is killed i.e. formally not active any more
			select {
			case <-time.After(timeout):
				state.activeTasksMu.Lock()
				delete(state.activeTasks, e.GetTaskID())
				state.activeTasksMu.Unlock()
			}
		} else {
			state.activeTasksMu.Lock()
			delete(state.activeTasks, e.GetTaskID())
			state.activeTasksMu.Unlock()
		}
	}()

	return nil
}
