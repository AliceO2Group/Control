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
	"fmt"

	"github.com/AliceO2Group/Control/coconut/control"
	"github.com/spf13/cobra"
)

// repoDefaultRevision sets global and per-repo default revision
var repoDefaultRevisionCmd = &cobra.Command{
	Use:   "default-revision",
	Short: "set default global and per-repository revision",
	Long: fmt.Sprintf(`The repository default-revision command sets the global default
repository revision as well as the per-repository default revision.`),
	Run:   control.WrapCall(control.SetDefaultRevision),
}

func init() {
	repoCmd.AddCommand(repoDefaultRevisionCmd)
}
