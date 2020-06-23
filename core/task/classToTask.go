/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2020 CERN and copyright holders of ALICE O².
 * Author: Ayaan Zaidi <azaidi@cern.ch>
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

package task

import "github.com/AliceO2Group/Control/common"

func ClassToTask(input *Class, parent parentRole) *Task {
	output := Task{
		parent:       parent,
		className:    "",
		name:         "",
		hostname:     "",
		agentId:      "",
		offerId:      "",
		taskId:       "",
		executorId:   "",
		localBindMap: nil,
		status:       0,
		state:        0,
		safeToStop:   false,
		properties:   nil,
		GetTaskClass: func() *Class {
			return input
		},
		commandInfo: &common.TaskCommandInfo{
			CommandInfo: common.CommandInfo{
				Env:       nil,
				Shell:     nil,
				Value:     nil,
				User:      nil,
				Arguments: nil,
			},
			ControlPort: 0,
			ControlMode: 0,
		},
	}

	return &output
}
