/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2021 CERN and copyright holders of ALICE O².
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

package odc

import (
	"context"
	"time"

	"github.com/AliceO2Group/Control/common/logger"
	"github.com/AliceO2Group/Control/common/monitoring"
	odcpb "github.com/AliceO2Group/Control/core/integration/odc/protos"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/keepalive"
)

var log = logger.New(logrus.StandardLogger(), "odcclient")

func newMetric() monitoring.Metric {
	return monitoring.NewMetric("odc")
}

type RpcClient struct {
	odcpb.ODCClient
	conn   *grpc.ClientConn
	cancel context.CancelFunc
}

func NewClient(cxt context.Context, cancel context.CancelFunc, endpoint string) *RpcClient {
	log.WithFields(logrus.Fields{
		"endpoint": endpoint,
	}).Debug("dialing ODC endpoint")

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
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(ODC_MAX_INBOUND_MESSAGE_SIZE)),
	}
	if !viper.GetBool("odcUseSystemProxy") {
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
				log.Debugf("ODC client %s", connState.String())
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
		ODCClient: odcpb.NewODCClient(conn),
		conn:      conn,
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

func (m *RpcClient) Initialize(ctx context.Context, in *odcpb.InitializeRequest, opts ...grpc.CallOption) (*odcpb.GeneralReply, error) {
	metric := newMetric()
	metric.AddTag("method", "Initialize")
	defer monitoring.TimerSend(&metric, monitoring.Milliseconds)()
	return m.ODCClient.Initialize(ctx, in, opts...)
}

func (m *RpcClient) Submit(ctx context.Context, in *odcpb.SubmitRequest, opts ...grpc.CallOption) (*odcpb.GeneralReply, error) {
	metric := newMetric()
	metric.AddTag("method", "Submit")
	defer monitoring.TimerSend(&metric, monitoring.Milliseconds)()
	return m.ODCClient.Submit(ctx, in, opts...)
}

func (m *RpcClient) Activate(ctx context.Context, in *odcpb.ActivateRequest, opts ...grpc.CallOption) (*odcpb.GeneralReply, error) {
	metric := newMetric()
	metric.AddTag("method", "Activate")
	defer monitoring.TimerSend(&metric, monitoring.Milliseconds)()
	return m.ODCClient.Activate(ctx, in, opts...)
}

func (m *RpcClient) Run(ctx context.Context, in *odcpb.RunRequest, opts ...grpc.CallOption) (*odcpb.GeneralReply, error) {
	metric := newMetric()
	metric.AddTag("method", "Run")
	defer monitoring.TimerSend(&metric, monitoring.Milliseconds)()
	return m.ODCClient.Run(ctx, in, opts...)
}

func (m *RpcClient) Update(ctx context.Context, in *odcpb.UpdateRequest, opts ...grpc.CallOption) (*odcpb.GeneralReply, error) {
	metric := newMetric()
	metric.AddTag("method", "Update")
	defer monitoring.TimerSend(&metric, monitoring.Milliseconds)()
	return m.ODCClient.Update(ctx, in, opts...)
}

func (m *RpcClient) Configure(ctx context.Context, in *odcpb.ConfigureRequest, opts ...grpc.CallOption) (*odcpb.StateReply, error) {
	metric := newMetric()
	metric.AddTag("method", "Configure")
	defer monitoring.TimerSend(&metric, monitoring.Milliseconds)()
	return m.ODCClient.Configure(ctx, in, opts...)
}

func (m *RpcClient) SetProperties(ctx context.Context, in *odcpb.SetPropertiesRequest, opts ...grpc.CallOption) (*odcpb.GeneralReply, error) {
	metric := newMetric()
	metric.AddTag("method", "SetProperties")
	defer monitoring.TimerSend(&metric, monitoring.Milliseconds)()
	return m.ODCClient.SetProperties(ctx, in, opts...)
}

func (m *RpcClient) GetState(ctx context.Context, in *odcpb.StateRequest, opts ...grpc.CallOption) (*odcpb.StateReply, error) {
	metric := newMetric()
	metric.AddTag("method", "GetState")
	defer monitoring.TimerSend(&metric, monitoring.Milliseconds)()
	return m.ODCClient.GetState(ctx, in, opts...)
}

func (m *RpcClient) Start(ctx context.Context, in *odcpb.StartRequest, opts ...grpc.CallOption) (*odcpb.StateReply, error) {
	metric := newMetric()
	metric.AddTag("method", "Start")
	defer monitoring.TimerSend(&metric, monitoring.Milliseconds)()
	return m.ODCClient.Start(ctx, in, opts...)
}

func (m *RpcClient) Stop(ctx context.Context, in *odcpb.StopRequest, opts ...grpc.CallOption) (*odcpb.StateReply, error) {
	metric := newMetric()
	metric.AddTag("method", "Stop")
	defer monitoring.TimerSend(&metric, monitoring.Milliseconds)()
	return m.ODCClient.Stop(ctx, in, opts...)
}

func (m *RpcClient) Reset(ctx context.Context, in *odcpb.ResetRequest, opts ...grpc.CallOption) (*odcpb.StateReply, error) {
	metric := newMetric()
	metric.AddTag("method", "Reset")
	defer monitoring.TimerSend(&metric, monitoring.Milliseconds)()
	return m.ODCClient.Reset(ctx, in, opts...)
}

func (m *RpcClient) Terminate(ctx context.Context, in *odcpb.TerminateRequest, opts ...grpc.CallOption) (*odcpb.StateReply, error) {
	metric := newMetric()
	metric.AddTag("method", "Terminate")
	defer monitoring.TimerSend(&metric, monitoring.Milliseconds)()
	return m.ODCClient.Terminate(ctx, in, opts...)
}

func (m *RpcClient) Shutdown(ctx context.Context, in *odcpb.ShutdownRequest, opts ...grpc.CallOption) (*odcpb.GeneralReply, error) {
	metric := newMetric()
	metric.AddTag("method", "Shutdown")
	defer monitoring.TimerSend(&metric, monitoring.Milliseconds)()
	return m.ODCClient.Shutdown(ctx, in, opts...)
}

func (m *RpcClient) Status(ctx context.Context, in *odcpb.StatusRequest, opts ...grpc.CallOption) (*odcpb.StatusReply, error) {
	metric := newMetric()
	metric.AddTag("method", "Status")
	defer monitoring.TimerSend(&metric, monitoring.Milliseconds)()
	return m.ODCClient.Status(ctx, in, opts...)
}
