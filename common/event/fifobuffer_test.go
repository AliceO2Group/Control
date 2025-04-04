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
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("FifoBuffer", func() {
	When("Poping lower amount of items than inside of a buffer", func() {
		It("returns requested items", func() {
			buffer := NewFifoBuffer[int]()
			buffer.Push(1)
			buffer.Push(2)
			buffer.Push(3)

			Expect(buffer.Length()).To(Equal(3))

			results := buffer.PopMultiple(2)
			Expect(results).To(Equal([]int{1, 2}))
		})
	})

	When("Poping higher amount of items than inside of a buffer", func() {
		It("returns only available items", func() {
			buffer := NewFifoBuffer[int]()
			buffer.Push(1)

			results := buffer.PopMultiple(2)
			Expect(results).To(Equal([]int{1}))
		})
	})

	When("We use buffer with multiple goroutines pushing first (PopMultiple)", func() {
		It("is synchronised properly", func() {
			buffer := NewFifoBuffer[int]()
			channel := make(chan struct{})

			wg := sync.WaitGroup{}
			wg.Add(2)

			go func() {
				buffer.Push(1)
				channel <- struct{}{}
				wg.Done()
			}()

			go func() {
				<-channel
				result := buffer.PopMultiple(42)
				Expect(result, 1)
				wg.Done()
			}()

			wg.Wait()
		})
	})

	When("We use buffer with multiple goroutines popping first", func() {
		It("is synchronised properly", func() {
			buffer := NewFifoBuffer[int]()
			channel := make(chan struct{})

			wg := sync.WaitGroup{}
			wg.Add(2)

			go func() {
				// Pop is blocking is we have empty buffer, so we notify before
				channel <- struct{}{}
				result := buffer.PopMultiple(42)
				Expect(result, 1)
				wg.Done()
			}()

			go func() {
				<-channel
				buffer.Push(1)
				wg.Done()
			}()

			wg.Wait()
		})
	})

	When("We block FifoBuffer without data and call Release", func() {
		It("releases goroutines properly", func() {
			buffer := NewFifoBuffer[int]()
			everythingDone := sync.WaitGroup{}
			channel := make(chan struct{})

			everythingDone.Add(1)
			go func() {
				channel <- struct{}{}
				buffer.PopMultiple(42)
				everythingDone.Done()
			}()
			<-channel
			time.Sleep(100 * time.Millisecond)
			buffer.ReleaseGoroutines()
			everythingDone.Wait()
		})
	})
})
