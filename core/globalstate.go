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
	"sync"

	"github.com/AliceO2Group/Control/common/event"
	"github.com/AliceO2Group/Control/core/environment"

	"github.com/AliceO2Group/Control/core/task"
	"github.com/AliceO2Group/Control/core/the"
	"github.com/spf13/viper"
)

func newGlobalState(shutdown func()) (*globalState, error) {
	if viper.GetBool("veryVerbose") {
		dump, err := the.ConfSvc().RawGetRecursive(viper.GetString("consulBasePath"))
		if err != nil {
			return nil, err
		}
		log.WithField("data", dump).Trace("configuration dump")
	}

	state := &globalState{
		shutdown:     shutdown,
		environments: nil,
	}

	internalEventCh := make(chan event.Event)
	taskman, err := task.NewManager(shutdown, internalEventCh)
	if err != nil {
		return nil, err
	}

	state.taskman = taskman
	//state.taskman.Start()
	state.environments = environment.NewEnvManager(state.taskman, internalEventCh)

	return state, nil
}

type globalState struct {
	sync.RWMutex

	shutdown func()

	// uses locks, so thread safe
	environments *environment.Manager
	taskman      *task.Manager
}
