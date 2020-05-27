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

//go:generate go run ../schemata/includeSchemata.go

package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/AliceO2Group/Control/walnut/validate"
	"github.com/spf13/cobra"
)

// checkCmd represents the check command
var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "check the file passed against a specified schema.",
	Long: `The check command validates the given file against a specified 
schema. This file can be a task template, a workflow template or an O² 
DPL Dump. Each of those have a schema provided to validate against. 

Usage:
  walnut check --format [format] [file]

Example:
  walnut check --format workflow_template readout-sftb.yaml

Valid schemata:
  workflow  task  dpl_dump`,

	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("check called with arg: " + strings.Join(args, " "))

		format, _ := cmd.Flags().GetString("format")
		for _, filename := range args {
			rawYAML, err := ioutil.ReadFile(filename)
			if err != nil {
				fmt.Printf("failed to open file %s: %v", filename, err)
				os.Exit(1)
			}
			err = validate.CheckSchema(rawYAML, format)
			if err != nil {
				fmt.Printf("validation failed: %v", err)
				os.Exit(1)
			}
		}
	},
	Args: cobra.ExactArgs(1),
}

func init() {
	rootCmd.AddCommand(checkCmd)

	checkCmd.Flags().StringP("format", "f", "", "format to validate against")
	checkCmd.MarkFlagRequired("format")
}
