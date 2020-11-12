/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2020 CERN and copyright holders of ALICE O².
 * Author: Miltiadis Alexis <miltiadis.alexis@cern.ch>
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
var environmentCreateAutoCmd = &cobra.Command{
	Use:   "auto",
	Aliases: []string{"a",},
	Short: "create auto run environment",
	Long: fmt.Sprintf(`The environment create auto command requests from %s the
creation of a new autorun environment.

The operation may or may not be successful depending on available resources and configuration.`, product.PRETTY_SHORTNAME),
	Run:   control.WrapCall(control.CreateAutoEnvironment),

}

func init() {
	environmentCreateCmd.AddCommand(environmentCreateAutoCmd)

	environmentCreateAutoCmd.Flags().StringP("workflow-template", "w", "", "workflow to be loaded in the new environment")
	environmentCreateAutoCmd.MarkFlagRequired("workflow-template")

	environmentCreateAutoCmd.Flags().StringP("extra-vars", "e", "", "values passed using key=value CSV or JSON syntax, interpreted as strings `key1=val1,key2=val2` or `{\"key1\": \"value1\", \"key2\": \"value2\"}`")
}
