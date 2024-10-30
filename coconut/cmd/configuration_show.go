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
	Use:     "show <component> <entry>",
	Aliases: []string{"s"},
	Example: `coconut conf show <component> <entry> 
coconut conf show <component> <entry> -t <timestamp>
coconut conf show <component>/<run type>/<machine role>/<entry>
coconut conf show <component>/<run type>/<machine role>/<entry> -t <timestamp>
coconut conf show <component>/<run type>/<machine role>/<entry>@<timestamp>
coconut conf show <component> <entry> -r <run type> -l <machine role> -t <timestamp>'
coconut conf show <component> <entry> -s -e '{"key1": "value1", "key2": "value2"}'`,
	Short: "Show configuration for the component and entry specified",
	Long: `The configuration show command returns the most recent 
configuration revision for the specified component and entry. 
It can also return a specific revision, requested with the --timestamp/-t flag`,
	Run:  configuration.WrapCall(configuration.Show),
	Args: cobra.RangeArgs(0, 3),
}

func init() {
	configurationCmd.AddCommand(configurationShowCmd)
	configurationShowCmd.Flags().StringP("timestamp", "t", "", "request configuration for this timestamp")
	configurationShowCmd.Flags().StringP("runtype", "r", "", "request configuration for this run type (e.g. PHYSICS, TECHNICAL, etc.)")
	configurationShowCmd.Flags().StringP("role", "l", "", "request configuration for this O² machine role")
	configurationShowCmd.Flags().BoolP("simulate", "s", false, "simulate runtime template processing on the configuration payload")
	// The following only applies if simulate is set:
	configurationShowCmd.Flags().StringP("extra-vars", "e", "", "values passed using key=value CSV or JSON syntax, interpreted as strings `key1=val1,key2=val2` or `{\"key1\": \"value1\", \"key2\": \"value2\"}`")
}
