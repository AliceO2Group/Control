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
	"math/rand"
	"time"
	"sync"

	"github.com/mesos/mesos-go/api/v1/lib"
	"github.com/mesos/mesos-go/api/v1/lib/backoff"
	"github.com/mesos/mesos-go/api/v1/lib/scheduler/calls"
	"github.com/looplab/fsm"
	"github.com/AliceO2Group/Control/core/environment"
	"github.com/AliceO2Group/Control/configuration"
	"encoding/json"
	"github.com/AliceO2Group/Control/core/controlcommands"
	"context"
)

func newInternalState(cfg Config, shutdown func()) (*internalState, error) {
	metricsAPI := initMetrics(cfg)
	executorInfo, err := prepareExecutorInfo(
		cfg.executor,
		cfg.mesosExecutorImage,
		buildWantsExecutorResources(cfg),
		cfg.mesosJobRestartDelay,
		metricsAPI,
	)
	if err != nil {
		return nil, err
	}
	creds, err := loadCredentials(cfg.mesosCredentials)
	if err != nil {
		return nil, err
	}

	cfgman, err := configuration.NewConfiguration(cfg.configurationUri)
	if cfg.veryVerbose {
		cfgDump, _ := cfgman.GetRecursive("o2")
		cfgBytes, _ := json.MarshalIndent(cfgDump,"", "\t")
		log.WithField("data", string(cfgBytes)).Debug("configuration dump")
	}
	if err != nil {
		return nil, err
	}

	resourceOffersDone := make(chan environment.Roles)
	rolesToDeploy := make(chan map[string]environment.RoleCfg)
	reviveOffersTrg := make(chan struct{})

	state := &internalState{
		config:             cfg,
		reviveTokens:       backoff.BurstNotifier(cfg.mesosReviveBurst, cfg.mesosReviveWait, cfg.mesosReviveWait, nil),
		resourceOffersDone: resourceOffersDone,
		rolesToDeploy:      rolesToDeploy,
		reviveOffersTrg:    reviveOffersTrg,
		wantsTaskResources: mesos.Resources{},
		executor:           executorInfo,
		metricsAPI:         metricsAPI,
		cli:                buildHTTPSched(cfg, creds),
		random:             rand.New(rand.NewSource(time.Now().Unix())),
		shutdown:           shutdown,
		environments:       environment.NewEnvManager(nil),
		cfgman:             cfgman,
	}

	state.commandqueue = &controlcommands.CommandQueue{
		SendFunc: func(command controlcommands.MesosCommand, receiver controlcommands.MesosCommandReceiver) (*controlcommands.SingleResponse, error) {
			return SendCommand(context.TODO(), state, command, receiver)
		},
	}

	roleman := environment.NewRoleManager(state.environments, cfgman, resourceOffersDone,
		rolesToDeploy, reviveOffersTrg, state.commandqueue)
	err = roleman.RefreshRoleClasses()
	if err != nil {
		log.WithField("error", err).Warning("bad configuration, some roleClasses were not refreshed")
	}
	state.roleman = roleman

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
	resourceOffersDone chan environment.Roles
	rolesToDeploy      chan map[string]environment.RoleCfg
	reviveOffersTrg    chan struct{}
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
	environments	   *environment.EnvManager
	roleman            *environment.RoleManager
	cfgman             configuration.Configuration
	commandqueue       *controlcommands.CommandQueue
}

