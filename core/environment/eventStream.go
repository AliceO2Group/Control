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

package environment

import (
	"sync"

	"github.com/AliceO2Group/Control/common/event"
	pb "github.com/AliceO2Group/Control/core/protos"
)

type Subscription interface {
	Unsubscribe()
	GetFeed() chan *pb.Event
	Send(event.Event)
	Err() <-chan error
}

type eventStream struct {
	stream chan *pb.Event
	mu     sync.Mutex
}

func SubscribeToStream(ch chan *pb.Event) Subscription {
	return &eventSub{
		feed: &eventStream{
			stream: ch,
		},
		err: make(chan error),
	}
}

func (e *eventStream) send(data *pb.Event) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.stream != nil {
		e.stream <- data
	}
}

func (e *eventStream) closeStream() {
	e.mu.Lock()
	defer e.mu.Unlock()
	close(e.stream)
	e.stream = nil
}

type eventSub struct {
	feed *eventStream
	once sync.Once
	err  chan error
}

func (s *eventSub) Unsubscribe() {
	s.once.Do(func() {
		s.feed.closeStream()
		close(s.err)
	})
}

func (s *eventSub) GetFeed() chan *pb.Event {
	return s.feed.stream
}

func (s *eventSub) Send(ev event.Event) {
	var data *pb.Event

	switch typedEvent := ev.(type) {
	case *event.RoleEvent:
		re := pb.Event_RoleEvent{
			RoleEvent: &pb.Ev_RoleEvent{
				Name:     typedEvent.GetName(),
				State:    typedEvent.GetState(),
				Status:   typedEvent.GetStatus(),
				RolePath: typedEvent.GetRolePath(),
			},
		}
		data = pb.WrapEvent(&re)
	case *event.TaskEvent:
		re := pb.Event_TaskEvent{
			TaskEvent: &pb.Ev_TaskEvent{
				Name:      typedEvent.GetName(),
				Taskid:    typedEvent.GetTaskID(),
				State:     typedEvent.GetState(),
				Status:    typedEvent.GetStatus(),
				Hostname:  typedEvent.GetHostname(),
				ClassName: typedEvent.GetClassName(),
			},
		}
		data = pb.WrapEvent(&re)
	case *event.EnvironmentEvent:
		re := pb.Event_EnvironmentEvent{
			EnvironmentEvent: &pb.Ev_EnvironmentEvent{
				EnvironmentId:    typedEvent.GetName(),
				State:            typedEvent.GetState(),
				CurrentRunNumber: typedEvent.GetRun(),
				Error:            typedEvent.GetError(),
				Message:          typedEvent.GetMessage(),
			},
		}
		data = pb.WrapEvent(&re)
	default:
		// noop
	}
	s.feed.send(data)
}

func (s *eventSub) Err() <-chan error {
	return s.err
}
