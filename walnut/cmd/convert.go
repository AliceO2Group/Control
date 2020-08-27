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
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/AliceO2Group/Control/common/utils"
	"github.com/AliceO2Group/Control/core/task"
	"github.com/AliceO2Group/Control/core/workflow"
	"github.com/AliceO2Group/Control/walnut/converter"
)

var outputDir string
var modules []string
var graft string
var workflowName string
var configURIVarname string
var staticConfigURI string
var extraVars string
var taskNamePrefix string


// convertCmd represents the convert command
var convertCmd = &cobra.Command{
	Use:   "convert",
	Short: "converts a DPL Dump to the required formats",
	Long: `The convert command takes a DPL input and outputs task and workflow templates. Optional flags can be provided to
specify which modules should be used when generating task templates. Control-OCCPlugin is always used as module.`,

	Run: func(cmd *cobra.Command, args []string) {
		for _, dumpFile := range args {
			// Strip .json from end of filename
			nameOfDump := filepath.Base(dumpFile)[:len(filepath.Base(dumpFile))-5]
			defaults := map[string]string {
				"user": "flp",
				nameOfDump + "_monitoring_url": "no-op://",
				"dpl_config": "",
			}

			// Open specified DPL Dump
			file, err := ioutil.ReadFile(dumpFile)
			if err != nil {
				err = fmt.Errorf("failed to open file &s: &w", dumpFile, err)
				fmt.Println(err.Error())
				os.Exit(1)
			}

			if configURIVarname != "" {
				defaults[configURIVarname] = staticConfigURI
			} else {
				defaults[nameOfDump + "_config_uri"] = staticConfigURI
			}


			// Import the dump and convert it to []*task.Class
			dplDump, err := converter.DPLImporter(file)
			taskClass, err := converter.ExtractTaskClasses(dplDump, taskNamePrefix, modules)

			if outputDir == "" {
				outputDir, _ = os.Getwd()
			}
			outputDir, _ = homedir.Expand(outputDir)

			extraVars = strings.TrimSpace(extraVars)
			if cmd.Flags().Changed("extra-vars") && len(extraVars) == 0 {
				fmt.Println(errors.New("empty list of extra-vars supplied").Error())
				os.Exit(1)
			}

			extraVarsMap, err := utils.ParseExtraVars(extraVars)
			if err != nil {
				fmt.Println(err.Error())
				os.Exit(1)
			}

			fmt.Printf("OPENED: %s", dumpFile)

			if graft == "" {
				if workflowName == "" {
					workflowName = nameOfDump
				}

				// If not grafting, simply convert dump to WFTs and TTs
				err = WriteTemplates(taskClass, workflowName, extraVarsMap, defaults)
				if err != nil {
					fmt.Println(err.Error())
					os.Exit(1)
				}
			} else {
				rolePath := strings.Split(graft, ":")
				workflowPath := rolePath[0]
				targetRolePath := rolePath[1]

				if workflowName == "" {
					workflowName = filepath.Base(workflowPath)
				}

				// Open existing workflow
				f, err := ioutil.ReadFile(workflowPath)
				if err != nil {
					log.Fatal(err)
				}
				root, err := workflow.LoadWorkflow(f)
				if err != nil {
					log.Fatal(err)
				}

				// Convert DPL to yaml.Node
				roleToGraft, err := workflow.LoadDPL(taskClass, dumpFile[:len(dumpFile)-5], extraVarsMap)
				if err != nil {
					log.Fatal(err)
				}
				roleData, err := workflow.RoleToYAML(roleToGraft)
				if err != nil {
					log.Fatal(err)
				}

				grafted, err := workflow.Graft(&root, targetRolePath, roleData, workflowName)
				if err != nil {
					log.Fatal(err)
				}
				path := filepath.Join(outputDir, workflowName+".yaml")
				fmt.Println("Writing to: ", path)
				err = ioutil.WriteFile(path, grafted, 0644)
				if err != nil {
					log.Fatal(err)
				}
			}
		}

		isGitRepo, _ := strconv.ParseBool(strings.TrimSpace(
			runGitCmd([]string{"rev-parse", "--is-inside-work-tree"})))

		if isGitRepo {
			fmt.Printf(runGitCmd([]string{"status"}))

			result := true
			prompt := &survey.Confirm{
				Message: "Would you like to view the git diff?",
				Default: true,
			}
			_ = survey.AskOne(prompt, &result)

			if result {
				fmt.Printf(runGitCmd([]string{"diff"}))
			}
		}
	},
	Args: cobra.MinimumNArgs(1),
}

func runGitCmd(args []string) string {
	cmd := exec.Command("git", args...)
	cmd.Dir = outputDir
	out, err := cmd.Output()
	if err != nil {
		log.Fatal(err)
	}

	return string(out)
}

func WriteTemplates(taskClass []*task.Class, nameOfDump string, extraVarsMap map[string]string, defaults map[string]string) (err error) {
	err = converter.GenerateTaskTemplate(taskClass, outputDir, defaults)
	if err != nil {
		return fmt.Errorf("conversion to task failed for %s: %w", nameOfDump, err)
	}

	role, err := workflow.LoadDPL(taskClass, nameOfDump, extraVarsMap)
	err = converter.GenerateWorkflowTemplate(role, outputDir)
	if err != nil {
		return fmt.Errorf("conversion to workflow failed for %s: %w", nameOfDump, err)
	}

	return nil
}

func init() {
	convertCmd.Flags().StringArrayVarP(&modules, "modules", "m", []string{}, "modules to load")
	_ = viper.BindPFlag("modules", convertCmd.Flags().Lookup("modules"))

	convertCmd.PersistentFlags().StringVarP(&outputDir, "output-dir", "o", "",
		"optional output directory")
	_ = viper.BindPFlag("output-dir", rootCmd.Flags().Lookup("output-dir"))

	convertCmd.PersistentFlags().StringVarP(&graft, "graft", "g", "",
		"graft converted DPL to an existing template")
	_ = viper.BindPFlag("graft", rootCmd.Flags().Lookup("graft"))

	convertCmd.PersistentFlags().StringVarP(&workflowName, "workflow-name", "w", "",
		"workflow to graft to")
	_ = viper.BindPFlag("workflow-name", rootCmd.Flags().Lookup("workflow-name"))

	convertCmd.PersistentFlags().StringVarP(&configURIVarname, "config-uri-varname", "", "",
		"name of config uri var")
	_ = viper.BindPFlag("config-uri-varname", rootCmd.Flags().Lookup("config-uri-varname"))

	convertCmd.PersistentFlags().StringVarP(&staticConfigURI, "static-config-uri", "", "",
		"path of config uri")
	_ = viper.BindPFlag("static-config-uri", rootCmd.Flags().Lookup("static-config-uri"))

	convertCmd.PersistentFlags().StringVarP(&extraVars, "extra-vars", "", "",
		"extra vars")
	_ = viper.BindPFlag("extra-vars", rootCmd.Flags().Lookup("extra-vars"))

	convertCmd.PersistentFlags().StringVarP(&taskNamePrefix, "task-name-prefix", "", "",
		"prefix for task name fields")
	_ = viper.BindPFlag("task-name-prefix", rootCmd.Flags().Lookup("task-name-prefix"))

	rootCmd.AddCommand(convertCmd)
}
