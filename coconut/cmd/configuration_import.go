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

var configurationImportCmd = &cobra.Command{
	Use:   "import <component> <entry> <file_path>",
	Aliases: []string{"i", "imp"},
	Example: `coconut conf import <component> <entry> <file_path>
coconut conf import <component/entry> <file_path>
coconut conf import <component> <entry> <file_path> -n
coconut conf import <component/entry> <file_path> --new-component
coconut conf import <component/entry> <file_path> --format=json
coconut conf import <component/entry> <file_path>.json
`,
	Short: "Import a configuration file for the component and entry specified",
	Long: `The configuration import command will generate a timestamp and save
the configuration file under the component/entry/timestamp path in Consul. Default
accepted file extensions are JSON, YAML, TOML, INI`,
	Run: configuration.WrapCall(configuration.Import),
	Args:  cobra.ExactArgs(3),
}

func init() {
	configurationCmd.AddCommand(configurationImportCmd)
	configurationImportCmd.Flags().BoolP("new-component", "n",  false, "used to add a new component")
	configurationImportCmd.Flags().StringP("format", "f",  "", "used to specify a different extension file than default (JSON, YAML, INI, TOML)")

}
