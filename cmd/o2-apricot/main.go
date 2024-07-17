/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2017-2021 CERN and copyright holders of ALICE O².
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
	"os"

	"github.com/AliceO2Group/Control/apricot"
	"github.com/AliceO2Group/Control/common/logger/infologger"
	log "github.com/sirupsen/logrus"
	"github.com/teo/logrus-prefixed-formatter"
)

func init() {
	log.SetFormatter(&prefixed.TextFormatter{
		FullTimestamp: true,
		SpacePadding:  20,
		PrefixPadding: 12,

		// Needed for colored stdout/stderr in GoLand, IntelliJ, etc.
		ForceColors:     true,
		ForceFormatting: true,
	})
	log.SetOutput(os.Stdout)
}

func main() {
	if err := apricot.NewConfig(); err != nil {
		log.Fatal(err)
	}

	ilHook, err := infologger.NewDirectHook("ECS", "apricot", nil)
	if err == nil {
		log.AddHook(ilHook)
	}

	if err := apricot.Run(); err != nil {
		log.Fatal(err)
	}
}
