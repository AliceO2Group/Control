/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2018 CERN and copyright holders of ALICE O².
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

package environment

import (
	"errors"
	"strconv"

	"github.com/AliceO2Group/Control/common/logger/infologger"
	"github.com/AliceO2Group/Control/core/controlcommands"
	"github.com/AliceO2Group/Control/core/task"
	"github.com/AliceO2Group/Control/core/the"
)

func NewStartActivityTransition(taskman *task.ManagerV2) Transition {
	return &StartActivityTransition{
		baseTransition: baseTransition{
			name:    "START_ACTIVITY",
			taskman: taskman,
		},
	}
}

type StartActivityTransition struct {
	baseTransition
}

func (t StartActivityTransition) do(env *Environment) (err error) {
	if env == nil {
		return errors.New("cannot transition in NIL environment")
	}

	var runNumber uint32
	runNumber, err = the.ConfSvc().NewRunNumber()
	if err != nil {
		return
	}

	log.WithField(infologger.Run, runNumber).Info("starting new run")

	env.currentRunNumber = runNumber
	args := controlcommands.PropertyMap{
		"runNumber": strconv.FormatUint(uint64(runNumber), 10 ),
	}

	taskmanMessage := task.NewTransitionTaskMessage(
						env.Workflow().GetTasks(),
						task.CONFIGURED.String(),
						task.START.String(),
						task.RUNNING.String(),
						args,
					)
	t.taskman.MessageChannel <- taskmanMessage

	// err = t.taskman.TransitionTasks(
	// 	env.Workflow().GetTasks(),
	// 	task.CONFIGURED.String(),
	// 	task.START.String(),
	// 	task.RUNNING.String(),
	// 	args,
	// )

	// if err != nil {
	// 	env.currentRunNumber = 0
	// 	return
	// }
	log.WithField(infologger.Run, env.currentRunNumber).Info("run started")

	return
}
