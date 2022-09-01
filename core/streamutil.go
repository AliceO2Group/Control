/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2020-2022 CERN and copyright holders of ALICE O².
 * Author: Miltiadis Alexis <miltiadis.alexis@cern.ch>
 *         Teo Mrnjavac <teo.mrnjavac@cern.ch>
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

package core

import (
	"sync"

	pb "github.com/AliceO2Group/Control/core/protos"
)

// SafeStreamsMap is a safe map where the key is usually a
// subscriptionID received from the grpc call and as a value
// a channel where get events from the environment
// and we stream them to the grpc client.
type SafeStreamsMap struct {
	mu      sync.RWMutex
	streams map[string]chan *pb.Event
}

func (s *SafeStreamsMap) add(id string, ch chan *pb.Event) {
	s.mu.Lock()
	s.streams[id] = ch
	s.mu.Unlock()
}

func (s *SafeStreamsMap) delete(id string) {
	s.mu.Lock()
	delete(s.streams, id)
	s.mu.Unlock()
}

func (s *SafeStreamsMap) GetChannel(id string) (ch chan *pb.Event, ok bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ch, ok = s.streams[id]
	return
}

func newSafeStreamsMap() SafeStreamsMap {
	return SafeStreamsMap{
		streams: make(map[string]chan *pb.Event),
	}
}
