/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2020-2021 CERN and copyright holders of ALICE O².
 * Author: Teo Mrnjavac <teo.mrnjavac@cern.ch>
 * Author: George Raduta <george.raduta@cern.ch>
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

// Package componentcfg provides component configuration management functionality,
// including query handling and template processing for O² components.
package componentcfg

import (
	apricotpb "github.com/AliceO2Group/Control/apricot/protos"
	"strings"
)

const (
	ConfigComponentsPath = "o2/components/"
)

const (
	SEPARATOR      = "/"
	SEPARATOR_RUNE = '/'
)

// Checks whether the input string is a valid component name
func IsInputValidComponentName(input string) bool {
	return !strings.Contains(input, "/")
}

// Checks whether the input string is a valid entry name
func IsInputValidEntryName(input string) bool {
	return !strings.HasSuffix(input, "resolve")
}

// keys must be a slice containing all the full keys in /o2/components
func GetComponentsMapFromKeysList(keys []string) map[string]bool {
	var componentsMap = make(map[string]bool)
	for _, value := range keys {
		if !strings.HasPrefix(value, ConfigComponentsPath) {
			continue
		}
		value := strings.TrimPrefix(value, ConfigComponentsPath)
		component := strings.Split(value, "/")[0]
		if len(component) > 0 {
			componentsMap[component] = true
		}
	}
	return componentsMap
}

func GetEntriesMapOfComponentFromKeysList(component string, runtype apricotpb.RunType, rolename string, keys []string) map[string]bool {
	var entriesMap = make(map[string]bool)
	runtypeString, ok := apricotpb.RunType_name[int32(runtype)]
	if !ok {
		runtypeString = "ANY" // this should never happen
	}
	for _, value := range keys {
		value := strings.TrimPrefix(value, ConfigComponentsPath)
		parts := strings.Split(value, "/")
		if len(parts) < 4 {
			continue
		}
		if parts[0] == component &&
			parts[1] == runtypeString &&
			parts[2] == rolename {
			entriesMap[parts[3]] = true
		}
	}
	return entriesMap
}
