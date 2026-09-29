/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2026 CERN and copyright holders of ALICE O².
 * Author: Michal Tichak <michal.tichak@cern.ch>
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

package controller

import (
	"context"
	"errors"
	"net"
	"strings"
	"sync"

	"google.golang.org/grpc"

	pb "github.com/AliceO2Group/Control/control-operator/internal/controller/protos/generated"
)

// occServerStub is an in-process implementation of the OCC gRPC service defined
// in occ/protos/occ.proto. Embedding pb.UnimplementedOccServer satisfies the
// generated interface and keeps this type valid if the service gains methods.
// The state is guarded because the gRPC server answers on its own goroutines while
// the test reads and writes it.
type occServerStub struct {
	pb.UnimplementedOccServer

	returnError bool
	mu          sync.RWMutex
	state       string
}

func (s *occServerStub) setState(state string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state = state
}

func (s *occServerStub) getState() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state
}

func (s *occServerStub) GetState(ctx context.Context, req *pb.GetStateRequest) (*pb.GetStateReply, error) {
	if s.returnError == true {
		return nil, errors.New("returning error as requested by user")
	}
	return &pb.GetStateReply{State: s.getState()}, nil
}

// Transition answers with the state the requested event leads to, looked up in the
// same FSM table the controller used to pick the event, and moves the stub to that
// state so a following GetState reports it.
func (s *occServerStub) Transition(ctx context.Context, req *pb.TransitionRequest) (*pb.TransitionReply, error) {
	if s.returnError == true {
		return nil, errors.New("returning error as requested by user")
	}

	src, err := StateFromString(strings.ToLower(req.GetSrcState()))
	if err != nil {
		return nil, err
	}
	event, err := TransitionFromString(strings.ToLower(req.GetTransitionEvent()))
	if err != nil {
		return nil, err
	}

	for fromTo, transition := range fromStatesToTransition {
		if fromTo.from == src && transition == event {
			newState := fromTo.to.String()
			s.setState(newState)
			return &pb.TransitionReply{
				Trigger:         pb.StateChangeTrigger_EXECUTOR,
				State:           newState,
				TransitionEvent: req.GetTransitionEvent(),
				Ok:              true,
			}, nil
		}
	}

	// No rule for this event in this state, so the device stays where it is.
	return &pb.TransitionReply{
		Trigger:         pb.StateChangeTrigger_EXECUTOR,
		State:           src.String(),
		TransitionEvent: req.GetTransitionEvent(),
		Ok:              false,
	}, nil
}

func (s *occServerStub) StateStream(req *pb.StateStreamRequest, srv grpc.ServerStreamingServer[pb.StateStreamReply]) error {
	<-srv.Context().Done()
	return nil
}

func (s *occServerStub) EventStream(req *pb.EventStreamRequest, srv grpc.ServerStreamingServer[pb.EventStreamReply]) error {
	<-srv.Context().Done()
	return nil
}

// startOccServerStub serves a stub on a kernel-assigned loopback port and
// returns it together with that port and a stop function.
func startOccServerStub() (*occServerStub, int, func(), error) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, 0, nil, err
	}

	stub := &occServerStub{}
	srv := grpc.NewServer()
	pb.RegisterOccServer(srv, stub)

	go func() {
		_ = srv.Serve(lis)
	}()

	return stub, lis.Addr().(*net.TCPAddr).Port, srv.Stop, nil
}
