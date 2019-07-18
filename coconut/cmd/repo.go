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

//go:generate protoc -I ../../core --gofast_out=plugins=grpc:.. protos/o2control.proto

package cmd

import (
	"github.com/spf13/cobra"
)

// repoCmd represents the repository command
var repoCmd = &cobra.Command{
	Use:   "repository",
	Aliases: []string{"repo"},
	Short: "modify or list git repos for task and workflow configuration",
	Long: `The repository command allows you to perform operations on the repos used for task and workflow configuration.`,
}

func init() {
	rootCmd.AddCommand(repoCmd)
}
