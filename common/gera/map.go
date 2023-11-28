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

// Package gera implements a hierarchical key-value store.
//
// A gera.Map uses a map[string]interface{} as backing store, and it can wrap other gera.Map instances.
// Values in child maps override any value provided by a gera.Map that's wrapped in the hierarchy.
package gera

import (
	"dario.cat/mergo"
)

type Map interface {
	Wrap(m Map) Map
	IsHierarchyRoot() bool
	HierarchyContains(m Map) bool
	Unwrap() Map

	Has(key string) bool
	Len() int

	Get(key string) (interface{}, bool)
	Set(key string, value interface{}) bool

	Flattened() (map[string]interface{}, error)
	WrappedAndFlattened(m Map) (map[string]interface{}, error)

	Raw() map[string]interface{}
}

func MakeMap() Map {
	return &WrapMap{
		theMap: make(map[string]interface{}),
		parent: nil,
	}
}

func MakeMapWithMap(fromMap map[string]interface{}) Map {
	myMap := &WrapMap{
		theMap: fromMap,
		parent: nil,
	}
	return myMap
}

func MakeMapWithMapCopy(fromMap map[string]interface{}) Map {
	newBackingMap := make(map[string]interface{})
	for k, v := range fromMap {
		newBackingMap[k] = v
	}

	return MakeMapWithMap(newBackingMap)
}

type WrapMap struct {
	theMap map[string]interface{}
	parent Map
}

func (w *WrapMap) UnmarshalYAML(unmarshal func(interface{}) error) error {
	m := make(map[string]interface{})
	err := unmarshal(&m)
	if err == nil {
		*w = WrapMap{
			theMap: m,
			parent: nil,
		}
	}
	return err
}

func (w *WrapMap) IsHierarchyRoot() bool {
	if w == nil || w.parent != nil {
		return false
	}
	return true
}

func (w *WrapMap) HierarchyContains(m Map) bool {
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
func (w *WrapMap) Wrap(m Map) Map {
	if w == nil {
		return nil
	}
	w.parent = m
	return w
}

// Unwraps this map from its parent.
// Returns a pointer to the former parent which was just unwrapped.
func (w *WrapMap) Unwrap() Map {
	if w == nil {
		return nil
	}
	p := w.parent
	w.parent = nil
	return p
}

func (w *WrapMap) Get(key string) (value interface{}, ok bool) {
	if w == nil || w.theMap == nil {
		return nil, false
	}
	if val, ok := w.theMap[key]; ok {
		return val, true
	}
	if w.parent != nil {
		return w.parent.Get(key)
	}
	return nil, false
}

func (w *WrapMap) Set(key string, value interface{}) (ok bool) {
	if w == nil || w.theMap == nil {
		return false
	}
	w.theMap[key] = value
	return true
}

func (w *WrapMap) Has(key string) bool {
	_, ok := w.Get(key)
	return ok
}

func (w *WrapMap) Len() int {
	if w == nil || w.theMap == nil {
		return 0
	}
	flattened, err := w.Flattened()
	if err != nil {
		return 0
	}
	return len(flattened)
}

func (w *WrapMap) Flattened() (map[string]interface{}, error) {
	if w == nil {
		return nil, nil
	}

	out := make(map[string]interface{})
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

func (w *WrapMap) WrappedAndFlattened(m Map) (map[string]interface{}, error) {
	if w == nil {
		return nil, nil
	}

	out := make(map[string]interface{})
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

func (w *WrapMap) Raw() map[string]interface{} {
	if w == nil {
		return nil
	}
	return w.theMap
}
