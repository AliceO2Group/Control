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

	"github.com/AliceO2Group/Control/common/event"
	"github.com/AliceO2Group/Control/common/logger/infologger"
	"github.com/AliceO2Group/Control/core/controlcommands"
	"github.com/AliceO2Group/Control/core/task"
	"github.com/AliceO2Group/Control/core/workflow"
	"github.com/iancoleman/strcase"
)

func NewStopActivityTransition(taskman *task.Manager) Transition {
	return &StopActivityTransition{
		baseTransition: baseTransition{
			name:    "STOP_ACTIVITY",
			taskman: taskman,
		},
	}
}

type StopActivityTransition struct {
	baseTransition
}

func (t StopActivityTransition) do(env *Environment) (err error) {
	if env == nil {
		return errors.New("cannot transition in NIL environment")
	}

	log.WithField(infologger.Run, env.currentRunNumber).
		WithField("partition", env.Id().String()).
		Info("stopping run")

	args := controlcommands.PropertyMap{}

	// Get a handle to the consolidated var stack of the root role of the env's workflow
	if wf := env.Workflow(); wf != nil {
		if cvs, cvsErr := wf.ConsolidatedVarStack(); cvsErr == nil {

			// Propagate run end time to all tasks
			if value, ok := cvs["run_end_time_ms"]; ok {
				args[strcase.ToLowerCamel("run_end_time_ms")] = value
			}
		}
	}

	taskmanMessage := task.NewTransitionTaskMessage(
		workflow.GetActiveTasks(env.Workflow()),
		task.RUNNING.String(),
		task.STOP.String(),
		task.CONFIGURED.String(),
		args,
		env.Id(),
	)
	t.taskman.MessageChannel <- taskmanMessage

	incomingEv := <-env.stateChangedCh
	// If some tasks failed to transition
	if tasksStateErrors := incomingEv.GetTasksStateChangedError(); tasksStateErrors != nil {
		return tasksStateErrors
	}
	env.sendEnvironmentEvent(&event.EnvironmentEvent{EnvironmentID: env.Id().String(), State: "CONFIGURED"})

	return
}
