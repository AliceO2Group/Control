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
	"strings"
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

	runNumber := env.currentRunNumber

	log.WithField(infologger.Run, runNumber).
		WithField("partition", env.Id().String()).
		Info("starting new run")

	args := controlcommands.PropertyMap{
		"runNumber": strconv.FormatUint(uint64(runNumber), 10),
	}

	flps := env.GetFLPs()
	dd_enabled, _ := strconv.ParseBool(env.GetKV("", "dd_enabled"))
	dcs_enabled, _ := strconv.ParseBool(env.GetKV("", "dcs_enabled"))
	epn_enabled, _ := strconv.ParseBool(env.GetKV("", "epn_enabled"))
	odc_topology := env.GetKV("", "odc_topology")
	// GetString of active detectors and pass it to the BK API
	detectors := strings.Join(env.GetActiveDetectors().StringList(), ",")
	/*
		the.BookkeepingAPI().CreateRun(env.Id().String(), len(env.GetActiveDetectors()), 0, len(flps), int32(runNumber), env.GetRunType().String(), time.Now(), time.Now(), dd_enabled, dcs_enabled, epn_enabled, odc_topology, detectors)
		for _, flp := range flps {
			the.BookkeepingAPI().CreateFlp(flp, flp, int32(runNumber))
		}

		// According to documentation the 1st input should
		// a text Log entry that is written by the shifter
		// TODO (malexis): we need to implement a way to
		// get the text from the shifter
		// parentlogId = -1 to create a new log on each run
		// the.BookkeepingAPI().CreateLog(env.Id().String(), fmt.Sprintf("Log for run %s and environment %s",runNumberStr,env.Id().String()), runNumberStr, -1)
		the.BookkeepingAPI().CreateLog(env.GetVarsAsString(), fmt.Sprintf("Log for run %s and environment %s",args["runNumber"],env.Id().String()), args["runNumber"], -1)
	*/
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
		the.BookkeepingAPI().UpdateRun(int32(runNumber), "bad", time.Now(), time.Now())
		env.currentRunNumber = 0
		return tasksStateErrors
	}

	log.WithField(infologger.Run, env.currentRunNumber).
		WithField("partition", env.Id().String()).
		Info("run started")
	env.sendEnvironmentEvent(&event.EnvironmentEvent{
		EnvironmentID: env.Id().String(),
		State:         "RUNNING",
		Run:           env.currentRunNumber,
	})

	return
}
