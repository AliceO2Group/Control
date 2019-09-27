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

var configurationShowCmd = &cobra.Command{
	Use:   "show <component> <entry>",
	Aliases: []string{"s"},
	Example: `coconut conf show <component> <entry> 
coconut conf show <component> <entry> -t <timestamp>
coconut conf show <component>/<entry>
coconut conf show <component>/<entry> -t <timestamp>
coconut conf show <component>/<entry>@<timestamp>`,
	Short: "Show configuration for the component and entry specified",
	Long: `The configuration show command requests by default the latest 
configuration for the specified component and entry. It can request exact 
time configuration by specifying wanted timestamp as flag`,
	Run: configuration.WrapCall(configuration.Show),
	Args:  cobra.RangeArgs(0, 3),
}

func init() {
	configurationCmd.AddCommand(configurationShowCmd)
	configurationShowCmd.Flags().StringP("timestamp", "t",  "", "request configuration for this timestamp")
}
