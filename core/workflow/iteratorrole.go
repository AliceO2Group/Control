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
	"errors"
	"strconv"

	"github.com/AliceO2Group/Control/core/task"
	"github.com/AliceO2Group/Control/core/task/constraint"
	"github.com/gobwas/glob"
)

type iteratorRole struct {
	aggregator
	For      iteratorInfo            `yaml:"for,omitempty"`
	template roleTemplate
}

type templateMap map[string]interface{}

type roleTemplate interface {
	Role
	generateRole(t templateMap) (Role, error)
}

func (i *iteratorRole) UnmarshalYAML(unmarshal func(interface{}) error) (err error) {
	auxUnion := _unionTypeProbe{}
	role := iteratorRole{}
	err = unmarshal(&auxUnion)
	if err != nil {
		return
	}
	auxFor := struct {
		For         iteratorInfo            `yaml:"for"`
	}{}
	err = unmarshal(&auxFor)
	if err != nil {
		return
	}

	var template roleTemplate
	switch {
	case auxUnion.Roles != nil && auxUnion.Task == nil:
		template = &aggregatorTemplate{}
	case auxUnion.Task != nil && auxUnion.Roles == nil:
		template = &taskTemplate{}
	default:
		err = errors.New("invalid template role in iterator")
		return
	}
	err = unmarshal(template)
	if err != nil {
		return
	}

	role.template = template
	role.For = auxFor.For

	// FIXME: if Name does not contain {{ }}, we must bail!

	err = role.expandTemplate()
	if err != nil {
		return
	}
	*i = role
	return
}

type iteratorInfo struct {
	Begin       int                     `yaml:"begin"`
	End         int                     `yaml:"end"`
	Var         string                  `yaml:"var"`
}

func (f *iteratorInfo) UnmarshalYAML(unmarshal func(interface{}) error) (err error) {
	aux := struct{
		Begin       string                  `yaml:"begin"`
		End         string                  `yaml:"end"`
		Var         string                  `yaml:"var"`
	}{}
	err = unmarshal(&aux)
	if err != nil {
		return
	}

	f.Begin, err = strconv.Atoi(aux.Begin)
	if err != nil {
		return
	}
	f.End, err = strconv.Atoi(aux.End)
	if err != nil {
		return
	}
	f.Var = aux.Var
	return
}

func (i *iteratorRole) GlobFilter(g glob.Glob) (rs []Role) {
	rs = make([]Role, 0)
	for _, chr := range i.Roles {
		chrs := chr.GlobFilter(g)
		if len(chrs) != 0 {
			rs = append(rs, chrs...)
		}
	}
	return
}

func (i *iteratorRole) ProcessTemplates() (err error) {
	if i == nil {
		return errors.New("role tree error when processing templates")
	}

	for _, role := range i.Roles {
		err = role.ProcessTemplates()
		if err != nil {
			return
		}
	}
	return
}

func (i *iteratorRole) expandTemplate() (err error) {
	values := make(templateMap)

	roles := make([]Role, 0)

	for j := i.For.Begin; j <= i.For.End; j++ {
		values[i.For.Var] = strconv.Itoa(j)
		var newRole Role
		newRole, err = i.template.generateRole(values)
		if err != nil {
			return
		}
		roles = append(roles, newRole)
	}

	i.Roles = roles
	return
}

func (i *iteratorRole) GetParent() Updatable {
	if i == nil || i.template == nil {
		return nil
	}
	return i.template.GetParent()
}

func (i *iteratorRole) GetParentRole() Role {
	if i == nil || i.template == nil {
		return nil
	}
	return i.template.GetParentRole()
}

func (i *iteratorRole) GetName() string {
	if i == nil || i.template == nil {
		return ""
	}
	return i.template.GetName()
}

func (i *iteratorRole) GetPath() string {
	if i == nil || i.template == nil {
		return ""
	}
	return i.template.GetPath()
}

func (i *iteratorRole) GetStatus() task.Status {
	panic("implement me")
}

func (i *iteratorRole) GetState() task.State {
	panic("implement me")
}

func (i *iteratorRole) setParent(role Updatable) {
	i.template.setParent(role)
	for _, v := range i.Roles {
		v.setParent(role)
	}
}

func (i *iteratorRole) getConstraints() (cts constraint.Constraints) {
	if i == nil {
		return
	}
	if parentRole := i.GetParentRole(); parentRole != nil {
		cts = parentRole.getConstraints()
	}
	return
}
