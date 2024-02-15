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
	"github.com/spf13/cobra"
)

// repoCmd represents the repository command
var repoCmd = &cobra.Command{
	Use:     "repository",
	Aliases: []string{"repo"},
	Short:   "manage git repositories for task and workflow configuration",
	Long: `The repository command performs operations on the repositories used for task and workflow configuration.

A valid workflow configuration repository must contain the directories ` + "`tasks`" + ` and ` + "`workflows`" + ` in its ` + "`master`" + ` branch.

When referencing a repository, the clone method should never be prepended. Supported repo backends and their expected format are:
- https: [hostname]/[repo_path]
- ssh: [hostname]:[repo_path]
- local [repo_path] (local repo entries are ephemeral and will not survive a core restart)

Examples of valid repository identifiers:

` + "```" + `
github.com/AliceO2Group/ControlWorkflows (https)
gitlab.cern.ch/tmrnjava/AliECS_conf/ (https)
alio2-cr1-hv-gw01.cern.ch:/opt/git/ControlWorkflows (ssh)
/home/flp/git/ControlWorkflows (local filesystem - (*entry does not survive a core restart*))
` + "```" + `

By default, all short task and workflow names are assumed to be in the default repository (see ` + "`coconut repo list`" + ` command).

Any workflow from any repository can be loaded by providing a full and unique path, e.g. the following two are different workflows:
` + "```" + `
github.com/AliceO2Group/ControlWorkflows/workflows/readout-qc-1
gitlab.cern.ch/tmrnjava/AliECS_conf/workflows/readout-qc-1
` + "```" + `

By default a workflow is loaded from its state at HEAD in the master branch. A request to load a workflow can further be qualified with a branch, tag or commit hash:
` + "```" + `
readout-qc-1@readout-testing
gitlab.cern.ch/tmrnjava/AliECS_conf/workflows/readout-qc-1@5c7f1c1f
` + "```" + `

Make sure to run ` + "`coconut repo refresh`" + ` if you make changes to a configuration repository.`,
}

func init() {
	rootCmd.AddCommand(repoCmd)
}
