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
    "github.com/AliceO2Group/Control/core/task/constraint"
)

func Graft(root Role, toAdd Role) (output Role){
    roles := root.(*aggregatorRole)
    roles.aggregator.Roles = append(roles.aggregator.Roles, toAdd)

    output = roles

    return output
}

var TestRoleBase = roleBase{
    Name:        "readout-{{ it }}",
    Connect:     nil,
    Constraints: constraint.Constraints{
        constraint.Constraint{
            Attribute: "machine_id",
            Value:     "{{ it }}",
        },
    },
    Defaults:    nil,
    Vars:        gera.MakeStringMapWithMap(map[string]string{
        "readout_cfg_uri": "file:/home/flp/readout.cfg",
    }),
    UserVars:    nil,
    Locals:      nil,
    Bind:        nil,
}

var TestAggregatorRole = aggregatorRole{
    roleBase:   TestRoleBase,
    aggregator: aggregator{},
}

var TestRole Role = &TestAggregatorRole