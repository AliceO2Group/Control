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
	"time"

	"github.com/AliceO2Group/Control/common/event"
	"github.com/AliceO2Group/Control/common/logger/infologger"
	"github.com/AliceO2Group/Control/core/controlcommands"
	"github.com/AliceO2Group/Control/core/task"
	"github.com/AliceO2Group/Control/core/the"
)

func NewStartActivityTransition(taskman *task.Manager) Transition {
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
		"runNumber": strconv.FormatUint(uint64(runNumber), 10),
	}

	flps := env.GetFLPs()
	the.BookkeepingAPI().CreateRun(env.Id().String(), 0, 0, len(flps), int(runNumber), env.GetRunType(), time.Now().Unix(), time.Now().Unix())
	for _, flp := range flps {
		the.BookkeepingAPI().CreateFlp(flp, flp, int64(runNumber))
	}

	// According to documentation the 1st input should
	// a text Log entry that is written by the shifter
	// TODO (malexis): we need to implement a way to
	// get the text from the shifter
	// runNumberStr := strconv.FormatUint(uint64(runNumber), 10 )
	// the.BookkeepingAPI().CreateLog(env.Id().String(), fmt.Sprintf("Log for run %s and environment %s",runNumberStr,env.Id().String()), runNumberStr, -1)

	taskmanMessage := task.NewTransitionTaskMessage(
						env.Workflow().GetTasks(),
						task.CONFIGURED.String(),
						task.START.String(),
						task.RUNNING.String(),
						args,
						env.Id(),
					)
	t.taskman.MessageChannel <- taskmanMessage

	incomingEv := <-env.stateChangedCh
	// If some tasks failed to transition
	if tasksStateErrors := incomingEv.GetTasksStateChangedError(); tasksStateErrors != nil {
		the.BookkeepingAPI().UpdateRun(int(runNumber), "bad", time.Now().Unix(), time.Now().Unix())
		env.currentRunNumber = 0
		return tasksStateErrors
	}

	log.WithField(infologger.Run, env.currentRunNumber).Info("run started")
	env.sendEnvironmentEvent(&event.EnvironmentEvent{
		EnvironmentID: env.Id().String(),
		State: "RUNNING",
		Run: env.currentRunNumber,
	})

	return
}
