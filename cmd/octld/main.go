/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2017-2018 CERN and copyright holders of ALICE O².
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

package main

import (
	"flag"
	"os"

	"github.com/AliceO2Group/Control/core"
	log "github.com/sirupsen/logrus"
	"github.com/teo/logrus-prefixed-formatter"
)

func init() {
	log.SetFormatter(&prefixed.TextFormatter{
		FullTimestamp:   true,
		SpacePadding:    20,
		PrefixPadding:   12,

		// Needed for colored stdout/stderr in GoLand, IntelliJ, etc.
		ForceColors:     true,
		ForceFormatting: true,
	})
}

func main() {
	cfg := core.NewConfig()
	fs := flag.NewFlagSet("scheduler", flag.ExitOnError)
	cfg.AddFlags(fs)
	fs.Parse(os.Args[1:])

	if err := core.Run(cfg); err != nil {
		log.Fatal(err)
	}
}
