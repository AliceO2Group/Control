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

var configurationListCmd = &cobra.Command{
	Use:     "list [component]",
	Aliases: []string{"l", "ls"},
	Example: `coconut conf list
coconut conf list <component>
coconut conf list <component> -t`,
	Short: "List all existing O² components in Consul",
	Long: `The configuration list command requests all components 
from O² Configuration as a list and displays it on the standard output`,
	Run:  configuration.WrapCall(configuration.List),
	Args: cobra.MaximumNArgs(1),
}

func init() {
	configurationCmd.AddCommand(configurationListCmd)
	configurationListCmd.Flags().StringP("output", "o", "yaml", "output format for the configuration list (yaml/json)")
	configurationListCmd.Flags().BoolP("timestamp", "t", false, "display latest timestamp entries for the requested component")
}
