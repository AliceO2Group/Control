/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2020 CERN and copyright holders of ALICE O².
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

var  (
	inputFullRegex = regexp.MustCompile(`^([a-zA-Z0-9-]+)(\/[a-z-A-Z0-9-]+){1}(\@[0-9]+)?$`)
)

func IsInputCompEntryTsValid(input string) bool {
	return inputFullRegex.MatchString(input)
}

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
func GetLatestTimestamp(keys []string, component string, entry string)(timestamp string, err error) {
	keyPrefix := ConfigComponentsPath + component + "/" + entry
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
	return strconv.FormatUint(maxTimeStamp, 10), nil
}

// Method to split component, entry and timestamp when being passed a key from consul
// e.g. of key o2/components/quality-control/cru-demo/12345678
func GetComponentEntryTimestampFromConsul(key string)(string, string, string) {
	key = strings.TrimPrefix(key, ConfigComponentsPath)
	key = strings.TrimPrefix(key, "/'")
	key = strings.TrimSuffix(key, "/")
	elements := strings.Split(key, "/")
	if len(elements) == 3 {
		return elements[0], elements[1], elements[2]
	} else {
		return elements[0], elements[1], "unversioned"
	}
}


func GetComponentsMapFromKeysList(keys []string) map[string]bool {
	var componentsMap = make(map[string]bool)
	for _,value := range keys {
		value := strings.TrimPrefix(value, ConfigComponentsPath)
		component := strings.Split(value, "/" )[0]
		componentsMap[component] = true
	}
	return componentsMap
}

func GetEntriesMapOfComponentFromKeysList(component string, keys []string) map[string]bool  {
	var entriesMap = make(map[string]bool)
	for _,value := range keys {
		value := strings.TrimPrefix(value, ConfigComponentsPath)
		parts := strings.Split(value, "/" )
		if parts[0] == component {
			entriesMap[parts[1]] = true
		}
	}
	return entriesMap
}
