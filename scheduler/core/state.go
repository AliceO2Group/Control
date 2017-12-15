/*
 * === This file is part of octl <https://github.com/teo/octl> ===
 *
 * Copyright 2017 CERN and copyright holders of ALICE O².
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

package core

import (
	"math/rand"
	"time"
	"sync"

	"github.com/mesos/mesos-go/api/v1/lib"
	"github.com/mesos/mesos-go/api/v1/lib/backoff"
	"github.com/mesos/mesos-go/api/v1/lib/scheduler/calls"
	"github.com/looplab/fsm"
	"github.com/teo/octl/scheduler/core/environment"
	"github.com/pborman/uuid"
)

func newInternalState(cfg Config, shutdown func()) (*internalState, error) {
	metricsAPI := initMetrics(cfg)
	executorInfo, err := prepareExecutorInfo(
		cfg.executor,
		cfg.execImage,
		cfg.server,
		buildWantsExecutorResources(cfg),
		cfg.jobRestartDelay,
		metricsAPI,
	)
	if err != nil {
		return nil, err
	}
	creds, err := loadCredentials(cfg.credentials)
	if err != nil {
		return nil, err
	}
	state := &internalState{
		config:             cfg,
		reviveTokens:       backoff.BurstNotifier(cfg.reviveBurst, cfg.reviveWait, cfg.reviveWait, nil),
		resourceOffersDone: make(chan []uuid.Array),
		envToDeploy:        make(chan uuid.Array),
		wantsTaskResources: mesos.Resources{},
		executor:           executorInfo,
		metricsAPI:         metricsAPI,
		cli:                buildHTTPSched(cfg, creds),
		random:             rand.New(rand.NewSource(time.Now().Unix())),
		shutdown:           shutdown,
		environments:		environment.NewEnvironments(),
	}
	return state, nil
}

type internalState struct {
	sync.RWMutex

	// needs locking:
	wantsTaskResources mesos.Resources
	tasksLaunched      int
	tasksFinished      int
	err                error

	// not used in multiple goroutines:
	executor           *mesos.ExecutorInfo
	reviveTokens       <-chan struct{}
	resourceOffersDone chan []uuid.Array
	envToDeploy        chan uuid.Array
	random             *rand.Rand

	// shouldn't change at runtime, so thread safe:
	role               string
	cli                calls.Caller
	config             Config
	shutdown           func()

	// uses prometheus counters, so thread safe
	metricsAPI         *metricsAPI

	// uses locks, so thread safe
	sm                 *fsm.FSM
	environments	   *environment.Environments
}

