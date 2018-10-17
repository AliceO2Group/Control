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

// Package workflow defines the Role interface, along with tooling to build
// the control tree.
// A workflow is a tree of Roles, and it's loaded from Configuration with a
// combination of YAML unmarshaling and Go template execution.
package workflow

import (
	"github.com/AliceO2Group/Control/core/task"
	"github.com/AliceO2Group/Control/core/task/constraint"
	"github.com/pborman/uuid"
)

type Role interface {
	copyable
	GetParentRole() Role
	GetRoles() []Role
	GetName() string
	GetStatus() task.Status
	GetState() task.State
	GetTasks() []*task.Task
	GenerateTaskDescriptors() task.Descriptors
	getConstraints() constraint.Constraints
	setParent(role Updatable)
}

type Updatable interface {
	updateStatus(s task.Status)
	updateState(s task.State) //string?
	GetEnvironmentId() uuid.Array
	GetPath() string
}

type updatableRole interface {
	Role
	Updatable
}

type controllableRole interface {
	Role
	GetPath() string
	//doTransition(transition Transition) (task.Status, task.State)
}

type copyable interface {
	copy() copyable
}
