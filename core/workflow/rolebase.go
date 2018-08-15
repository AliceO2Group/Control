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

package workflow

import (
	"github.com/AliceO2Group/Control/core/task/channel"
	"github.com/AliceO2Group/Control/common/logger"
	"github.com/sirupsen/logrus"
	"github.com/jinzhu/copier"
	"fmt"
	"github.com/AliceO2Group/Control/core/task"
	"github.com/pborman/uuid"
	"github.com/AliceO2Group/Control/core/task/constraint"
)

var log = logger.New(logrus.StandardLogger(), "workflow")

const PATH_SEPARATOR = "."

type roleBase struct {
	Name        string                  `yaml:"name"`
	parent      updatable
	Vars        task.VarMap             `yaml:"vars,omitempty"`
	Connect     []channel.Outbound      `yaml:"connect,omitempty"`
	Constraints constraint.Constraints  `yaml:"constraints,omitempty"`
	status      SafeStatus
	state       SafeState
}

func (r *roleBase) UnmarshalYAML(unmarshal func(interface{}) error) (err error) {
	role := roleBase{
		status: SafeStatus{status:task.INACTIVE},
		state:  SafeState{state:task.STANDBY},
	}
	err = unmarshal(&role)
	if err == nil {
		*r = role
	}
	return
}

func (r *roleBase) copy() copyable {
	rCopy := roleBase{
		Name: r.Name,
		parent: r.parent,
		Vars: make(task.VarMap),
		Connect: make([]channel.Outbound, len(r.Connect)),
		status: r.status,
		state: r.state,
	}
	err := copier.Copy(rCopy.Vars, r.Vars)
	if err != nil {
		log.WithField("role", r.GetPath()).WithError(err).Error("role copy error")
	}
	copied := copy(rCopy.Connect, r.Connect)
	if copied != len(r.Connect) {
		log.WithField("role", r.GetPath()).
			WithError(fmt.Errorf("slice copy copied %d items, %d expected", copied, len(r.Connect))).
			Error("role copy error")
	}
	return &rCopy
}

func (r *roleBase) GetParentRole() Role {
	if r == nil {
		return nil
	}
	parentRole, ok := r.parent.(Role)
	if ok {
		return parentRole
	}
	return nil
}

func (r *roleBase) GetName() string {
	if r == nil {
		return ""
	}
	return r.Name
}

func (r* roleBase) GetEnvironmentId() uuid.Array {
	if r.parent == nil {
		return uuid.NIL.Array()
	}
	return r.parent.GetEnvironmentId()
}

func (r *roleBase) GetPath() string {
	if r == nil {
		return ""
	}
	if r.parent == nil {
		return r.Name
	}

	parentPath := r.parent.GetPath()
	if len(parentPath) > 0 {
		return parentPath + PATH_SEPARATOR + r.Name
	}

	return r.Name
}

func (r *roleBase) GetStatus() task.Status {
	if r == nil {
		return ""
	}
	return r.status.get()
}

func (r *roleBase) GetState() task.State {
	if r == nil {
		return ""
	}
	return r.state.get()
}

func (r *roleBase) getConstraints() (cts constraint.Constraints) {
	if r == nil {
		return
	}

	if r.Constraints == nil {
		cts = make(constraint.Constraints, 0)
	} else {
		cts = make(constraint.Constraints, len(r.Constraints))
		copy(cts, r.Constraints)
	}

	if r.parent == nil {
		return
	}
	if parentRole := r.GetParentRole(); parentRole != nil {
		cts = cts.MergeParent(parentRole.getConstraints())
	}

	return
}