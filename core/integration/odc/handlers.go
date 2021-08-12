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

package odc

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/AliceO2Group/Control/common/logger/infologger"
	"github.com/AliceO2Group/Control/common/utils"
	"github.com/AliceO2Group/Control/core/integration/odc/odcutils"
	odcpb "github.com/AliceO2Group/Control/core/integration/odc/protos"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

func handleGetState(ctx context.Context, odcClient *RpcClient, envId string) (string, error) {
	defer utils.TimeTrackFunction(time.Now(), log.WithPrefix("odcclient"))
	req := &odcpb.StateRequest{
		Partitionid: envId,
		Path:     "",
		Detailed: false,
	}

	var (
		err error = nil
		newState = "UNKNOWN"
		rep *odcpb.StateReply
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
	if replyStatus := rep.Reply.Status; replyStatus != odcpb.ReplyStatus_SUCCESS {
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
	return odcutils.StateForOdcState(newState), err
}

func handleStart(ctx context.Context, odcClient *RpcClient, arguments map[string]string, envId string) error {
	defer utils.TimeTrackFunction(time.Now(), log.WithPrefix("odcclient"))
	req := &odcpb.StartRequest{
		Request:              &odcpb.StateRequest{
			Partitionid: envId,
			Path:     "",
			Detailed: false,
		},
	}

	var err error = nil
	var rep *odcpb.StateReply

	if envId == "" {
		return errors.New("cannot proceed with empty environment id")
	}

	// SetProperties before START
	setPropertiesRequest := &odcpb.SetPropertiesRequest{
		Partitionid: envId,
		Path:       "",
		Properties: make([]*odcpb.Property, len(arguments)),
	}

	// Extract relevant parameters from Arguments payload
	// and build payload for SetProperty+Start
	i := 0
	for k, v := range arguments {
		setPropertiesRequest.Properties[i] = &odcpb.Property{
			Key:   k,
			Value: v,
		}
		i++
	}

	var setPropertiesResponse *odcpb.GeneralReply
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
	if replyStatus := setPropertiesResponse.Status; replyStatus != odcpb.ReplyStatus_SUCCESS {
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

	// The actual START operation starts here
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
	if replyStatus := rep.Reply.Status; replyStatus != odcpb.ReplyStatus_SUCCESS {
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

func handleStop(ctx context.Context, odcClient *RpcClient, arguments map[string]string, envId string) error {
	defer utils.TimeTrackFunction(time.Now(), log.WithPrefix("odcclient"))
	req := &odcpb.StopRequest{
		Request:              &odcpb.StateRequest{
			Partitionid: envId,
			Path:     "",
			Detailed: false,
		},
	}

	var err error = nil
	var rep *odcpb.StateReply

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
	if replyStatus := rep.Reply.Status; replyStatus != odcpb.ReplyStatus_SUCCESS {
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

func handleReset(ctx context.Context, odcClient *RpcClient, arguments map[string]string, envId string) error {
	defer utils.TimeTrackFunction(time.Now(), log.WithPrefix("odcclient"))
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

func handleCleanup(ctx context.Context, odcClient *RpcClient, arguments map[string]string, envId string) error {
	defer utils.TimeTrackFunction(time.Now(), log.WithPrefix("odcclient"))
	if envId == "" {
		return errors.New("cannot proceed with empty environment id")
	}

	// This function tries to perform the regular teardown sequence.
	// Since Shutdown is supposed to work in any state, we don't bail on error.
	err := doReset(ctx, odcClient, arguments, envId)
	if err != nil {
		log.WithError(printGrpcError(err)).
			WithField("level", infologger.IL_Devel).
			WithField("partition", envId).
			Warn("ODC Reset call failed")
	}

	err = doTerminate(ctx, odcClient, arguments, envId)
	if err != nil {
		log.WithError(printGrpcError(err)).
			WithField("level", infologger.IL_Devel).
			WithField("partition", envId).
			Warn("ODC Terminate call failed")
	}

	err = doShutdown(ctx, odcClient, arguments, envId)
	if err != nil {
		log.WithError(printGrpcError(err)).
			WithField("level", infologger.IL_Devel).
			WithField("partition", envId).
			Warn("ODC Shutdown call failed")
	}
	return nil // We clobber the error because nothing can be done for a failed cleanup
}

func doReset(ctx context.Context, odcClient *RpcClient, arguments map[string]string, envId string) error {
	// RESET
	req := &odcpb.ResetRequest{
		Request:              &odcpb.StateRequest{
			Partitionid: envId,
			Path:     "",
			Detailed: false,
		},
	}

	var err error = nil
	var rep *odcpb.StateReply

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
	if replyStatus := rep.Reply.Status; replyStatus != odcpb.ReplyStatus_SUCCESS {
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

func doTerminate(ctx context.Context, odcClient *RpcClient, arguments map[string]string, envId string) error {
	// TERMINATE
	req := &odcpb.TerminateRequest{
		Request:              &odcpb.StateRequest{
			Partitionid: envId,
			Path:     "",
			Detailed: false,
		},
	}

	var err error = nil
	var rep *odcpb.StateReply

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
	if replyStatus := rep.Reply.Status; replyStatus != odcpb.ReplyStatus_SUCCESS {
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

func doShutdown(ctx context.Context, odcClient *RpcClient, arguments map[string]string, envId string) error{
	// SHUTDOWN
	shutdownRequest := &odcpb.ShutdownRequest{
		Partitionid: envId,
	}

	var err error = nil
	var shutdownResponse *odcpb.GeneralReply
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
	if replyStatus := shutdownResponse.Status; replyStatus != odcpb.ReplyStatus_SUCCESS {
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

func handleExit(ctx context.Context, odcClient *RpcClient, arguments map[string]string ) error {
	return nil
}

func handleRun(ctx context.Context, odcClient *RpcClient, arguments map[string]string, envId string) error {
	defer utils.TimeTrackFunction(time.Now(), log.WithPrefix("odcclient"))
	if envId == "" {
		return errors.New("cannot proceed with empty environment id")
	}

	log.Trace("BEGIN handleRun")
	defer log.Trace("END handleRun")

	// RUN request, includes INITIALIZE+SUBMIT+ACTIVATE
	var topology, plugin, resources string
	exists := false

	topology, exists = arguments["topology"]
	if !exists || len(topology) == 0 {
		return errors.New("empty topology received")
	}
	plugin, exists = arguments["plugin"]
	if !exists || len(plugin) == 0 {
		return errors.New("empty plugin received")
	}
	resources, exists = arguments["resources"]
	if !exists || len(resources) == 0 {
		return errors.New("empty resources received")
	}

	runRequest := &odcpb.RunRequest{
		Partitionid: envId,
		Topology: topology,
		Plugin: plugin,
		Resources: resources,
	}

	var err error = nil
	var runResponse *odcpb.GeneralReply

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
	if replyStatus := runResponse.Status; replyStatus != odcpb.ReplyStatus_SUCCESS {
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


func handleConfigure(ctx context.Context, odcClient *RpcClient, arguments map[string]string, topology string, plugin string, resources string, envId string) error {
	defer utils.TimeTrackFunction(time.Now(), log.WithPrefix("odcclient"))
	if envId == "" {
		return errors.New("cannot proceed with empty environment id")
	}

	var err error = nil

	// SetProperties before CONFIGURE
	setPropertiesRequest := &odcpb.SetPropertiesRequest{
		Partitionid: envId,
		Path:       "",
		Properties: make([]*odcpb.Property, len(arguments)),
	}

	// Extract relevant parameters from Arguments payload
	// and build payload for SetProperty+Configure
	i := 0
	for k, v := range arguments {
		setPropertiesRequest.Properties[i] = &odcpb.Property{
			Key:   k,
			Value: v,
		}
		i++
	}

	err = handleRun(ctx, odcClient, map[string]string{
		"topology": topology,
		"plugin": plugin,
		"resources": resources,
	}, envId)
	if err != nil {
		return printGrpcError(err)
	}


	var setPropertiesResponse *odcpb.GeneralReply
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
	if replyStatus := setPropertiesResponse.Status; replyStatus != odcpb.ReplyStatus_SUCCESS {
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
	configureRequest := &odcpb.ConfigureRequest{
		Request:              &odcpb.StateRequest{
			Partitionid: envId,
			Path:     "",
			Detailed: false,
		},
	}

	var configureResponse *odcpb.StateReply
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
	if replyStatus := configureResponse.Reply.Status; replyStatus != odcpb.ReplyStatus_SUCCESS {
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
			}).
			Trace("ODC call error")
		err = fmt.Errorf("ODC returned %s: %s", grpcStatus.Code().String(), grpcStatus.Message())
	} else {
		err = errors.New("invalid gRPC status")
		log.WithField("error", "invalid gRPC status").
			Trace("ODC call error")
	}
	return err
}