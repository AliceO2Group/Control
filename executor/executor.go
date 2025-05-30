/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2018-2020 CERN and copyright holders of ALICE O².
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

//go:generate protoc -I ../occ --go_out=. --go-grpc_out=. --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative protos/occ.proto

// Package executor implements the O² Control executor binary.
package executor

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"sync"
	"syscall"
	"time"

	"github.com/AliceO2Group/Control/common/logger"
	"github.com/AliceO2Group/Control/common/logger/infologger"
	"github.com/AliceO2Group/Control/common/utils/uid"
	"github.com/AliceO2Group/Control/executor/executable"
	"github.com/AliceO2Group/Control/executor/executorutil"
	"github.com/mesos/mesos-go/api/v1/lib"
	"github.com/mesos/mesos-go/api/v1/lib/backoff"
	"github.com/mesos/mesos-go/api/v1/lib/encoding/codecs"
	"github.com/mesos/mesos-go/api/v1/lib/executor"
	"github.com/mesos/mesos-go/api/v1/lib/executor/calls"
	"github.com/mesos/mesos-go/api/v1/lib/executor/config"
	"github.com/mesos/mesos-go/api/v1/lib/executor/events"
	"github.com/mesos/mesos-go/api/v1/lib/httpcli"
	"github.com/mesos/mesos-go/api/v1/lib/httpcli/httpexec"
	"github.com/pborman/uuid"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
)

const (
	apiPath     = "/api/v1/executor"
	httpTimeout = 10 * time.Second
)

var log = logger.New(logrus.StandardLogger(), "executor")

var errMustAbort = errors.New("executor received abort signal from Mesos, will attempt to re-subscribe")

// internalState of the executor.
type internalState struct {
	activeTasksMu  sync.RWMutex
	cli            calls.Sender
	cfg            config.Config
	framework      mesos.FrameworkInfo
	executor       mesos.ExecutorInfo
	agent          mesos.AgentInfo
	unackedTasks   map[mesos.TaskID]mesos.TaskInfo
	unackedUpdates map[string]executor.Call_Update
	failedTasks    map[mesos.TaskID]mesos.TaskStatus // send updates for these as we can
	killedTasks    map[mesos.TaskID]mesos.TaskStatus
	activeTasks    map[mesos.TaskID]executable.Task
	shouldQuit     bool

	statusCh  chan mesos.TaskStatus
	messageCh chan []byte
}

// maybeReconnect returns a backoff.Notifier chan if framework checkpointing is enabled.
func maybeReconnect(cfg config.Config) <-chan struct{} {
	if cfg.Checkpoint {
		return backoff.Notifier(1*time.Second, cfg.SubscriptionBackoffMax*3/4, nil)
	}
	return nil
}

// Run is the actual entry point of the executor.
func Run(cfg config.Config) {
	// Set memlock limit for child processes to unlimited. This only needs to
	// happen once per executor instance.
	// See O2-1459 O2-1412
	_ = syscall.Setrlimit(8 /* memlock magic number */, &syscall.Rlimit{
		Cur: ^uint64(0),
		Max: ^uint64(0),
	})

	var (
		apiURL = url.URL{
			Scheme: "http",
			Host:   cfg.AgentEndpoint,
			Path:   apiPath,
		}
		client = httpcli.New(
			httpcli.Endpoint(apiURL.String()),
			httpcli.Codec(codecs.ByMediaType[codecs.MediaTypeProtobuf]),
			httpcli.Do(httpcli.With(
				httpcli.Timeout(httpTimeout),
				httpcli.Transport(func(transport *http.Transport) {
					transport.Proxy = func(request *http.Request) (*url.URL, error) {
						return nil, nil
					}
				}),
			)),
		)

		// Fill in the Framework and Executor IDs as call parameters
		callOptions = executor.CallOptions{
			calls.Framework(cfg.FrameworkID),
			calls.Executor(cfg.ExecutorID),
		}
		state = &internalState{
			// With this we inject the callOptions into all outgoing calls
			cli: calls.SenderWith(
				httpexec.NewSender(client.Send),
				callOptions...,
			),
			// The executor keeps lists of unacknowledged tasks and unacknowledged updates. In case of unexpected
			// disconnection, the executor should SUBSCRIBE again and send these lists to Mesos in the SUBSCRIBE
			// call.
			unackedTasks:   make(map[mesos.TaskID]mesos.TaskInfo),
			unackedUpdates: make(map[string]executor.Call_Update),
			failedTasks:    make(map[mesos.TaskID]mesos.TaskStatus),
			killedTasks:    make(map[mesos.TaskID]mesos.TaskStatus),

			// The executor keeps a map of controlled tasks.
			activeTasks: make(map[mesos.TaskID]executable.Task),

			statusCh:  make(chan mesos.TaskStatus, 1024),
			messageCh: make(chan []byte),
		}
		subscriber = calls.SenderWith(
			// Here too, callOptions for all outgoing subscriber calls
			httpexec.NewSender(client.Send, httpcli.Close(true)),
			callOptions...,
		)

		// Chan which receives a struct every once in a while
		shouldReconnect = maybeReconnect(cfg)
		disconnected    = time.Now()
		handler         = buildEventHandler(state)
	)

	// Chmod executor directory from 750 to 755 because of OCTRL-766
	_ = syscall.Chmod(cfg.Directory, 0755)

	// Main loop for (re)subscription. Once we're subscribed, we jump into the event loop for handling the agent.
	for {
		func() {
			// We create the subscription call. If we haven't had an unclean disconnect, the lists of unacknowledged
			// tasks and updates are empty.
			subscribe := calls.Subscribe(unacknowledgedTasks(state), unacknowledgedUpdates(state))

			log.WithField("executorId", state.executor.GetExecutorID().Value).
				Debug("subscribing to agent for events")
			//                           ↓ empty context       ↓ we get a plain RequestFunc from the executor.Call value
			resp, err := subscriber.Send(context.TODO(), calls.NonStreaming(subscribe))
			if resp != nil {
				defer resp.Close()
			}
			if err == nil {
				log.WithField("level", infologger.IL_Support).
					WithField("executorId", state.executor.GetExecutorID().Value).
					Info("executor subscribed, ready to receive events")
				// We're officially connected, start decoding events
				err = eventLoop(state, resp, handler)
				// If we're out of the eventLoop, means a disconnect happened, willingly or not.
				disconnected = time.Now()
				log.WithField("level", infologger.IL_Support).
					WithField("executorId", state.executor.GetExecutorID().Value).
					Info("event loop finished")
			}
			if err != nil && err != io.EOF {
				log.WithField("error", err).
					WithField("executorId", state.executor.GetExecutorID().Value).
					Error("executor disconnected with error")
			} else {
				log.WithField("executorId", state.executor.GetExecutorID().Value).
					Info("executor disconnected")
			}
		}()
		if state.shouldQuit {
			log.WithField("executorId", state.executor.GetExecutorID().Value).
				Info("gracefully shutting down on request")
			return
		}
		// The purpose of checkpointing is to handle recovery when mesos-agent exits.
		if !cfg.Checkpoint {
			log.WithField("executorId", state.executor.GetExecutorID().Value).
				Info("gracefully exiting because framework checkpointing is not enabled")
			return
		}
		if time.Now().Sub(disconnected) > cfg.RecoveryTimeout {
			log.WithField("timeout", cfg.RecoveryTimeout).
				WithField("executorId", state.executor.GetExecutorID().Value).
				Error("failed to re-establish subscription with agent within recovery timeout, aborting")
			return
		}
		log.WithField("executorId", state.executor.GetExecutorID().Value).
			Debug("waiting for reconnect timeout")
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

// buildEventHandler builds an events.Handler, whose HandleEvent is triggered from the eventLoop.
func buildEventHandler(state *internalState) events.Handler {
	return events.HandlerFuncs{
		executor.Event_SUBSCRIBED: func(_ context.Context, e *executor.Event) error {
			log.WithField("event", e.Type.String()).Trace("handling event")

			// With this event we get FrameworkInfo, ExecutorInfo, AgentInfo:
			state.framework = e.Subscribed.FrameworkInfo
			state.executor = e.Subscribed.ExecutorInfo
			state.agent = e.Subscribed.AgentInfo
			return nil
		},
		executor.Event_LAUNCH: func(_ context.Context, e *executor.Event) error {
			// Launch one task. We're not handling LAUNCH_GROUP.
			log.WithField("event", e.Type.String()).Trace("handling event")

			return handleLaunchEvent(state, e.Launch.Task)
		},
		executor.Event_KILL: func(_ context.Context, e *executor.Event) error {
			log.WithField("event", e.Type.String()).Trace("handling event")

			return handleKillEvent(state, e.Kill)
		},
		executor.Event_ACKNOWLEDGED: func(_ context.Context, e *executor.Event) error {
			log.WithField("event", e.Type.String()).Trace("handling event")

			delete(state.unackedTasks, e.Acknowledged.TaskID)
			delete(state.unackedUpdates, string(e.Acknowledged.UUID))

			return nil
		},
		executor.Event_MESSAGE: func(_ context.Context, e *executor.Event) error {
			log.WithField("event", e.Type.String()).Trace("handling event")
			log.WithFields(logrus.Fields{
				"length":  len(e.Message.Data),
				"message": string(e.Message.Data[:]),
			}).
				Trace("received message data")
			err := handleMessageEvent(state, e.Message.Data)
			if err != nil {
				log.WithField("error", err.Error()).
					Debug("incoming MESSAGE handler error")
			}
			return nil // failed message handler (e.g. no such task) shouldn't cause an executor disconnect
		},
		executor.Event_SHUTDOWN: func(_ context.Context, e *executor.Event) error {
			log.WithField("event", e.Type.String()).Trace("handling event")
			state.shouldQuit = true
			return nil
		},
		executor.Event_ERROR: func(_ context.Context, e *executor.Event) error {
			log.WithField("event", e.Type.String()).Trace("handling event")
			return errMustAbort
		},
		executor.Event_HEARTBEAT: func(_ context.Context, e *executor.Event) error {
			log.WithField("event", e.Type.String()).Trace("heartbeat received from Mesos")
			return nil
		},
	}.Otherwise(func(_ context.Context, e *executor.Event) error {
		log.Warn("unexpected event received", e)
		return nil
	})
}

// update sends UPDATE to agent.
func update(state *internalState, status mesos.TaskStatus) error {
	envId := executorutil.GetEnvironmentIdFromLabelerType(&status)

	status.Timestamp = proto.Float64(float64(time.Now().Unix()))
	log.WithFields(logrus.Fields{
		"partition": envId.String(),
		"status":    status.State.String(),
		"id":        status.TaskID.Value,
	}).
		Debug("sending UPDATE on task status")
	upd := calls.Update(status)
	resp, err := state.cli.Send(context.TODO(), calls.NonStreaming(upd))
	if resp != nil {
		resp.Close()
	}
	if err != nil {
		log.WithField("error", err).
			WithField("partition", envId.String()).
			Error("executor failed to send update")
		debugJSON(upd)
	} else {
		state.unackedUpdates[string(status.UUID)] = *upd.Update
	}
	return err
}

// newStatus constructs a new mesos.TaskStatus to describe a task.
func newStatus(envId uid.ID, state *internalState, id mesos.TaskID) mesos.TaskStatus {
	envIdS := envId.String()
	return mesos.TaskStatus{
		TaskID:     id,
		Source:     mesos.SOURCE_EXECUTOR.Enum(),
		ExecutorID: &state.executor.ExecutorID,
		UUID:       []byte(uuid.NewRandom()),
		Labels: &mesos.Labels{
			Labels: []mesos.Label{{Key: "environmentId", Value: &envIdS}},
		},
	}
}
