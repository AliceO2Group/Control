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
	"reflect"
	"strconv"
)

type ItemType int

const (
	IT_Value ItemType = iota
	IT_Map
	IT_Array
)

type Item interface {
	Type() ItemType
	Value() string
	Map() Map
	Array() Array
	DeepCopy() Item
}
type Map map[string]Item
type Array []Item
type String string

func (m Map) Type() ItemType {
	return IT_Map
}
func (m Map) Value() string {
	return ""
}
func (m Map) Map() Map {
	return m
}
func (m Map) Array() Array {
	return nil
}
func (m Map) DeepCopy() Item {
	item := make(Map)
	for k, v := range m {
		vCopy := v.DeepCopy()
		item[k] = vCopy
	}
	return item
}

func (m Array) Type() ItemType {
	return IT_Array
}
func (m Array) Value() string {
	return ""
}
func (m Array) Map() Map {
	return nil
}
func (m Array) Array() Array {
	return m
}
func (m Array) DeepCopy() Item {
	item := make(Array, len(m))
	for i, v := range m {
		vCopy := v.DeepCopy()
		item[i] = vCopy
	}
	return item
}

func (s String) Type() ItemType {
	return IT_Value
}
func (s String) Value() string {
	return string(s)
}
func (s String) Map() Map {
	return nil
}
func (s String) Array() Array {
	return nil
}
func (s String) DeepCopy() Item {
	item := String(s.Value())
	return item
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
	} else if v.Kind() == reflect.Slice {
		a := make(Array, v.Len())
		for i := 0; i < v.Len(); i++ {
			val := v.Index(i)
			a[i], err = intfToItem(val.Interface())
			if err != nil {
				return
			}
		}
		item = a
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
