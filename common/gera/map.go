/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2020-2024 CERN and copyright holders of ALICE O².
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
	"sync"

	"dario.cat/mergo"
)

type Map[K comparable, V any] interface {
	Wrap(m Map[K, V]) Map[K, V]
	IsHierarchyRoot() bool
	HierarchyContains(m Map[K, V]) bool
	Unwrap() Map[K, V]

	Has(key K) bool
	Len() int

	Get(key K) (V, bool)
	Set(key K, value V) bool
	Del(key K) bool

	Flattened() (map[K]V, error)
	FlattenedParent() (map[K]V, error)
	WrappedAndFlattened(m Map[K, V]) (map[K]V, error)

	Raw() map[K]V
	Copy() Map[K, V]
	RawCopy() map[K]V
}

func MakeMap[K comparable, V any]() *WrapMap[K, V] {
	return &WrapMap[K, V]{
		theMap: make(map[K]V),
		parent: nil,
	}
}

func MakeMapWithMap[K comparable, V any](fromMap map[K]V) *WrapMap[K, V] {
	myMap := &WrapMap[K, V]{
		theMap: fromMap,
		parent: nil,
	}
	return myMap
}

func FlattenStack[K comparable, V any](maps ...Map[K, V]) (flattened map[K]V, err error) {
	flattenedMap := MakeMap[K, V]()
	for _, oneMap := range maps {
		var localFlattened map[K]V
		localFlattened, err = oneMap.Flattened()
		if err != nil {
			return
		}
		flattenedMap = MakeMapWithMap(localFlattened).Wrap(flattenedMap).(*WrapMap[K, V])
	}

	flattened, err = flattenedMap.Flattened()
	return
}

func MakeMapWithMapCopy[K comparable, V any](fromMap map[K]V) *WrapMap[K, V] {
	newBackingMap := make(map[K]V)
	for k, v := range fromMap {
		newBackingMap[k] = v
	}

	return MakeMapWithMap(newBackingMap)
}

type WrapMap[K comparable, V any] struct {
	theMap map[K]V
	parent Map[K, V]

	unmarshalYAML func(w Map[K, V], unmarshal func(interface{}) error) error
	marshalYAML   func(w Map[K, V]) (interface{}, error)

	mu sync.RWMutex
}

func (w *WrapMap[K, V]) WithUnmarshalYAML(unmarshalYAML func(w Map[K, V], unmarshal func(interface{}) error) error) *WrapMap[K, V] {
	w.unmarshalYAML = unmarshalYAML
	return w
}

func (w *WrapMap[K, V]) WithMarshalYAML(marshalYAML func(w Map[K, V]) (interface{}, error)) *WrapMap[K, V] {
	w.marshalYAML = marshalYAML
	return w
}

func (w *WrapMap[K, V]) UnmarshalYAML(unmarshal func(interface{}) error) error {
	if w.unmarshalYAML != nil {
		return w.unmarshalYAML(w, unmarshal)
	}

	m := make(map[K]V)
	err := unmarshal(&m)
	if err == nil {
		*w = WrapMap[K, V]{
			theMap: m,
			parent: nil,
		}
	}
	return err
}

func (w *WrapMap[K, V]) MarshalYAML() (interface{}, error) {
	if w.marshalYAML != nil {
		return w.marshalYAML(w)
	}

	return w.theMap, nil
}

func (w *WrapMap[K, V]) IsHierarchyRoot() bool {
	if w == nil || w.parent != nil {
		return false
	}
	return true
}

func (w *WrapMap[K, V]) HierarchyContains(m Map[K, V]) bool {
	if w == nil || w.parent == nil {
		return false
	}
	if w == m {
		return true
	}
	if w.parent == m {
		return true
	}
	return w.parent.HierarchyContains(m)
}

// Wraps this map around the gera.Map m, which becomes the new parent.
// Returns a pointer to the composite map (i.e. to itself in its new state).
func (w *WrapMap[K, V]) Wrap(m Map[K, V]) Map[K, V] {
	if w == nil {
		return nil
	}
	w.parent = m
	return w
}

// Unwraps this map from its parent.
// Returns a pointer to the former parent which was just unwrapped.
func (w *WrapMap[K, V]) Unwrap() Map[K, V] {
	if w == nil {
		return nil
	}
	p := w.parent
	w.parent = nil
	return p
}

func (w *WrapMap[K, V]) Get(key K) (value V, ok bool) {
	if w == nil || w.theMap == nil {
		return value, false
	}

	w.mu.RLock()
	defer w.mu.RUnlock()

	if val, ok := w.theMap[key]; ok {
		return val, true
	}
	if w.parent != nil {
		return w.parent.Get(key)
	}
	return value, false
}

func (w *WrapMap[K, V]) Set(key K, value V) (ok bool) {
	if w == nil || w.theMap == nil {
		return false
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	w.theMap[key] = value
	return true
}

func (w *WrapMap[K, V]) Del(key K) (ok bool) {
	if w == nil || w.theMap == nil {
		return false
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	if _, exists := w.theMap[key]; exists {
		delete(w.theMap, key)
	}
	return true
}

func (w *WrapMap[K, V]) Has(key K) bool {
	_, ok := w.Get(key)
	return ok
}

func (w *WrapMap[K, V]) Len() int {
	if w == nil || w.theMap == nil {
		return 0
	}
	flattened, err := w.Flattened()
	if err != nil {
		return 0
	}
	return len(flattened)
}

func (w *WrapMap[K, V]) Flattened() (map[K]V, error) {
	if w == nil {
		return nil, nil
	}

	w.mu.RLock()
	defer w.mu.RUnlock()

	thisMapCopy := make(map[K]V)
	for k, v := range w.theMap {
		thisMapCopy[k] = v
	}
	if w.parent == nil {
		return thisMapCopy, nil
	}

	flattenedParent, err := w.parent.Flattened()
	if err != nil {
		return thisMapCopy, err
	}

	err = mergo.Merge(&flattenedParent, thisMapCopy, mergo.WithOverride)
	return flattenedParent, err
}

func (w *WrapMap[K, V]) FlattenedParent() (map[K]V, error) {
	if w == nil {
		return nil, nil
	}

	if w.parent == nil {
		return make(map[K]V), nil
	}

	return w.parent.Flattened()
}

func (w *WrapMap[K, V]) WrappedAndFlattened(m Map[K, V]) (map[K]V, error) {
	if w == nil {
		return nil, nil
	}

	w.mu.RLock()

	thisMapCopy := make(map[K]V)
	for k, v := range w.theMap {
		thisMapCopy[k] = v
	}

	w.mu.RUnlock()

	if m == nil {
		return thisMapCopy, nil
	}

	flattenedM, err := m.Flattened()
	if err != nil {
		return thisMapCopy, err
	}

	err = mergo.Merge(&flattenedM, thisMapCopy, mergo.WithOverride)
	return flattenedM, err
}

func (w *WrapMap[K, V]) Raw() map[K]V { // allows unmutexed access to map, can be unsafe!
	if w == nil {
		return nil
	}
	return w.theMap
}

func (w *WrapMap[K, V]) Copy() Map[K, V] {
	if w == nil {
		return nil
	}

	w.mu.RLock()
	defer w.mu.RUnlock()

	newMap := &WrapMap[K, V]{
		theMap: make(map[K]V, len(w.theMap)),
		parent: w.parent,
	}
	for k, v := range w.theMap {
		newMap.theMap[k] = v
	}
	return newMap
}

func (w *WrapMap[K, V]) RawCopy() map[K]V { // always safe
	if w == nil {
		return nil
	}
	return w.Copy().Raw()
}
