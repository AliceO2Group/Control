/*
 * === This file is part of octl <https://github.com/teo/octl> ===
 *
 * Copyright 2018 CERN and copyright holders of ALICE O².
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

//go:generate protoc --go_out=plugins=grpc:. protos/octlserver.proto
package core

import (
	"golang.org/x/net/context"
    "google.golang.org/grpc"

	"github.com/teo/octl/scheduler/core/protos"
	"google.golang.org/grpc/reflection"
	"github.com/mesos/mesos-go/api/v1/lib/extras/store"
	"runtime"
	"fmt"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/codes"
	"github.com/looplab/fsm"
	"github.com/pborman/uuid"
)


func NewServer(state *internalState, fidStore store.Singleton) *grpc.Server {
	s := grpc.NewServer()
	pb.RegisterOctlServer(s, &RpcServer{
		state: state,
		fidStore: fidStore,
	})
	// Register reflection service on gRPC server.
	reflection.Register(s)
	return s
}

func (m *RpcServer) logMethod() {
	if !m.state.config.verbose {
		return
	}
	pc, _, _, ok := runtime.Caller(0)
	if !ok {
		return
	}
	fun := runtime.FuncForPC(pc)
	if fun == nil {
		return
	}
	log.WithPrefix("rpcserver").
		WithField("method", fun.Name()).
		Debug("handling RPC request")
}

// Implements interface pb.OctlServer
type RpcServer struct {
	state       *internalState
	fidStore    store.Singleton
}

func (*RpcServer) TrackStatus(*pb.StatusRequest, pb.Octl_TrackStatusServer) error {
	panic("implement me")
}

func (m *RpcServer) GetFrameworkInfo(context.Context, *pb.GetFrameworkInfoRequest) (*pb.GetFrameworkInfoReply, error) {
	m.logMethod()
	m.state.RLock()
	defer m.state.RUnlock()
	r := &pb.GetFrameworkInfoReply{
		FrameworkId:        store.GetIgnoreErrors(m.fidStore)(),
		EnvironmentsCount:  int32(len(m.state.environments.Ids())),
		RolesCount:         int32(m.state.roleman.RoleCount()),
		State:              m.state.sm.Current(),
	}
	return r, nil
}

func (*RpcServer) Teardown(context.Context, *pb.TeardownRequest) (*pb.TeardownReply, error) {
	panic("implement me")
}

func (m *RpcServer) GetEnvironments(context.Context, *pb.GetEnvironmentsRequest) (*pb.GetEnvironmentsReply, error) {
	m.logMethod()
	m.state.RLock()
	defer m.state.RUnlock()
	r := &pb.GetEnvironmentsReply{
		FrameworkId: store.GetIgnoreErrors(m.fidStore)(),
		Environments: make([]*pb.EnvironmentInfo, 0, 0),
	}
	for _, id := range m.state.environments.Ids() {
		env, err := m.state.environments.Environment(id)
		if err != nil {
			log.WithPrefix("rpcserver").
				WithField("error", err).
				WithField("envId", id.String()).
				Error("cannot get environment")
			continue
		}
		e := &pb.EnvironmentInfo{
			Id:             env.Id().String(),
			CreatedWhen:    env.CreatedWhen().String(),
			State:          env.CurrentState(),
			Roles:          env.Roles(),
		}
		r.Environments = append(r.Environments, e)
	}
	return r, nil
}

func (m *RpcServer) NewEnvironment(cxt context.Context, request *pb.NewEnvironmentRequest) (*pb.NewEnvironmentReply, error) {
	m.logMethod()
	// NEW_ENVIRONMENT transition
	// The following should
	// 1) Create a new value of type Environment struct
	// 2) Build the topology and ask Mesos to run all the processes
	// 3) Acquire the status of the processes to ascertain that they are indeed running and
	//    in their STANDBY state
	// 4) Execute the CONFIGURE transition on all the processes, and recheck their status to
	//    make sure they are now successfully in CONFIGURED
	// 5) Report back here with the new environment id and error code, if needed.

	if m.state.sm.Cannot("NEW_ENVIRONMENT") {
		msg := fmt.Sprintf("NEW_ENVIRONMENT transition impossible, current state: %s",
			m.state.sm.Current())
		return nil, status.Error(codes.Internal, msg)
	}
	err := m.state.sm.Event("NEW_ENVIRONMENT") //Async until Transition call
	defer m.state.sm.Transition()

	if _, ok := err.(fsm.NoTransitionError); !ok && err != nil {
		return nil, status.Newf(codes.Internal, "cannot create new environment: %s", err.Error()).Err()
	}

	// Create new Environment instance with some roles, we get back a UUID
	id, err := m.state.environments.CreateEnvironment(request.Roles)
	if err != nil {
		return nil, status.Newf(codes.Internal, "cannot create new environment: %s", err.Error()).Err()
	}

	newEnv, err := m.state.environments.Environment(id)
	if err != nil {
		return nil, status.Newf(codes.Internal, "cannot get newly created environment: %s", err.Error()).Err()
	}

	r := &pb.NewEnvironmentReply{
		Id: id.String(),
		State: newEnv.Sm.Current(),
	}

	return r, nil
}

func (m *RpcServer) GetEnvironment(cxt context.Context, req *pb.GetEnvironmentRequest) (*pb.GetEnvironmentReply, error) {
	m.logMethod()
	m.state.RLock()
	defer m.state.RUnlock()

	if req == nil || len(req.Id) == 0 {
		return nil, status.New(codes.InvalidArgument, "received nil request").Err()
	}

	env, err := m.state.environments.Environment(uuid.Parse(req.Id))
	if err != nil {
		return nil, status.Newf(codes.NotFound, "environment not found: %s", err.Error()).Err()
	}

	r := &pb.GetEnvironmentReply{
		Environment: &pb.EnvironmentInfo{
			Id: env.Id().String(),
			CreatedWhen: env.CreatedWhen().String(),
			State: env.CurrentState(),
			Roles: env.Roles(),
		},
	}
	return r, nil
}

func (*RpcServer) ControlEnvironment(context.Context, *pb.ControlEnvironmentRequest) (*pb.ControlEnvironmentReply, error) {
	panic("implement me")
}

func (*RpcServer) ModifyEnvironment(context.Context, *pb.ModifyEnvironmentRequest) (*pb.ModifyEnvironmentReply, error) {
	panic("implement me")
}

func (*RpcServer) DestroyEnvironment(context.Context, *pb.DestroyEnvironmentRequest) (*pb.DestroyEnvironmentReply, error) {
	panic("implement me")
}

func (m *RpcServer) GetRoles(context.Context, *pb.GetRolesRequest) (*pb.GetRolesReply, error) {
	m.logMethod()
	m.state.RLock()
	defer m.state.RUnlock()

	r := &pb.GetRolesReply{
		Roles: make([]*pb.RoleInfo, 0, 0),
	}

	for _, role := range m.state.roleman.GetRoles() {
		ri := &pb.RoleInfo{
			Locked: role.IsLocked(),
			Hostname: role.GetHostname(),
			Name: role.GetName(),
		}
		r.Roles = append(r.Roles, ri)
	}
	return r, nil
}
