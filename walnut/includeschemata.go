/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2020 CERN and copyright holders of ALICE O².
 * Author: Ayaan Zaidi <azaidi@cern.ch>
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

// Generate latest schemas from .json files
//go:generate go run includeschemata.go

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

// Reads all .json files in the current folder
// and encodes them as strings literals in schemata.go
func main() {

	schemasPath := "./schemata/"

	fs, err := ioutil.ReadDir(schemasPath)
	if err != nil {
		return
	}

	out, err := os.Create(filepath.Join(schemasPath, "schemata.go"))
	if err != nil {
		return
	}

	out.Write([]byte("package schemata \n\nconst (\n"))
	for _, f := range fs {
		if strings.HasSuffix(f.Name(), ".json") {
			out.Write([]byte(strings.TrimSuffix(f.Name(), ".json") + " = `\n"))
			f, _ := os.Open(filepath.Join(schemasPath, f.Name()))
			io.Copy(out, f)
			out.Write([]byte("`\n"))
		}
	}
	out.Write([]byte(")\n"))

}
