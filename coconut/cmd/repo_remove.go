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

// repoRemoveCmd represents the repo remove command
var repoRemoveCmd = &cobra.Command{
	Use:     "remove <repo id>",
	Aliases: []string{"r", "delete", "del", "d"},
	Short:   "remove a git repository",
	Long: `The repository remove command removes a git repository from the catalogue of workflow configuration sources.
A repository is referenced by its repo id, as reported by` + "`coconut repo list`",
	Example: ` * ` + "`coconut repo remove 1`" + `
 * ` + "`coconut repo del 2`",
	Run:  control.WrapCall(control.RemoveRepo),
	Args: cobra.ExactArgs(1),
}

func init() {
	repoCmd.AddCommand(repoRemoveCmd)
}
