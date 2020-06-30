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
	"github.com/AliceO2Group/Control/core/workflow"
	"github.com/AliceO2Group/Control/walnut/converter"
	"io/ioutil"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// convertCmd represents the convert command
var convertCmd = &cobra.Command{
	Use:   "convert",
	Short: "Converts a DPL Dump to the required formats.",
	Long: `The convert command takes a DPL input and outputs task and workflow templates. Optional flags can be provided to
selectively output task or workflow templates. By default, both templates are produced.
`,

	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("convert called with args: " + strings.Join(args, " "))

		template, _ := cmd.Flags().GetString("template")
		for _, filename := range args {
			file, err := ioutil.ReadFile(filename)
			if err != nil {
				fmt.Errorf("failed to open file &s: &v", filename, err)
				os.Exit(1)
			}

			dplDump, err := converter.JSONImporter(file)
			taskClass, err := converter.ExtractTaskClasses(dplDump)

			if template == "task" {
				err = converter.TaskToYAML(taskClass)
				if err != nil {
					fmt.Errorf("conversion to task failed for %s: %v", filename, err)
					os.Exit(1)
				}
			} else if template == "workflow" {
				workflowRole, err := workflow.LoadDPL(taskClass)
				err = converter.RoleToYAML(workflowRole)
				if err != nil {
					fmt.Errorf("conversion to workflow failed for %s: %v", filename, err)
					os.Exit(1)
				}
			} else {
				err = converter.TaskToYAML(taskClass)
				if err != nil {
					fmt.Errorf("conversion to task failed for %s: %v", filename, err)
					os.Exit(1)
				}

				workflowRole, err := workflow.LoadDPL(taskClass)
				err = converter.RoleToYAML(workflowRole)
				if err != nil {
					fmt.Errorf("conversion to workflow failed for %s: %v", filename, err)
					os.Exit(1)
				}
			}
		}
	},
	Args: cobra.ExactArgs(1),
}

func init() {
	rootCmd.AddCommand(convertCmd)

	convertCmd.Flags().StringP("template", "t", "", "template to generate")
	convertCmd.MarkFlagRequired("filename")
}
