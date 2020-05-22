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
	"strings"

	"github.com/AliceO2Group/Control/walnut/validate"

	"github.com/spf13/cobra"
)

// checkCmd represents the check command
var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "check the file passed against a specified schema.",
	Long: `The check command validates the given file against a specified schema. 

Usage:
  walnut check --format [template] [file]

Example:
  walnut check --format workflow_template readout-sftb.yaml

Valid schemas:
  workflow_template  task_template  dpl_dump`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("check called with arg: " + strings.Join(args, " "))
		validate.Template(args[0], args[1]) //TODO: Handle two arguments in cobra
	},
	// Args: cobra.ExactArgs(1),
}

func init() {
	rootCmd.AddCommand(checkCmd)

	checkCmd.Flags().StringP("format", "f", "", "format to validate against")
	checkCmd.MarkFlagRequired("format")
}
