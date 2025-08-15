/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2018 CERN and copyright holders of ALICE O².
 * Author: Teo Mrnjavac <teo.mrnjavac@cern.ch>
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

//go:generate protoc -I=../../core -I=../../common --go_out=.. --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative --go-grpc_out=require_unimplemented_servers=false:.. protos/o2control.proto

// Package cmd implements the command line interface for coconut, providing
// various subcommands for managing O² Control environments and configurations.
package cmd

import (
	"fmt"

	"github.com/AliceO2Group/Control/common/product"
	"github.com/spf13/cobra"
)

// environmentCmd represents the environment command
var environmentCmd = &cobra.Command{
	Use:     "environment",
	Aliases: []string{"env", "e"},
	Short:   fmt.Sprintf("create, destroy and manage %s environments", product.PRETTY_SHORTNAME),
	Long: `The environments command allows you to perform operations on environments.

An environment is an instance of a data-driven workflow of tasks, along with its workflow configuration, task configuration and state.

Tasks are logically grouped into roles. Each environment has a distributed state machine, which aggregates the state of its constituent roles and tasks.

An environment can be created, it can be configured and reconfigured multiple times, and it can be started and stopped multiple times.

` + "```" + `
-> STANDBY -(CONFIGURE)-> CONFIGURED -(START_ACTIVITY)-> RUNNING
    |  ↑                   |  |  ↑                        |
    |   ------(RESET)------   |   ----(STOP_ACTIVITY)-----
    |                         |
    |-------------------------
  (EXIT)
    ↓
   DONE
` + "```" + `

If the current state is ` + "`RUNNING`" + `, the environment represents a ` + "`RUN`" + ` and has a run number. This number is only valid until the next ` + "`STOP_ACTIVITY`" + ` transition, each subsequent ` + "`START_ACTIVITY`" + ` transition will yield a new run number.

For more information on the behavior of coconut environments, see the subcommands linked below.`,
}

func init() {
	rootCmd.AddCommand(environmentCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// environmentCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// environmentCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
