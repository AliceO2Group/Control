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

package occserver

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"github.com/AliceO2Group/Control/common/logger"
	"github.com/AliceO2Group/Control/odcshim/odcclient"
	pb "github.com/AliceO2Group/Control/odcshim/protos"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

var log = logger.New(logrus.StandardLogger(), "o2-aliecs-odc-shim")

const CALL_TIMEOUT = 60*time.Second

type OccServerImpl struct {
	odcHost   string
	odcPort   int
	odcClient *odcclient.RpcClient
}

func (s *OccServerImpl) ensureClientConnected() error {
	// create the clientconn
	// instantiate the odcClient and Initialize
	// report error or nil
	endpoint := fmt.Sprintf("%s:%d", s.odcHost, s.odcPort)

	cxt, cancel := context.WithTimeout(context.Background(), CALL_TIMEOUT)

	s.odcClient = odcclient.NewClient(cxt, cancel, endpoint)
	if s.odcClient == nil {
		return fmt.Errorf("cannot dial ODC endpoint: %s", endpoint)
	}
	return nil
}

func NewServer(host string, port int) *grpc.Server {
	grpcServer := grpc.NewServer()
	srvImpl := &OccServerImpl{
		odcHost: host,
		odcPort: port,
	}
	pb.RegisterOccServer(grpcServer, srvImpl)
	// Register reflection service on gRPC server.
	reflection.Register(grpcServer)

	return grpcServer
}

func (s *OccServerImpl) logMethod() {
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
	log.WithField("method", fun.Name()).
		Debug("handling RPC request")
}

func (*OccServerImpl) EventStream(req *pb.EventStreamRequest, srv pb.Occ_EventStreamServer) error {
	return status.Errorf(codes.Unimplemented, "method EventStream not implemented")
}
func (*OccServerImpl) StateStream(req *pb.StateStreamRequest, srv pb.Occ_StateStreamServer) error {
	return status.Errorf(codes.Unimplemented, "method StateStream not implemented")
}
func (*OccServerImpl) GetState(ctx context.Context, req *pb.GetStateRequest) (*pb.GetStateReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetState not implemented")
}
func (s *OccServerImpl) Transition(ctx context.Context, req *pb.TransitionRequest) (*pb.TransitionReply, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "bad incoming request")
	}

	// Provisional response
	rep := &pb.TransitionReply{
		Trigger:         pb.StateChangeTrigger_EXECUTOR,
		State:           req.SrcState,
		TransitionEvent: req.TransitionEvent,
		Ok:              false,
	}

	if s.odcClient == nil || s.odcClient.GetConnState() != connectivity.Ready {
		err := s.ensureClientConnected()
		if err != nil {
			log.WithError(err).
				WithFields(logrus.Fields{
					"host": s.odcHost,
					"port": s.odcPort,
				}).
				Error("cannot connect to ODC endpoint")
			return nil, err
		}
	}

	var err error = nil
	switch event := req.TransitionEvent; event {
	case "CONFIGURE":
		err = handleConfigure(ctx, s.odcClient, req.Arguments)
		if err == nil {
			rep.Ok = true
			rep.State = "CONFIGURED"
		}
	}
	return rep, err
}
