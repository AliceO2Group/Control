/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2020 CERN and copyright holders of ALICE O².
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

var configurationImportCmd = &cobra.Command{
	Use:     "import <component> <entry> <file_path>",
	Aliases: []string{"i", "imp"},
	Example: `coconut conf import <component> <entry> <file_path>
coconut conf import <component>/<run type>/<machine role>/<entry> <file_path>
coconut conf import <component> <entry> <file_path> --new-component
coconut conf import <component>/<run type>/<machine role>/<entry> <file_path> --format=json
coconut conf import <component> <entry> <file_path>.json
coconut conf import <component> <entry> <file_path> 
coconut conf import <component> <entry> <file_path> --new-component
`,
	Short: "Import a configuration file for the specified component and entry",
	Long: `The configuration import command generates a timestamp and saves
the configuration file to Consul under the <component>/<entry> path. 
Supported configuration file types are JSON, YAML, TOML and INI, 
and their file extensions are recognized automatically.`,
	Run:  configuration.WrapCall(configuration.Import),
	Args: cobra.RangeArgs(2, 3),
}

func init() {
	configurationCmd.AddCommand(configurationImportCmd)
	configurationImportCmd.Flags().BoolP("new-component", "n", false, "create a new configuration component while importing entry")
	configurationImportCmd.Flags().StringP("format", "f", "", "force a specific configuration file type, overriding any file extension")
	configurationImportCmd.Flags().StringP("runtype", "r", "", "request configuration for this run type (e.g. PHYSICS, TECHNICAL, etc.)")
	configurationImportCmd.Flags().StringP("role", "l", "", "request configuration for this O² machine role")
}
