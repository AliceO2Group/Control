/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2020 CERN and copyright holders of ALICE O².
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

// A Processor and ReposItory for COnfiguration Templates
package remote

import (
	"context"
	"runtime"

	apricotpb "github.com/AliceO2Group/Control/apricot/protos"
	"github.com/AliceO2Group/Control/common/logger"
	"github.com/AliceO2Group/Control/configuration"
	"github.com/AliceO2Group/Control/configuration/componentcfg"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

var log = logger.New(logrus.StandardLogger(),"apricot")

var(
	E_OK = status.New(codes.OK, "")
	E_CONFIGURATION_BACKEND_UNAVAILABLE = status.Errorf(codes.Internal, "configuration backend unavailable")
	E_BAD_INPUT = status.Errorf(codes.InvalidArgument, "bad request received")
)

type RpcServer struct {
	service configuration.Service
}

func NewServer(service configuration.Service) *grpc.Server {
	s := grpc.NewServer()
	apricotpb.RegisterApricotServer(s, &RpcServer{
		service: service,
	})
	// Register reflection service on gRPC server.
	reflection.Register(s)
	return s
}

func (m *RpcServer) NewRunNumber(_ context.Context, _ *apricotpb.Empty) (*apricotpb.RunNumberResponse, error) {
	if m == nil || m.service == nil {
		return nil, E_CONFIGURATION_BACKEND_UNAVAILABLE
	}
	m.logMethod()
	rn, err := m.service.NewRunNumber()
	return &apricotpb.RunNumberResponse{RunNumber: rn}, err
}

func (m *RpcServer) GetDefaults(_ context.Context, _ *apricotpb.Empty) (*apricotpb.StringMap, error) {
	if m == nil || m.service == nil {
		return nil, E_CONFIGURATION_BACKEND_UNAVAILABLE
	}
	m.logMethod()
	varStack := m.service.GetDefaults()
	return &apricotpb.StringMap{StringMap: varStack}, E_OK.Err()
}

func (m *RpcServer) GetVars(_ context.Context, _ *apricotpb.Empty) (*apricotpb.StringMap, error) {
	if m == nil || m.service == nil {
		return nil, E_CONFIGURATION_BACKEND_UNAVAILABLE
	}
	m.logMethod()
	varStack := m.service.GetVars()
	return &apricotpb.StringMap{StringMap: varStack}, E_OK.Err()
}

func (m *RpcServer) GetComponentConfiguration(_ context.Context, request *apricotpb.ComponentRequest) (*apricotpb.ComponentResponse, error) {
	if m == nil || m.service == nil {
		return nil, E_CONFIGURATION_BACKEND_UNAVAILABLE
	}
	m.logMethod()

	if request == nil {
		return nil, E_BAD_INPUT
	}

	var path *componentcfg.Query
	if rawPath := request.GetPath(); len(rawPath) > 0 {
		var err error
		path, err = componentcfg.NewQuery(rawPath)
		if err != nil {
			return nil, E_BAD_INPUT
		}
	} else if query := request.GetQuery(); query != nil {
		path = &componentcfg.Query{
			Component: query.Component,
			Flavor:    query.RunType,
			Rolename:  query.MachineRole,
			EntryKey:  query.Entry,
			Timestamp: query.Timestamp,
		}
	} else {
		return nil, E_BAD_INPUT
	}

	var payload string
	var err error
	if request.ProcessTemplate {
		payload, err = m.service.GetAndProcessComponentConfiguration(path, request.GetVarStack())
	} else {
		payload, err = m.service.GetComponentConfiguration(path)
	}

	if err != nil {
		return nil, err
	}
	return &apricotpb.ComponentResponse{Payload: payload}, E_OK.Err()
}

func (m *RpcServer) GetRuntimeEntry(ctx context.Context, request *apricotpb.GetRuntimeEntryRequest) (*apricotpb.ComponentResponse, error) {
	if m == nil || m.service == nil {
		return nil, E_CONFIGURATION_BACKEND_UNAVAILABLE
	}
	m.logMethod()
	if request == nil {
		return nil, E_BAD_INPUT
	}

	payload, err := m.service.GetRuntimeEntry(request.Component, request.Key)
	if err != nil {
		return nil, err
	}
	return &apricotpb.ComponentResponse{Payload: payload}, E_OK.Err()
}

func (m *RpcServer) SetRuntimeEntry(ctx context.Context, request *apricotpb.SetRuntimeEntryRequest) (*apricotpb.Empty, error) {
	if m == nil || m.service == nil {
		return nil, E_CONFIGURATION_BACKEND_UNAVAILABLE
	}
	m.logMethod()
	if request == nil {
		return nil, E_BAD_INPUT
	}

	err := m.service.SetRuntimeEntry(request.Component, request.Key, request.Value)
	if err != nil {
		return nil, err
	}
	return &apricotpb.Empty{}, E_OK.Err()
}

func (m *RpcServer) RawGetRecursive(ctx context.Context, request *apricotpb.RawGetRecursiveRequest) (*apricotpb.ComponentResponse, error) {
	if m == nil || m.service == nil {
		return nil, E_CONFIGURATION_BACKEND_UNAVAILABLE
	}
	m.logMethod()

	if request == nil {
		return nil, E_BAD_INPUT
	}

	payload, err := m.service.RawGetRecursive(request.RawPath)
	if err != nil {
		return &apricotpb.ComponentResponse{Payload: ""}, err
	}
	return &apricotpb.ComponentResponse{Payload: payload}, nil
}

func (m *RpcServer) logMethod() {
	if !viper.GetBool("verbose") {
		return
	}
	pc, _, _, ok := runtime.Caller(1)
	if !ok {
		return
	}
	fun := runtime.FuncForPC(pc)
	if fun == nil {
		return
	}
	log.WithPrefix("apricot").
		WithField("method", fun.Name()).
		Trace("handling RPC request")
}
