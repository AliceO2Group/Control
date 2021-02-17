/*
 * === This file is part of ALICE O² ===
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

// Package coconut implements the O² Control and Configuration Utility.
package coconut

import (
	"google.golang.org/grpc"
	"github.com/AliceO2Group/Control/common/logger"
	"github.com/AliceO2Group/Control/coconut/protos"
	"github.com/sirupsen/logrus"
	"context"
)

var log = logger.New(logrus.StandardLogger(), "coconut")

const GrpcMaxCallRecvSize = 10*1024*1024
type internalState struct {
}

func NewClient(cxt context.Context, cancel context.CancelFunc, endpoint string) *RpcClient {
	conn, err := grpc.DialContext(
		cxt,
		endpoint,
		grpc.WithInsecure(),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(GrpcMaxCallRecvSize)),
	)
	if err != nil {
		log.WithField("error", err.Error()).
			WithField("endpoint", endpoint).
			Errorf("cannot dial RPC endpoint")
		cancel()
		return nil
	}

	state := &internalState{}

	client := &RpcClient {
		ControlClient: pb.NewControlClient(conn),
		state: state,
		conn: conn,
	}

	return client
}

type RpcClient struct {
	pb.ControlClient
	state   *internalState
	conn    *grpc.ClientConn
}

func (m *RpcClient) Close() error {
	return m.conn.Close()
}