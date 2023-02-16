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

package cmd

import (
	"fmt"

	"github.com/AliceO2Group/Control/coconut/control"
	"github.com/AliceO2Group/Control/common/product"
	"github.com/spf13/cobra"
)

// environmentCreateCmd represents the environment list command
var environmentCreateCmd = &cobra.Command{
	Use:     "create",
	Aliases: []string{"new", "c", "n"},
	Short:   "create a new environment",
	Long: fmt.Sprintf(`The environment create command requests from %s the
creation of a new environment.

The operation may or may not be successful depending on available resources and configuration.

A valid workflow template (sometimes called simply "workflow" for brevity) must be passed to this command via the mandatory workflow-template flag.

Workflows and tasks are managed with a git based configuration system, so the workflow template may be provided simply by name or with repository and branch/tag/hash constraints.
Examples:
 * `+"`coconut env create -w myworkflow`"+` - loads workflow `+"`myworkflow`"+` from default configuration repository at HEAD of master branch
 * `+"`coconut env create -w github.com/AliceO2Group/MyConfRepo/myworkflow`"+` - loads a workflow from a specific git repository, HEAD of master branch
 * `+"`coconut env create -w myworkflow@rev`"+` - loads a workflow from default repository, on branch, tag or revision `+"`rev`"+`
 * `+"`coconut env create -w github.com/AliceO2Group/MyConfRepo/myworkflow@rev`"+` - loads a workflow from a specific git repository, on branch, tag or revision `+"`rev`"+`
 * `+"`coconut env create -c /home/myrepo/myconfigfile.json -e '{\"hosts\":\"[\\\"my-test-machine\\\"]\"}'`"+`
 * `+"`coconut env create -c consul:///o2/runtime/COG-v1/TECHNICAL -w readout-dataflow@myBranch -e '{\"hosts\":\"[\\\"my-test-machine\\\"]\"}'`"+`
 * `+"`coconut env create -c TECHNICAL -e '{\"hosts\":\"[\\\"my-test-machine\\\"]\"}'`"+`

For more information on the %s workflow configuration system, see documentation for the `+"`coconut repository`"+` command.`, product.PRETTY_SHORTNAME, product.PRETTY_SHORTNAME),
	Run: control.WrapCall(control.CreateEnvironment),
}

func init() {
	environmentCmd.AddCommand(environmentCreateCmd)

	environmentCreateCmd.Flags().StringP("configuration", "c", "", "high-level configuration payload to be loaded for the new environment")
	environmentCreateCmd.Flags().StringP("workflow-template", "w", "", "workflow to be loaded in the new environment")

	environmentCreateCmd.Flags().BoolP("auto", "a", false, "create an autorun environment")
	environmentCreateCmd.Flags().BoolP("public", "p", true, "control public rights of the environment")
	environmentCreateCmd.Flags().BoolP("asynchronous", "y", false, "use asynchronous mode for environment creation")

	environmentCreateCmd.Flags().StringP("extra-vars", "e", "", "values passed using key=value CSV or JSON syntax, interpreted as strings `key1=val1,key2=val2` or `{\"key1\": \"value1\", \"key2\": \"value2\"}`")
}
