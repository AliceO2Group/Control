/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2020 CERN and copyright holders of ALICE O².
 * Author: Ayaan Zaidi <azaidi@cern.ch>
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
	"github.com/AliceO2Group/Control/common/gera"
	"github.com/AliceO2Group/Control/core/task"
	"gopkg.in/yaml.v3"
)

func LoadDPL(tasks []*task.Class, rootRoleName string) (workflow Role, err error) {
	// FIXME: base roleBase of root defaults to all empty values
	root := new(aggregatorRole)
	root.roleBase.Name = rootRoleName

	for _, taskItem := range tasks {
		SingleTaskRole := taskRole{
			roleBase: roleBase{
				Name:        taskItem.Identifier.Name,
				parent:      root,
				Connect:     nil,
				Constraints: nil,
				Defaults:    gera.MakeStringMap(),
				Vars:        gera.MakeStringMap(),
				UserVars:    gera.MakeStringMap(),
				Locals:      nil,
				Bind:        nil,
			},
		}

		SingleTaskRole.Connect = append(SingleTaskRole.Connect, taskItem.Connect...)
		SingleTaskRole.Constraints = append(SingleTaskRole.Constraints, taskItem.Constraints...)
		SingleTaskRole.Defaults = gera.MakeStringMapWithMap(taskItem.Defaults.Raw())
		SingleTaskRole.Bind = append(SingleTaskRole.Bind, taskItem.Bind...)
		SingleTaskRole.Task = task.ClassToTask(taskItem, &SingleTaskRole)

		root.aggregator.Roles = append(root.aggregator.Roles, &SingleTaskRole)
	}

	workflow = root

	// FIXME: either get rid of err or add handling of errors
	return workflow, nil
}

func RoleToYAML(input Role) ([]byte, error) {
	// Auxiliary struct for marshalling
	auxRole := aggregatorRole{
		roleBase: roleBase{
			Name:        input.GetName(),
			Constraints: input.getConstraints(),
			Defaults:    gera.MakeStringMapWithMap(input.GetDefaults().Raw()),
			Vars:        gera.MakeStringMapWithMap(input.GetVars().Raw()),
			UserVars:    gera.MakeStringMapWithMap(input.GetUserVars().Raw()),
		},
		aggregator: aggregator{
			Roles: input.GetRoles(),
		},
	}

	yamlDATA, err := yaml.Marshal(&auxRole)
	return yamlDATA, err
}
