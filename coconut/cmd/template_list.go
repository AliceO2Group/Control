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
	"github.com/AliceO2Group/Control/coconut/control"
	"github.com/spf13/cobra"
)

// templateListCmd represents the template list command
var templateListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"list", "ls", "l"},
	Short:   "list available workflow templates",
	Long: `The template list command shows a list of available workflow templates.
These workflow templates can then be loaded to create an environment.

` + "`coconut templ list` " + `can be called with 
1) a combination of the ` + "`--repo` " + `, ` + "`--revision` " + `, ` + "`--all-branches` " + `, ` + "`--all-tags` " + `, ` + "`--all-workflows` " + `flags, or with
2) an argument in the form of [repo-pattern]@[revision-pattern], where the patterns are globbing.`,
	Example: ` * ` + "`coconut templ list`" + ` lists templates from the HEAD of master for all git repositories
 * ` + "`coconut templ list --all-workflows`" + ` lists all templates (including non-public ones) from the HEAD of master for all git repositories
 * ` + "`coconut templ list '*AliceO2Group*'`" + ` lists all templates coming from the HEAD of master of git repositories that match the pattern *AliceO2Group*
 * ` + "`coconut templ list '*@v*'`" + ` lists templates coming from revisions matching the ` + "`v*`" + `pattern for all git repositories
 * ` + "`coconut templ list --repository='*AliceO2Group*'`" + ` lists all templates coming from the HEAD of master of git repositories that match the pattern *AliceO2Group*
 * ` + "`coconut templ list --revision='dev*'`" + ` lists templates coming from revisions matching the ` + "`dev*`" + `pattern for all git repositories
 * ` + "`coconut templ list --repository='*gitlab.cern.ch*' --revision=master`" + ` lists templates for revisions ` + "`master`" + `for git repositories matching ` + "`*gitlab.cern.ch*`" + `
 * ` + "`coconut templ list --all-branches`" + ` lists templates from all branches for all git repositories
 * ` + "`coconut templ list --repository='*github.com*' --all-tags`" + ` lists templates from all tags for git repositories which match the *github.com* pattern
 * ` + "`coconut templ list --revision=5c7f1c1fded1b87243998579ed876c8035a08377 `" + ` lists templates from the commit corresponding to the hash for all git repositories`,

	Run: control.WrapCall(control.ListWorkflowTemplates),
}

func init() {
	templateCmd.AddCommand(templateListCmd)

	templateListCmd.Flags().StringP("repository", "r", "", "repositories to list templates from")
	templateListCmd.Flags().StringP("revision", "i", "", "revisions (branches/tags) to list templates from")
	templateListCmd.Flags().BoolP("all-branches", "b", false, "list templates from all branches")
	templateListCmd.Flags().BoolP("all-tags", "t", false, "list templates from all tags")
	templateListCmd.Flags().BoolP("all-workflows", "a", false, "list all templates, even non-public ones")
	templateListCmd.Flags().BoolP("show-description", "d", false, "show the description of each template")
}
