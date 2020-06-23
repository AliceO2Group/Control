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
	"errors"
	"fmt"
	"runtime"
	"time"

	"github.com/AliceO2Group/Control/common/logger"
	"github.com/AliceO2Group/Control/odcshim/odcclient"
	odc "github.com/AliceO2Group/Control/odcshim/odcprotos"
	pb "github.com/AliceO2Group/Control/odcshim/protos"
	"github.com/k0kubun/pp"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"golang.org/x/net/context"
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
	switch event := req.TransitionEvent; event {
	case "CONFIGURE":
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

		// Extract relevant parameters from Arguments payload
		// and build payload for SetProperty+Configure
		var envId string
		var topology string
		configureMap := make(map[string]string)
		for _, entry := range req.Arguments {
			if entry.Key == "environment_id" {
				envId = entry.Value
			} else if entry.Key == "topology" {
				topology = entry.Value
			} else {
				configureMap[entry.Key] = entry.Value
			}

		}

		// Provisional response
		rep := &pb.TransitionReply{
			Trigger:         pb.StateChangeTrigger_EXECUTOR,
			State:           req.SrcState,
			TransitionEvent: req.TransitionEvent,
			Ok:              false,
		}

		// INITIALIZE
		initializeRequest := &odc.InitializeRequest{
			Runid:                0,	//FIXME: not available at this time
			Sessionid:            envId,
		}

		initializeResponse, err := s.odcClient.Initialize(ctx, initializeRequest, grpc.EmptyCallOption{})
		if err != nil {
			// We must process the error explicitly here, otherwise we get an error because gRPC's
			// Status is different from what gogoproto expects.
			status, ok := status.FromError(err)
			if ok {
				log.WithFields(logrus.Fields{
						"code": status.Code().String(),
						"message": status.Message(),
						"details": status.Details(),
						"error": status.Err().Error(),
						"ppStatus": pp.Sprint(status),
						"ppErr": pp.Sprint(err),
					}).
					Error("transition call error")
				err = fmt.Errorf("occplugin returned %s: %s", status.Code().String(), status.Message())
			} else {
				err = errors.New("invalid gRPC status")
				log.WithField("error", "invalid gRPC status").Error("transition call error")
			}
			return nil, err
		}

		if initializeResponse != nil {
			if odcErr := initializeResponse.GetError(); odcErr != nil {
				return rep, fmt.Errorf("code %d from ODC: %s", odcErr.GetCode(), odcErr.GetMsg())
			}
			if replyStatus := initializeResponse.Status; replyStatus != odc.ReplyStatus_SUCCESS {
				return rep, fmt.Errorf("status %s from ODC", replyStatus.String())
			}
			log.WithFields(logrus.Fields{
				"odcMsg": initializeResponse.Msg,
				"odcStatus": initializeResponse.Status.String(),
				"odcExectime": initializeResponse.Exectime,
				"odcRunid": initializeResponse.Runid,
				"odcSessionid": initializeResponse.Sessionid,
			}).
			Debug("call to ODC complete")
		}

		// SUBMIT
		submitRequest := &odc.SubmitRequest{}

		submitResponse, err := s.odcClient.Submit(ctx, submitRequest, grpc.EmptyCallOption{})
		if err != nil {
			// We must process the error explicitly here, otherwise we get an error because gRPC's
			// Status is different from what gogoproto expects.
			status, ok := status.FromError(err)
			if ok {
				log.WithFields(logrus.Fields{
						"code": status.Code().String(),
						"message": status.Message(),
						"details": status.Details(),
						"error": status.Err().Error(),
						"ppStatus": pp.Sprint(status),
						"ppErr": pp.Sprint(err),
					}).
					Error("transition call error")
				err = fmt.Errorf("occplugin returned %s: %s", status.Code().String(), status.Message())
			} else {
				err = errors.New("invalid gRPC status")
				log.WithField("error", "invalid gRPC status").Error("transition call error")
			}
			return nil, err
		}

		if submitResponse != nil {
			if odcErr := submitResponse.GetError(); odcErr != nil {
				return rep, fmt.Errorf("code %d from ODC: %s", odcErr.GetCode(), odcErr.GetMsg())
			}
			if replyStatus := submitResponse.Status; replyStatus != odc.ReplyStatus_SUCCESS {
				return rep, fmt.Errorf("status %s from ODC", replyStatus.String())
			}
			log.WithFields(logrus.Fields{
					"odcMsg": submitResponse.Msg,
					"odcStatus": submitResponse.Status.String(),
					"odcExectime": submitResponse.Exectime,
					"odcRunid": submitResponse.Runid,
					"odcSessionid": submitResponse.Sessionid,
				}).
				Debug("call to ODC complete")
		}

		// ACTIVATE
		activateRequest := &odc.ActivateRequest{
			Topology: topology,
		}

		activateResponse, err := s.odcClient.Activate(ctx, activateRequest, grpc.EmptyCallOption{})
		if err != nil {
			// We must process the error explicitly here, otherwise we get an error because gRPC's
			// Status is different from what gogoproto expects.
			status, ok := status.FromError(err)
			if ok {
				log.WithFields(logrus.Fields{
						"code": status.Code().String(),
						"message": status.Message(),
						"details": status.Details(),
						"error": status.Err().Error(),
						"ppStatus": pp.Sprint(status),
						"ppErr": pp.Sprint(err),
					}).
					Error("transition call error")
				err = fmt.Errorf("occplugin returned %s: %s", status.Code().String(), status.Message())
			} else {
				err = errors.New("invalid gRPC status")
				log.WithField("error", "invalid gRPC status").Error("transition call error")
			}
			return nil, err
		}

		if activateResponse != nil {
			if odcErr := activateResponse.GetError(); odcErr != nil {
				return rep, fmt.Errorf("code %d from ODC: %s", odcErr.GetCode(), odcErr.GetMsg())
			}
			if replyStatus := activateResponse.Status; replyStatus != odc.ReplyStatus_SUCCESS {
				return rep, fmt.Errorf("status %s from ODC", replyStatus.String())
			}
			log.WithFields(logrus.Fields{
					"odcMsg": activateResponse.Msg,
					"odcStatus": activateResponse.Status.String(),
					"odcExectime": activateResponse.Exectime,
					"odcRunid": activateResponse.Runid,
					"odcSessionid": activateResponse.Sessionid,
				}).
				Debug("call to ODC complete")
		}

		// FIXME: implement SetProperty calls before Configure

		// CONFIGURE
		configureRequest := &odc.ConfigureRequest{
			Request:              &odc.StateChangeRequest{
				Path:     "",
				Detailed: false,
			},
		}

		configureResponse, err := s.odcClient.Configure(ctx, configureRequest, grpc.EmptyCallOption{})
		if err != nil {
			// We must process the error explicitly here, otherwise we get an error because gRPC's
			// Status is different from what gogoproto expects.
			status, ok := status.FromError(err)
			if ok {
				log.WithFields(logrus.Fields{
						"code": status.Code().String(),
						"message": status.Message(),
						"details": status.Details(),
						"error": status.Err().Error(),
						"ppStatus": pp.Sprint(status),
						"ppErr": pp.Sprint(err),
					}).
					Error("transition call error")
				err = fmt.Errorf("occplugin returned %s: %s", status.Code().String(), status.Message())
			} else {
				err = errors.New("invalid gRPC status")
				log.WithField("error", "invalid gRPC status").Error("transition call error")
			}
			return nil, err
		}

		if configureResponse != nil && configureResponse.Reply != nil {
			if odcErr := configureResponse.Reply.GetError(); odcErr != nil {
				return rep, fmt.Errorf("code %d from ODC: %s", odcErr.GetCode(), odcErr.GetMsg())
			}
			if replyStatus := configureResponse.Reply.Status; replyStatus != odc.ReplyStatus_SUCCESS {
				return rep, fmt.Errorf("status %s from ODC", replyStatus.String())
			}
			log.WithFields(logrus.Fields{
					"odcMsg": configureResponse.Reply.Msg,
					"odcStatus": configureResponse.Reply.Status.String(),
					"odcExectime": configureResponse.Reply.Exectime,
					"odcRunid": configureResponse.Reply.Runid,
					"odcSessionid": configureResponse.Reply.Sessionid,
				}).
				Debug("call to ODC complete")
		}



	}
	return nil, status.Errorf(codes.Unimplemented, "method Transition not implemented")
}
