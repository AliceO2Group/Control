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
	"errors"
	"fmt"

	"github.com/AliceO2Group/Control/odcshim/odcclient"
	odc "github.com/AliceO2Group/Control/odcshim/odcprotos"
	pb "github.com/AliceO2Group/Control/odcshim/protos"
	"github.com/k0kubun/pp"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

func handleConfigure(ctx context.Context, odcClient *odcclient.RpcClient, arguments []*pb.ConfigEntry ) error {
	// Extract relevant parameters from Arguments payload
	// and build payload for SetProperty+Configure
	var topology string
	configureMap := make(map[string]string)
	for _, entry := range arguments {
		if entry.Key == "topology" {
			topology = entry.Value
		} else {
			configureMap[entry.Key] = entry.Value
		}
	}

	// RUN request, includes INITIALIZE+SUBMIT+ACTIVATE
	runRequest := &odc.RunRequest{
		Topology: topology,
	}

	var err error = nil
	var runResponse *odc.GeneralReply

	runResponse, err = odcClient.Run(ctx, runRequest, grpc.EmptyCallOption{})
	if err != nil {
		return printGrpcError(err)
	}

	if runResponse == nil {
		// We got a nil response with nil error, this should never happen
		return errors.New("nil response error")
	}

	if odcErr := runResponse.GetError(); odcErr != nil {
		return fmt.Errorf("code %d from ODC: %s", odcErr.GetCode(), odcErr.GetMsg())
	}
	if replyStatus := runResponse.Status; replyStatus != odc.ReplyStatus_SUCCESS {
		return fmt.Errorf("status %s from ODC", replyStatus.String())
	}
	log.WithFields(logrus.Fields{
			"odcMsg":       runResponse.Msg,
			"odcStatus":    runResponse.Status.String(),
			"odcExectime":  runResponse.Exectime,
			"odcRunid":     runResponse.Runid,
			"odcSessionid": runResponse.Sessionid,
		}).
		Debug("call to ODC complete")

	// SetProperties before CONFIGURE
	setPropertiesRequest := &odc.SetPropertiesRequest{
		Path:       "",
		Properties: make([]*odc.Property, len(configureMap)),
	}
	i := 0
	for k, v := range configureMap {
		setPropertiesRequest.Properties[i] = &odc.Property{
			Key:   k,
			Value: v,
		}
		i++
	}
	var setPropertiesResponse *odc.GeneralReply
	setPropertiesResponse, err = odcClient.SetProperties(ctx, setPropertiesRequest, grpc.EmptyCallOption{})
	if err != nil {
		return printGrpcError(err)
	}

	if setPropertiesResponse == nil {
		// We got a nil response with nil error, this should never happen
		return errors.New("nil response error")
	}

	if odcErr := setPropertiesResponse.GetError(); odcErr != nil {
		return fmt.Errorf("code %d from ODC: %s", odcErr.GetCode(), odcErr.GetMsg())
	}
	if replyStatus := setPropertiesResponse.Status; replyStatus != odc.ReplyStatus_SUCCESS {
		return fmt.Errorf("status %s from ODC", replyStatus.String())
	}
	log.WithFields(logrus.Fields{
			"odcMsg":       setPropertiesResponse.Msg,
			"odcStatus":    setPropertiesResponse.Status.String(),
			"odcExectime":  setPropertiesResponse.Exectime,
			"odcRunid":     setPropertiesResponse.Runid,
			"odcSessionid": setPropertiesResponse.Sessionid,
		}).
		Debug("call to ODC complete")


	// CONFIGURE
	configureRequest := &odc.ConfigureRequest{
		Request:              &odc.StateRequest{
			Path:     "",
			Detailed: false,
		},
	}

	var configureResponse *odc.StateReply
	configureResponse, err = odcClient.Configure(ctx, configureRequest, grpc.EmptyCallOption{})
	if err != nil {
		return printGrpcError(err)
	}

	if configureResponse == nil || configureResponse.Reply == nil {
		// We got a nil response with nil error, this should never happen
		return errors.New("nil response error")
	}

	if odcErr := configureResponse.Reply.GetError(); odcErr != nil {
		return fmt.Errorf("code %d from ODC: %s", odcErr.GetCode(), odcErr.GetMsg())
	}
	if replyStatus := configureResponse.Reply.Status; replyStatus != odc.ReplyStatus_SUCCESS {
		return fmt.Errorf("status %s from ODC", replyStatus.String())
	}
	log.WithFields(logrus.Fields{
			"odcMsg": configureResponse.Reply.Msg,
			"odcStatus": configureResponse.Reply.Status.String(),
			"odcExectime": configureResponse.Reply.Exectime,
			"odcRunid": configureResponse.Reply.Runid,
			"odcSessionid": configureResponse.Reply.Sessionid,
		}).
		Debug("call to ODC complete")
	return err
}

func printGrpcError(err error) error {
	// We must process the error explicitly here, otherwise we get an error because gRPC's
	// Status is different from what gogoproto expects.
	grpcStatus, ok := status.FromError(err)
	if ok {
		log.WithFields(logrus.Fields{
			"code": grpcStatus.Code().String(),
			"message": grpcStatus.Message(),
			"details": grpcStatus.Details(),
			"error": grpcStatus.Err().Error(),
			"ppStatus": pp.Sprint(grpcStatus),
			"ppErr": pp.Sprint(err),
		}).
			Error("transition call error")
		err = fmt.Errorf("occplugin returned %s: %s", grpcStatus.Code().String(), status.Message())
	} else {
		err = errors.New("invalid gRPC status")
		log.WithField("error", "invalid gRPC status").Error("transition call error")
	}
	return err
}