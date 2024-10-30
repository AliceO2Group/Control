/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2017 CERN and copyright holders of ALICE O².
 * Author: Teo Mrnjavac <teo.mrnjavac@cern.ch>
 *
 * Portions from examples in <https://github.com/mesos/mesos-go>:
 *     Copyright 2013-2015, Mesosphere, Inc.
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

package schedutil

import (
	"io/ioutil"
	"os"
)

func LoadCredentials(username string, password string) (result credentials, err error) {
	result = credentials{username, password}
	if result.password != "" {
		// this is the path to a file containing the password
		_, err = os.Stat(result.password)
		if err != nil {
			return
		}
		var f *os.File
		f, err = os.Open(result.password)
		if err != nil {
			return
		}
		defer f.Close()
		var bytes []byte
		bytes, err = ioutil.ReadAll(f)
		if err != nil {
			return
		}
		result.password = string(bytes)
	}
	return
}
