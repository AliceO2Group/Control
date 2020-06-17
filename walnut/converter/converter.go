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
	"fmt"
	"io/ioutil"
	"os"

	"github.com/AliceO2Group/Control/common"
	"github.com/AliceO2Group/Control/common/controlmode"
	"github.com/AliceO2Group/Control/common/gera"
	"github.com/AliceO2Group/Control/core/task"
	"github.com/AliceO2Group/Control/core/task/channel"
	"github.com/AliceO2Group/Control/core/task/constraint"
	"gopkg.in/yaml.v3"
)

// return pointer to float64
func create(x float64) *float64 {
	return &x
}

type TaskTemplate struct {
	Identifier task.TaskClassIdentifier `yaml:"name"`
	Defaults   gera.StringMap           `yaml:"defaults"`
	Control    struct {
		Mode controlmode.ControlMode `yaml:"mode"`
	} `yaml:"control"`
	Command     *common.CommandInfo     `yaml:"command"`
	Wants       task.ResourceWants      `yaml:"wants"`
	Bind        []channel.Inbound       `yaml:"bind"`
	Properties  gera.StringMap          `yaml:"properties"`
	Constraints []constraint.Constraint `yaml:"constraints"`
	Arguments   []string                `yaml:"arguments"`
}

// ExtractTaskClasses takes in a DPL Dump string and extracts
// an array of Tasks
func ExtractTaskClasses(DPL Dump) (tasks []*TaskTemplate, err error) {

	var channelName string
	for index := range DPL.Workflows {
		if index+1 == len(DPL.Workflows) {
			channelName = DPL.Workflows[index].Name
		} else {
			channelName = "from_"+DPL.Workflows[index].Name+"_to_"+DPL.Workflows[index+1].Name
		}
		workflowName := DPL.Workflows[index].Name
		defaultBindChannel := channel.Inbound{
			Channel: channel.Channel{
				Name:        channelName,
				Type:        channel.ChannelType(""),
				SndBufSize:  1000,
				RcvBufSize:  1000,
				RateLogging: 60,
				Transport:   channel.TransportType("shmem"),
			},
			Addressing: "ipc",
		}

		/*
			defaultConnectChannel := channel.Outbound{
				Channel: channel.Channel{
					Name:        workflowName,
					Type:        channel.ChannelType(""),
					SndBufSize:  1000,
					RcvBufSize:  1000,
					RateLogging: 60,
					Transport:   channel.TransportType("shmem"),
				},
				// Target: "", No default value
			}
		*/

		var arguments []string
		for _, arg := range DPL.Metadatas[index+1].CmdlLineArgs {
			arg = fmt.Sprintf("%q", arg)
			arguments = append(arguments, arg)
		}

		task := TaskTemplate{
			Identifier: task.TaskClassIdentifier{
				Name: workflowName,
			},
			Defaults: gera.MakeStringMapWithMap(map[string]string{
				"user": "flp",
			}),
			Control: struct {
				Mode controlmode.ControlMode "yaml:\"mode\""
			}{Mode: controlmode.FAIRMQ},
			Wants: task.ResourceWants{
				Cpu:    create(0.15),
				Memory: create(128),
				Ports:  task.Ranges{}, //begin - end OR range
			},
			Bind: []channel.Inbound{defaultBindChannel},
			Properties: gera.MakeStringMapWithMap(map[string]string{
				"severity": "trace",
				"color":    "false",
			}),
			// Connect:   []channel.Outbound{defaultConnectChannel},
			Arguments: arguments,
		}
		fmt.Printf("\nTASK:\n%v\n", task)
		tasks = append(tasks, &task)
	}
	return tasks, nil
}

func taskToYAML(extractedTasks []*TaskTemplate) (err error) {

	_, err = os.Stat("tasks")
 
	if os.IsNotExist(err) {
		errDir := os.MkdirAll("tasks", 0755)
		if errDir != nil {
			return fmt.Errorf("create dir failed: %w", err)
		}
	}

	for _, SingleTask := range extractedTasks {
		YAMLData, err := yaml.Marshal(&SingleTask)
		if err != nil {
			return fmt.Errorf("Marshal failed: %w", err)
		}

		// Write marshaled YAML to file
		err = ioutil.WriteFile("tasks/"+SingleTask.Identifier.Name+".yaml", YAMLData, 0644)
		if err != nil {
			return fmt.Errorf("Creating file failed: %w", err)
		}

	}
	return
}

func (t *TaskTemplate) MarshalYAML(marshal func(interface{}) error) (err error) {
	type _class struct {
		Identifier task.TaskClassIdentifier
		Defaults   map[string]string `yaml:"defaults"`
		Control    struct {
			Mode controlmode.ControlMode `yaml:"mode"`
		} `yaml:"control"`
		Command     *common.CommandInfo     `yaml:"command"`
		Wants       task.ResourceWants      `yaml:"wants"`
		Bind        []channel.Inbound       `yaml:"bind"`
		Properties  map[string]string       `yaml:"properties"`
		Constraints []constraint.Constraint `yaml:"constraints"`
		// Connect     []channel.Outbound      `yaml:"connect"`
	}

	aux := _class{
		Defaults:   make(map[string]string),
		Properties: make(map[string]string),
	}
	err = marshal(&aux)

	if err == nil {
		*t = TaskTemplate{
			Identifier:  aux.Identifier,
			Defaults:    gera.MakeStringMapWithMap(aux.Defaults),
			Control:     aux.Control,
			Command:     aux.Command,
			Wants:       aux.Wants,
			Bind:        aux.Bind,
			Properties:  gera.MakeStringMapWithMap(aux.Properties),
			Constraints: aux.Constraints,
			// Connect:     aux.Connect,
		}
	}
	return
}
