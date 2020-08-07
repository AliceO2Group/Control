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
	"github.com/AliceO2Group/Control/common/logger/infologger"
	"github.com/AliceO2Group/Control/core/task"
)

func NewStopActivityTransition(taskman *task.ManagerV2) Transition {
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

	log.WithField(infologger.Run, env.currentRunNumber).Info("stopping run")

	env.currentRunNumber = 0
	
	taskmanMessage := task.NewtransitionTaskMessage(
						env.Workflow().GetTasks(),
						task.RUNNING.String(),
						task.STOP.String(),
						task.CONFIGURED.String(),
						nil,
					)
	t.taskman.MessageChannel <- taskmanMessage

	// err = t.taskman.TransitionTasks(
	// 	env.Workflow().GetTasks(),
	// 	task.RUNNING.String(),
	// 	task.STOP.String(),
	// 	task.CONFIGURED.String(),
	// 	nil,
	// )

	// if err != nil {
	// 	return
	// }

	return
}
