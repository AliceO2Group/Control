/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2020 CERN and copyright holders of ALICE O².
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

package componentcfg

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	ConfigComponentsPath = "o2/components/"
)

const(
	SEPARATOR = "/"
	SEPARATOR_RUNE = '/'
)

var  (
	//                                       component        /RUNTYPE          /rolename             /entry                @timestamp
	inputFullRegex = regexp.MustCompile(`^([a-zA-Z0-9-_]+)(\/[A-Z0-9-_]+){1}(\/[a-z-A-Z0-9-_]+){1}(\/[a-z-A-Z0-9-_]+){1}(\@[0-9]+)?$`)
)

type Path struct {
	Component string
	Flavor string
	Rolename string
	EntryKey string
	Timestamp string
}

func NewPath(path string) (p *Path, err error) {
	p = &Path{
		Component: "",
		Flavor:    "",
		Rolename:  "",
		EntryKey:  "",
		Timestamp:  "",
	}
	if IsInputCompEntryTsValid(path) {
		if strings.Contains(path, "@") {
			// coconut conf show component/FLAVOR/rolename/entry@timestamp
			arg := strings.Replace(path, "@", SEPARATOR, 1)
			params := strings.Split(arg, SEPARATOR)
			p.Component = params[0]
			p.Flavor = params[1]
			p.Rolename = params[2]
			p.EntryKey = params[3]
			p.Timestamp = params[4]
		} else if strings.Contains(path, SEPARATOR) {
			// coconut conf show component/FLAVOR/rolename/entry
			params := strings.Split(path, SEPARATOR)
			p.Component = params[0]
			p.Flavor = params[1]
			p.Rolename = params[2]
			p.EntryKey = params[3]
			// and if we received a raw path (with / instead of @ before timestamp):
			if len(params) > 4 && len(params[4]) > 0 {
				p.Timestamp = params[4]
			}
		}
	} else {
		err = errors.New("bad component configuration key format")
		return
	}

	return p, nil
}

func (p *Path) Path() string {
	path := p.WithoutTimestamp()
	if len(p.Timestamp) > 0 {
		return path + "@" + p.Timestamp
	}
	return path
}

func (p *Path) Raw() string {
	path := p.WithoutTimestamp()
	if len(p.Timestamp) > 0 {
		return path + SEPARATOR + p.Timestamp
	}
	return path
}

func (p *Path) WithoutTimestamp() string {
	return p.Component + SEPARATOR + p.Flavor + SEPARATOR + p.Rolename + SEPARATOR + p.EntryKey
}

func (p *Path) AbsoluteRaw() string {
	return ConfigComponentsPath + p.Raw()
}

func (p *Path) AbsoluteWithoutTimestamp() string {
	return ConfigComponentsPath + p.WithoutTimestamp()
}

func IsInputCompEntryTsValid(input string) bool {
	return inputFullRegex.MatchString(input)
}

// Checks whether the input string is a valid Consul path element on its own
func IsInputSingleValidWord(input string) bool {
	return !strings.Contains(input, "/") && !strings.Contains(input, "@")
}

// Method to parse a timestamp in the specified format
func GetTimestampInFormat(timestamp string, timeFormat string)(string, error){
	timeStampAsInt, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return "", errors.New("unable to identify timestamp")
	}
	tm := time.Unix(timeStampAsInt, 0)
	return  tm.Format(timeFormat), nil
}

// Method to return the latest timestamp for a specified component & entry
// If no keys were passed an error and code exit 3 will be returned
func GetLatestTimestamp(keys []string, p *Path)(timestamp string, err error) {
	keyPrefix := p.AbsoluteWithoutTimestamp()
	if len(keys) == 0 {
		err = errors.New("no keys found")
		return "", err
	}

	var maxTimeStamp uint64
	for _, key := range keys {
		componentTimestamp, err := strconv.ParseUint(strings.TrimPrefix(key, keyPrefix + "/"), 10, 64)
		if err == nil {
			if componentTimestamp > maxTimeStamp  {
				maxTimeStamp = componentTimestamp
			}
		}
	}
	if maxTimeStamp == 0 {
		return "", nil
	}
	return strconv.FormatUint(maxTimeStamp, 10), nil
}

// keys must be a slice containing all the full keys in /o2/components
func GetComponentsMapFromKeysList(keys []string) map[string]bool {
	var componentsMap = make(map[string]bool)
	for _,value := range keys {
		value := strings.TrimPrefix(value, ConfigComponentsPath)
		component := strings.Split(value, "/" )[0]
		componentsMap[component] = true
	}
	return componentsMap
}

func GetEntriesMapOfComponentFromKeysList(component string, runtype string, rolename string, keys []string) map[string]bool  {
	var entriesMap = make(map[string]bool)
	for _,value := range keys {
		value := strings.TrimPrefix(value, ConfigComponentsPath)
		parts := strings.Split(value, "/" )
		if parts[0] == component &&
			parts[1] == runtype &&
			parts[2] == rolename {
			entriesMap[parts[3]] = true
		}
	}
	return entriesMap
}
