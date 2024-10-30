/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2019 CERN and copyright holders of ALICE O².
 * Author: Kostas Alexopoulos <kostas.alexopoulos@cern.ch>
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

// repoListCmd represents the repo list command
var repoListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"l"},
	Short:   "list git repositories",
	Long:    "The repostory list command lists available git repositories to be used for task and workflow configuration.",
	Run:     control.WrapCall(control.ListRepos),
}

func init() {
	repoCmd.AddCommand(repoListCmd)
}
