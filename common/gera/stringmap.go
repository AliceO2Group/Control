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

package gera

import (
	"github.com/imdario/mergo"
	"gopkg.in/yaml.v2"
)

type StringMap interface {
	yaml.Unmarshaler
	Wrap(m StringMap) StringMap
	IsHierarchyRoot() bool
	HierarchyContains(m StringMap) bool
	Unwrap() StringMap

	Has(key string) bool
	Len() int

	Get(key string) (string, bool)
	Set(key string, value string) bool

	Flattened() (map[string]string, error)
	FlattenedParent() (map[string]string, error)
	WrappedAndFlattened(m StringMap) (map[string]string, error)

    Raw() map[string]string
	Copy() StringMap
}

func MakeStringMap() *StringWrapMap {
	return &StringWrapMap{
		theMap: make(map[string]string),
		parent: nil,
	}
}

func MakeStringMapWithMap(fromMap map[string]string) *StringWrapMap {
	myMap := &StringWrapMap{
		theMap: fromMap,
		parent: nil,
	}
	return myMap
}

func FlattenStack(stringMaps ...StringMap) (flattened map[string]string, err error) {
	flattenedSM := MakeStringMap()
	for _, stringMap := range stringMaps {
		var localFlattened map[string]string
		localFlattened, err = stringMap.Flattened()
		if err != nil {
			return
		}
		flattenedSM = MakeStringMapWithMap(localFlattened).Wrap(flattenedSM).(*StringWrapMap)
	}

	flattened, err = flattenedSM.Flattened()
	return
}

func MakeStringMapWithMapCopy(fromMap map[string]string) *StringWrapMap {
	newBackingMap := make(map[string]string)
	for k, v := range fromMap {
		newBackingMap[k] = v
	}

	return MakeStringMapWithMap(newBackingMap)
}

type StringWrapMap struct {
	theMap map[string]string
	parent StringMap
}

func (w *StringWrapMap) UnmarshalYAML(unmarshal func(interface{}) error) error {
	m := make(map[string]string)
	err := unmarshal(&m)
	if err == nil {
		*w = StringWrapMap{
			theMap: m,
			parent: nil,
		}
	}
	return err
}

func (w *StringWrapMap) IsHierarchyRoot() bool {
	if w == nil || w.parent != nil {
		return false
	}
	return true
}

func (w *StringWrapMap) HierarchyContains(m StringMap) bool {
	if w == nil || w.parent == nil {
		return false
	}
	if w.parent == m {
		return true
	}
	return w.parent.HierarchyContains(m)
}

// Wraps this map around the gera.Map m, which becomes the new parent.
// Returns a pointer to the composite map (i.e. to itself in its new state).
func (w *StringWrapMap) Wrap(m StringMap) StringMap {
	if w == nil {
		return nil
	}
	w.parent = m
	return w
}

// Unwraps this map from its parent.
// Returns a pointer to the former parent which was just unwrapped.
func (w *StringWrapMap) Unwrap() StringMap {
	if w == nil {
		return nil
	}
	p := w.parent
	w.parent = nil
	return p
}

func (w *StringWrapMap) Get(key string) (value string, ok bool) {
	if w == nil || w.theMap == nil {
		return "", false
	}
	if val, ok := w.theMap[key]; ok {
		return val, true
	}
	if w.parent != nil {
		return w.parent.Get(key)
	}
	return "", false
}

func (w *StringWrapMap) Set(key string, value string) (ok bool) {
	if w == nil || w.theMap == nil {
		return false
	}
	w.theMap[key] = value
	return true
}

func (w *StringWrapMap) Has(key string) bool {
	_, ok := w.Get(key)
	return ok
}

func (w *StringWrapMap) Len() int {
	if w == nil || w.theMap == nil {
		return 0
	}
	flattened, err := w.Flattened()
	if err != nil {
		return 0
	}
	return len(flattened)
}

func (w *StringWrapMap) Flattened() (map[string]string, error) {
	if w == nil {
		return nil, nil
	}

	out := make(map[string]string)
	for k, v := range w.theMap {
		out[k] = v
	}
	if w.parent == nil {
		return out, nil
	}

	flattenedParent, err := w.parent.Flattened()
	if err != nil {
		return out, err
	}

	err = mergo.Merge(&out, flattenedParent)
	return out, err
}

func (w *StringWrapMap) FlattenedParent() (map[string]string, error) {
	if w == nil {
		return nil, nil
	}

	if w.parent == nil {
		return make(map[string]string), nil
	}

	return w.parent.Flattened()
}

func (w *StringWrapMap) WrappedAndFlattened(m StringMap) (map[string]string, error) {
	if w == nil {
		return nil, nil
	}

	out := make(map[string]string)
	for k, v := range w.theMap {
		out[k] = v
	}
	if m == nil {
		return out, nil
	}

	flattenedM, err := m.Flattened()
	if err != nil {
		return out, err
	}

	err = mergo.Merge(&out, flattenedM)
	return out, err
}

func (w *StringWrapMap) Raw() map[string]string {
	if w == nil {
		return nil
	}
	return w.theMap
}

func (w *StringWrapMap) Copy() StringMap {
	newMap := &StringWrapMap{
		theMap: make(map[string]string, len(w.theMap)),
		parent: w.parent,
	}
	for k,v := range w.theMap {
		newMap.theMap[k] = v
	}
	return newMap
}
