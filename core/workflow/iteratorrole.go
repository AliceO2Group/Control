/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2018-2020 CERN and copyright holders of ALICE O².
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

	"github.com/AliceO2Group/Control/common/gera"
	"github.com/AliceO2Group/Control/core/repos"

	"github.com/AliceO2Group/Control/core/task"
	"github.com/AliceO2Group/Control/core/task/constraint"
	"github.com/gobwas/glob"
)

type iteratorRole struct {
	aggregator
	For      iteratorRange `yaml:"for,omitempty"`
	template roleTemplate
}

type templateMap map[string]interface{}

type roleTemplate interface {
	Role
	generateRole(localVars map[string]string) (Role, error)
}

func (i *iteratorRole) UnmarshalYAML(unmarshal func(interface{}) error) (err error) {
	auxUnion := _unionTypeProbe{}
	role := iteratorRole{}
	err = unmarshal(&auxUnion)
	if err != nil {
		return
	}

	type _iteratorRangeUnion struct {
		Begin       *string                  `yaml:"begin"`
		End         *string                  `yaml:"end"`
		Var         *string                  `yaml:"var"`
		Range       *string                  `yaml:"range"`
	}
	auxForUnion := struct {
		For _iteratorRangeUnion `yaml:"for"`
	}{}
	err = unmarshal(&auxForUnion)
	if err != nil {
		return
	}

	var forBlock iteratorRange
	switch {
	case auxForUnion.For.Begin != nil && auxForUnion.For.End != nil && auxForUnion.For.Var != nil:
		auxFor := struct {
			For *iteratorRangeFor `yaml:"for"`
		}{}
		err = unmarshal(&auxFor)
		if err != nil {
			return
		}
		forBlock = auxFor.For
	case auxForUnion.For.Range != nil && auxForUnion.For.Var != nil:
		auxFor := struct {
			For *iteratorRangeExpr `yaml:"for"`
		}{}
		err = unmarshal(&auxFor)
		if err != nil {
			return
		}
		forBlock = auxFor.For
	default:
		err = errors.New("invalid range specifier in iterator")
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
	role.For = forBlock

	// FIXME: if Name does not contain {{ }}, we must bail!
	*i = role
	return
}

func (i *iteratorRole) MarshalYAML() (interface{}, error) {
	aux := make(map[string]interface{})

	aux["aggregator"] = i.aggregator
	aux["for"]        = i.For
	aux["roles"]      = i.template

	return aux, nil
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

func (i *iteratorRole) ProcessTemplates(workflowRepo *repos.Repo) (err error) {
	if i == nil {
		return errors.New("role tree error when processing templates")
	}

	err = i.expandTemplate()
	if err != nil {
		return
	}

	for _, role := range i.Roles {
		err = role.ProcessTemplates(workflowRepo)
		if err != nil {
			return
		}
	}
	return
}

func (i *iteratorRole) expandTemplate() (err error) {
	varStack := make(map[string]string)
	if parent := i.GetParent(); parent != nil {
		varStack, _ = gera.FlattenStack(
			parent.GetDefaults(),
			parent.GetVars(),
			parent.GetUserVars(),
		)
	}

	roles := make([]Role, 0)

	var ran []string
	ran, err = i.For.GetRange(varStack)
	if err != nil {
		return
	}

	for _, localValue := range ran {
		locals := make(map[string]string)
		locals[i.For.GetVar()] = localValue
		var newRole Role
		newRole, err = i.template.generateRole(locals)
		if err != nil {
			return
		}
		roles = append(roles, newRole)
	}

	i.Roles = roles
	for j := 0; j < len(i.Roles); j++ {
		i.Roles[j].setParent(i.GetParent())
	}
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

func (i *iteratorRole) GetDefaults() gera.StringMap {
	if i == nil {
		return nil
	}
	return i.template.GetDefaults()
}

func (i *iteratorRole) GetVars() gera.StringMap {
	if i == nil {
		return nil
	}
	return i.template.GetVars()
}

func (i *iteratorRole) GetUserVars() gera.StringMap {
	if i == nil {
		return nil
	}
	return i.template.GetUserVars()
}

func (i *iteratorRole) SetRuntimeVar(key string, value string) {
	if i == nil {
		return
	}
	i.template.SetRuntimeVar(key, value)
}

func (i *iteratorRole) SetRuntimeVars(kv map[string]string) {
	if i == nil {
		return
	}
	i.template.SetRuntimeVars(kv)
}
