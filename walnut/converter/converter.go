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

	"github.com/AliceO2Group/Control/common/controlmode"
	"github.com/AliceO2Group/Control/core/task"
	"github.com/AliceO2Group/Control/core/task/channel"
)

// return pointer to float64
func create(x float64) *float64 {
	return &x
}

// ExtractTaskClasses takes in a DPL Dump string and extracts
// an array of Tasks
func ExtractTaskClasses(DPL Dump) ([]*task.Class, error) {
	var tasks []*task.Class
	for index := range DPL.Workflows {

		defaultBindChannel := channel.Inbound{
			Channel: channel.Channel{
				Name:        DPL.Workflows[index].Name,
				Type:        channel.ChannelType(""),
				SndBufSize:  1000,
				RcvBufSize:  1000,
				RateLogging: 60,
				Transport:   channel.TransportType("shmem"),
			},
			Addressing: "ipc",
		}

		defaultConnectChannel := channel.Outbound{
			Channel: channel.Channel{
				Name:        DPL.Workflows[index].Name,
				Type:        channel.ChannelType(""),
				SndBufSize:  1000,
				RcvBufSize:  1000,
				RateLogging: 60,
				Transport:   channel.TransportType("shmem"),
			},
			Target: "",
		}

		var task = task.Class{
			Identifier: task.TaskClassIdentifier{
				Name: DPL.Workflows[index].Name,
			},
			Defaults: nil,
			Control: struct {
				Mode controlmode.ControlMode "yaml:\"mode\""
			}{Mode: 0},
			Wants: task.ResourceWants{
				Cpu:    create(0.15),
				Memory: create(128),
				Ports:  nil, //begin - end OR range
			},
			Bind:       []channel.Inbound{defaultBindChannel},
			Properties: nil,
			Connect:    []channel.Outbound{defaultConnectChannel},
		}
		fmt.Printf("Task: \n%v\n", task)
		tasks = append(tasks, &task)
	}
	return tasks, nil
}
