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
	"github.com/AliceO2Group/Control/coconut/configuration"
	"github.com/spf13/cobra"
)

// configurationDumpCmd represents the configuration list command
var configurationDumpCmd = &cobra.Command{
	Use:     "dump [key]",
	Aliases: []string{"d"},
	Short:   "dump configuration subtree",
	Long: `The configuration dump command requests from O² Configuration 
a subtree of key-values, and dumps it to standard output in the specified 
format. This command has full read access to the O² Configuration store 
and performs a raw query with no additional processing or access control
semantics.`,
	Run:  configuration.WrapCall(configuration.Dump),
	Args: cobra.ExactArgs(1),
}

func init() {
	configurationCmd.AddCommand(configurationDumpCmd)

	configurationDumpCmd.Flags().StringP("format", "f", "yaml", "output format for the configuration dump")

}
