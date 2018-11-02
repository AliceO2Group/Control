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
	"github.com/k0kubun/pp"
	"google.golang.org/grpc"
	"fmt"

	"github.com/AliceO2Group/Control/executor/protos"
	"time"
	"github.com/AliceO2Group/Control/common/logger"
	"github.com/sirupsen/logrus"
	"github.com/AliceO2Group/Control/executor/executorcmd/transitioner"
	"google.golang.org/grpc/status"
	"errors"
	"encoding/json"
	"strconv"
	"github.com/AliceO2Group/Control/common/controlmode"
)

var log = logger.New(logrus.StandardLogger(), "executorcmd")

func NewClient(controlPort uint64, controlMode controlmode.ControlMode) *RpcClient {
	endpoint := fmt.Sprintf("127.0.0.1:%d", controlPort)
	log.WithField("endpoint", endpoint).Debug("starting new gRPC client")

	cxt, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	conn, err := grpc.DialContext(cxt, endpoint, grpc.WithInsecure())
	if err != nil {
		log.WithField("error", err.Error()).
			WithField("endpoint", endpoint).
			Errorf("gRPC client can't dial")
		cancel()
		return nil
	}

	client := &RpcClient {
		OccClient: pb.NewOccClient(conn),
		conn: conn,
	}

	log.WithFields(logrus.Fields{"endpoint": endpoint, "controlMode": controlMode.String()}).Debug("instantiating new transitioner")
	client.ctrl = transitioner.NewTransitioner(controlMode, client.doTransition)
	return client
}

type RpcClient struct {
	pb.OccClient
	conn *grpc.ClientConn
	ctrl transitioner.Transitioner
}

func (r *RpcClient) Close() error {
	return r.conn.Close()
}

func (r *RpcClient) FromDeviceState(state string) string {
	return r.ctrl.FromDeviceState(state)
}

func (r *RpcClient) UnmarshalTransition(data []byte) (cmd *ExecutorCommand_Transition, err error) {
	cmd = new(ExecutorCommand_Transition)
	cmd.rc = r
	err = json.Unmarshal(data, cmd)
	if err != nil {
		cmd = nil
	}
	return
}

func (r *RpcClient) doTransition(ei transitioner.EventInfo) (newState string, err error) {
	log.WithField("event", ei.Evt).Debug("executor<->occplugin interface requesting transition")
	var response *pb.TransitionReply
	response, err = r.Transition(context.TODO(), &pb.TransitionRequest{
		Event: ei.Evt,
		Arguments: func() (cfg []*pb.ConfigEntry) {
			cfg = make([]*pb.ConfigEntry, 0)
			if len(ei.Args) == 0 {
				return
			}
			for k, v := range ei.Args {
				cfg = append(cfg, &pb.ConfigEntry{Key: k, Value: v})
			}
			return
		}(),
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
		response.GetEvent() == ei.Evt &&
		response.GetState() == ei.Dst {
		newState = response.GetState()
		err = nil
	} else if response != nil {
		newState = response.GetState()
		err = errors.New(fmt.Sprintf("transition unsuccessful: ok: %s, trigger: %s, event: %s, state: %s",
			strconv.FormatBool(response.GetOk()),
			response.GetTrigger().String(),
			response.GetEvent(),
			response.GetState()))
	} else {
		newState = ""
		err = errors.New("transition unsuccessful: invalid response but no gRPC error")
	}
	return
}