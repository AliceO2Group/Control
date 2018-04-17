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

package configuration

import (
	"gopkg.in/yaml.v2"
	"strings"
	"io/ioutil"
	"errors"
	"fmt"
)

type YamlConfiguration struct {
	uri  string
	data Map
}

func newYamlConfiguration(uri string) (yc *YamlConfiguration, err error) {
	yc = &YamlConfiguration{
		uri: uri,
		data: nil,
	}
	err = yc.refresh()

	return
}

func (yc *YamlConfiguration) refresh() (err error) {
	yamlFile, err := ioutil.ReadFile(pathForUri(yc.uri))
	if err != nil {
		return
	}

	var intf interface{}
	err = yaml.Unmarshal(yamlFile, &intf)
	if err != nil {
		yc.data = nil
		return
	}
	item, err := intfToItem(intf)
	if err != nil {
		yc.data = nil
		return
	}
	if !item.IsMap() {
		yc.data = nil
		err = errors.New("bad configuration file format, top level item should be a map")
		return
	}
	yc.data = item.Map()
	return
}

func (yc *YamlConfiguration) flush() (err error) {
	yamlFile, err := yaml.Marshal(yc.data)
	if err != nil {
		return
	}

	err = ioutil.WriteFile(pathForUri(yc.uri), yamlFile, 0644)
	return
}

func (yc *YamlConfiguration) Get(key string) (value string, err error) {
	err = yc.refresh()
	if err != nil {
		return
	}
	requestKey := yamlFormatKey(key)
	keysPath := strings.Split(requestKey, "/")
	currentMap := yc.data
	for i, k := range keysPath {
		if currentMap[k] == nil {
			err = errors.New(fmt.Sprintf("no value for key %s", key))
			return
		} else {
			it := currentMap[k]
			if i == len(keysPath) - 1 { // last iteration, so we should have a string for this key
				if it.IsValue() {
					value = it.Value()
					break
				} else {
					err = errors.New(fmt.Sprintf("found map at key %s but string was expected", key))
					return
				}
			} else {
				if it.IsMap() {
					currentMap = it.Map()
				} else {
					err = errors.New(fmt.Sprintf("found string at key %s but map was expected", key))
					return
				}
			}
		}
	}
	return
}

func (yc *YamlConfiguration) GetRecursive(key string) (value Map, err error) {
	err = yc.refresh()
	if err != nil {
		return
	}
	requestKey := yamlFormatKey(key)
	keysPath := strings.Split(requestKey, "/")
	currentMap := yc.data
	for _, k := range keysPath {
		if currentMap[k] == nil {
			err = errors.New(fmt.Sprintf("no value for key %s", key))
			return
		} else {
			it := currentMap[k]
			if it.IsMap() {
				currentMap = it.Map()
			} else {
				err = errors.New(fmt.Sprintf("found string at key %s but map was expected", key))
				return
			}
		}
	}
	value = currentMap
	return
}

func (yc *YamlConfiguration) Put(key string, value string) (err error) {
	err = yc.refresh()
	if err != nil {
		return
	}
	requestKey := yamlFormatKey(key)
	keysPath := strings.Split(requestKey, "/")
	currentMap := yc.data
	for i, k := range keysPath {
		if i == len(keysPath) - 1 {
			currentMap[k] = String(value)
			break
		}
		if currentMap[k] == nil {
			currentMap[k] = make(Map)
		}

		it := currentMap[k]
		if it.IsMap() {
			currentMap = it.Map()
		} else {
			err = errors.New(fmt.Sprintf("found string at key %s but map was expected", key))
			return
		}
	}
	err = yc.flush()
	return
}

func (yc *YamlConfiguration) Exists(key string) (exists bool, err error) {
	err = yc.refresh()
	if err != nil {
		return
	}
	requestKey := yamlFormatKey(key)
	keysPath := strings.Split(requestKey, "/")
	currentMap := yc.data
	for i, k := range keysPath {
		if currentMap[k] == nil {
			exists = false
			return
		} else {
			it := currentMap[k]
			if i == len(keysPath) - 1 { // last iteration, so we should have a string for this key
				exists = it != nil
				return
			} else {
				if it.IsMap() {
					currentMap = it.Map()
				} else {
					exists = false
					return
				}
			}
		}
	}
	return
}

func yamlFormatKey(key string) (consulKey string) {
	// Trim leading slashes
	consulKey = strings.TrimLeft(key, "/")
	return
}

func pathForUri(uri string) string {
	return strings.TrimPrefix(uri, "file://")
}
