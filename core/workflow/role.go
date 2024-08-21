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
	"github.com/AliceO2Group/Control/common/event"
	"github.com/AliceO2Group/Control/common/gera"
	"github.com/AliceO2Group/Control/common/utils/uid"
	"github.com/AliceO2Group/Control/core/repos"
	"github.com/AliceO2Group/Control/core/task"
	"github.com/AliceO2Group/Control/core/task/channel"
	"github.com/AliceO2Group/Control/core/task/constraint"
	"github.com/AliceO2Group/Control/core/task/sm"
	"github.com/AliceO2Group/Control/core/workflow/callable"
	"github.com/gobwas/glob"
)

type Role interface {
	copyable
	VarNode
	GetParent() Updatable
	GetParentRole() Role
	GetRootRole() Role
	GetRoles() []Role
	GetPath() string
	GetName() string
	GetStatus() task.Status
	GetState() sm.State
	GetTasks() task.Tasks
	GetTaskClasses() []string
	GenerateTaskDescriptors() task.Descriptors
	getConstraints() constraint.Constraints
	setParent(role Updatable)
	ProcessTemplates(workflowRepo repos.IRepo, loadSubworkflow LoadSubworkflowFunc, baseConfigStack map[string]string) error
	GlobFilter(g glob.Glob) []Role
	SetRuntimeVar(key string, value string)
	SetRuntimeVars(kv map[string]string)
	DeleteRuntimeVar(key string)
	DeleteRuntimeVars(keys []string)
	GetHooksMapForTrigger(trigger string) callable.HooksMap
	GetAllHooks() callable.Hooks
	IsEnabled() bool
	GetCurrentRunNumber() uint32
	ConsolidatedVarStack() (varStack map[string]string, err error)
	ConsolidatedVarMaps() (defaults, vars, userVars map[string]string, err error)
	GetEnvironmentId() uid.ID
	IsCritical() bool
}

type PublicUpdatable interface {
	UpdateStatus(s task.Status)
	UpdateState(s sm.State)
}

type Updatable interface {
	VarNode
	GetParent() Updatable
	updateStatus(s task.Status)
	updateState(s sm.State) //string?
	GetEnvironmentId() uid.ID
	GetPath() string
	CollectOutboundChannels() []channel.Outbound
	CollectInboundChannels() []channel.Inbound
	SendEvent(event.Event)
	GetCurrentRunNumber() uint32
}

type VarNode interface {
	GetDefaults() gera.Map[string, string]
	GetVars() gera.Map[string, string]
	GetUserVars() gera.Map[string, string]
}

type copyable interface {
	copy() copyable
}
