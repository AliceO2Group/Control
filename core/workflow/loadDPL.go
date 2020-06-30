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
	"github.com/AliceO2Group/Control/core/task/channel"
	"github.com/AliceO2Group/Control/core/task/constraint"
	"gopkg.in/yaml.v3"

	"github.com/AliceO2Group/Control/common/gera"
	"github.com/AliceO2Group/Control/core/task"
)

func LoadDPL(tasks []*task.Class) (workflow Role, err error) {
	// FIXME: base roleBase of root defaults to all empty values
	root := new(aggregatorRole)

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

		SingleTaskRole.Connect     = append(SingleTaskRole.Connect, taskItem.Connect...)
		SingleTaskRole.Constraints = append(SingleTaskRole.Constraints, taskItem.Constraints...)
		SingleTaskRole.Defaults    = gera.MakeStringMapWithMap(taskItem.Defaults.Raw())
		SingleTaskRole.Bind        = append(SingleTaskRole.Bind, taskItem.Bind...)
		SingleTaskRole.Task        = task.ClassToTask(taskItem, &SingleTaskRole)

		root.aggregator.Roles = append(root.aggregator.Roles, &SingleTaskRole)
	}

	workflow = root

	// FIXME: either get rid of err or add handling of errors
	return workflow, nil
}

// Aux struct to fulfil export requirement by yaml.Marshal
type auxAggregatorRole struct {
	RoleBase   roleBase
	Aggregator aggregator
}

func RoleToYAML(input Role) ([]byte, error) {
	auxRole := auxAggregatorRole{
		RoleBase:   roleBase{},
		Aggregator: aggregator{
			Roles: input.GetRoles(),
		},
	}

	yamlDATA, err := yaml.Marshal(&auxRole)
	return yamlDATA, err
}

// Cannot invoke MarshalYAML on aggregatorRole (unexported members)
// auxAggregatorRole flattens roleBase and aggregator to have them
// marshalled at the same depth
func (a *auxAggregatorRole) MarshalYAML() (interface{}, error) {
	type _task struct {
		Load       string                   `yaml:"load"`
	}

	type _role struct {
		Name       string                   `yaml:"name"`
		Connect    []*channel.Outbound
		Task       _task                    `yaml:"task"`
	}

	type flatAggregatorRole struct {
		Name        string                  `yaml:"name"`
		Constraints constraint.Constraints  `yaml:"constraints,omitempty"`
		Defaults    gera.StringMap          `yaml:"defaults"`
		Vars        gera.StringMap          `yaml:"vars"`
		Aggregator  []_role                 `yaml:"roles"`
	}

	type rootWorkflow struct {
		Name        string                  `yaml:"name"`
		Defaults    gera.StringMap          `yaml:"defaults"`
		Roles       []flatAggregatorRole    `yaml:"roles"`
	}

	root := rootWorkflow{
		Name:     a.RoleBase.Name,
		Defaults: nil,
	}

	aux := flatAggregatorRole{
		Name:        a.RoleBase.Name,
		Constraints: nil,
		Defaults:    gera.MakeStringMap(),
		Vars:        gera.MakeStringMap(),
		Aggregator:  nil,
	}

	var auxAggregator []_role

	for _, eachTask := range a.Aggregator.GetTasks(){
		taskClass := *eachTask.GetTaskClass()
		auxRole := _role{
			Name:    taskClass.Identifier.Name,
			Task:    _task{
				Load: taskClass.Identifier.Name,
			},
		}

		for _, eachConnect := range taskClass.Connect {
			auxRole.Connect = append(auxRole.Connect, &eachConnect)
		}

		auxAggregator = append(auxAggregator, auxRole)
	}

	aux.Constraints = a.RoleBase.Constraints
	aux.Defaults    = a.RoleBase.Defaults
	aux.Vars        = a.RoleBase.Vars
	aux.Aggregator  = auxAggregator

	root.Roles = append(root.Roles, aux)

	return root, nil
}

