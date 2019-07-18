/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2018-2019 CERN and copyright holders of ALICE O².
 * Author: Teo Mrnjavac <teo.mrnjavac@cern.ch>
 *
 * Portions from examples in <https://github.com/mesos/mesos-go>:
 *     Copyright 2013-2015, Mesosphere, Inc.
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

//go:generate protoc -I ../occ --gofast_out=plugins=grpc:. protos/occ.proto

// Package executor implements the O² Control executor binary.
package executor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/AliceO2Group/Control/common"
	"github.com/AliceO2Group/Control/common/event"
	"github.com/AliceO2Group/Control/common/logger"
	"github.com/AliceO2Group/Control/core/controlcommands"
	"github.com/AliceO2Group/Control/executor/executorcmd"
	"github.com/AliceO2Group/Control/executor/protos"
	"github.com/golang/protobuf/proto"
	"github.com/mesos/mesos-go/api/v1/lib"
	"github.com/mesos/mesos-go/api/v1/lib/backoff"
	"github.com/mesos/mesos-go/api/v1/lib/encoding"
	"github.com/mesos/mesos-go/api/v1/lib/encoding/codecs"
	"github.com/mesos/mesos-go/api/v1/lib/executor"
	"github.com/mesos/mesos-go/api/v1/lib/executor/calls"
	"github.com/mesos/mesos-go/api/v1/lib/executor/config"
	"github.com/mesos/mesos-go/api/v1/lib/executor/events"
	"github.com/mesos/mesos-go/api/v1/lib/httpcli"
	"github.com/mesos/mesos-go/api/v1/lib/httpcli/httpexec"
	"github.com/pborman/uuid"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

const (
	apiPath                = "/api/v1/executor"
	httpTimeout            = 10 * time.Second
	startupPollingInterval = 500 * time.Millisecond
	startupTimeout         = 30 * time.Second
)

var log = logger.New(logrus.StandardLogger(), "executor")

var errMustAbort = errors.New("executor received abort signal from Mesos, will attempt to re-subscribe")


// maybeReconnect returns a backoff.Notifier chan if framework checkpointing is enabled.
func maybeReconnect(cfg config.Config) <-chan struct{} {
	if cfg.Checkpoint {
		return backoff.Notifier(1*time.Second, cfg.SubscriptionBackoffMax*3/4, nil)
	}
	return nil
}

// Run is the actual entry point of the executor.
func Run(cfg config.Config) {
	var (
		apiURL = url.URL{
			Scheme: "http",
			Host:   cfg.AgentEndpoint,
			Path:   apiPath,
		}
		http = httpcli.New(
			httpcli.Endpoint(apiURL.String()),
			httpcli.Codec(codecs.ByMediaType[codecs.MediaTypeProtobuf]),
			httpcli.Do(httpcli.With(httpcli.Timeout(httpTimeout))),
		)

		// Fill in the Framework and Executor IDs as call parameters
		callOptions = executor.CallOptions{
			calls.Framework(cfg.FrameworkID),
			calls.Executor(cfg.ExecutorID),
		}
		state = &internalState{
			// With this we inject the callOptions into all outgoing calls
			cli: calls.SenderWith(
				httpexec.NewSender(http.Send),
				callOptions...,
			),
			// The executor keeps lists of unacknowledged tasks and unacknowledged updates. In case of unexpected
			// disconnection, the executor should SUBSCRIBE again and send these lists to Mesos in the SUBSCRIBE
			// call.
			unackedTasks:   make(map[mesos.TaskID]mesos.TaskInfo),
			unackedUpdates: make(map[string]executor.Call_Update),
			failedTasks:    make(map[mesos.TaskID]mesos.TaskStatus),
			killedTasks:    make(map[mesos.TaskID]mesos.TaskStatus),
			rpcClients:     make(map[mesos.TaskID]*executorcmd.RpcClient),
		}
		subscriber = calls.SenderWith(
			// Here too, callOptions for all outgoing subscriber calls
			httpexec.NewSender(http.Send, httpcli.Close(true)),
			callOptions...,
		)

		// Chan which receives a struct every once in a while
		shouldReconnect = maybeReconnect(cfg)
		disconnected    = time.Now()
		handler         = buildEventHandler(state)
	)

	// Main loop for (re)subscription. Once we're subscribed, we jump into the event loop for handling the agent.
	for {
		func() {
			// We create the subscription call. If we haven't had an unclean disconnect, the lists of unacknowledged
			// tasks and updates are empty.
			state.mu.RLock()
			subscribe := calls.Subscribe(unacknowledgedTasks(state), unacknowledgedUpdates(state))
			state.mu.RUnlock()

			log.Debug("subscribing to agent for events")
			//                           ↓ empty context       ↓ we get a plain RequestFunc from the executor.Call value
			resp, err := subscriber.Send(context.TODO(), calls.NonStreaming(subscribe))
			if resp != nil {
				defer resp.Close()
			}
			if err == nil {
				log.Info("executor subscribed, ready to receive events")
				// We're officially connected, start decoding events
				err = eventLoop(state, resp, handler)
				// If we're out of the eventLoop, means a disconnect happened, willingly or not.
				disconnected = time.Now()
				log.Debug("event loop finished")
			}
			if err != nil && err != io.EOF {
				log.WithField("error", err).Error("executor disconnected with error")
			} else {
				log.Info("executor disconnected")
			}
		}()
		if state.shouldQuit {
			log.Info("gracefully shutting down on request")
			return
		}
		// The purpose of checkpointing is to handle recovery when mesos-agent exits.
		if !cfg.Checkpoint {
			log.Info("gracefully exiting because framework checkpointing is not enabled")
			return
		}
		if time.Now().Sub(disconnected) > cfg.RecoveryTimeout {
			log.WithField("timeout", cfg.RecoveryTimeout).
				Error("failed to re-establish subscription with agent within recovery timeout, aborting")
			return
		}
		log.Debug("waiting for reconnect timeout")
		<-shouldReconnect // wait for some amount of time before retrying subscription
	}
}

// unacknowledgedTasks generates the value of the UnacknowledgedTasks field of a Subscribe call.
func unacknowledgedTasks(state *internalState) (result []mesos.TaskInfo) {
	if n := len(state.unackedTasks); n > 0 {
		result = make([]mesos.TaskInfo, 0, n)
		for k := range state.unackedTasks {
			result = append(result, state.unackedTasks[k])
		}
	}
	return
}

// unacknowledgedUpdates generates the value of the UnacknowledgedUpdates field of a Subscribe call.
func unacknowledgedUpdates(state *internalState) (result []executor.Call_Update) {
	if n := len(state.unackedUpdates); n > 0 {
		result = make([]executor.Call_Update, 0, n)
		for k := range state.unackedUpdates {
			result = append(result, state.unackedUpdates[k])
		}
	}
	return
}

// eventLoop dispatches incoming events from mesos-agent to the events.Handler (built in buildEventhandler).
func eventLoop(state *internalState, decoder encoding.Decoder, h events.Handler) (err error) {
	log.Debug("listening for events from agent")
	ctx := context.TODO() // dummy context
	for err == nil && !state.shouldQuit {
		log.Debug("begin new event loop iteration")
		// housekeeping
		state.mu.Lock()
		sendFailedTasks(state)
		state.mu.Unlock()

		var e executor.Event
		if err = decoder.Decode(&e); err == nil {
			err = h.HandleEvent(ctx, &e)
		}
	}
	return err
}

// buildEventHandler builds an events.Handler, whose HandleEvent is triggered from the eventLoop.
func buildEventHandler(state *internalState) events.Handler {
	return events.HandlerFuncs{
		executor.Event_SUBSCRIBED: func(_ context.Context, e *executor.Event) error {
			log.WithField("event", e.Type.String()).Debug("handling event")

			state.mu.Lock()
			// With this event we get FrameworkInfo, ExecutorInfo, AgentInfo:
			state.framework = e.Subscribed.FrameworkInfo
			state.executor = e.Subscribed.ExecutorInfo
			state.agent = e.Subscribed.AgentInfo
			state.mu.Unlock()
			return nil
		},
		executor.Event_LAUNCH: func(_ context.Context, e *executor.Event) error {
			// Launch one task. We're not handling LAUNCH_GROUP.
			log.WithField("event", e.Type.String()).Debug("handling event")
			launch(state, e.Launch.Task)
			return nil
		},
		executor.Event_KILL: func(_ context.Context, e *executor.Event) error {
			log.WithField("event", e.Type.String()).Debug("handling event")

			return kill(state, e.Kill)
		},
		executor.Event_ACKNOWLEDGED: func(_ context.Context, e *executor.Event) error {
			log.WithField("event", e.Type.String()).Debug("handling event")

			state.mu.Lock()
			delete(state.unackedTasks, e.Acknowledged.TaskID)
			delete(state.unackedUpdates, string(e.Acknowledged.UUID))
			state.mu.Unlock()

			return nil
		},
		executor.Event_MESSAGE: func(_ context.Context, e *executor.Event) error {
			log.WithField("event", e.Type.String()).Debug("handling event")
			log.WithFields(logrus.Fields{
				"length":  len(e.Message.Data),
				"message": string(e.Message.Data[:]),
			}).
			Debug("received message data")
			err := handleMessage(state, e.Message.Data)
			if err != nil {
				log.WithField("error", err.Error()).Debug("MESSAGE handler error")
			}
			return err
		},
		executor.Event_SHUTDOWN: func(_ context.Context, e *executor.Event) error {
			log.WithField("event", e.Type.String()).Debug("handling event")
			state.shouldQuit = true
			return nil
		},
		executor.Event_ERROR: func(_ context.Context, e *executor.Event) error {
			log.WithField("event", e.Type.String()).Debug("handling event")
			return errMustAbort
		},
	}.Otherwise(func(_ context.Context, e *executor.Event) error {
		log.Error("unexpected event", e)
		return nil
	})
}

// sendFailedTasks runs on every iteration of eventLoop to send an UPDATE on any failed tasks to the agent.
func sendFailedTasks(state *internalState) {
	for taskID, status := range state.failedTasks {
		updateErr := update(state, status)
		if updateErr != nil {
			log.WithFields(logrus.Fields{
				"taskId": taskID.Value,
				"error":  updateErr,
			}).
				Error("failed to send status update for task")
		} else {
			// If we have successfully notified Mesos, we clear our list of failed tasks.
			delete(state.failedTasks, taskID)
		}
	}
}

func handleMessage(state *internalState, data []byte) (err error) {
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

	switch incoming.Name {
	case "MesosCommand_Transition":
		taskId := incoming.TargetList[0].TaskId

		state.mu.RLock()
		// Check whether we have a control connection to the running task
		_, ok := state.rpcClients[taskId]
		if !ok {
			err = fmt.Errorf("no RPC client for taskId %s", taskId.Value)
			log.WithFields(logrus.Fields{
					"name": incoming.Name,
					"message": string(data[:]),
					"error": err.Error(),
				}).
				Error("cannot unmarshal incoming MESSAGE")
			state.mu.RUnlock()
			return
		}

		var cmd *executorcmd.ExecutorCommand_Transition
		cmd, err = state.rpcClients[taskId].UnmarshalTransition(data)
		if err != nil {
			log.WithFields(logrus.Fields{
					"name": cmd.Name,
					"message": string(data[:]),
					"error": err.Error(),
				}).
				Error("cannot unmarshal incoming MESSAGE")
			state.mu.RUnlock()
			return
		}
		state.mu.RUnlock()

		if cmd.Name == "CONFIGURE" {
			log.WithFields(logrus.Fields{"map": cmd.Arguments, "taskId": taskId}).Debug("CONFIGURE pushing FairMQ properties")
		}

		newState, transitionError := cmd.Commit()

		go func() {
			response := cmd.PrepareResponse(transitionError, newState, taskId.Value)
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
			state.mu.Lock()
			defer state.mu.Unlock()
			state.cli.Send(context.TODO(), calls.NonStreaming(calls.Message(data)))
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

// launch tries to launch a task described by a mesos.TaskInfo.
func launch(state *internalState, task mesos.TaskInfo) {
	state.mu.Lock()

	state.unackedTasks[task.TaskID] = task
	jsonTask, _ := json.MarshalIndent(task, "", "\t")
	log.WithField("payload", fmt.Sprintf("%s", jsonTask[:])).Debug("received task to launch")

	status := newStatus(state, task.TaskID)

	var commandInfo common.TaskCommandInfo

	tciData := task.GetData()

	log.WithField("json", string(tciData[:])).Debug("received TaskCommandInfo")
	if err := json.Unmarshal(tciData, &commandInfo); tciData != nil && err == nil {
		log.WithFields(logrus.Fields{
				"shell": *commandInfo.Shell,
				"value": *commandInfo.Value,
				"args":  commandInfo.Arguments,
				"task": task.Name,
			}).
			Info("launching task")
	} else {
		if err != nil {
			log.WithError(err).WithField("task", task.Name).Error("could not launch task")
		} else {
			log.WithError(errors.New("command data is nil")).WithField("task", task.Name).Error("could not launch task")
		}
		status.State = mesos.TASK_FAILED.Enum()
		status.Message = protoString("TaskInfo.Data is nil")
		state.failedTasks[task.TaskID] = status
		state.mu.Unlock()
		return
	}
	state.mu.Unlock()

	var taskCmd *exec.Cmd
	if *commandInfo.Shell {
		rawCommand := strings.Join(append([]string{*commandInfo.Value}, commandInfo.Arguments...), " ")
		taskCmd = exec.Command("/bin/sh", []string{"-c", rawCommand}...)
	} else {
		taskCmd = exec.Command(*commandInfo.Value, commandInfo.Arguments...)
	}
	taskCmd.Env = append(os.Environ(), commandInfo.Env...)

	var errStdout, errStderr error
	stdoutIn, _ := taskCmd.StdoutPipe()
	stderrIn, _ := taskCmd.StderrPipe()

	log.WithField("payload", string(task.GetData()[:])).WithField("task", task.Name).Debug("starting task")
	err := taskCmd.Start()
	if err != nil {
		log.WithFields(logrus.Fields{
				"id":      task.TaskID.Value,
				"task":    task.Name,
				"error":   err,
				"command": *commandInfo.Value,
			}).
			Error("failed to run task")
		status.State = mesos.TASK_FAILED.Enum()
		status.Message = protoString(err.Error())
		state.mu.Lock()
		state.failedTasks[task.TaskID] = status
		state.mu.Unlock()
		return
	}
	log.WithField("id", task.TaskID.Value).WithField("task", task.Name).Debug("task started")

	go func() {
		_, errStdout = io.Copy(log.WithPrefix("task-stdout").WithField("task", task.Name).Writer(), stdoutIn)
	}()
	go func() {
		_, errStderr = io.Copy(log.WithPrefix("task-stderr").WithField("task", task.Name).Writer(), stderrIn)
	}()

	state.mu.Lock()
	log.WithFields(logrus.Fields{
			"controlPort": commandInfo.ControlPort,
			"controlMode": commandInfo.ControlMode.String(),
			"task": task.Name,
			"id": task.TaskID.Value,
		}).
		Debug("starting gRPC client")
	state.rpcClients[task.TaskID] = executorcmd.NewClient(commandInfo.ControlPort, commandInfo.ControlMode)
	state.rpcClients[task.TaskID].TaskCmd = taskCmd
	state.mu.Unlock()

	go func() {
		elapsed := 0 * time.Second
		for {
			log.WithFields(logrus.Fields{
					"id":      task.TaskID.Value,
					"task":    task.Name,
					"elapsed": elapsed.String(),
				}).
				Debug("polling task for IDLE state reached")
			state.mu.RLock()
			response, err := state.rpcClients[task.TaskID].GetState(context.TODO(), &pb.GetStateRequest{}, grpc.EmptyCallOption{})
			if err != nil {
				log.WithError(err).WithField("task", task.Name).Info("cannot query task status")
			} else {
				log.WithField("state", response.GetState()).WithField("task", task.Name).Debug("task status queried")
			}
			// NOTE: we acquire the transitioner-dependent STANDBY equivalent state
			reachedState := state.rpcClients[task.TaskID].FromDeviceState(response.GetState())
			state.mu.RUnlock()

			if reachedState == "STANDBY" && err == nil {
				log.WithField("id", task.TaskID.Value).
					WithField("task", task.Name).
					Debug("task running and ready for control input")
				break
			} else if elapsed >= startupTimeout {
				err = errors.New("timeout while waiting for task startup")
				log.WithField("task", task.Name).Error(err.Error())
				status.State = mesos.TASK_FAILED.Enum()
				status.Message = protoString(err.Error())

				state.mu.Lock()
				state.failedTasks[task.TaskID] = status
				state.rpcClients[task.TaskID].Close()
				delete(state.rpcClients, task.TaskID)
				state.mu.Unlock()
				return
			} else {
				log.WithField("task", task.Name).Debugf("task not ready yet, waiting %s", startupPollingInterval.String())
				time.Sleep(startupPollingInterval)
				elapsed += startupPollingInterval
			}
		}

		// Set up event stream from task
		state.mu.RLock()
		esc, err := state.rpcClients[task.TaskID].EventStream(context.TODO(), &pb.EventStreamRequest{}, grpc.EmptyCallOption{})
		state.mu.RUnlock()
		if err != nil {
			log.WithField("task", task.Name).WithError(err).Error("cannot set up event stream from task")
			status.State = mesos.TASK_FAILED.Enum()
			status.Message = protoString(err.Error())

			state.mu.Lock()
			state.failedTasks[task.TaskID] = status
			state.rpcClients[task.TaskID].Close()
			delete(state.rpcClients, task.TaskID)
			state.mu.Unlock()
			return
		}

		log.WithField("task", task.Name).Debug("notifying of task running state")
		status.State = mesos.TASK_RUNNING.Enum()

		// send RUNNING
		state.mu.Lock()
		err = update(state, status)
		if err != nil {
			log.WithFields(logrus.Fields{
					"id":  task.TaskID.Value,
					"task": task.Name,
					"error": err.Error(),
				}).
				Error("failed to send TASK_RUNNING")
			status.State = mesos.TASK_FAILED.Enum()
			status.Message = protoString(err.Error())

			state.failedTasks[task.TaskID] = status
			state.rpcClients[task.TaskID].Close()
			delete(state.rpcClients, task.TaskID)

			if taskCmd.Process != nil {
				log.WithFields(logrus.Fields{
						"process": taskCmd.Process.Pid,
						"id":      task.TaskID.Value,
						"task":    task.Name,
					}).
					Warning("killing leftover process")
				err = taskCmd.Process.Kill()
				if err != nil {
					log.WithFields(logrus.Fields{
							"process": taskCmd.Process.Pid,
							"id":      task.TaskID.Value,
							"task":    task.Name,
						}).
						Error("cannot kill process")
				}
			}
			state.mu.Unlock()
			return
		}
		state.mu.Unlock()

		// Process events from task
		go func() {
			deo := event.DeviceEventOrigin{
				AgentId: task.AgentID,
				ExecutorId: task.GetExecutor().ExecutorID,
				TaskId: task.TaskID,
			}
			for {
				if _, ok := state.rpcClients[task.TaskID]; !ok {
					log.WithError(err).Warning("event stream done")
					break
				}
				esr, err := esc.Recv()
				if err == io.EOF {
					log.WithError(err).Warning("event stream EOF")
					break
				}
				if err != nil {
					log.WithError(err).Warning("error receiving event from task")
					continue
				}
				ev := esr.GetEvent()

				deviceEvent := event.NewDeviceEvent(deo, ev.GetType())
				if deviceEvent == nil {
					log.Debug("nil DeviceEvent received (NULL_DEVICE_EVENT) - closing stream")
					break
				}

				jsonEvent, err := json.Marshal(deviceEvent)
				if err != nil {
					log.WithError(err).Warning("error marshaling event from task")
					continue
				}

				state.mu.RLock()
				state.cli.Send(context.TODO(), calls.NonStreaming(calls.Message(jsonEvent)))
				log.WithFields(logrus.Fields{
						"task": task.TaskID.Value,
						"event": string(jsonEvent),
					}).
					Debug("event sent")
				state.mu.RUnlock()
			}
		}()

		err = taskCmd.Wait()

		state.mu.Lock()
		defer state.mu.Unlock()
		if _, ok := state.rpcClients[task.TaskID]; ok {
			state.rpcClients[task.TaskID].Close() // NOTE: might return non-nil error, but we don't care much
			log.Debug("rpc client closed")
			delete(state.rpcClients, task.TaskID)
			log.Debug("rpc client removed")
		}

		if _, ok := state.killedTasks[task.TaskID]; !ok && err != nil {
			log.WithFields(logrus.Fields{
					"id":    task.TaskID.Value,
					"task":  task.Name,
					"error": err.Error(),
				}).
				Error("process terminated with error")
			status.State = mesos.TASK_FAILED.Enum()
			status.Message = protoString(err.Error())
			state.failedTasks[task.TaskID] = status
			return
		}

		if errStdout != nil || errStderr != nil {
			log.WithFields(logrus.Fields{
					"errStderr": errStderr,
					"errStdout": errStdout,
					"id":        task.TaskID.Value,
					"task":      task.Name,
				}).
				Warning("failed to capture stdout or stderr of task")
		}

		if _, ok := state.killedTasks[task.TaskID]; ok {
			status = state.killedTasks[task.TaskID]
			delete(state.killedTasks, task.TaskID)
		} else {
			status = newStatus(state, task.TaskID)
			status.State = mesos.TASK_FAILED.Enum()
		}
		log.WithField("task", task.Name).
			WithField("status", status.State.String()).
			Debug("sending final status update")

		err = update(state, status)
		if err != nil {
			log.WithFields(logrus.Fields{
					"id":    task.TaskID.Value,
					"name":  task.Name,
					"error": err.Error(),
				}).
				Error("failed to send final status update")
			status.State = mesos.TASK_FAILED.Enum()
			status.Message = protoString(err.Error())
			state.failedTasks[task.TaskID] = status
		}
	}()
	log.WithField("task", task.Name).Debug("gRPC client running, handler forked")
}

func kill(state *internalState, e *executor.Event_Kill) error {
	state.mu.RLock()
	rpcClient, ok := state.rpcClients[e.GetTaskID()]
	if !ok {
		state.mu.RUnlock()
		return errors.New("invalid task ID")
	}
	response, err := rpcClient.GetState(context.TODO(), &pb.GetStateRequest{}, grpc.EmptyCallOption{})
	if err != nil {
		log.WithError(err).WithField("taskId", e.GetTaskID()).Error("cannot query task status")
	} else {
		log.WithField("state", response.GetState()).WithField("taskId", e.GetTaskID()).Debug("task status queried")
	}
	// NOTE: we acquire the transitioner-dependent STANDBY equivalent state
	reachedState := rpcClient.FromDeviceState(response.GetState())
	state.mu.RUnlock()

	nextTransition := func(currentState string) (exc *executorcmd.ExecutorCommand_Transition) {
		log.WithField("currentState", currentState).
			Debug("nextTransition(currentState) BEGIN")
		var evt, destination string
		switch currentState {
		case "RUNNING":
			evt = "STOP"
			destination = "CONFIGURED"
		case "CONFIGURED":
			evt = "RESET"
			destination = "STANDBY"
		case "ERROR":
			evt = "RECOVER"
			destination = "STANDBY"
		case "STANDBY":
			evt = "EXIT"
			destination = "DONE"
		}
		log.WithField("evt", evt).
			WithField("dst", destination).
			Debug("nextTransition(currentState) BEGIN")

		exc = executorcmd.NewLocalExecutorCommand_Transition(
			rpcClient,
			[]controlcommands.MesosCommandTarget{
				{
					AgentId: *state.agent.GetID(),
					ExecutorId: state.executor.GetExecutorID(),
					TaskId: e.GetTaskID(),
				},
			},
			reachedState,
			evt,
			destination,
			nil,
		)
		return
	}

	for reachedState != "DONE" {
		cmd := nextTransition(reachedState)
		log.WithFields(logrus.Fields{
				"evt": cmd.Event,
				"src": cmd.Source,
				"dst": cmd.Destination,
				"targetList": cmd.TargetList,
			}).
			Debug("state DONE not reached, about to commit transition")
		newState, transitionError := cmd.Commit()
		log.WithField("newState", newState).
			WithError(transitionError).
			Debug("transition committed")
		if transitionError != nil || len(cmd.Event) == 0 {
			log.WithError(transitionError).Error("cannot gracefully end task")
			break
		}
		reachedState = newState
	}

	log.Debug("end transition loop done")

	state.mu.Lock()
	log.Debug("state locked")

	status := newStatus(state, e.GetTaskID())

	_ = rpcClient.TaskCmd.Process.Kill()

	if reachedState == "DONE" {
		log.Debug("task exited correctly")
		status.State = mesos.TASK_FINISHED.Enum()
	} else { // something went wrong
		log.Debug("task killed")
		status.State = mesos.TASK_KILLED.Enum()
	}
	state.killedTasks[e.GetTaskID()] = status

	log.Debug("unlocking state")
	state.mu.Unlock()
	return err
}

// helper func to package strings up nicely for protobuf
func protoString(s string) *string { return &s }

// update sends UPDATE to agent.
func update(state *internalState, status mesos.TaskStatus) error {
	status.Timestamp = proto.Float64(float64(time.Now().Unix()))
	log.WithFields(logrus.Fields{
			"status": status.State.String(),
			"id":     status.TaskID.Value,
		}).
		Debug("sending UPDATE on task status")
	upd := calls.Update(status)
	resp, err := state.cli.Send(context.TODO(), calls.NonStreaming(upd))
	if resp != nil {
		resp.Close()
	}
	if err != nil {
		log.WithField("error", err).Error("failed to send update")
		debugJSON(upd)
	} else {
		state.unackedUpdates[string(status.UUID)] = *upd.Update
	}
	return err
}

// newStatus constructs a new mesos.TaskStatus to describe a task.
func newStatus(state *internalState, id mesos.TaskID) mesos.TaskStatus {
	return mesos.TaskStatus{
		TaskID:     id,
		Source:     mesos.SOURCE_EXECUTOR.Enum(),
		ExecutorID: &state.executor.ExecutorID,
		UUID:       []byte(uuid.NewRandom()),
	}
}

// internalState of the executor.
type internalState struct {
	mu             sync.RWMutex
	cli            calls.Sender
	cfg            config.Config
	framework      mesos.FrameworkInfo
	executor       mesos.ExecutorInfo
	agent          mesos.AgentInfo
	unackedTasks   map[mesos.TaskID]mesos.TaskInfo
	unackedUpdates map[string]executor.Call_Update
	failedTasks    map[mesos.TaskID]mesos.TaskStatus // send updates for these as we can
	killedTasks    map[mesos.TaskID]mesos.TaskStatus
	rpcClients     map[mesos.TaskID]*executorcmd.RpcClient
	shouldQuit     bool
}
