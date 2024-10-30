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
	"github.com/AliceO2Group/Control/coconut/app"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"time"
)

// aboutCmd represents the about command
var aboutCmd = &cobra.Command{
	Use:     "about",
	Aliases: []string{},
	Short:   fmt.Sprintf("about %s", app.NAME),
	Long:    `The about command shows some basic information on this utility.`,
	Run: func(*cobra.Command, []string) {
		color.Set(color.FgHiWhite)
		fmt.Print(app.PRETTY_SHORTNAME + " *** ")
		color.Set(color.FgHiGreen)
		fmt.Printf("The ALICE %s\n", app.PRETTY_FULLNAME)
		color.Unset()
		fmt.Printf(`
version:         %s
config:          %s
endpoint:        %s
config_endpoint: %s
`,
			color.HiGreenString(viper.GetString("version")),
			color.HiGreenString(func() string {
				if len(viper.ConfigFileUsed()) > 0 {
					return viper.ConfigFileUsed()
				}
				return "builtin"
			}()),
			color.HiGreenString(viper.GetString("endpoint")),
			color.HiGreenString(viper.GetString("config_endpoint")))

		color.Set(color.FgHiBlue)
		fmt.Printf("\nCopyright 2017-%d CERN and the copyright holders of ALICE O².\n"+
			"This program is free software: you can redistribute it and/or modify \n"+
			"it under the terms of the GNU General Public License as published by \n"+
			"the Free Software Foundation, either version 3 of the License, or \n"+
			"(at your option) any later version.\n", time.Now().Year())
		color.Unset()

		fmt.Printf(`
bugs:            %s
code:            %s
maintainer:      %s
`,

			color.HiBlueString("https://alice.its.cern.ch/jira/projects/OCTRL"),
			color.HiBlueString("https://github.com/AliceO2Group/Control"),
			color.HiBlueString("Teo Mrnjavac, CERN EP-AID-DA <teo.m@cern.ch>"))
	},
}

func init() {
	rootCmd.AddCommand(aboutCmd)
}
