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

// environmentControlCmd represents the environment list command
var environmentControlCmd = &cobra.Command{
	Use:     "control [environment id]",
	Aliases: []string{"ctl", "ct", "t"},
	Short:   "control the state machine of an environment",
	Long: `The environment control command triggers an event in the state 
machine of an existing O² environment. The event, if valid, starts a transition. 
The reached state is returned.

An event name must be passed via the mandatory event flag.
Valid events:
  CONFIGURE            RESET                EXIT
  START_ACTIVITY       STOP_ACTIVITY

Not all events are available in all states.`,
	Run:  control.WrapCall(control.ControlEnvironment),
	Args: cobra.ExactArgs(1),
}

func init() {
	environmentCmd.AddCommand(environmentControlCmd)

	environmentControlCmd.Flags().StringP("event", "e", "", "environment state machine event to trigger")
	environmentControlCmd.MarkFlagRequired("event")
}
