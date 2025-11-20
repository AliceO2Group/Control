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

	"github.com/AliceO2Group/Control/core/task/sm"
	"github.com/AliceO2Group/Control/core/workflow"

	"github.com/AliceO2Group/Control/common/event"
	"github.com/AliceO2Group/Control/common/logger/infologger"
	"github.com/AliceO2Group/Control/core/controlcommands"
	"github.com/AliceO2Group/Control/core/task"
	"github.com/iancoleman/strcase"
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
		WithField(infologger.Level, infologger.IL_Support).
		Info("starting new run")

	cleanupCount := 0
	cleanupCountS, ok := env.GlobalVars.Get("__fmq_cleanup_count")
	if ok && len(cleanupCountS) > 0 {
		var parseErr error
		cleanupCount, parseErr = strconv.Atoi(cleanupCountS)
		if parseErr != nil {
			cleanupCount = 1 // something was there, even though non-parsable, so we signal to clean up
		}
	}

	args := controlcommands.PropertyMap{
		"runNumber": strconv.FormatUint(uint64(runNumber), 10),
		"cleanup":   strconv.Itoa(cleanupCount),
	}

	// Get a handle to the consolidated var stack of the root role of the env's workflow
	if wf := env.Workflow(); wf != nil {
		if cvs, cvsErr := wf.ConsolidatedVarStack(); cvsErr == nil {
			for _, key := range []string{
				"fill_info_fill_number",
				"fill_info_filling_scheme",
				"fill_info_beam_type",
				"fill_info_stable_beams_start_ms",
				"fill_info_stable_beams_end_ms",
				"run_type",
				"run_start_time_ms",
				"run_end_time_ms", // included to ensure that a cleared SOEOR timestamp is propagated to all tasks during START-STOP-START
				"lhc_period",
				"pdp_beam_type",
				"pdp_override_run_start_time",
				"original_run_number",
			} {
				if value, ok := cvs[key]; ok {
					// we push the above parameters with both camelCase and snake_case identifiers for convenience
					args[strcase.ToLowerCamel(key)] = value
					args[key] = value
				}
			}
		}
	}

	taskmanMessage := task.NewTransitionTaskMessage(
		workflow.GetActiveTasks(env.Workflow()),
		sm.CONFIGURED.String(),
		sm.START.String(),
		sm.RUNNING.String(),
		args,
		env.Id(),
	)
	t.taskman.MessageChannel <- taskmanMessage

	incomingEv := <-env.stateChangedCh
	// If some tasks failed to transition
	if tasksStateErrors := incomingEv.GetTasksStateChangedError(); tasksStateErrors != nil {
		return tasksStateErrors
	}

	log.WithField(infologger.Run, env.currentRunNumber).
		WithField("partition", env.Id().String()).
		WithField(infologger.Level, infologger.IL_Support).
		Info("run started")
	env.sendEnvironmentEvent(&event.EnvironmentEvent{
		EnvironmentID: env.Id().String(),
		State:         "RUNNING",
		Run:           env.currentRunNumber,
	})

	return
}
