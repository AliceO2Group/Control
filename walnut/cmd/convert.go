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
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/AliceO2Group/Control/core/workflow"
	"github.com/AliceO2Group/Control/walnut/converter"
)

var outputDir string
var modules []string

// convertCmd represents the convert command
var convertCmd = &cobra.Command{
	Use:   "convert",
	Short: "converts a DPL Dump to the required formats",
	Long: `The convert command takes a DPL input and outputs task and workflow templates. Optional flags can be provided to
specify which modules should be used when generating task templates. Control-OCCPlugin is always used as module.`,

	Run: func(cmd *cobra.Command, args []string) {
		for _, dumpFile := range args {
			// Strip .json from end of filename
			nameOfDump := dumpFile[:len(dumpFile)-5]

			file, err := ioutil.ReadFile(dumpFile)
			if err != nil {
				err = fmt.Errorf("failed to open file &s: &w", dumpFile, err)
				fmt.Println(err.Error())
				os.Exit(1)
			}

			dplDump, err := converter.DPLImporter(file)
			taskClass, err := converter.ExtractTaskClasses(dplDump, modules)

			if outputDir == "" {
				outputDir, _ = os.Getwd()
			}
			outputDir, _ = homedir.Expand(outputDir)

			runGitCmd := func(args []string) string {
				cmd := exec.Command("git", args...)
				cmd.Dir = outputDir
				out, err := cmd.Output()
				if err != nil {
					log.Fatal(err)
				}

				return string(out)
			}

			isGitRepo, _ := strconv.ParseBool(strings.TrimSpace(
				runGitCmd([]string{"rev-parse", "--is-inside-work-tree"})))

			err = converter.GenerateTaskTemplate(taskClass, outputDir)
			if err != nil {
				err = fmt.Errorf("conversion to task failed for %s: %w", dumpFile, err)
				fmt.Println(err.Error())
				os.Exit(1)
			}

			role, err := workflow.LoadDPL(taskClass, nameOfDump)
			err = converter.GenerateWorkflowTemplate(role, outputDir)
			if err != nil {
				err = fmt.Errorf("conversion to workflow failed for %s: %w", dumpFile, err)
				fmt.Println(err.Error())
				os.Exit(1)
			}

			if isGitRepo {
				fmt.Printf(runGitCmd([]string{"status"}))

				result := true
				prompt := &survey.Confirm{
					Message: "Would you like to view the git diff?",
				}
				_ = survey.AskOne(prompt, &result)

				if result {
					fmt.Printf(runGitCmd([]string{"diff"}))
				}
			}
		}
	},
	Args: cobra.MinimumNArgs(1),
}

func init() {
	convertCmd.Flags().StringArrayVarP(&modules, "modules", "m", []string{}, "modules to load")
	_ = viper.BindPFlag("modules", convertCmd.Flags().Lookup("modules"))

	convertCmd.PersistentFlags().StringVarP(&outputDir, "output-dir", "o", "",
		"optional output directory")
	viper.BindPFlag("output-dir", rootCmd.Flags().Lookup("output-dir"))

	rootCmd.AddCommand(convertCmd)
}
