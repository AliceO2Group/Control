package api

/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2022 CERN and copyright holders of ALICE O².
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

import (
	"runtime"
	"time"

	"github.com/AliceO2Group/Control/chili/protos"
	"github.com/AliceO2Group/Control/common/logger"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

var log = logger.New(logrus.StandardLogger(), "core")

// Implements interface pb.ChiliServer
type RpcServer struct {
}

func NewServer() *grpc.Server {
	s := grpc.NewServer()
	pb.RegisterChiliServer(s, &RpcServer{})
	// Register reflection service on gRPC server.
	reflection.Register(s)
	return s
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
	log.WithPrefix("rpcserver").
		WithField("method", fun.Name()).
		Trace("handling RPC request")
}

func (m *RpcServer) EventStream(request *pb.EventStreamRequest, server pb.Chili_EventStreamServer) error {
	m.logMethod()
	cxt := server.Context()

OUTER:
	for i := 0; i < 1000; i++ {
		select {
		case <-cxt.Done():
			log.Info("stream closed client side")
			break OUTER
		case <-time.After(1 * time.Second):
		}
		err := server.Send(&pb.Event{
			Timestamp: time.Now().Format(time.RFC3339),
			Payload:   &pb.Event_MetaEvent{MetaEvent: &pb.Ev_MetaEvent_Subscribed{ClientId: "foo"}},
		})
		if err != nil {
			log.WithError(err).Error("stream broken client side")
			return status.New(codes.OK, "all done").Err()
		}
		log.Printf("tick %d\n", i)
	}

	return status.New(codes.OK, "all done").Err()
}
