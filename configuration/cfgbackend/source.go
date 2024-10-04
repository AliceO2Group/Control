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

// Package configuration defines the Source interface as the
// main access point to O² Configuration backends.
// Consul and YAML backends are also provided.
package cfgbackend

import (
	"errors"
	"strings"
)

// Read-Only Source
type ROSource interface {
	Get(string) (string, error)
	GetKeysByPrefix(string) ([]string, error)
	GetRecursive(string) (Item, error)
	GetRecursiveYaml(string) ([]byte, error)
	Exists(string) (bool, error)
	IsDir(string) bool
}

type Source interface {
	ROSource
	Put(string, string) error
	PutRecursive(string, Item) error
	PutRecursiveYaml(string, []byte) error
}

func NewSource(uri string) (configuration Source, err error) {
	if strings.HasPrefix(uri, "consul://") {
		configuration, err = NewConsulSource(strings.TrimPrefix(uri, "consul://"))
		return
	} else if strings.HasPrefix(uri, "file://") &&
		(strings.HasSuffix(uri, ".yaml") || strings.HasSuffix(uri, ".json")) {
		configuration, err = newYamlSource(uri)
		return
	} else if strings.HasPrefix(uri, "mock://") {
		configuration, err = NewMockSource()
		return
	}

	err = errors.New("bad URI for configuration source")
	return
}
