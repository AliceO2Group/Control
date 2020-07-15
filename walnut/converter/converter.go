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

package converter

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/mitchellh/go-homedir"
	"gopkg.in/yaml.v3"

	"github.com/AliceO2Group/Control/common"
	"github.com/AliceO2Group/Control/common/controlmode"
	"github.com/AliceO2Group/Control/common/gera"
	"github.com/AliceO2Group/Control/core/task"
	"github.com/AliceO2Group/Control/core/task/channel"
	"github.com/AliceO2Group/Control/core/workflow"
)

// return pointer to float64
func createFloat(x float64) *float64 {
	return &x
}

// return pointer to bool
func createBool(x bool) *bool {
	return &x
}

// return pointer to string
func createString(x string) *string {
	return &x
}

// ExtractTaskClasses takes in a DPL Dump string and extracts
// an array of Tasks
func ExtractTaskClasses(dplDump Dump, envModules []string) (tasks []*task.Class, err error) {
	envModules = append(envModules, "Control-OCCPlugin")

	for index := range dplDump.Workflows {
		taskName := dplDump.Workflows[index].Name
		correspondingMetadata := index + 1 // offset to match workflowEntry with correct metadataEntry
		channelNames := dplDump.Metadata[correspondingMetadata].Channels

		task := task.Class{
			Identifier: task.TaskClassIdentifier{
				Name: taskName,
			},
			Defaults: gera.MakeStringMapWithMap(map[string]string{
				"user": "flp",
			}),
			Control: struct {
				Mode controlmode.ControlMode "yaml:\"mode\""
			}{Mode: controlmode.FAIRMQ},
			Command: &common.CommandInfo{
				Env:       []string{}, // -> Default to empty array
				Shell:     createBool(true),
				Arguments: sanitizeCmdLineArgs(dplDump.Metadata[correspondingMetadata].CmdlLineArgs,
					taskName),
				User:      createString("flp"),
			},
			Wants: task.ResourceWants{
				Cpu:    createFloat(0.15),
				Memory: createFloat(128),
				Ports:  task.Ranges{}, // begin - end OR range
			},
			Properties: gera.MakeStringMapWithMap(map[string]string{
				"severity": "trace",
				"color":    "false",
			}),
		}

		value := fmt.Sprintf("eval `aliswmod load %s` &&\n%s", strings.Join(envModules, " "),
			dplDump.Metadata[correspondingMetadata].Executable)
		task.Command.Value = &value

		for _, channelName := range channelNames {
			// To avoid duplication, only "push" channels are included
			if strings.Contains(channelName, "from_"+taskName) {
				singleBind := channel.Inbound{
					Channel: channel.Channel{
						Name:      channelName,
						Type:      channel.ChannelType("push"),
						Transport: channel.TransportType("shmem"),
					},
					Addressing: "ipc",
				}
				task.Bind = append(task.Bind, singleBind)
			}

			if strings.Contains(channelName, "to_"+taskName) {
				// String manipulation to generate channel target of the form:
				// {{ Parent().Path }}.taskName:from_{initiator}_to_{receiver}
				initiator := channelName[5:strings.Index(channelName, "_to")]
				singleConnect := channel.Outbound{
					Channel: channel.Channel{
						Name:      channelName,
						Type:      channel.ChannelType("pull"),
						Transport: channel.TransportType("shmem"),
					},
					Target: fmt.Sprintf("{{ Parent().Path }}.%s:%s", initiator, channelName),
				}
				task.Connect = append(task.Connect, singleConnect)
			}
		}

		tasks = append(tasks, &task)
	}
	return tasks, nil
}

func sanitizeCmdLineArgs (input []string, taskName string) (output []string) {
	for index, value := range input {
		// Check args for dump arguments and remove them
		if  !strings.Contains(value, "--dump-workflow") ||
			!strings.Contains(value, "--dump-workflow-file") ||
			!strings.Contains(value, ".json") && input[index-1] != "--dump-workflow-file" {
				output = append(output, value)
		}
	}
	// Add --id parameter along with name of task to arguments
	output = append(output, "--id", taskName)

	return output
}

// GenerateTaskTemplate takes as input an array of pointers to task.Class
// and writes them to a AliECS friendly YAML file
func GenerateTaskTemplate(extractedTasks []*task.Class, outputDir string) (err error) {
	var dir string
	if outputDir == "" {
		dir, _ = os.Getwd()
	} else {
		dir = outputDir
	}
	path := filepath.Join(dir, "tasks")
	path, _ = homedir.Expand(path)
	os.MkdirAll(path, os.ModePerm)

	for _, SingleTask := range extractedTasks {
		YAMLData, err := yaml.Marshal(&SingleTask)
		if err != nil {
			return fmt.Errorf("marshal failed: %v", err)
		}

		fileName := filepath.Join(path, SingleTask.Identifier.Name+".yaml")
		if _, err := os.Stat(fileName); !os.IsNotExist(err) {
			if !confirmOverwrite(fmt.Sprintf("%s already exists, overwrite?", fileName)) {
				continue
			}
		}

		f, err := os.Create(fileName)
		defer f.Close()

		// Write marshaled YAML to file
		err = ioutil.WriteFile(fileName, YAMLData, 0644)
		if err != nil {
			return fmt.Errorf("creating file failed: %v", err)
		}
	}
	return
}

// confirmOverwrite takes an input string message to show the user and receive a yes/no response
// if anything other than y/n provided, keep asking
func confirmOverwrite (ask string) bool {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Printf("%s [y/n]", ask)
		response, err := reader.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}
		response = strings.ToLower(strings.TrimSpace(response))

		if response == "y" || response == "yes" {
			return true
		} else if response == "n" || response == "no" {
			return false
		} else {
			fmt.Printf("Invalid response\n")
		}
	}
}

func GenerateWorkflowTemplate(input workflow.Role, outputDir string) (err error) {
	var dir string
	if outputDir == "" {
		dir, _ = os.Getwd()
	} else {
		dir = outputDir
	}
	path := filepath.Join(dir, "workflows")
	path, _ = homedir.Expand(path)
	os.MkdirAll(path, os.ModePerm)

	yamlDATA, err := workflow.RoleToYAML(input)
	if err != nil {
		return fmt.Errorf("error converting role to YAML: %v", err)
	}

	fileName := filepath.Join(path, input.GetName()+".yaml")
	if _, err := os.Stat(fileName); !os.IsNotExist(err) {
		if !confirmOverwrite(fmt.Sprintf("%s already exists, overwrite?", fileName)) {
			return nil
		}
	}

	err = ioutil.WriteFile(fileName, yamlDATA, 0644)
	if err != nil {
		return fmt.Errorf("error writing role to file: %v", err)
	}

	return
}
