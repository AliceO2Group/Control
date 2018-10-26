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
	"bytes"
	"fmt"
	"text/template"

	"github.com/AliceO2Group/Control/common/logger"
	"github.com/AliceO2Group/Control/core/task/channel"
	"github.com/jinzhu/copier"
	"github.com/sirupsen/logrus"

	"github.com/AliceO2Group/Control/core/task"
	"github.com/AliceO2Group/Control/core/task/constraint"
	"github.com/pborman/uuid"
)

var log = logger.New(logrus.StandardLogger(), "workflow")

const PATH_SEPARATOR = "."

type roleBase struct {
	Name        string                  `yaml:"name"`
	parent      Updatable
	Vars        task.VarMap             `yaml:"vars,omitempty"`
	Connect     []channel.Outbound      `yaml:"connect,omitempty"`
	Constraints constraint.Constraints  `yaml:"constraints,omitempty"`
	status      SafeStatus
	state       SafeState
}

func (r *roleBase) CollectOutboundChannels() (channels []channel.Outbound) {
	if r.parent == nil {
		channels = make([]channel.Outbound, 0)
	} else {
		channels = r.parent.CollectOutboundChannels()
	}
	for _, v := range r.Connect {
		channels = append(channels, v)
		// FIXME: this does not take into account child roles with outbound channels with the same name
		// as an outbound channel in the parent.
		// The correct behaviour would be OVERRIDE, currently the behavior is UNDEFINED.
	}
	return
}

func (r *roleBase) UnmarshalYAML(unmarshal func(interface{}) error) (err error) {
	// NOTE: the local type alias is a necessary step, otherwise unmarshal(&role) would
	//       recurse back to this function forever
	type _roleBase roleBase
	role := _roleBase{
		status: SafeStatus{status:task.INACTIVE},
		state:  SafeState{state:task.STANDBY},
	}
	err = unmarshal(&role)
	if err == nil {
		*r = roleBase(role)
	}
	return
}

func (r *roleBase) resolveOutboundChannelTargets() {
	type _parentRole interface {
		GetParentRole() Role
		GetPath() string
	}

	funcMap := template.FuncMap{
		"this": func() string {
			return r.GetPath()
		},
		"parent": func() string {
			p := r.GetParentRole()
			if p == nil {
				log.WithFields(logrus.Fields{"error": "role has no parent", "role": r.GetPath()}).Error("workflow configuration error")
				return ""
			}
			return p.GetPath()
		},
		"up": func(levels int) string {
			if levels <= 0 {
				return r.GetPath()
			}
			var p _parentRole = r
			for i := 0; i < levels; i++ {
				p = p.GetParentRole()
				if p == nil {
					log.WithFields(logrus.Fields{"error": "role has no ancestor", "role": r.GetPath()}).Error("workflow configuration error")
					return ""
				}
			}
			return p.GetPath()
		},
	}

	for i, ch := range r.Connect {
		tmpl := template.New(r.GetPath())
		parsed, err := tmpl.Funcs(funcMap).Parse(ch.Target)
		if err != nil {
			log.WithError(err).WithFields(logrus.Fields{"role": r.GetPath(), "channel": ch.Name, "target": ch.Target}).Error("cannot parse template for outbound channel target")
			continue
		}
		buf := new(bytes.Buffer)
		err = parsed.Execute(buf, struct{}{})
		if err != nil {
			log.WithError(err).WithFields(logrus.Fields{"role": r.GetPath(), "channel": ch.Name, "target": ch.Target}).Error("cannot execute template for outbound channel target")
			continue
		}
		// Finally we write the result back to the target string
		r.Connect[i].Target = buf.String()
	}
}

func (r *roleBase) copy() copyable {
	rCopy := roleBase{
		Name: r.Name,
		parent: r.parent,
		Vars: make(task.VarMap),
		Connect: make([]channel.Outbound, len(r.Connect)),
		Constraints: make(constraint.Constraints, len(r.Constraints)),
		status: r.status,
		state: r.state,
	}

	err := copier.Copy(&rCopy.Vars, &r.Vars)
	if err != nil {
		log.WithField("role", r.GetPath()).WithError(err).Error("role copy error")
	}

	copied := copy(rCopy.Connect, r.Connect)
	if copied != len(r.Connect) {
		log.WithField("role", r.GetPath()).
			WithError(fmt.Errorf("slice copy copied %d items, %d expected", copied, len(r.Connect))).
			Error("role copy error")
	}

	copied = copy(rCopy.Constraints, r.Constraints)
	if copied != len(r.Constraints) {
		log.WithField("role", r.GetPath()).
			WithError(fmt.Errorf("slice copy copied %d items, %d expected", copied, len(r.Constraints))).
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
		return task.UNDEFINED
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