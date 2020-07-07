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
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"io/ioutil"
	"os"
)

// convertCmd represents the convert command
var convertCmd = &cobra.Command{
	Use:   "convert",
	Short: "converts a DPL Dump to the required formats",
	Long: `The convert command takes a DPL input and outputs task and workflow templates. Optional flags can be provided to
specify which modules should be used when generating task templates. Control-OCCPlugin is always used as module.`,
	Run: func(cmd *cobra.Command, args []string) {
		filename, _ := cmd.Flags().GetStringArray("filename")
		// FIXME: only accepting first string
		modules, _ := cmd.Flags().GetStringArray("modules")

		for _, dump := range filename {
			file, err := ioutil.ReadFile(dump)
			if err != nil {
				fmt.Errorf("failed to open file &s: &v", file, err)
				os.Exit(1)
			}

			dplDump, err := converter.JSONImporter(file)
			taskClass, err := converter.ExtractTaskClasses(dplDump, modules)

			err = converter.TaskToYAML(taskClass)
			if err != nil {
				fmt.Errorf("conversion to task failed for %s: %v", file, err)
				os.Exit(1)
			}

			role, err := workflow.LoadDPL(taskClass, dump)
			err = converter.RoleToYAML(role)
			if err != nil {
				fmt.Errorf("conversion to workflow failed for %s: %v", file, err)
				os.Exit(1)
			}
		}
	},
}

var modules []string

func init() {
	rootCmd.AddCommand(convertCmd)

	convertCmd.Flags().StringArrayP("filename", "f", []string{}, "DPL dump to convert")
	viper.BindPFlag("filename", convertCmd.Flags().Lookup("filename"))
	convertCmd.MarkFlagRequired("filename")

	convertCmd.Flags().StringArrayP("modules", "m", []string{}, "modules to include")
	viper.BindPFlag("modules", convertCmd.Flags().Lookup("modules"))
}
