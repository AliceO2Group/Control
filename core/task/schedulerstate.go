/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2017-2018 CERN and copyright holders of ALICE O².
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

package task

import (
	"context"
	"math/rand"
	"sync"
	"time"

	"github.com/AliceO2Group/Control/common/logger/infologger"
	"github.com/AliceO2Group/Control/core/controlcommands"
	"github.com/AliceO2Group/Control/core/task/schedutil"
	"github.com/gogo/protobuf/proto"
	"github.com/looplab/fsm"
	mesos "github.com/mesos/mesos-go/api/v1/lib"
	"github.com/mesos/mesos-go/api/v1/lib/backoff"
	"github.com/mesos/mesos-go/api/v1/lib/extras/scheduler/callrules"
	"github.com/mesos/mesos-go/api/v1/lib/extras/store"
	"github.com/mesos/mesos-go/api/v1/lib/scheduler"
	"github.com/mesos/mesos-go/api/v1/lib/scheduler/calls"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

const (
	MAX_CONCURRENT_DEPLOY_REQUESTS           = 100
	MAX_ATTEMPTS_PER_DEPLOY_REQUEST          = 5
	SLEEP_LENGTH_BETWEEN_PER_DEPLOY_REQUESTS = 1 // in seconds
)

type schedulerState struct {
	sync.RWMutex

	sm *fsm.FSM

	fidStore store.Singleton

	// needs locking:
	wantsTaskResources mesos.Resources
	err                error

	// not used in multiple goroutines:
	executor      *mesos.ExecutorInfo
	reviveTokens  <-chan struct{}
	tasksToDeploy chan *ResourceOffersDeploymentRequest

	reviveOffersTrg chan struct{}
	random          *rand.Rand

	// shouldn't change at runtime, so thread safe:
	role     string
	cli      calls.Caller
	shutdown func()

	// uses prometheus counters, so thread safe
	metricsAPI *metricsAPI

	// uses locks, so thread safe
	servent      *controlcommands.Servent
	commandqueue *controlcommands.CommandQueue
	taskman      *Manager
}

func NewScheduler(taskman *Manager, fidStore store.Singleton, shutdown func()) (*schedulerState, error) {
	metricsAPI := initMetrics()
	executorInfo, err := schedutil.PrepareExecutorInfo(
		viper.GetString("executor"),
		viper.GetString("mesosExecutorImage"),
		schedutil.BuildWantsExecutorResources(viper.GetFloat64("executorCPU"),
			viper.GetFloat64("executorMemory")),
		viper.GetDuration("mesosJobRestartDelay"),
	)
	if err != nil {
		return nil, err
	}
	creds, err := schedutil.LoadCredentials(viper.GetString("mesosCredentials.username"),
		viper.GetString("mesosCredentials.password"))
	if err != nil {
		return nil, err
	}

	tasksToDeploy := make(chan *ResourceOffersDeploymentRequest, MAX_CONCURRENT_DEPLOY_REQUESTS)

	reviveOffersTrg := make(chan struct{})

	state := &schedulerState{
		taskman:  taskman,
		fidStore: fidStore,
		reviveTokens: backoff.BurstNotifier(
			viper.GetInt("mesosReviveBurst"),
			viper.GetDuration("mesosReviveWait"),
			viper.GetDuration("mesosReviveWait"),
			nil),
		tasksToDeploy:      tasksToDeploy,
		reviveOffersTrg:    reviveOffersTrg,
		wantsTaskResources: mesos.Resources{},
		executor:           executorInfo,
		metricsAPI:         metricsAPI,
		cli:                schedutil.BuildHTTPSched(creds),
		random:             rand.New(rand.NewSource(time.Now().Unix())),
		shutdown:           shutdown,
	}

	state.servent = controlcommands.NewServent(
		func(command controlcommands.MesosCommand, receiver controlcommands.MesosCommandTarget) error {
			return state.sendCommand(context.Background(), command, receiver)
		},
	)
	state.commandqueue = controlcommands.NewCommandQueue(state.servent)

	state.commandqueue.Start() // FIXME: should there be 1 cq per env?

	state.sm = fsm.NewFSM(
		"INITIAL",
		fsm.Events{
			{Name: "CONNECT", Src: []string{"INITIAL"}, Dst: "CONNECTED"},
			{Name: "NEW_ENVIRONMENT", Src: []string{"CONNECTED"}, Dst: "CONNECTED"},
			{Name: "GO_ERROR", Src: []string{"CONNECTED"}, Dst: "ERROR"},
			{Name: "RESET", Src: []string{"ERROR"}, Dst: "INITIAL"},
			{Name: "EXIT", Src: []string{"CONNECTED"}, Dst: "FINAL"},
		},
		fsm.Callbacks{
			"before_event": func(_ context.Context, e *fsm.Event) {
				log.WithFields(logrus.Fields{
					"event": e.Event,
					"src":   e.Src,
					"dst":   e.Dst,
				}).Debug("state.sm starting transition")
			},
			"enter_state": func(_ context.Context, e *fsm.Event) {
				log.WithFields(logrus.Fields{
					"event": e.Event,
					"src":   e.Src,
					"dst":   e.Dst,
				}).Debug("state.sm entering state")
			},
			"leave_CONNECTED": func(_ context.Context, e *fsm.Event) {
				log.Debug("leave_CONNECTED")
			},
			"before_NEW_ENVIRONMENT": func(_ context.Context, e *fsm.Event) {
				log.Debug("before_NEW_ENVIRONMENT")
				e.Async() // transition frozen until the corresponding fsm.Transition call
			},
			"enter_CONNECTED": func(_ context.Context, e *fsm.Event) {
				log.Debug("enter_CONNECTED")
				log.WithField("level", infologger.IL_Support).
					Info("scheduler connected")
			},
			"after_NEW_ENVIRONMENT": func(_ context.Context, e *fsm.Event) {
				log.Debug("after_NEW_ENVIRONMENT")
			},
		},
	)

	return state, nil
}

func (state *schedulerState) setupCli() calls.Caller {
	// callrules.New returns a Rules and accept a bunch of Rule values as arguments.
	// WithFrameworkID returns a Rule which injects a frameworkID to outgoing calls.
	// logCalls returns a rule which prints to the log all calls of type SUBSCRIBE.
	// callMetrics logs metrics for every outgoing call.
	state.cli = callrules.New(
		callrules.WithFrameworkID(store.GetIgnoreErrors(state.fidStore)),
		logCalls(map[scheduler.Call_Type]string{scheduler.Call_SUBSCRIBE: "subscribe connecting"}),
		callMetrics(state.metricsAPI, time.Now, viper.GetBool("summaryMetrics")),
	).Caller(state.cli)

	return state.cli
}

func (state *schedulerState) GetFrameworkID() string {
	return store.GetIgnoreErrors(state.fidStore)()
}

func (state *schedulerState) Start(ctx context.Context) {
	// Async start of the scheduler controller. This runs in parallel with the grpc server.
	go func() {
		err := runSchedulerController(ctx, state, state.fidStore)
		state.RLock()
		defer state.RUnlock()
		if state.err != nil {
			err = state.err
			log.WithField("error", err.Error()).Debug("scheduler quit with error, main state machine GO_ERROR")
			state.sm.Event(context.Background(), "GO_ERROR", err) // TODO: use error information in GO_ERROR
		} else {
			log.Debug("scheduler quit, no errors")
			state.sm.Event(context.Background(), "EXIT")
		}
	}()
}

func (state *schedulerState) CopyExecutorInfo() *mesos.ExecutorInfo {
	return proto.Clone(state.executor).(*mesos.ExecutorInfo)
}
