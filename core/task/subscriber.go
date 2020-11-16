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


package task

import (
    "sync"
    
    pb "github.com/AliceO2Group/Control/core/protos"
)

type Subscription interface {
	Unsubscribe()
	Err() <-chan error
}

type EventFeed struct {
    mu sync.RWMutex
    streams []chan *pb.Event
}

func (e *EventFeed) Send(data *pb.Event) {
    e.mu.RLock()
    defer e.mu.RUnlock()
    for _, stream := range e.streams {
        stream <- data
    }
}

func (e *EventFeed) Subscribe(stream chan *pb.Event) Subscription {
    e.mu.Lock()
    defer e.mu.Unlock()
    e.streams = append(e.streams, stream)
    return &sub{
        feed: e,
        channelIndex: len(e.streams)-1,
        channel: stream,
        err: make(chan error, 1),
    }
}

// removes an channel at index i efficiently as order does not
// matter for listeners we keep track of.
func (e *EventFeed) remove(i int) {
    e.mu.Lock()
    close(e.streams[i])
    e.streams[i] = e.streams[len(e.streams)-1]
    e.streams = e.streams[:len(e.streams)-1]
    e.mu.Unlock()
}

func NewEventFeed() *EventFeed {
    return &EventFeed{
        streams: make([]chan *pb.Event, 0),
    }
}

type sub struct {
    feed         *EventFeed
    channelIndex int
    channel      chan *pb.Event
    once         sync.Once
    err          chan error
}

func (s *sub) Unsubscribe() {
    s.once.Do(func() {
        s.feed.remove(s.channelIndex)
        close(s.err)
    })
}

func (s *sub) Err() <-chan error {
    return s.err
}
