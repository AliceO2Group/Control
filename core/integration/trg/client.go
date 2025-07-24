/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2021 CERN and copyright holders of ALICE O².
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

package trg

import (
	"context"
	"time"

	"github.com/AliceO2Group/Control/common/logger"
	"github.com/AliceO2Group/Control/common/monitoring"
	trgecspb "github.com/AliceO2Group/Control/core/integration/trg/protos"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/keepalive"
)

var log = logger.New(logrus.StandardLogger(), "trgclient")

func convertMethodName(method string) (converted string) {
	switch method {
	case trgecspb.CTPd_PrepareForRun_FullMethodName:
		converted = "PrepareForRun"
	case trgecspb.CTPd_RunLoad_FullMethodName:
		converted = "RunLoad"
	case trgecspb.CTPd_RunUnload_FullMethodName:
		converted = "RunUnload"
	case trgecspb.CTPd_RunStart_FullMethodName:
		converted = "RunStart"
	case trgecspb.CTPd_RunStatus_FullMethodName:
		converted = "RunStatus"
	case trgecspb.CTPd_RunList_FullMethodName:
		converted = "RunList"
	case trgecspb.CTPd_RunStop_FullMethodName:
		converted = "RunStop"
	case trgecspb.CTPd_RunConfig_FullMethodName:
		converted = "RunConfig"
	case trgecspb.CTPd_RunCleanup_FullMethodName:
		converted = "RunCleanup"
	case trgecspb.CTPd_TPCReset_FullMethodName:
		converted = "TPCReset"
	default:
		converted = "Unknown"
	}
	return
}

type RpcClient struct {
	trgecspb.CTPdClient
	conn *grpc.ClientConn
}

func NewClient(cxt context.Context, cancel context.CancelFunc, endpoint string) *RpcClient {
	log.WithFields(logrus.Fields{
		"endpoint": endpoint,
	}).Debug("dialing TRG endpoint")

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
		grpc.WithUnaryInterceptor(monitoring.SetupUnaryClientInterceptor("trg", convertMethodName)),
	}
	if !viper.GetBool("trgServiceUseSystemProxy") {
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
				log.Debugf("TRG client %s", connState.String())
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
		CTPdClient: trgecspb.NewCTPdClient(conn),
		conn:       conn,
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
	return m.conn.Close()
}
