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
	"reflect"
	"errors"
	"fmt"
	"strconv"
)

type Item interface {
	IsValue() bool
	IsMap() bool
	Value() string
	Map() Map
}
type Map	 map[string]Item
type String	 string

func (m Map) IsValue() bool {
	return false
}
func (m Map) IsMap() bool {
	return true
}
func (m Map) Value() string {
	return ""
}
func (m Map) Map() Map {
	return m
}

func intfToItem(intf interface{}) (item Item, err error) {
	v := reflect.ValueOf(intf)
	if v.Kind() == reflect.Map {
		m := make(Map)
		for _, key := range v.MapKeys() {
			val := v.MapIndex(key)
			m[key.Interface().(string)], err = intfToItem(val.Interface())
			if err != nil {
				return
			}
		}
		item = m
		return
	} else if v.Kind() == reflect.String {
		item = String(v.String())
		return
	} else if v.Kind() == reflect.Int {
		item = String(strconv.FormatInt(int64(v.Int()), 10))
		return
	} else if v.Kind() == reflect.Float64 {
		item = String(strconv.FormatFloat(v.Float(), 'f', -1, 64))
		return
	} else if v.Kind() == reflect.Float32 {
		item = String(strconv.FormatFloat(v.Float(), 'f', -1, 32))
		return
	} else if v.Kind() == reflect.Bool {
		item = String(strconv.FormatBool(v.Bool()))
		return
	} else {
		fmt.Println(v.Kind())
		fmt.Println(v.String())
		fmt.Println(v)

		err = errors.New("bad configuration format, item is neither String nor Map")
		return
	}
	return
}

func (s String) IsValue() bool {
	return true
}
func (s String) IsMap() bool {
	return false
}
func (s String) Value() string {
	return string(s)
}
func (s String) Map() Map {
	return nil
}
