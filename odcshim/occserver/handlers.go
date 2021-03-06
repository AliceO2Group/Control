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

func handleGetState(ctx context.Context, odcClient *odcclient.RpcClient, envId string) (string, error) {
	req := &odc.StateRequest{
		Partitionid: envId,
		Path:     "",
		Detailed: false,
	}

	var (
		err error = nil
		newState = "UNKNOWN"
		rep *odc.StateReply
	)

	if envId == "" {
		return newState, errors.New("cannot proceed with empty environment id")
	}

	rep, err = odcClient.GetState(ctx, req, grpc.EmptyCallOption{})
	if err != nil {
		return newState, printGrpcError(err)
	}

	if rep == nil || rep.Reply == nil {
		// We got a nil response with nil error, this should never happen
		return newState, errors.New("nil response error")
	}

	newState = rep.Reply.State

	if odcErr := rep.Reply.GetError(); odcErr != nil {
		return newState, fmt.Errorf("code %d from ODC: %s", odcErr.GetCode(), odcErr.GetMsg())
	}
	if replyStatus := rep.Reply.Status; replyStatus != odc.ReplyStatus_SUCCESS {
		return newState, fmt.Errorf("status %s from ODC", replyStatus.String())
	}
	log.WithFields(logrus.Fields{
			"odcMsg": rep.Reply.Msg,
			"odcStatus": rep.Reply.Status.String(),
			"odcExectime": rep.Reply.Exectime,
			"odcRunid": rep.Reply.Partitionid,
			"odcSessionid": rep.Reply.Sessionid,
			"odcState": rep.Reply.State,
		}).
		Debug("call to ODC complete")
	return stateForOdcState(newState), err
}

func handleStart(ctx context.Context, odcClient *odcclient.RpcClient, arguments []*pb.ConfigEntry, envId string) error {
	req := &odc.StartRequest{
		Request:              &odc.StateRequest{
			Partitionid: envId,
			Path:     "",
			Detailed: false,
		},
	}

	var err error = nil
	var rep *odc.StateReply

	if envId == "" {
		return errors.New("cannot proceed with empty environment id")
	}

	rep, err = odcClient.Start(ctx, req, grpc.EmptyCallOption{})
	if err != nil {
		return printGrpcError(err)
	}

	if rep == nil || rep.Reply == nil {
		// We got a nil response with nil error, this should never happen
		return errors.New("nil response error")
	}

	if odcErr := rep.Reply.GetError(); odcErr != nil {
		return fmt.Errorf("code %d from ODC: %s", odcErr.GetCode(), odcErr.GetMsg())
	}
	if replyStatus := rep.Reply.Status; replyStatus != odc.ReplyStatus_SUCCESS {
		return fmt.Errorf("status %s from ODC", replyStatus.String())
	}
	log.WithFields(logrus.Fields{
			"odcMsg": rep.Reply.Msg,
			"odcStatus": rep.Reply.Status.String(),
			"odcExectime": rep.Reply.Exectime,
			"odcRunid": rep.Reply.Partitionid,
			"odcSessionid": rep.Reply.Sessionid,
		}).
		Debug("call to ODC complete")
	return err
}

func handleStop(ctx context.Context, odcClient *odcclient.RpcClient, arguments []*pb.ConfigEntry, envId string) error {
	req := &odc.StopRequest{
		Request:              &odc.StateRequest{
			Partitionid: envId,
			Path:     "",
			Detailed: false,
		},
	}

	var err error = nil
	var rep *odc.StateReply

	if envId == "" {
		return errors.New("cannot proceed with empty environment id")
	}

	rep, err = odcClient.Stop(ctx, req, grpc.EmptyCallOption{})
	if err != nil {
		return printGrpcError(err)
	}

	if rep == nil || rep.Reply == nil {
		// We got a nil response with nil error, this should never happen
		return errors.New("nil response error")
	}

	if odcErr := rep.Reply.GetError(); odcErr != nil {
		return fmt.Errorf("code %d from ODC: %s", odcErr.GetCode(), odcErr.GetMsg())
	}
	if replyStatus := rep.Reply.Status; replyStatus != odc.ReplyStatus_SUCCESS {
		return fmt.Errorf("status %s from ODC", replyStatus.String())
	}
	log.WithFields(logrus.Fields{
			"odcMsg": rep.Reply.Msg,
			"odcStatus": rep.Reply.Status.String(),
			"odcExectime": rep.Reply.Exectime,
			"odcRunid": rep.Reply.Partitionid,
			"odcSessionid": rep.Reply.Sessionid,
		}).
		Debug("call to ODC complete")
	return err
}

func handleReset(ctx context.Context, odcClient *odcclient.RpcClient, arguments []*pb.ConfigEntry, envId string) error {
	if envId == "" {
		return errors.New("cannot proceed with empty environment id")
	}

	err := doReset(ctx, odcClient, arguments, envId)
	if err != nil {
		return printGrpcError(err)
	}

	err = doTerminate(ctx, odcClient, arguments, envId)
	if err != nil {
		return printGrpcError(err)
	}

	err = doShutdown(ctx, odcClient, arguments, envId)
	if err != nil {
		return printGrpcError(err)
	}
	return nil
}

func doReset(ctx context.Context, odcClient *odcclient.RpcClient, arguments []*pb.ConfigEntry, envId string) error {
	// RESET
	req := &odc.ResetRequest{
		Request:              &odc.StateRequest{
			Partitionid: envId,
			Path:     "",
			Detailed: false,
		},
	}

	var err error = nil
	var rep *odc.StateReply

	rep, err = odcClient.Reset(ctx, req, grpc.EmptyCallOption{})
	if err != nil {
		return printGrpcError(err)
	}

	if rep == nil || rep.Reply == nil {
		// We got a nil response with nil error, this should never happen
		return errors.New("nil response error")
	}

	if odcErr := rep.Reply.GetError(); odcErr != nil {
		return fmt.Errorf("code %d from ODC: %s", odcErr.GetCode(), odcErr.GetMsg())
	}
	if replyStatus := rep.Reply.Status; replyStatus != odc.ReplyStatus_SUCCESS {
		return fmt.Errorf("status %s from ODC", replyStatus.String())
	}
	log.WithFields(logrus.Fields{
			"odcMsg": rep.Reply.Msg,
			"odcStatus": rep.Reply.Status.String(),
			"odcExectime": rep.Reply.Exectime,
			"odcRunid": rep.Reply.Partitionid,
			"odcSessionid": rep.Reply.Sessionid,
		}).
		Debug("call to ODC complete")
	return err
}

func doTerminate(ctx context.Context, odcClient *odcclient.RpcClient, arguments []*pb.ConfigEntry, envId string) error {
	// TERMINATE
	req := &odc.TerminateRequest{
		Request:              &odc.StateRequest{
			Partitionid: envId,
			Path:     "",
			Detailed: false,
		},
	}

	var err error = nil
	var rep *odc.StateReply

	rep, err = odcClient.Terminate(ctx, req, grpc.EmptyCallOption{})
	if err != nil {
		return printGrpcError(err)
	}

	if rep == nil || rep.Reply == nil {
		// We got a nil response with nil error, this should never happen
		return errors.New("nil response error")
	}

	if odcErr := rep.Reply.GetError(); odcErr != nil {
		return fmt.Errorf("code %d from ODC: %s", odcErr.GetCode(), odcErr.GetMsg())
	}
	if replyStatus := rep.Reply.Status; replyStatus != odc.ReplyStatus_SUCCESS {
		return fmt.Errorf("status %s from ODC", replyStatus.String())
	}
	log.WithFields(logrus.Fields{
			"odcMsg": rep.Reply.Msg,
			"odcStatus": rep.Reply.Status.String(),
			"odcExectime": rep.Reply.Exectime,
			"odcRunid": rep.Reply.Partitionid,
			"odcSessionid": rep.Reply.Sessionid,
		}).
		Debug("call to ODC complete")
	return err
}

func doShutdown(ctx context.Context, odcClient *odcclient.RpcClient, arguments []*pb.ConfigEntry, envId string) error{
	// SHUTDOWN
	shutdownRequest := &odc.ShutdownRequest{
		Partitionid: envId,
	}

	var err error = nil
	var shutdownResponse *odc.GeneralReply
	shutdownResponse, err = odcClient.Shutdown(ctx, shutdownRequest, grpc.EmptyCallOption{})
	if err != nil {
		return printGrpcError(err)
	}

	if shutdownResponse == nil {
		// We got a nil response with nil error, this should never happen
		return errors.New("nil response error")
	}

	if odcErr := shutdownResponse.GetError(); odcErr != nil {
		return fmt.Errorf("code %d from ODC: %s", odcErr.GetCode(), odcErr.GetMsg())
	}
	if replyStatus := shutdownResponse.Status; replyStatus != odc.ReplyStatus_SUCCESS {
		return fmt.Errorf("status %s from ODC", replyStatus.String())
	}
	log.WithFields(logrus.Fields{
		"odcMsg": shutdownResponse.Msg,
		"odcStatus": shutdownResponse.Status.String(),
		"odcExectime": shutdownResponse.Exectime,
		"odcRunid": shutdownResponse.Partitionid,
		"odcSessionid": shutdownResponse.Sessionid,
	}).
		Debug("call to ODC complete")
	return err
}

func handleExit(ctx context.Context, odcClient *odcclient.RpcClient, arguments []*pb.ConfigEntry ) error {
	return nil
}

func handleRun(ctx context.Context, odcClient *odcclient.RpcClient, arguments []*pb.ConfigEntry, envId string) error {
	if envId == "" {
		return errors.New("cannot proceed with empty environment id")
	}

	log.Trace("BEGIN handleRun")
	defer log.Trace("END handleRun")
	// RUN request, includes INITIALIZE+SUBMIT+ACTIVATE
	var topology string
	for _, entry := range arguments {
		if entry.Key == "topology" {
			topology = entry.Value
		}
	}

	if len(topology) == 0 {
		return errors.New("empty topology received")
	}

	runRequest := &odc.RunRequest{
		Partitionid: envId,
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
		err = fmt.Errorf("code %d from ODC: %s", odcErr.GetCode(), odcErr.GetMsg())
	}
	if replyStatus := runResponse.Status; replyStatus != odc.ReplyStatus_SUCCESS {
		return fmt.Errorf("status %s from ODC with error %w", replyStatus.String(), err)
	}
	log.WithFields(logrus.Fields{
			"odcMsg":       runResponse.Msg,
			"odcStatus":    runResponse.Status.String(),
			"odcExectime":  runResponse.Exectime,
			"odcRunid":     runResponse.Partitionid,
			"odcSessionid": runResponse.Sessionid,
		}).
		Debug("call to ODC complete")
	return err
}


func handleConfigure(ctx context.Context, odcClient *odcclient.RpcClient, arguments []*pb.ConfigEntry, topology string, envId string) error {
	if envId == "" {
		return errors.New("cannot proceed with empty environment id")
	}

	var err error = nil

	// SetProperties before CONFIGURE
	setPropertiesRequest := &odc.SetPropertiesRequest{
		Partitionid: envId,
		Path:       "",
		Properties: make([]*odc.Property, len(arguments)),
	}

	// Extract relevant parameters from Arguments payload
	// and build payload for SetProperty+Configure
	for i, entry := range arguments {
		setPropertiesRequest.Properties[i] = &odc.Property{
			Key:   entry.Key,
			Value: entry.Value,
		}
	}

	err = handleRun(ctx, odcClient, []*pb.ConfigEntry{{
		Key:   "topology",
		Value: topology,
	}}, envId)
	if err != nil {
		return printGrpcError(err)
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
			"odcRunid":     setPropertiesResponse.Partitionid,
			"odcSessionid": setPropertiesResponse.Sessionid,
		}).
		Debug("call to ODC complete")


	// CONFIGURE
	configureRequest := &odc.ConfigureRequest{
		Request:              &odc.StateRequest{
			Partitionid: envId,
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
			"odcRunid": configureResponse.Reply.Partitionid,
			"odcSessionid": configureResponse.Reply.Sessionid,
		}).
		Debug("call to ODC complete")
	return err
}

func printGrpcError(err error) error {
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
		err = fmt.Errorf("occplugin returned %s: %s", grpcStatus.Code().String(), grpcStatus.Message())
	} else {
		err = errors.New("invalid gRPC status")
		log.WithField("error", "invalid gRPC status").Error("transition call error")
	}
	return err
}