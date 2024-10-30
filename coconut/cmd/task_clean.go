/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2019 CERN and copyright holders of ALICE O².
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

package cmd

import (
	"github.com/AliceO2Group/Control/coconut/control"
	"github.com/spf13/cobra"
)

// taskCleanCmd represents the task clean command
var taskCleanCmd = &cobra.Command{
	Use:     "clean",
	Aliases: []string{"clean", "cleanup", "cl"},
	Short:   "clean up idle O² tasks",
	Long: `The task clean command removes all tasks that aren't currently associated with an environment. 
This includes AliECS tasks in any state. 
Alternatively, a list of task IDs to remove can be passed as a space-separated sequence of parameters.`,
	Run:  control.WrapCall(control.CleanTasks),
	Args: cobra.ArbitraryArgs,
}

func init() {
	taskCmd.AddCommand(taskCleanCmd)
}
