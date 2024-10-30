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

// repoRemoveCmd represents the repository remove command
var repoDefaultCmd = &cobra.Command{
	Use:   "default <repo id>",
	Short: "set a git repository as default",
	Long: `The repository default command sets a git repository as the default repository for incoming workflow deployment requests.
A repository is referenced through its repo id, as reported by ` + "`coconut repo list`.",
	Example: ` * ` + "`coconut repo default 2`",
	Run:     control.WrapCall(control.SetDefaultRepo),
	Args:    cobra.ExactArgs(1),
}

func init() {
	repoCmd.AddCommand(repoDefaultCmd)
}
