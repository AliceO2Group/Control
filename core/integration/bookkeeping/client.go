/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2022 CERN and copyright holders of ALICE O².
 * Author: Claire Guyot <claire.guyot@cern.ch>
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

package bookkeeping

import (
	"context"
	"time"

	"github.com/AliceO2Group/Control/common/logger"
	bkpb "github.com/AliceO2Group/Control/core/integration/bookkeeping/protos"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/keepalive"
)

var log = logger.New(logrus.StandardLogger(), "bookkeepingclient")

type RpcClient struct {
	bkpb.EnvironmentServiceClient
	bkpb.FlpServiceClient
	bkpb.LogServiceClient
	bkpb.RunServiceClient
	conn   *grpc.ClientConn
	cancel context.CancelFunc
}

func NewClient(cxt context.Context, cancel context.CancelFunc, endpoint string) *RpcClient {
	log.WithFields(logrus.Fields{
		"endpoint": endpoint,
	}).Debug("dialing Bookkeeping endpoint")

	dialOptions := []grpc.DialOption{
		grpc.WithInsecure(),
		grpc.WithConnectParams(grpc.ConnectParams{
			Backoff: backoff.Config{
				BaseDelay:  backoff.DefaultConfig.BaseDelay,
				Multiplier: backoff.DefaultConfig.Multiplier,
				Jitter:     backoff.DefaultConfig.Jitter,
				MaxDelay:   20 * time.Second,
			},
			MinConnectTimeout: 10 * time.Second,
		}),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                5 * time.Second,
			Timeout:             time.Second,
			PermitWithoutStream: true,
		}),
	}
	if !viper.GetBool("bookkeepingUseSystemProxy") {
		dialOptions = append(dialOptions, grpc.WithNoProxy())
	}
	conn, err := grpc.DialContext(cxt,
		endpoint,
		dialOptions...,
	)
	if err != nil {
		log.WithField("error", err.Error()).
			WithField("endpoint", endpoint).
			Errorf("cannot dial RPC endpoint")
		cancel()
		return nil
	}

	go func() {
		connState := connectivity.Idle
		stateChangedNotify := make(chan bool)
		notifyFunc := func(st connectivity.State) {
			ok := conn.WaitForStateChange(cxt, st)
			stateChangedNotify <- ok
		}
		go notifyFunc(connState)

		for {
			select {
			case ok := <-stateChangedNotify:
				if !ok {
					return
				}
				connState = conn.GetState()
				log.Debugf("Bookkeeping client %s", connState.String())
				go notifyFunc(connState)
			case <-time.After(2 * time.Minute):
				if conn.GetState() != connectivity.Ready {
					conn.ResetConnectBackoff()
				}
			case <-cxt.Done():
				return
			}
		}
	}()

	client := &RpcClient{
		EnvironmentServiceClient: bkpb.NewEnvironmentServiceClient(conn),
		FlpServiceClient:         bkpb.NewFlpServiceClient(conn),
		LogServiceClient:         bkpb.NewLogServiceClient(conn),
		RunServiceClient:         bkpb.NewRunServiceClient(conn),
		conn:                     conn,
		cancel:                   cancel,
	}

	return client
}

func (m *RpcClient) GetConnState() connectivity.State {
	if m.conn == nil {
		return connectivity.Idle
	}
	return m.conn.GetState()
}

func (m *RpcClient) Close() error {
	m.cancel()
	return m.conn.Close()
}
