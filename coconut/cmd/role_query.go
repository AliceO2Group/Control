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

package cmd

import (
	"github.com/AliceO2Group/Control/coconut/control"
	"github.com/spf13/cobra"
)

// roleQueryCmd represents the role list command
var roleQueryCmd = &cobra.Command{
	Use:     "query [environment id] [query path]",
	Aliases: []string{"query", "q"},
	Short:   "query O² roles",
	Long: `The role query command returns one or more role subtrees within a given environment.

It allows the user to inspect the environment's workflow at runtime, after template processing operations, including
iterator expansion, just-in-time subworkflow inclusion, and variable substitution, have taken place.

For the target role or roles, it also prints the locally defined variables, as well as the consolidated variable
stack from the point of view of that role.

The role query command accepts two arguments: the environment ID and the query path. The query path is a dot-separated
walk through the role tree of the given environment, starting from the root role. Wildcard expressions are allowed, as
per https://github.com/gobwas/glob syntax.

Examples:
 * ` + "`coconut role query 2rE9AV3m1HL readout-dataflow`" + ` - queries the role ` + "`readout-dataflow`" + ` in environment ` + "`2rE9AV3m1HL`" + `, prints the full tree, along with the variables defined in the root role
 * ` + "`coconut role query 2rE9AV3m1HL readout-dataflow.host-aido2-bld4-lab102`" + ` - queries the role ` + "`readout-dataflow.host-aido2-bld4-lab102`" + `, prints the subtree of that role, along with the variables defined in it
 * ` + "`coconut role query 2rE9AV3m1HL readout-dataflow.host-aido2-bld4-lab102.data-distribution.stfs`" + ` - queries the role at the given path, it is a task role so there is no subtree, prints the variables defined in that role
 * ` + "`coconut role query 2rE9AV3m1HL readout-dataflow.host-aido2-bld4-lab*`" + ` - queries the roles matching the given glob expression, prints all the subtrees and variables
 * ` + "`coconut role query 2rE9AV3m1HL readout-dataflow.host-aido2-bld4-lab*.data-distribution.*`" + ` - queries the task roles matching the given glob expression, prints all the variables`,
	Run:  control.WrapCall(control.QueryRoles),
	Args: cobra.ExactArgs(2),
}

func init() {
	roleCmd.AddCommand(roleQueryCmd)
}
