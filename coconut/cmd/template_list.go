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
	"github.com/spf13/cobra"
	"github.com/AliceO2Group/Control/coconut/control"
)

// templateListCmd represents the template list command
var templateListCmd = &cobra.Command{
	Use:   "list",
	Aliases: []string{"list", "ls", "l"},
	Short: "list available workflow templates",
	Long: `The template list command shows a list of available workflow templates.
These workflow templates can then be loaded to create an environment.`,
	Run:   control.WrapCall(control.ListWorkflowTemplates), //TODO: Add help information
}

func init() {
	templateCmd.AddCommand(templateListCmd)

	templateListCmd.Flags().StringP("repo", "r", "*", "repositories to list templates from")
	templateListCmd.Flags().StringP("revision", "b", "*", "revisions (branches/tags) to list templates from") //TODO: b is ambiguous here (can also be tag)
}
