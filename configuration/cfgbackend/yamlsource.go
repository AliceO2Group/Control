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

package cfgbackend

import (
	"errors"
	"fmt"
	"io/ioutil"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type YamlSource struct {
	uri  string
	data Map
}

func newYamlSource(uri string) (yc *YamlSource, err error) {
	yc = &YamlSource{
		uri:  uri,
		data: nil,
	}
	err = yc.refresh()

	return
}

func (yc *YamlSource) refresh() (err error) {
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
	if item.Type() != IT_Map {
		yc.data = nil
		err = errors.New("bad configuration file format, top level item should be a map")
		return
	}
	yc.data = item.Map()
	return
}

func (yc *YamlSource) flush() (err error) {
	yamlFile, err := yaml.Marshal(yc.data)
	if err != nil {
		return
	}

	err = ioutil.WriteFile(pathForUri(yc.uri), yamlFile, 0644)
	return
}

func (yc *YamlSource) Get(key string) (value string, err error) {
	err = yc.refresh()
	if err != nil {
		return
	}
	requestKey := yamlFormatKey(key)
	keysPath := strings.Split(requestKey, "/")

	currentMap := yc.data
	for i, k := range keysPath {
		currentKey := k
		arrayIndex := -1
		isArray := regexp.MustCompile("\\w+\\[\\d+\\]")
		if isArray.MatchString(k) {
			// we have an array index on our hands
			split := strings.Split(k, "[")
			currentKey = split[0]
			arrayIndex, err = strconv.Atoi(strings.Trim(split[1], "[] "))
			if err != nil {
				return
			}
		}
		if currentMap[currentKey] == nil {
			err = errors.New(fmt.Sprintf("no value for key %s", key))
			return
		} else {
			it := currentMap[currentKey]
			if i == len(keysPath)-1 { // last iteration, so we should have a string for this key
				if it.Type() == IT_Value {
					value = it.Value()
					break
				} else if it.Type() == IT_Array {
					if arrayIndex > -1 {
						value = it.Array()[arrayIndex].Value()
						break
					} else {
						err = errors.New(fmt.Sprintf("found array at key %s but string was expected", key))
						return
					}
				} else {
					err = errors.New(fmt.Sprintf("found map at key %s but string was expected", key))
					return
				}
			} else {
				if it.Type() == IT_Map {
					currentMap = it.Map()
				} else if it.Type() == IT_Array {
					if arrayIndex > -1 {
						currentMap = it.Array()[arrayIndex].Map()
					} else {
						err = errors.New(fmt.Sprintf("found array at key %s but map was expected", key))
						return
					}
				} else {
					err = errors.New(fmt.Sprintf("found string at key %s but map was expected", key))
					return
				}
			}
		}
	}

	return
}

func (yc *YamlSource) GetKeysByPrefix(keyPrefix string) (keys []string, err error) {
	var recursive Item
	recursive, err = yc.GetRecursive(keyPrefix)
	if err != nil {
		return
	}

	keys = []string{}
	if recursive.Type() != IT_Map {
		err = errors.New("cannot get keys of non-map type")
		return
	}

	// we collect all the keys recursively and mark nodes with "/" at the end, just as the consul backend does
	var collectKeys func(Item, string)
	collectKeys = func(item Item, path string) {
		if item.Type() == IT_Map {
			for k, v := range item.Map() {
				newPath := yamlFormatKey(path) + "/" + k
				// i'm not sure what to do with arrays, but the consul backend does not support them anyway
				if v.Type() == IT_Map {
					keys = append(keys, newPath+"/")
				} else {
					keys = append(keys, newPath)
				}
				collectKeys(v, newPath)
			}
		}
	}
	collectKeys(recursive, keyPrefix)

	return
}

func (yc *YamlSource) IsDir(key string) (isDir bool) {
	recursive, err := yc.GetRecursive(key)
	if err != nil {
		return false
	}
	return recursive.Type() == IT_Map
}

func (yc *YamlSource) GetRecursive(key string) (value Item, err error) {
	err = yc.refresh()
	if err != nil {
		return
	}
	requestKey := yamlFormatKey(key)
	if len(requestKey) == 0 {
		// we request the root
		value = yc.data
		return
	}
	keysPath := strings.Split(requestKey, "/")
	currentMap := yc.data
	for i, k := range keysPath {
		currentKey := k
		arrayIndex := -1
		isArray := regexp.MustCompile("\\w+\\[\\d+\\]")
		if isArray.MatchString(k) {
			// we have an array index on our hands
			split := strings.Split(k, "[")
			currentKey = split[0]
			arrayIndex, err = strconv.Atoi(strings.Trim(split[1], "[] "))
			if err != nil {
				return
			}
		}
		if currentMap[currentKey] == nil {
			err = errors.New(fmt.Sprintf("no value for key '%s'", key))
			return
		} else {
			it := currentMap[currentKey]
			if i == len(keysPath)-1 { // last iteration
				if it.Type() == IT_Value {
					value = it
					return
				} else if it.Type() == IT_Array {
					if arrayIndex > -1 {
						value = it.Array()[arrayIndex]
						return
					} else {
						value = it
						return
					}
				} else {
					value = it
					return
				}
			} else {
				if it.Type() == IT_Map {
					currentMap = it.Map()
				} else if it.Type() == IT_Array {
					if arrayIndex > -1 {
						item := it.Array()[arrayIndex]
						if item.Type() == IT_Map {
							currentMap = item.Map()
						} else if item.Type() == IT_Array {
							err = errors.New(fmt.Sprintf("found array at key %s but map was expected", key))
							return
						} else {
							err = errors.New(fmt.Sprintf("found string at key %s but map was expected", key))
							return
						}
					}
				} else {
					err = errors.New(fmt.Sprintf("found string at key %s but map was expected", key))
					return
				}
			}
		}
	}
	value = currentMap
	return
}

func (yc *YamlSource) GetRecursiveYaml(key string) (value []byte, err error) {
	var item Item
	item, err = yc.GetRecursive(key)
	if err != nil {
		return
	}
	value, err = yaml.Marshal(item)
	return
}

func (yc *YamlSource) Put(key string, value string) (err error) {
	err = yc.refresh()
	if err != nil {
		return
	}
	requestKey := yamlFormatKey(key)
	keysPath := strings.Split(requestKey, "/")
	currentMap := yc.data
	for i, k := range keysPath {
		currentKey := k
		arrayIndex := -1
		isArray := regexp.MustCompile("\\w+\\[\\d+\\]")
		if isArray.MatchString(k) {
			// we have an array index on our hands
			split := strings.Split(k, "[")
			currentKey = split[0]
			arrayIndex, err = strconv.Atoi(strings.Trim(split[1], "[] "))
			if err != nil {
				return
			}
		}

		if i == len(keysPath)-1 {
			if arrayIndex > -1 {
				if currentMap[currentKey] == nil || currentMap[currentKey].Type() != IT_Array {
					err = errors.New(fmt.Sprintf("found nil at key %s but array was expected", key))
					return
				}
				ar := currentMap[currentKey].Array()
				ar[arrayIndex] = String(value)
				currentMap[currentKey] = ar
			} else {
				currentMap[currentKey] = String(value)
				break
			}
		} else {
			if currentMap[currentKey] == nil {
				if arrayIndex > -1 {
					err = errors.New(fmt.Sprintf("found nil at key %s but array was expected", key))
					return
				}
				currentMap[currentKey] = make(Map)
			}

			it := currentMap[currentKey]
			if it.Type() == IT_Map {
				currentMap = it.Map()
			} else if it.Type() == IT_Array {
				if arrayIndex > -1 {
					item := it.Array()[arrayIndex]
					if item.Type() == IT_Map {
						currentMap = item.Map()
					} else if item.Type() == IT_Array {
						err = errors.New(fmt.Sprintf("found array at key %s but map was expected", key))
						return
					} else {
						err = errors.New(fmt.Sprintf("found string at key %s but map was expected", key))
						return
					}
				}
			} else {
				err = errors.New(fmt.Sprintf("found string at key %s but map was expected", key))
				return
			}
		}
	}
	err = yc.flush()
	return
}

func (yc *YamlSource) PutRecursive(key string, value Item) (err error) {
	err = yc.refresh()
	if err != nil {
		return
	}
	requestKey := yamlFormatKey(key)
	keysPath := strings.Split(requestKey, "/")
	currentMap := yc.data
	for i, k := range keysPath {
		currentKey := k
		arrayIndex := -1
		isArray := regexp.MustCompile("\\w+\\[\\d+\\]")
		if isArray.MatchString(k) {
			// we have an array index on our hands
			split := strings.Split(k, "[")
			currentKey = split[0]
			arrayIndex, err = strconv.Atoi(strings.Trim(split[1], "[] "))
			if err != nil {
				return
			}
		}

		if i == len(keysPath)-1 {
			if arrayIndex > -1 {
				if currentMap[currentKey] == nil || currentMap[currentKey].Type() != IT_Array {
					err = errors.New(fmt.Sprintf("found nil at key %s but array was expected", key))
					return
				}
				ar := currentMap[currentKey].Array()
				ar[arrayIndex] = value.DeepCopy()
				currentMap[currentKey] = ar
			} else {
				currentMap[currentKey] = value.DeepCopy()
				break
			}
		} else {
			if currentMap[currentKey] == nil {
				if arrayIndex > -1 {
					err = errors.New(fmt.Sprintf("found nil at key %s but array was expected", key))
					return
				}
				currentMap[currentKey] = make(Map)
			}

			it := currentMap[currentKey]
			if it.Type() == IT_Map {
				currentMap = it.Map()
			} else if it.Type() == IT_Array {
				if arrayIndex > -1 {
					item := it.Array()[arrayIndex]
					if item.Type() == IT_Map {
						currentMap = item.Map()
					} else if item.Type() == IT_Array {
						err = errors.New(fmt.Sprintf("found array at key %s but map was expected", key))
						return
					} else {
						err = errors.New(fmt.Sprintf("found string at key %s but map was expected", key))
						return
					}
				}
			} else {
				err = errors.New(fmt.Sprintf("found string at key %s but map was expected", key))
				return
			}
		}
	}
	err = yc.flush()
	return
}

func (yc *YamlSource) PutRecursiveYaml(key string, value []byte) (err error) {
	var (
		raw    interface{}
		cooked Item
	)
	err = yaml.Unmarshal(value, &raw)
	if err != nil {
		return
	}

	cooked, err = intfToItem(raw)
	if err != nil {
		return
	}

	err = yc.PutRecursive(key, cooked)
	return
}

func (yc *YamlSource) Exists(key string) (exists bool, err error) {
	err = yc.refresh()
	if err != nil {
		return
	}
	requestKey := yamlFormatKey(key)
	keysPath := strings.Split(requestKey, "/")
	currentMap := yc.data
	for i, k := range keysPath {
		currentKey := k
		arrayIndex := -1
		isArray := regexp.MustCompile("\\w+\\[\\d+\\]")
		if isArray.MatchString(k) {
			// we have an array index on our hands
			split := strings.Split(k, "[")
			currentKey = split[0]
			arrayIndex, err = strconv.Atoi(strings.Trim(split[1], "[] "))
			if err != nil {
				return
			}
		}
		if currentMap[currentKey] == nil {
			exists = false
			return
		} else {
			it := currentMap[currentKey]
			if i == len(keysPath)-1 { // last iteration, so we should have a string for this key
				if arrayIndex > -1 {
					exists = it.Array()[arrayIndex] != nil
				}
				exists = it != nil
				return
			} else {
				if it.Type() == IT_Map {
					currentMap = it.Map()
				} else if it.Type() == IT_Array {
					if arrayIndex > -1 && arrayIndex < len(it.Array()) {
						currentMap = it.Array()[arrayIndex].Map()
					} else {
						exists = false
						return
					}
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
	// Trim leading and trailing slashes
	consulKey = strings.Trim(key, "/")
	return
}

func pathForUri(uri string) string {
	return strings.TrimPrefix(uri, "file://")
}
