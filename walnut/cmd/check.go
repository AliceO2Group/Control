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

package cmd

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/spf13/viper"

	"github.com/AliceO2Group/Control/walnut/schemata"
	"github.com/spf13/cobra"
)

// checkCmd represents the check command
var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "validate the file against a specified schema",
	Long: `The check command validates the given file against a specified schema. This file can be a task template, 
a workflow template or an O² DPL Dump. Each of these have a schema to validate against.

Valid schema formats:
  workflow  task  dpl_dump`,

	Run: func(cmd *cobra.Command, args []string) {
		format, _ := cmd.Flags().GetString("format")

		for _, filename := range args {
			file, err := ioutil.ReadFile(filename)
			if err != nil {
				fmt.Printf("failed to open file %s: %v", filename, err)
				os.Exit(1)
			}
			err = schemata.Validate(file, format)
			if err != nil {
				fmt.Printf("validation failed: %v", err)
				os.Exit(1)
			}
		}
	},
	Args: cobra.MinimumNArgs(1),
}

var format string

func init() {
	rootCmd.AddCommand(checkCmd)

	checkCmd.Flags().StringP("format", "f", "", "format to validate against")
	viper.BindPFlag("format", checkCmd.Flags().Lookup("format"))
	checkCmd.MarkFlagRequired("format")
}
