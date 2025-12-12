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

// Package executorcmd contains the gRPC client, as well as facilities
// for processing and committing incoming transition events.
package executorcmd

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"time"

	"github.com/AliceO2Group/Control/common/logger/infologger"
	"github.com/AliceO2Group/Control/executor/executorcmd/nopb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"

	"github.com/AliceO2Group/Control/common/controlmode"
	"github.com/AliceO2Group/Control/executor/executorcmd/transitioner"
	pb "github.com/AliceO2Group/Control/executor/protos"
	"github.com/sirupsen/logrus"
	grpcstatus "google.golang.org/grpc/status"
)

type ControlTransport uint32

const (
	ProtobufTransport = ControlTransport(0)
	JsonTransport     = ControlTransport(1)
)
const GRPC_DIAL_TIMEOUT = 45 * time.Second

func NewClient(
	controlPort uint64,
	controlMode controlmode.ControlMode,
	controlTransport ControlTransport,
	log *logrus.Entry,
) *RpcClient {
	endpoint := fmt.Sprintf("127.0.0.1:%d", controlPort)
	controlTransportS := "Protobuf"
	if controlTransport == JsonTransport {
		controlTransportS = "JSON"
	}

	cxt, cancel := context.WithTimeout(context.Background(), GRPC_DIAL_TIMEOUT)
	conn, err := grpc.DialContext(cxt, endpoint, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithReturnConnectionError(), grpc.WithConnectParams(grpc.ConnectParams{
		Backoff: backoff.Config{
			BaseDelay:  200 * time.Millisecond,
			Multiplier: 1.1,
			Jitter:     0.02,
			MaxDelay:   GRPC_DIAL_TIMEOUT,
		},
		MinConnectTimeout: 5 * time.Second,
	}))
	if err != nil {
		log.WithField("error", err.Error()).
			WithField("endpoint", endpoint).
			WithField("transport", controlTransportS).
			WithField("level", infologger.IL_Devel).
			Error("gRPC client can't dial")

		cancel()
		if conn != nil {
			log.WithField("endpoint", endpoint).
				WithField("transport", controlTransportS).
				Warn("gRPC client connection failure: cleaning up leftover connection")
			_ = conn.Close()
		}
		return nil
	} else {
		log.WithField("endpoint", endpoint).
			WithField("transport", controlTransportS).
			Debug("gRPC client dial successful")
	}

	var occClient pb.OccClient
	if controlTransport == JsonTransport {
		occClient = nopb.NewOccClient(conn)
	} else {
		occClient = pb.NewOccClient(conn)
	}

	client := &RpcClient{
		OccClient: occClient,
		conn:      conn,
	}

	client.Transitioner = transitioner.NewTransitioner(controlMode, client.doTransition)
	client.Log = log
	return client
}

type RpcClient struct {
	pb.OccClient
	conn         *grpc.ClientConn
	Transitioner transitioner.Transitioner
	TaskCmd      *exec.Cmd
	Log          *logrus.Entry
}

func (r *RpcClient) Close() error {
	if r != nil && r.conn != nil {
		return r.conn.Close()
	}
	return nil
}

func (r *RpcClient) FromDeviceState(state string) string {
	return r.Transitioner.FromDeviceState(state)
}

func (r *RpcClient) doTransition(ei transitioner.EventInfo) (newState string, err error) {
	r.Log.WithField("event", ei.Evt).
		Debug("executor<->occplugin interface requesting transition")

	var response *pb.TransitionReply

	argsToPush := func() (cfg []*pb.ConfigEntry) {
		cfg = make([]*pb.ConfigEntry, 0)
		if len(ei.Args) == 0 {
			return
		}
		for k, v := range ei.Args {
			cfg = append(cfg, &pb.ConfigEntry{Key: k, Value: v})
			r.Log.WithField("key", k).
				WithField("value", v).
				Trace("pushing argument")
		}
		return
	}()

	response, err = r.Transition(context.TODO(), &pb.TransitionRequest{
		TransitionEvent: ei.Evt,
		Arguments:       argsToPush,
		SrcState:        ei.Src,
	}, grpc.EmptyCallOption{})

	if err != nil {
		status, ok := grpcstatus.FromError(err)
		if ok {
			r.Log.WithFields(logrus.Fields{
				"code":    status.Code().String(),
				"message": status.Message(),
				"details": status.Details(),
				"error":   status.Err().Error(),
				"level":   infologger.IL_Devel,
			}).
				Error("transition call error")
			err = errors.New(fmt.Sprintf("occplugin returned %s: %s", status.Code().String(), status.Message()))
		} else {
			err = errors.New("invalid gRPC status")
			r.Log.WithField("error", "invalid gRPC status response received from occplugin").
				WithField("level", infologger.IL_Support).
				Error("transition call error")
		}
		return
	}

	taskId, _ := r.Log.Data["id"]
	if response != nil &&
		response.GetOk() &&
		response.GetTrigger() == pb.StateChangeTrigger_EXECUTOR &&
		response.GetTransitionEvent() == ei.Evt &&
		response.GetState() == ei.Dst {
		newState = response.GetState()
		err = nil
		r.Log.WithField("dst", newState).Debug("occ transition complete")
	} else if response != nil {
		newState = response.GetState()
		err = fmt.Errorf("transition unsuccessful: ok: %s, trigger: %s, event: %s, state: %s, id: %s",
			strconv.FormatBool(response.GetOk()),
			response.GetTrigger().String(),
			response.GetTransitionEvent(),
			response.GetState(),
			taskId)
	} else {
		newState = ""
		err = fmt.Errorf("transition unsuccessful: id: %s invalid response but no gRPC error", taskId)
	}
	return
}
