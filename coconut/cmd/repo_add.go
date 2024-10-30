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

// repoAddCmd represents the repo add command
var repoAddCmd = &cobra.Command{
	Use:     "add <repo url>",
	Aliases: []string{"new", "a"},
	Short:   "add a new git repository",
	Long: `The repository add command adds a git repository to the catalogue of repositories used for task and workflow configuration.
The default revision of the repository may be explicitly specified by passing the flag ` + "`--default-revision`" + ` . In any case,
the ensuing list is followed until a valid revision has been identified:

- explicitly set default revision (optional)
- global default revision
- ` + "`master`" + `
- ` + "`main`" + `

Exhaustion of the aforementioned list results in a repo add failure.

` + "`coconut repo add`" + ` can be called with
1) a repository identifier
2) a repository identifier coupled with the ` + "`--default-revision`" + ` flag (see examples below)

The protocol prefix should always be omitted.`,
	Example: ` * ` + "`coconut repo add github.com/AliceO2Group/ControlWorkflows`" + `
 * ` + "`coconut repo add github.com/AliceO2Group/ControlWorkflows --default-revision custom-rev`" + `
 * ` + "`coconut repo add alio2-cr1-hv-gw01.cern.ch:/opt/git/ControlWorkflows --default-revision custom-rev`" + `
 * ` + "`coconut repo add /home/flp/git/ControlWorkflows`",
	Run:  control.WrapCall(control.AddRepo),
	Args: cobra.ExactArgs(1),
}

func init() {
	repoCmd.AddCommand(repoAddCmd)

	repoAddCmd.Flags().StringP("default-revision", "d", "", "default revision for repository to add")
}
