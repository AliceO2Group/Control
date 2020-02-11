/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2018-2019 CERN and copyright holders of ALICE O².
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

package utils

import (
	"io"
	"os"
	"strings"
)

func IsDirEmpty(path string) (bool, error) {
	dir, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer dir.Close() //Read-only we don't care about the return value

	_, err = dir.Readdirnames(1)
	if err == io.EOF {
		return true, nil
	}

	return false, err
}

func EnsureTrailingSlash(path *string) {
	if !strings.HasSuffix(*path, "/") { //Add trailing '/'
		*path += "/"
	}
}

// helper func to package strings up nicely for protobuf
func ProtoString(s string) *string { return &s }

func RemoveTrailingSlash(path *string) {
	if strings.HasSuffix(*path, "/") { //Remove trailing '/'
		strings.TrimRight(*path, "/")
	}
}
