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

package workflow

import (
	"github.com/AliceO2Group/Control/configuration/template"
	"github.com/AliceO2Group/Control/core/task/constraint"
)

func WrapConstraints(items constraint.Constraints) template.Fields {
	fields := make(template.Fields, 0)
	for i, _ := range items {
		index := i // we need a local copy for the getter/setter closures
		fields = append(fields, &template.GenericWrapper{
			Getter: func() string {
				return items[index].Value
			},
			Setter: func(value string) {
				items[index].Value = value
			},
		})
	}
	return fields
}

func GetRoot(role Role) (root Role) {
	if role == nil {
		return nil
	}
	current := role
	for {
		if parent := current.GetParentRole(); parent != nil {
			// if there's a parent, we go up
			current = parent
			continue
		}

		// it has no parent, so...
		return current
	}
}

func Walk(root Role, do func(role Role)) {
	switch typed := root.(type) {
	case *aggregatorRole:
		do(typed)
		for _, child := range typed.Roles {
			Walk(child, do)
		}
	case *iteratorRole:
		do(typed)
		for _, child := range typed.Roles {
			Walk(child, do)
		}
	case *includeRole:
		do(typed)
		for _, child := range typed.Roles {
			Walk(child, do)
		}
	case *taskRole:
		do(typed)
	case *callRole:
		do(typed)
	}
}

func LeafWalk(root Role, do func(role Role)) {
	switch typed := root.(type) {
	case *aggregatorRole:
		for _, child := range typed.Roles {
			LeafWalk(child, do)
		}
	case *iteratorRole:
		for _, child := range typed.Roles {
			LeafWalk(child, do)
		}
	case *includeRole:
		for _, child := range typed.Roles {
			LeafWalk(child, do)
		}
	case *taskRole:
		do(typed)
	case *callRole:
		do(typed)
	}
}

// LinkChildrenToParents walks through the role tree and sets the correct parent for each child
func LinkChildrenToParents(root Role) {
	var setParents func(role Role, parent Role)
	setParents = func(role Role, parent Role) {
		if typedParent, ok := parent.(Updatable); ok {
			role.setParent(typedParent)
		}
		for _, child := range role.GetRoles() {
			setParents(child, role)
		}
	}

	setParents(root, nil)
}

func MakeDisabledRoleCallback(r Role) func(stage template.Stage, err error) error {
	return func(stage template.Stage, err error) error {
		if stage == template.STAGE0 { // only `enabled` has been processed so far
			if !r.IsEnabled() {
				rde := &template.RoleDisabledError{RolePath: r.GetPath()}
				return rde
			}
		}
		return err
	}
}
