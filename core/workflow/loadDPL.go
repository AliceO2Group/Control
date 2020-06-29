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
)

func LoadDPL(tasks []*task.Class) (workflow Role, err error) {
	root := new(aggregatorRole)

	for _, taskItem := range tasks {
		SingleTaskRole := taskRole{
			roleBase: roleBase{
				Name:        taskItem.Identifier.Name,
				parent:      root,
				Connect:     nil,
				Constraints: nil,
				Defaults:    &gera.StringWrapMap{},
				Vars:        &gera.StringWrapMap{},
				UserVars:    &gera.StringWrapMap{},
				Locals:      nil,
				Bind:        nil,
			},
		}

		for _, eachConnect := range taskItem.Connect {
			SingleTaskRole.Connect = append(SingleTaskRole.Connect, eachConnect)
		}

		for _, eachConstraint := range taskItem.Constraints {
			SingleTaskRole.Constraints = append(SingleTaskRole.Constraints, eachConstraint)
		}

		SingleTaskRole.Defaults = gera.MakeStringMapWithMap(taskItem.Defaults.Raw())

		for _, eachBind := range taskItem.Bind {
			SingleTaskRole.Bind = append(SingleTaskRole.Bind, eachBind)
		}

		SingleTaskRole.Task = task.ClassToTask(taskItem, &SingleTaskRole)

		root.aggregator.Roles = append(root.aggregator.Roles, &SingleTaskRole)
	}

	workflow = root
	return workflow, nil
}
