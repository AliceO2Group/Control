/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2022 CERN and copyright holders of ALICE O².
 * Author: Piotr Konopka <pkonopka@cern.ch>
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
	"github.com/AliceO2Group/Control/core/task"
	"github.com/AliceO2Group/Control/core/task/sm"
)

func NewGoErrorTransition(taskman *task.Manager) Transition {
	return &GoErrorTransition{
		baseTransition: baseTransition{
			name:    "GO_ERROR",
			taskman: taskman,
		},
	}
}

type GoErrorTransition struct {
	baseTransition
}

func (t GoErrorTransition) do(env *Environment) (err error) {

	// we stop all tasks which are in RUNNING
	toStop := env.Workflow().GetTasks().Filtered(func(t *task.Task) bool {
		t.SetSafeToStop(true)
		return t.IsSafeToStop()
	})
	if len(toStop) > 0 {
		taskmanMessage := task.NewTransitionTaskMessage(
			toStop,
			sm.RUNNING.String(),
			sm.STOP.String(),
			sm.CONFIGURED.String(),
			nil,
			env.Id(),
		)
		t.taskman.MessageChannel <- taskmanMessage
		<-env.stateChangedCh
	}

	return
}
