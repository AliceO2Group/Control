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
 *ù
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

package core

import (
	"context"
	"encoding/json"
	"math/rand"
	"sync"
	"time"

	"github.com/AliceO2Group/Control/core/controlcommands"
	"github.com/AliceO2Group/Control/core/environment"
	"github.com/AliceO2Group/Control/core/protos"
	"github.com/AliceO2Group/Control/core/task"
	"github.com/AliceO2Group/Control/core/the"
	"github.com/looplab/fsm"
	"github.com/mesos/mesos-go/api/v1/lib"
	"github.com/mesos/mesos-go/api/v1/lib/backoff"
	"github.com/mesos/mesos-go/api/v1/lib/scheduler/calls"
	"github.com/spf13/viper"
)

func newInternalState(shutdown func()) (*internalState, error) {
	metricsAPI := initMetrics()
	executorInfo, err := prepareExecutorInfo(
		viper.GetString("executor"),
		viper.GetString("mesosExecutorImage"),
		buildWantsExecutorResources(viper.GetFloat64("executorCPU"),
			viper.GetFloat64("executorMemory")),
		viper.GetDuration("mesosJobRestartDelay"),
		metricsAPI,
	)
	if err != nil {
		return nil, err
	}
	creds, err := loadCredentials(viper.GetString("mesosCredentials.username"),
		viper.GetString("mesosCredentials.password"))
	if err != nil {
		return nil, err
	}

	if viper.GetBool("veryVerbose") {
		cfgDump, err := the.ConfSvc().GetROSource().GetRecursive(the.ConfSvc().GetConsulPath())
		if err != nil {
			log.WithError(err).Fatal("cannot retrieve configuration")
			return nil, err
		}
		cfgBytes, err := json.MarshalIndent(cfgDump,"", "\t")
		if err != nil {
			log.WithError(err).Fatal("cannot marshal configuration dump")
			return nil, err
		}
		log.WithField("data", string(cfgBytes)).Trace("configuration dump")
	}

	resourceOffersDone := make(chan task.DeploymentMap)
	tasksToDeploy := make(chan task.Descriptors)
	reviveOffersTrg := make(chan struct{})
	Event := make(chan *pb.Event)

	state := &internalState{
		reviveTokens:       backoff.BurstNotifier(
			viper.GetInt("mesosReviveBurst"),
			viper.GetDuration("mesosReviveWait"),
			viper.GetDuration("mesosReviveWait"),
			nil),
		resourceOffersDone: resourceOffersDone,
		tasksToDeploy:      tasksToDeploy,
		reviveOffersTrg:    reviveOffersTrg,
		Event:              Event,
		wantsTaskResources: mesos.Resources{},
		executor:           executorInfo,
		metricsAPI:         metricsAPI,
		cli:                buildHTTPSched(creds),
		random:             rand.New(rand.NewSource(time.Now().Unix())),
		shutdown:           shutdown,
		environments:       nil,
	}

	state.servent = controlcommands.NewServent(
		func(command controlcommands.MesosCommand, receiver controlcommands.MesosCommandTarget) error {
			return SendCommand(context.TODO(), state, command, receiver)
		},
	)
	state.commandqueue = controlcommands.NewCommandQueue(state.servent)

	taskman := task.NewManagerV2(
		resourceOffersDone,
		tasksToDeploy,
		reviveOffersTrg,
		state.commandqueue,
		func(task *task.Task) error {
			return KillTask(context.TODO(), state, task.GetMesosCommandTarget())
		},
		buildHTTPSched(creds),
	)
	state.taskman = taskman
	state.taskman.Start()
	state.environments = environment.NewEnvManager(state.taskman)
	state.commandqueue.Start()	// FIXME: there should be 1 cq per env

	return state, nil
}

type internalState struct {
	sync.RWMutex

	// needs locking:
	wantsTaskResources mesos.Resources
	err                error

	// not used in multiple goroutines:
	executor           *mesos.ExecutorInfo
	reviveTokens       <-chan struct{}
	resourceOffersDone chan task.DeploymentMap
	tasksToDeploy      chan task.Descriptors
	reviveOffersTrg    chan struct{}
	random             *rand.Rand

	// shouldn't change at runtime, so thread safe:
	role               string
	cli                calls.Caller
	shutdown           func()

	// uses prometheus counters, so thread safe
	metricsAPI         *metricsAPI

	// uses locks, so thread safe
	sm           *fsm.FSM
	environments *environment.Manager
	taskman      *task.ManagerV2
	commandqueue *controlcommands.CommandQueue
	servent      *controlcommands.Servent

	Event chan *pb.Event
}
