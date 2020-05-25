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

	"github.com/AliceO2Group/Control/executor/executorcmd/nopb"
	"github.com/k0kubun/pp"
	"google.golang.org/grpc"

	"github.com/AliceO2Group/Control/common/controlmode"
	"github.com/AliceO2Group/Control/common/logger"
	"github.com/AliceO2Group/Control/executor/executorcmd/transitioner"
	"github.com/AliceO2Group/Control/executor/protos"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/status"
)

var log = logger.New(logrus.StandardLogger(), "executorcmd")

type ControlTransport uint32
const (
	ProtobufTransport = ControlTransport(0)
	JsonTransport = ControlTransport(1)
)

func NewClient(controlPort uint64, controlMode controlmode.ControlMode, controlTransport ControlTransport) *RpcClient {
	endpoint := fmt.Sprintf("127.0.0.1:%d", controlPort)
	controlTransportS := "Protobuf"
	if controlTransport == JsonTransport {
		controlTransportS = "JSON"
	}

	log.WithField("endpoint", endpoint).
		WithField("transport", controlTransportS).
		Debug("starting new gRPC client")

	cxt, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	conn, err := grpc.DialContext(cxt, endpoint, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.WithField("error", err.Error()).
			WithField("endpoint", endpoint).
			WithField("transport", controlTransportS).
			Errorf("gRPC client can't dial")
		cancel()
		return nil
	}

	var occClient pb.OccClient
	if controlTransport == JsonTransport {
		occClient = nopb.NewOccClient(conn)
	} else {
		occClient = pb.NewOccClient(conn)
	}

	client := &RpcClient {
		OccClient: occClient,
		conn: conn,
	}

	log.WithFields(logrus.Fields{"endpoint": endpoint, "controlMode": controlMode.String()}).Debug("instantiating new transitioner")
	client.Transitioner = transitioner.NewTransitioner(controlMode, client.doTransition)
	return client
}

type RpcClient struct {
	pb.OccClient
	conn         *grpc.ClientConn
	Transitioner transitioner.Transitioner
	TaskCmd      *exec.Cmd
}

func (r *RpcClient) Close() error {
	return r.conn.Close()
}

func (r *RpcClient) FromDeviceState(state string) string {
	return r.Transitioner.FromDeviceState(state)
}

func (r *RpcClient) doTransition(ei transitioner.EventInfo) (newState string, err error) {
	log.WithField("event", ei.Evt).
		Debug("executor<->occplugin interface requesting transition")

	var response *pb.TransitionReply

	argsToPush := func() (cfg []*pb.ConfigEntry) {
		cfg = make([]*pb.ConfigEntry, 0)
		if len(ei.Args) == 0 {
			return
		}
		for k, v := range ei.Args {
			cfg = append(cfg, &pb.ConfigEntry{Key: k, Value: v})
			log.WithField("key", k).
				WithField("value", v).
				Debug("pushing argument")
		}
		return
	}()

	response, err = r.Transition(context.TODO(), &pb.TransitionRequest{
		TransitionEvent: ei.Evt,
		Arguments: argsToPush,
		SrcState: ei.Src,
	}, grpc.EmptyCallOption{})

	log.Debug("response received, about to parse status")

	if err != nil {
		// We must process the error explicitly here, otherwise we get an error because gRPC's
		// Status is different from what gogoproto expects.
		status, ok := status.FromError(err)
		log.Debug("got status from error")
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
			err = errors.New(fmt.Sprintf("occplugin returned %s: %s", status.Code().String(), status.Message()))
		} else {
			err = errors.New("invalid gRPC status")
			log.WithField("error", "invalid gRPC status").Error("transition call error")
		}
		return
	}

	if response != nil &&
		response.GetOk() &&
		response.GetTrigger() == pb.StateChangeTrigger_EXECUTOR &&
		response.GetTransitionEvent() == ei.Evt &&
		response.GetState() == ei.Dst {
		newState = response.GetState()
		err = nil
	} else if response != nil {
		newState = response.GetState()
		err = errors.New(fmt.Sprintf("transition unsuccessful: ok: %s, trigger: %s, event: %s, state: %s",
			strconv.FormatBool(response.GetOk()),
			response.GetTrigger().String(),
			response.GetTransitionEvent(),
			response.GetState()))
	} else {
		newState = ""
		err = errors.New("transition unsuccessful: invalid response but no gRPC error")
	}
	return
}