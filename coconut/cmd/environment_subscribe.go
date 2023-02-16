/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2023 CERN and copyright holders of ALICE O².
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

// environmentSubscribeCmd represents the environment list command
var environmentSubscribeCmd = &cobra.Command{
	Use:     "subscribe [environment id]",
	Aliases: []string{"sub"},
	Short:   "subscribe to the event stream, either global or for a specific environment",
	Long: `The environment subscribe command subscribes to an event stream sent by
the AliECS core. If no argument is passed, all events are received, otherwise, the
user can pass an environment ID to only receive the events pertaining to that
environment.
The environment subscribe command does not terminate unless the core terminates it,
or the user presses Ctrl+C.`,
	Run:  control.WrapCallStream(control.Subscribe),
	Args: cobra.MaximumNArgs(1),
}

func init() {
	environmentCmd.AddCommand(environmentSubscribeCmd)

	environmentSubscribeCmd.Flags().BoolP("task-events", "t", false, "include task events in output")
}
