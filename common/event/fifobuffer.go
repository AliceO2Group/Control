/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2025 CERN and copyright holders of ALICE O².
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

package event

import (
	"math"
	"sync"
)

// This structure is meant to be used as a threadsafe FIFO with builtin waiting for new data
// in its Pop and PopMultiple functions. It is meant to be used with multiple goroutines, it is a
// waste of synchronization mechanisms if used synchronously.
type FifoBuffer[T any] struct {
	lock sync.Mutex
	cond sync.Cond

	buffer []T
}

func NewFifoBuffer[T any]() (result FifoBuffer[T]) {
	result = FifoBuffer[T]{
		lock: sync.Mutex{},
	}
	result.cond = *sync.NewCond(&result.lock)
	return
}

func (this *FifoBuffer[T]) Push(value T) {
	this.cond.L.Lock()
	this.buffer = append(this.buffer, value)
	this.cond.Signal()
	this.cond.L.Unlock()
}

// Blocks until it has some value in internal buffer
func (this *FifoBuffer[T]) PopMultiple(numberToPop uint) (result []T) {
	this.cond.L.Lock()
	defer this.cond.L.Unlock()

	for len(this.buffer) == 0 {
		this.cond.Wait()
		// this check is used when ReleaseGoroutines is called on waiting goroutine
		if len(this.buffer) == 0 {
			return
		}
	}

	result = make([]T, int(math.Min(float64(numberToPop), float64(len(this.buffer)))))
	copy(result, this.buffer[0:len(result)])
	this.buffer = this.buffer[len(result):]

	return
}

func (this *FifoBuffer[T]) Length() int {
	this.cond.L.Lock()
	defer this.cond.L.Unlock()
	return len(this.buffer)
}

func (this *FifoBuffer[T]) ReleaseGoroutines() {
	this.cond.L.Lock()
	this.cond.Broadcast()
	this.cond.L.Unlock()
}
