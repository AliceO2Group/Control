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

package task

import (
	"fmt"
	"strings"

	"github.com/AliceO2Group/Control/common/utils/uid"
)

type TaskError interface {
	error
	GetTaskId() string
}

type TasksError interface {
	error
	GetTaskIds() []string
}

type taskErrorBase struct {
	taskId string
}

func (r taskErrorBase) GetTaskId() string {
	return r.taskId
}

type tasksErrorBase struct {
	taskIds []string
}

func (r tasksErrorBase) GetTaskIds() []string {
	return r.taskIds
}

type GenericTaskError struct {
	taskErrorBase
	message string
}

func (r GenericTaskError) Error() string {
	return fmt.Sprintf("task %s error: %s", r.taskId, r.message)
}

type GenericTasksError struct {
	tasksErrorBase
	message string
}

func (r GenericTasksError) Error() string {
	return fmt.Sprintf("tasks [%s] error: %s", strings.Join(r.taskIds, ", "), r.message)
}

type TasksDeploymentError struct {
	tasksErrorBase
	failedNonCriticalDescriptors Descriptors
	failedCriticalDescriptors    Descriptors
}

func (r TasksDeploymentError) Error() string {
	return fmt.Sprintf("deployment failed for %d critical tasks, and %d non-critical tasks; critical tasks: [%s]; non-critical tasks: [%s]", len(r.failedCriticalDescriptors), len(r.failedNonCriticalDescriptors), r.failedCriticalDescriptors.String(), r.failedNonCriticalDescriptors.String())
}

type TaskAlreadyReleasedError taskErrorBase

func (r TaskAlreadyReleasedError) Error() string {
	return fmt.Sprintf("task %s already released", r.taskId)
}

type TaskNotFoundError taskErrorBase

func (r TaskNotFoundError) Error() string {
	return fmt.Sprintf("task %s not found", r.taskId)
}

type TaskLockedError struct {
	taskErrorBase
	envId uid.ID
}

func (r TaskLockedError) Error() string {
	return fmt.Sprintf("task %s is locked by environment %s", r.taskId, r.envId)
}
func (r TaskLockedError) EnvironmentId() uid.ID {
	return r.envId
}
