/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2020 CERN and copyright holders of ALICE O².
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

// repoDefaultRevision sets global and per-repo default revision
var repoDefaultRevisionCmd = &cobra.Command{
	Use:   "default-revision <global-default-revision | repo-id default-revision>",
	Short: "set default global and per-repository revision",
	Long: `The repository default-revision command sets the global default repository revision.'

To set a per repository default revision, the default revision specified needs to be preceded by the repository index (not its name), as is reported by ` + "`coconut repo list`.",
	Example: ` * ` + "`coconut repo default-revision basic-tasks`" + ` Sets ` + "`basic-tasks`" + `as the global default-revision
 * ` + "`coconut repo default-revision 0 master`" + ` Sets ` + "`master`" + `as the default-revision for repo with index 0
 * ` + "`coconut repo default-revision 2 vs-sftb`" + ` Sets ` + "`vs-sftb`" + `as the default-revision for repo with index 2`,
	Run:  control.WrapCall(control.SetDefaultRevision),
	Args: cobra.RangeArgs(1, 2),
}

func init() {
	repoCmd.AddCommand(repoDefaultRevisionCmd)
}
