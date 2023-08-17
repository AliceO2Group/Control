/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2018-2023 CERN and copyright holders of ALICE O².
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

package taskclass

import (
	"sync"
	"time"
)

type Classes struct {
	mu       sync.RWMutex
	classMap map[string]*Class
}

func (c *Classes) Do(f func(classMap *map[string]*Class) error) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return f(&c.classMap)
}

func (c *Classes) Foreach(do func(string, *Class) bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for taskClassIdentifier, classPtr := range c.classMap {
		ok := do(taskClassIdentifier, classPtr)
		if !ok {
			return
		}
	}
}

func (c *Classes) getMap() map[string]*Class {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.classMap
}

func (c *Classes) DeleteKey(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.classMap, key)
}

func (c *Classes) DeleteKeys(keys []string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, k := range keys {
		delete(c.classMap, k)
	}
}

func (c *Classes) UpdateClass(key string, class *Class) {
	c.mu.Lock()
	defer c.mu.Unlock()

	class.UpdatedTimestamp = time.Now() // used for invalidating stale classcache entries
	if _, ok := c.classMap[key]; ok {   //contains
		*c.classMap[key] = *class // update
	} else {
		c.classMap[key] = class // else add class as new entry
	}
}

func (c *Classes) GetClass(key string) (class *Class, ok bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	class, ok = c.classMap[key]
	return
}

func NewClasses() *Classes {
	return &Classes{
		classMap: make(map[string]*Class),
	}
}
