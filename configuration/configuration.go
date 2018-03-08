/*
 * === This file is part of octl <https://github.com/teo/octl> ===
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

package configuration

import (
	"strings"
	"errors"
)

type Configuration interface {
	Get(string) (string, error)
	GetRecursive(string) (Map, error)
	Put(string, string) error
	Exists(string) (bool, error)
}

func NewConfiguration(uri string) (configuration Configuration, err error) {
	if strings.HasPrefix(uri, "consul://") {
		configuration, err = newConsulConfiguration(uri)
		return
	} else if strings.HasPrefix(uri, "file://") && strings.HasSuffix(uri, ".yaml") {
		configuration, err = newYamlConfiguration(uri)
		return
	}

	err = errors.New("bad URI for configuration source")
	return
}
