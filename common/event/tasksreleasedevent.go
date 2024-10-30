/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2020 CERN and copyright holders of ALICE O².
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

package event

import (
	"github.com/AliceO2Group/Control/common/utils"
	"github.com/AliceO2Group/Control/common/utils/uid"
)

type TasksReleasedEvent struct {
	eventBase
	EnvironmentId     uid.ID           `json:"environmentId"`
	TaskIdsReleased   []string         `json:"taskIdsReleased"`
	TaskReleaseErrors map[string]error `json:"taskReleaseErrors"`
}

func (tr *TasksReleasedEvent) GetName() string {
	return "TASK_RELEASED"
}

func (tr *TasksReleasedEvent) GetEnvironmentId() uid.ID {
	if tr == nil {
		return ""
	}
	return tr.EnvironmentId
}

func (tr *TasksReleasedEvent) GetTaskIds() []string {
	if tr == nil {
		return nil
	}
	return tr.TaskIdsReleased
}

func (tr *TasksReleasedEvent) GetTaskReleaseErrors() map[string]error {
	if tr == nil {
		return nil
	}
	return tr.TaskReleaseErrors
}

func NewTasksReleasedEvent(envId uid.ID, taskIdsReleased []string, taskReleaseErrors map[string]error) (tr *TasksReleasedEvent) {
	tr = &TasksReleasedEvent{
		eventBase: eventBase{
			Timestamp:   utils.NewUnixTimestamp(),
			MessageType: "TasksReleasedEvent",
		},
		EnvironmentId:     envId,
		TaskIdsReleased:   taskIdsReleased,
		TaskReleaseErrors: taskReleaseErrors,
	}
	return tr
}
