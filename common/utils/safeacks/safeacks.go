/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2020 CERN and copyright holders of ALICE O².
 * Author: Miltiadis Alexis <miltiadis.alexis@cern.ch>
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

package safeacks

import (
	"fmt"
	"sync"
)

// SafeAcks is a thread safe structure which allows to handle acknowledgment exchanges
// with N senders and one receiver. The first sender succeeds, then an error is returned for the subsequent ones.
// This way, subsequent senders are not stuck sending an acknowledgment when nothing expects it anymore.
// The signaling design is inspired by point 2 in https://go101.org/article/channel-closing.html
// SafeAcks can be used to acknowledge that an action happened to the task such as task KILLED.
// At the moment we utilize SafeAcks to acknowledge that all the requested tasks were killed by mesos (task/manager.go).
type SafeAcks struct {
	mu   sync.RWMutex
	acks map[string]ackChannels
}

type ackChannels struct {
	// the channel to send the ack to
	ack chan struct{}
	// the channel to close when acks are no longer expected
	stop chan struct{}
}

func (a *SafeAcks) deleteKey(key string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	delete(a.acks, key)
}

func (a *SafeAcks) ExpectsAck(key string) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()

	_, ok := a.acks[key]

	return ok
}

func (a *SafeAcks) RegisterAck(key string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if _, hasKey := a.acks[key]; hasKey {
		return fmt.Errorf("an acknowledgment was already registered for key '%s'", key)
	}

	a.acks[key] = ackChannels{make(chan struct{}), make(chan struct{})}
	return nil
}

func (a *SafeAcks) getValue(key string) (ackChannels ackChannels, ok bool) {
	a.mu.Lock()
	defer a.mu.Unlock()

	ackChannels, ok = a.acks[key]
	return
}

// TrySendAck checks if an acknowledgment is expected and if it is, it blocks until it is received.
// If an acknowledgment is not expected at the moment of the call (or already was received), nil is returned.
// If more than one goroutine attempts to send an acknowledgment before it is received, all but one goroutines will
// receive an error.
func (a *SafeAcks) TrySendAck(key string) error {
	channels, ok := a.getValue(key)
	if !ok {
		// fixme: perhaps we should return an error also here, but returning nil preserves the original behaviour
		//  of safeAcks before the refactoring. Perhaps the rest of the code assumes it's ok to blindly try sending
		//  an ack "just in case", so I would not change it lightly.
		return nil
	}

	select {
	case <-channels.stop:
		return fmt.Errorf("an acknowledgment has been already received for key '%s'", key)
	case channels.ack <- struct{}{}:
		return nil
	}
}

// TryReceiveAck blocks until an acknowledgment is received and then returns true.
// It will return false if an acknowledgment for a given key is not expected.
func (a *SafeAcks) TryReceiveAck(key string) bool {
	channels, ok := a.getValue(key)
	if !ok {
		return false
	}
	<-channels.ack
	close(channels.stop)
	a.deleteKey(key)
	return true
}

func NewAcks() *SafeAcks {
	return &SafeAcks{
		acks: make(map[string]ackChannels),
	}
}
