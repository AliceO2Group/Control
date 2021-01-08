/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2019 CERN and copyright holders of ALICE O².
 * Author: George Raduta <george.raduta@cern.ch>
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

var configurationHistoryCmd = &cobra.Command{
	Use:   "history <query>",
	Aliases: []string{"h"},
	Example: `coconut conf history <component> <entry>
coconut conf history <component>/<run type>/<machine role>/<entry>`,
	Short: "List all existing entries with timestamps of a specified component in Consul",
	Long: `The configuration history command returns all timestamps for a specified component
and entry`,
	Run: configuration.WrapCall(configuration.History),
	Args: cobra.RangeArgs(1, 2),
}

func init() {
	configurationCmd.AddCommand(configurationHistoryCmd)
	configurationHistoryCmd.Flags().StringP("output", "o", "yaml", "output format for the returned entries (yaml/json)")
	configurationHistoryCmd.Flags().StringP("runtype", "r",  "", "request configuration for this run type (e.g. PHYSICS, TECHNICAL, etc.)")
	configurationHistoryCmd.Flags().StringP("role", "l",  "", "request configuration for this O² machine role")
}
