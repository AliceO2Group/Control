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

// environmentModifyCmd represents the environment list command
var environmentModifyCmd = &cobra.Command{
	Use:     "modify [environment id]",
	Aliases: []string{"mod", "m"},
	Short:   "modify an environment",
	Long: `The environment modify command changes the roles workflow of an 
existing O² environment.`,
	Run:  control.WrapCall(control.ModifyEnvironment),
	Args: cobra.ExactArgs(1),
}

func init() {
	//environmentCmd.AddCommand(environmentModifyCmd)
	//
	//environmentModifyCmd.Flags().StringArrayP("addroles", "a", []string{}, "a list of roles to add to the environment")
	//environmentModifyCmd.Flags().StringArrayP("removeroles", "r", []string{}, "a list of roles to remove from the environment")
	//environmentModifyCmd.Flags().BoolP("reconfigure", "c", false, "reconfigure all roles")
}
