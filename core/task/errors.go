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
	"github.com/pborman/uuid"
)

type TaskError interface {
	error
	GetTaskName() string
}

type TasksError interface {
	error
	GetTaskNames() []string
}

type taskErrorBase struct {
	taskName string
}
func (r taskErrorBase) GetTaskName() string {
	return r.taskName
}

type tasksErrorBase struct {
	taskNames []string
}
func (r tasksErrorBase) GetTaskNames() []string {
	return r.taskNames
}

type GenericTaskError struct {
	taskErrorBase
	message string
}
func (r GenericTaskError) Error() string {
	return fmt.Sprintf("task %s error: %s", r.taskName, r.message)
}

type GenericTasksError struct {
	tasksErrorBase
	message string
}
func (r GenericTasksError) Error() string {
	return fmt.Sprintf("tasks [%s] error: %s", strings.Join(r.taskNames, ", "), r.message)
}

type TasksDeploymentError tasksErrorBase
func (r TasksDeploymentError) Error() string {
	return fmt.Sprintf("deployment failed for tasks [%s]", r.taskNames)
}

type TaskAlreadyReleasedError taskErrorBase
func (r TaskAlreadyReleasedError) Error() string {
	return fmt.Sprintf("task %s already released", r.taskName)
}

type TaskNotFoundError taskErrorBase
func (r TaskNotFoundError) Error() string {
	return fmt.Sprintf("task %s not found", r.taskName)
}

type TaskLockedError struct {
	taskErrorBase
	envId uuid.Array
}
func (r TaskLockedError) Error() string {
	return fmt.Sprintf("task %s is locked by environment %s", r.taskName, r.envId)
}
func (r TaskLockedError) EnvironmentId() uuid.Array {
	return r.envId
}