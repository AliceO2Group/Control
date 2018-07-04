/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2018 CERN and copyright holders of ALICE O².
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

//go:generate protoc -I ../occplugin --gofast_out=plugins=grpc:. protos/OccPlugin.proto
//go:generate mv -f protos/OccPlugin.pb.go protos/occplugin.pb.go
package executor

import (
	"context"
	"errors"
	"io"
	"encoding/json"
	"net/url"
	"os/exec"
	"os"
	"time"

	"github.com/AliceO2Group/Control/common/logger"
	"github.com/mesos/mesos-go/api/v1/lib"
	"github.com/mesos/mesos-go/api/v1/lib/encoding"
	"github.com/mesos/mesos-go/api/v1/lib/executor"
	"github.com/mesos/mesos-go/api/v1/lib/executor/calls"
	"github.com/mesos/mesos-go/api/v1/lib/executor/config"
	"github.com/mesos/mesos-go/api/v1/lib/executor/events"
	"github.com/mesos/mesos-go/api/v1/lib/backoff"
	"github.com/mesos/mesos-go/api/v1/lib/encoding/codecs"
	"github.com/mesos/mesos-go/api/v1/lib/httpcli"
	"github.com/mesos/mesos-go/api/v1/lib/httpcli/httpexec"
	"github.com/pborman/uuid"
	"github.com/sirupsen/logrus"
	"github.com/AliceO2Group/Control/core/controlcommands"
	"github.com/AliceO2Group/Control/executor/protos"
	"google.golang.org/grpc"
	"fmt"
	"strconv"
)

const (
	apiPath     = "/api/v1/executor"
	httpTimeout = 10 * time.Second
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
			subscribe := calls.Subscribe(unacknowledgedTasks(state), unacknowledgedUpdates(state))

			log.Info("subscribing to agent for events")
			//                           ↓ empty context       ↓ we get a plain RequestFunc from the executor.Call value
			resp, err := subscriber.Send(context.TODO(), calls.NonStreaming(subscribe))
			if resp != nil {
				defer resp.Close()
			}
			if err == nil {
				// We're officially connected, start decoding events
				err = eventLoop(state, resp, handler)
				// If we're out of the eventLoop, means a disconnect happened, willingly or not.
				disconnected = time.Now()
			}
			if err != nil && err != io.EOF {
				log.WithField("error", err).Error("executor disconnected with error")
			} else {
				log.Info("executor disconnected")
			}
		}()
		if state.shouldQuit {
			log.Info("gracefully shutting down because we were told to")
			return
		}
		// The purpose of checkpointing is to handle recovery when mesos-agent exits.
		if !cfg.Checkpoint {
			log.Info("gracefully exiting because framework checkpointing is NOT enabled")
			return
		}
		if time.Now().Sub(disconnected) > cfg.RecoveryTimeout {
			log.WithField("timeout", cfg.RecoveryTimeout).
				Error("failed to re-establish subscription with agent within recovery timeout, aborting")
			return
		}
		log.Info("waiting for reconnect timeout")
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
	log.Info("listening for events from agent")
	ctx := context.TODO() // dummy context
	for err == nil && !state.shouldQuit {
		// housekeeping
		sendFailedTasks(state)

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
			// With this event we get FrameworkInfo, ExecutorInfo, AgentInfo:
			state.framework = e.Subscribed.FrameworkInfo
			state.executor = e.Subscribed.ExecutorInfo
			state.agent = e.Subscribed.AgentInfo
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
			// TODO: ask the process to kindly transition to the end state and quit, and if it doesn't do so gracefully,
			// kill it and report back with a MESSAGE.
			log.Warning("event KILL not implemented")
			return nil
		},
		executor.Event_ACKNOWLEDGED: func(_ context.Context, e *executor.Event) error {
			log.WithField("event", e.Type.String()).Debug("handling event")
			delete(state.unackedTasks, e.Acknowledged.TaskID)
			delete(state.unackedUpdates, string(e.Acknowledged.UUID))
			return nil
		},
		executor.Event_MESSAGE: func(_ context.Context, e *executor.Event) error {
			log.WithField("event", e.Type.String()).Debug("handling event")
			log.WithFields(logrus.Fields{
				"length":  len(e.Message.Data),
				"message": e.Message.Data,
			}).
			Debug("received message data")
			err := handleMessage(state, e.Message.Data)
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
		log.Fatal("unexpected event", e)
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
	var cmd struct {
		Name string `json:"name"`
	}
	err = json.Unmarshal(data, cmd)
	if err != nil {
		return
	}

	switch cmd.Name {
	case "MesosCommand_Transition":
		var cmd controlcommands.MesosCommand_Transition
		err = json.Unmarshal(data, cmd)
		if err != nil {
			return
		}
		return requestTransition(state, &cmd)
	default:
		return errors.New(fmt.Sprintf("unrecognized controlcommand %s", cmd.Name))
	}
}

func requestTransition(state *internalState, cmd *controlcommands.MesosCommand_Transition) (err error) {
	response, err := state.rpcClient.Transition(context.TODO(), &pb.TransitionRequest{
		Event: cmd.GetName(),
		Arguments: func() (cfg []*pb.ConfigEntry) {
			if len(cmd.Arguments) == 0 {
				return nil
			}
			for k, v := range cmd.Arguments {
				cfg = append(cfg, &pb.ConfigEntry{Key: k, Value: v})
			}
			return
		}(),
		SrcState: cmd.Source,
	}, grpc.EmptyCallOption{})

	if err != nil {
		return
	}
	if response.GetOk() &&
		response.GetTrigger() == pb.StateChangeTrigger_EXECUTOR &&
		response.GetEvent() == cmd.Event &&
		response.GetState() == cmd.Destination {
		return
	}

	return errors.New(fmt.Sprintf("transition unsuccessful: ok: %s, trigger: %s, event: %s, state: %s",
		strconv.FormatBool(response.GetOk()),
		response.GetTrigger().String(),
		response.GetEvent(),
		response.GetState()))
}

// launch tries to launch a task described by a mesos.TaskInfo.
func launch(state *internalState, task mesos.TaskInfo) {
	state.unackedTasks[task.TaskID] = task
	jsonTask, _ := json.MarshalIndent(task, "", "\t")
	log.WithField("task", string(jsonTask)).Debug("received task to launch")

	status := newStatus(state, task.TaskID)

	var commandInfo mesos.CommandInfo

	if err := json.Unmarshal(task.GetData(), &commandInfo); task.GetData() != nil && err == nil {
		log.WithFields(logrus.Fields{
			"shell": *commandInfo.Shell,
			"value": *commandInfo.Value,
			"args":  commandInfo.Arguments,
		}).
			Info("launching task")
	} else {
		if err != nil {
			log.WithField("error", err.Error()).Error("could not launch task")
		} else {
			log.WithField("error", "command data is nil").Error("could not launch task")
		}
		status.State = mesos.TASK_FAILED.Enum()
		status.Message = protoString("TaskInfo.Data is nil")
		state.failedTasks[task.TaskID] = status
		return
	}

	var taskCmd *exec.Cmd
	if *commandInfo.Shell {
		taskCmd = exec.Command("/bin/sh", append([]string{"-c"}, *commandInfo.Value)...)
	} else {
		taskCmd = exec.Command(*commandInfo.Value, commandInfo.Arguments...)
	}
	taskCmd.Env = append(os.Environ())

	var errStdout, errStderr error
	stdoutIn, _ := taskCmd.StdoutPipe()
	stderrIn, _ := taskCmd.StderrPipe()

	err := taskCmd.Start()
	if err != nil {
		log.WithFields(logrus.Fields{
			"task":    task.TaskID.Value,
			"error":   err,
			"command": *commandInfo.Value,
		}).
			Error("failed to run task")
		status.State = mesos.TASK_FAILED.Enum()
		status.Message = protoString(err.Error())
		state.failedTasks[task.TaskID] = status
		return
	}

	go func() {
		_, errStdout = io.Copy(log.WithPrefix("task-stdout").Writer(), stdoutIn)
	}()
	go func() {
		_, errStderr = io.Copy(log.WithPrefix("task-stderr").Writer(), stderrIn)
	}()

	// send RUNNING
	status.State = mesos.TASK_RUNNING.Enum()
	err = update(state, status)
	if err != nil {
		log.WithFields(logrus.Fields{
			"task":  task.TaskID.Value,
			"error": err,
		}).
			Error("failed to send TASK_RUNNING")
		status.State = mesos.TASK_FAILED.Enum()
		status.Message = protoString(err.Error())
		state.failedTasks[task.TaskID] = status
		if taskCmd.Process != nil {
			log.WithFields(logrus.Fields{
				"process": taskCmd.Process.Pid,
				"task":    task.TaskID.Value,
			}).
				Warning("killing leftover process")
			err = taskCmd.Process.Kill()
			if err != nil {
				log.WithFields(logrus.Fields{
					"process": taskCmd.Process.Pid,
					"task":    task.TaskID.Value,
				}).
					Error("cannot kill process")
			}
		}
		return
	}

	state.rpcClient = NewClient(state, 47101)

	err = taskCmd.Wait()
	if err != nil {
		log.WithFields(logrus.Fields{
			"task":  task.TaskID.Value,
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
		}).
			Warning("failed to capture stdout or stderr of task")
	}

	// send FINISHED
	status = newStatus(state, task.TaskID)
	status.State = mesos.TASK_FINISHED.Enum()
	err = update(state, status)
	if err != nil {
		log.WithFields(logrus.Fields{
			"task":  task.TaskID.Value,
			"error": err,
		}).
			Error("failed to send TASK_FINISHED")
		status.State = mesos.TASK_FAILED.Enum()
		status.Message = protoString(err.Error())
		state.failedTasks[task.TaskID] = status
	}
}

// helper func to package strings up nicely for protobuf
func protoString(s string) *string { return &s }

// update sends UPDATE to agent.
func update(state *internalState, status mesos.TaskStatus) error {
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
	cli            calls.Sender
	cfg            config.Config
	framework      mesos.FrameworkInfo
	executor       mesos.ExecutorInfo
	agent          mesos.AgentInfo
	unackedTasks   map[mesos.TaskID]mesos.TaskInfo
	unackedUpdates map[string]executor.Call_Update
	failedTasks    map[mesos.TaskID]mesos.TaskStatus // send updates for these as we can
	rpcClient      *RpcClient
	shouldQuit     bool
}
