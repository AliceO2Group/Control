/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2020-2024 CERN and copyright holders of ALICE O².
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
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/AliceO2Group/Control/common/logger/infologger"
	pb "github.com/AliceO2Group/Control/common/protos"
	"github.com/AliceO2Group/Control/common/utils"
	"github.com/AliceO2Group/Control/core/environment"
	"github.com/AliceO2Group/Control/core/integration/odc/odcutils"
	odcpb "github.com/AliceO2Group/Control/core/integration/odc/protos"
	"github.com/AliceO2Group/Control/core/the"
	"github.com/AliceO2Group/Control/core/workflow/callable"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

const ODC_ERROR_MAX_LENGTH = 250

func handleGetState(ctx context.Context, odcClient *RpcClient, envId string) (string, error) {
	defer utils.TimeTrackFunction(time.Now(), log.WithPrefix("odcclient"))
	req := &odcpb.StateRequest{
		Partitionid: envId,
		Path:        "",
		Detailed:    false,
	}

	var (
		err      error = nil
		newState       = "UNKNOWN"
		rep      *odcpb.StateReply
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
		return newState, fmt.Errorf("code %d from ODC: %s", odcErr.GetCode(), utils.TruncateString(odcErr.GetMsg(), ODC_ERROR_MAX_LENGTH))
	}
	if replyStatus := rep.Reply.Status; replyStatus != odcpb.ReplyStatus_SUCCESS {
		return newState, fmt.Errorf("status %s from ODC", replyStatus.String())
	}
	log.WithFields(logrus.Fields{
		"odcMsg":       rep.Reply.Msg,
		"odcStatus":    rep.Reply.Status.String(),
		"odcExectime":  rep.Reply.Exectime,
		"partition":    rep.Reply.Partitionid,
		"odcSessionid": rep.Reply.Sessionid,
		"odcState":     rep.Reply.State,
	}).
		Debug("call to ODC complete: odc.GetState")
	return odcutils.StateForOdcState(newState), err
}

func handleStart(ctx context.Context, odcClient *RpcClient, arguments map[string]string, paddingTimeout time.Duration, envId string, runNumber uint64, call *callable.Call) error {
	defer utils.TimeTrackFunction(time.Now(), log.WithPrefix("odcclient").WithField("partition", envId))

	var err error = nil
	var rep *odcpb.StateReply

	if envId == "" {
		return errors.New("cannot proceed with empty environment id")
	}

	// SetProperties before START
	setPropertiesRequest := &odcpb.SetPropertiesRequest{
		Partitionid: envId,
		Path:        "",
		Properties:  make([]*odcpb.Property, len(arguments)),
		Runnr:       runNumber,
	}
	// We ask this ODC call to complete within our own DEADLINE, minus 1 second
	ctxDeadline, ok := ctx.Deadline()
	if ok {
		setPropertiesRequest.Timeout = uint32((time.Until(ctxDeadline) - paddingTimeout).Seconds())
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

	payload := map[string]interface{}{
		"odcRequest": &setPropertiesRequest,
	}
	payloadJson, _ := json.Marshal(payload)
	the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
		Name:                call.GetName(),
		OperationName:       call.Func,
		OperationStatus:     pb.OpStatus_STARTED,
		OperationStep:       "perform ODC call: SetProperties",
		OperationStepStatus: pb.OpStatus_STARTED,
		EnvironmentId:       envId,
		Payload:             string(payloadJson[:]),
	})

	var setPropertiesResponse *odcpb.GeneralReply
	setPropertiesResponse, err = odcClient.SetProperties(ctx, setPropertiesRequest, grpc.EmptyCallOption{})
	if err != nil {
		the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
			Name:                call.GetName(),
			OperationName:       call.Func,
			OperationStatus:     pb.OpStatus_DONE_ERROR,
			OperationStep:       "perform ODC call: SetProperties",
			OperationStepStatus: pb.OpStatus_DONE_ERROR,
			EnvironmentId:       envId,
			Payload:             string(payloadJson[:]),
			Error:               err.Error(),
		})

		return printGrpcError(err)
	}

	if setPropertiesResponse == nil {
		err = fmt.Errorf("nil response error")
		log.WithField("partition", envId).WithError(err).
			Debugf("finished call odc.SetProperties, ERROR nil response")

		the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
			Name:                call.GetName(),
			OperationName:       call.Func,
			OperationStatus:     pb.OpStatus_DONE_ERROR,
			OperationStep:       "perform ODC call: SetProperties",
			OperationStepStatus: pb.OpStatus_DONE_ERROR,
			EnvironmentId:       envId,
			Payload:             string(payloadJson[:]),
			Error:               err.Error(),
		})

		// We got a nil response with nil error, this should never happen
		return err
	}

	// We nullify setPropertiesResponse.Hosts because the payload is too large to be included in the outgoing event
	setPropertiesResponse.Hosts = nil

	if odcErr := setPropertiesResponse.GetError(); odcErr != nil {
		log.WithField("partition", envId).
			WithError(err).
			Debugf("finished call odc.SetProperties, ERROR in response payload")

		err = fmt.Errorf("code %d from ODC: %s", odcErr.GetCode(), utils.TruncateString(odcErr.GetMsg(), ODC_ERROR_MAX_LENGTH))

		payload["odcResponse"] = &setPropertiesResponse
		payloadJson, _ = json.Marshal(payload)
		the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
			Name:                call.GetName(),
			OperationName:       call.Func,
			OperationStatus:     pb.OpStatus_DONE_ERROR,
			OperationStep:       "perform ODC call: SetProperties",
			OperationStepStatus: pb.OpStatus_DONE_ERROR,
			EnvironmentId:       envId,
			Payload:             string(payloadJson[:]),
			Error:               err.Error(),
		})

		return err
	}
	if replyStatus := setPropertiesResponse.Status; replyStatus != odcpb.ReplyStatus_SUCCESS {
		log.WithField("partition", envId).
			WithError(err).
			Debugf("finished call odc.SetProperties, bad status in response payload")

		err = fmt.Errorf("status %s from ODC", replyStatus.String())

		payload["odcResponse"] = &setPropertiesResponse
		payloadJson, _ = json.Marshal(payload)
		the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
			Name:                call.GetName(),
			OperationName:       call.Func,
			OperationStatus:     pb.OpStatus_DONE_ERROR,
			OperationStep:       "perform ODC call: SetProperties",
			OperationStepStatus: pb.OpStatus_DONE_ERROR,
			EnvironmentId:       envId,
			Payload:             string(payloadJson[:]),
			Error:               err.Error(),
		})

		return err
	}

	payload["odcResponse"] = &setPropertiesResponse
	payloadJson, _ = json.Marshal(payload)
	the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
		Name:                call.GetName(),
		OperationName:       call.Func,
		OperationStatus:     pb.OpStatus_ONGOING,
		OperationStep:       "perform ODC call: SetProperties",
		OperationStepStatus: pb.OpStatus_DONE_OK,
		EnvironmentId:       envId,
		Payload:             string(payloadJson[:]),
	})

	log.WithField("partition", envId).
		WithFields(logrus.Fields{
			"odcMsg":       setPropertiesResponse.Msg,
			"odcStatus":    setPropertiesResponse.Status.String(),
			"odcExectime":  setPropertiesResponse.Exectime,
			"partition":    setPropertiesResponse.Partitionid,
			"odcSessionid": setPropertiesResponse.Sessionid,
		}).
		Debug("call to ODC complete: odc.SetProperties")

	// The actual START operation starts here
	req := &odcpb.StartRequest{
		Request: &odcpb.StateRequest{
			Partitionid: envId,
			Path:        "",
			Detailed:    false,
			Runnr:       runNumber,
		},
	}
	// We ask this ODC call to complete within our own DEADLINE, minus 1 second
	ctxDeadline, ok = ctx.Deadline()
	if ok {
		req.Request.Timeout = uint32((time.Until(ctxDeadline) - paddingTimeout).Seconds())
	}

	payload = map[string]interface{}{
		"odcRequest": &req,
	}
	payloadJson, _ = json.Marshal(payload)
	the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
		Name:                call.GetName(),
		OperationName:       call.Func,
		OperationStatus:     pb.OpStatus_ONGOING,
		OperationStep:       "perform ODC call: Start",
		OperationStepStatus: pb.OpStatus_STARTED,
		EnvironmentId:       envId,
		Payload:             string(payloadJson[:]),
	})

	rep, err = odcClient.Start(ctx, req, grpc.EmptyCallOption{})
	if err != nil {
		the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
			Name:                call.GetName(),
			OperationName:       call.Func,
			OperationStatus:     pb.OpStatus_DONE_ERROR,
			OperationStep:       "perform ODC call: Start",
			OperationStepStatus: pb.OpStatus_DONE_ERROR,
			EnvironmentId:       envId,
			Payload:             string(payloadJson[:]),
			Error:               err.Error(),
		})

		return printGrpcError(err)
	}

	if rep == nil || rep.Reply == nil {
		err = fmt.Errorf("nil response error")

		the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
			Name:                call.GetName(),
			OperationName:       call.Func,
			OperationStatus:     pb.OpStatus_DONE_ERROR,
			OperationStep:       "perform ODC call: Start",
			OperationStepStatus: pb.OpStatus_DONE_ERROR,
			EnvironmentId:       envId,
			Payload:             string(payloadJson[:]),
			Error:               err.Error(),
		})

		// We got a nil response with nil error, this should never happen
		return err
	}

	// We nullify rep.Devices and rep.Reply.Hosts because the payload is too large to be included in the outgoing event
	rep.Devices = nil
	rep.Reply.Hosts = nil

	if odcErr := rep.Reply.GetError(); odcErr != nil {
		err = fmt.Errorf("code %d from ODC: %s", odcErr.GetCode(), utils.TruncateString(odcErr.GetMsg(), ODC_ERROR_MAX_LENGTH))

		payload["odcResponse"] = &rep
		payloadJson, _ = json.Marshal(payload)
		the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
			Name:                call.GetName(),
			OperationName:       call.Func,
			OperationStatus:     pb.OpStatus_DONE_ERROR,
			OperationStep:       "perform ODC call: Start",
			OperationStepStatus: pb.OpStatus_DONE_ERROR,
			EnvironmentId:       envId,
			Payload:             string(payloadJson[:]),
			Error:               err.Error(),
		})

		return err
	}
	if replyStatus := rep.Reply.Status; replyStatus != odcpb.ReplyStatus_SUCCESS {
		err = fmt.Errorf("status %s from ODC", replyStatus.String())

		payload["odcResponse"] = &rep
		payloadJson, _ = json.Marshal(payload)
		the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
			Name:                call.GetName(),
			OperationName:       call.Func,
			OperationStatus:     pb.OpStatus_DONE_ERROR,
			OperationStep:       "perform ODC call: Start",
			OperationStepStatus: pb.OpStatus_DONE_ERROR,
			EnvironmentId:       envId,
			Payload:             string(payloadJson[:]),
			Error:               err.Error(),
		})

		return err
	}

	payload["odcResponse"] = &rep
	payloadJson, _ = json.Marshal(payload)
	the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
		Name:                call.GetName(),
		OperationName:       call.Func,
		OperationStatus:     pb.OpStatus_DONE_OK,
		OperationStep:       "perform ODC call: Start",
		OperationStepStatus: pb.OpStatus_DONE_OK,
		EnvironmentId:       envId,
		Payload:             string(payloadJson[:]),
	})

	log.WithField("partition", envId).
		WithFields(logrus.Fields{
			"odcMsg":       rep.Reply.Msg,
			"odcStatus":    rep.Reply.Status.String(),
			"odcExectime":  rep.Reply.Exectime,
			"partition":    rep.Reply.Partitionid,
			"odcSessionid": rep.Reply.Sessionid,
		}).
		Debug("call to ODC complete: odc.Start")
	return err
}

func handleStop(ctx context.Context, odcClient *RpcClient, arguments map[string]string, paddingTimeout time.Duration, envId string, runNumber uint64, call *callable.Call) error {
	defer utils.TimeTrackFunction(time.Now(), log.WithPrefix("odcclient").WithField("partition", envId))
	req := &odcpb.StopRequest{
		Request: &odcpb.StateRequest{
			Partitionid: envId,
			Path:        "",
			Detailed:    false,
			Runnr:       runNumber,
		},
	}
	// We ask this ODC call to complete within our own DEADLINE, minus 1 second
	ctxDeadline, ok := ctx.Deadline()
	if ok {
		req.Request.Timeout = uint32((time.Until(ctxDeadline) - paddingTimeout).Seconds())
	}

	var err error = nil
	var rep *odcpb.StateReply

	if envId == "" {
		return errors.New("cannot proceed with empty environment id")
	}

	payload := map[string]interface{}{
		"odcRequest": &req,
	}
	payloadJson, _ := json.Marshal(payload)
	the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
		Name:                call.GetName(),
		OperationName:       call.Func,
		OperationStatus:     pb.OpStatus_STARTED,
		OperationStep:       "perform ODC call: Stop",
		OperationStepStatus: pb.OpStatus_STARTED,
		EnvironmentId:       envId,
		Payload:             string(payloadJson[:]),
	})

	rep, err = odcClient.Stop(ctx, req, grpc.EmptyCallOption{})
	if err != nil {
		the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
			Name:                call.GetName(),
			OperationName:       call.Func,
			OperationStatus:     pb.OpStatus_DONE_ERROR,
			OperationStep:       "perform ODC call: Stop",
			OperationStepStatus: pb.OpStatus_DONE_ERROR,
			EnvironmentId:       envId,
			Payload:             string(payloadJson[:]),
			Error:               err.Error(),
		})

		return printGrpcError(err)
	}

	if rep == nil || rep.Reply == nil {
		err = fmt.Errorf("nil response error")

		the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
			Name:                call.GetName(),
			OperationName:       call.Func,
			OperationStatus:     pb.OpStatus_DONE_ERROR,
			OperationStep:       "perform ODC call: Stop",
			OperationStepStatus: pb.OpStatus_DONE_ERROR,
			EnvironmentId:       envId,
			Payload:             string(payloadJson[:]),
			Error:               err.Error(),
		})

		// We got a nil response with nil error, this should never happen
		return err
	}

	// We nullify rep.Devices and rep.Reply.Hosts because the payload is too large to be included in the outgoing event
	rep.Devices = nil
	rep.Reply.Hosts = nil

	if odcErr := rep.Reply.GetError(); odcErr != nil {
		err = fmt.Errorf("code %d from ODC: %s", odcErr.GetCode(), utils.TruncateString(odcErr.GetMsg(), ODC_ERROR_MAX_LENGTH))

		payload["odcResponse"] = &rep
		payloadJson, _ = json.Marshal(payload)
		the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
			Name:                call.GetName(),
			OperationName:       call.Func,
			OperationStatus:     pb.OpStatus_DONE_ERROR,
			OperationStep:       "perform ODC call: Stop",
			OperationStepStatus: pb.OpStatus_DONE_ERROR,
			EnvironmentId:       envId,
			Payload:             string(payloadJson[:]),
			Error:               err.Error(),
		})

		return err
	}
	if replyStatus := rep.Reply.Status; replyStatus != odcpb.ReplyStatus_SUCCESS {
		err = fmt.Errorf("status %s from ODC", replyStatus.String())

		payload["odcResponse"] = &rep
		payloadJson, _ = json.Marshal(payload)
		the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
			Name:                call.GetName(),
			OperationName:       call.Func,
			OperationStatus:     pb.OpStatus_DONE_ERROR,
			OperationStep:       "perform ODC call: Stop",
			OperationStepStatus: pb.OpStatus_DONE_ERROR,
			EnvironmentId:       envId,
			Payload:             string(payloadJson[:]),
			Error:               err.Error(),
		})

		return err
	}

	payload["odcResponse"] = &rep
	payloadJson, _ = json.Marshal(payload)
	the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
		Name:                call.GetName(),
		OperationName:       call.Func,
		OperationStatus:     pb.OpStatus_DONE_OK,
		OperationStep:       "perform ODC call: Stop",
		OperationStepStatus: pb.OpStatus_DONE_OK,
		EnvironmentId:       envId,
		Payload:             string(payloadJson[:]),
	})

	log.WithField("partition", envId).
		WithFields(logrus.Fields{
			"odcMsg":       rep.Reply.Msg,
			"odcStatus":    rep.Reply.Status.String(),
			"odcExectime":  rep.Reply.Exectime,
			"partition":    rep.Reply.Partitionid,
			"odcSessionid": rep.Reply.Sessionid,
		}).
		Debug("call to ODC complete: odc.Stop")
	return err
}

func handlePartitionTerminate(ctx context.Context, odcClient *RpcClient, arguments map[string]string, paddingTimeout time.Duration, envId string, call *callable.Call) error {
	defer utils.TimeTrackFunction(time.Now(), log.WithPrefix("odcclient").WithField("partition", envId))
	if envId == "" {
		return errors.New("cannot proceed with empty environment id")
	}

	err := doTerminate(ctx, odcClient, arguments, paddingTimeout, envId, call)
	if err != nil {
		return printGrpcError(err)
	}

	err = doShutdown(ctx, odcClient, arguments, paddingTimeout, envId, call)
	if err != nil {
		return printGrpcError(err)
	}
	return nil
}

func handleReset(ctx context.Context, odcClient *RpcClient, arguments map[string]string, paddingTimeout time.Duration, envId string, call *callable.Call) error {
	defer utils.TimeTrackFunction(time.Now(), log.WithPrefix("odcclient").WithField("partition", envId))
	if envId == "" {
		return errors.New("cannot proceed with empty environment id")
	}

	err := doReset(ctx, odcClient, arguments, paddingTimeout, envId, call)
	if err != nil {
		return printGrpcError(err)
	}

	return nil
}

func handleCleanupLegacy(ctx context.Context, odcClient *RpcClient, arguments map[string]string, paddingTimeout time.Duration, envId string, call *callable.Call) error {
	defer utils.TimeTrackFunction(time.Now(), log.WithPrefix("odcclient").WithField("partition", envId))
	if envId == "" {
		return errors.New("cannot proceed with empty environment id")
	}

	// This function tries to perform the regular teardown sequence.
	// Since Shutdown is supposed to work in any state, we don't bail on error.
	err := doReset(ctx, odcClient, arguments, paddingTimeout, envId, call)
	if err != nil {
		log.WithError(printGrpcError(err)).
			WithField("level", infologger.IL_Devel).
			WithField("partition", envId).
			Warn("ODC Reset call failed")
	}

	err = doTerminate(ctx, odcClient, arguments, paddingTimeout, envId, call)
	if err != nil {
		log.WithError(printGrpcError(err)).
			WithField("level", infologger.IL_Devel).
			WithField("partition", envId).
			Warn("ODC Terminate call failed")
	}

	err = doShutdown(ctx, odcClient, arguments, paddingTimeout, envId, call)
	if err != nil {
		log.WithError(printGrpcError(err)).
			WithField("level", infologger.IL_Devel).
			WithField("partition", envId).
			Warn("ODC Shutdown call failed")
	}
	return nil // We clobber the error because nothing can be done for a failed cleanup
}

func handleCleanup(ctx context.Context, odcClient *RpcClient, arguments map[string]string, paddingTimeout time.Duration, envId string, call *callable.Call) error {
	log.WithField("partition", envId).
		Debug("handleCleanup starting")
	defer utils.TimeTrackFunction(time.Now(), log.WithPrefix("odcclient").WithField("partition", envId))

	// First we query ODC for the full list of active partitions
	req := &odcpb.StatusRequest{Running: true}

	var err error = nil
	var rep *odcpb.StatusReply

	payload := map[string]interface{}{
		"odcRequest": &req,
	}
	payloadJson, _ := json.Marshal(payload)
	the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
		Name:                call.GetName(),
		OperationName:       call.Func,
		OperationStatus:     pb.OpStatus_STARTED,
		OperationStep:       "perform ODC call: Status",
		OperationStepStatus: pb.OpStatus_STARTED,
		EnvironmentId:       envId,
		Payload:             string(payloadJson[:]),
	})

	rep, err = odcClient.Status(ctx, req, grpc.EmptyCallOption{})
	if err != nil {
		the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
			Name:                call.GetName(),
			OperationName:       call.Func,
			OperationStatus:     pb.OpStatus_DONE_ERROR,
			OperationStep:       "perform ODC call: Status",
			OperationStepStatus: pb.OpStatus_DONE_ERROR,
			EnvironmentId:       envId,
			Payload:             string(payloadJson[:]),
			Error:               err.Error(),
		})

		return printGrpcError(err)
	}

	if rep == nil || rep.GetStatus() == odcpb.ReplyStatus_UNKNOWN {
		err = fmt.Errorf("nil response error")

		the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
			Name:                call.GetName(),
			OperationName:       call.Func,
			OperationStatus:     pb.OpStatus_DONE_ERROR,
			OperationStep:       "perform ODC call: Status",
			OperationStepStatus: pb.OpStatus_DONE_ERROR,
			EnvironmentId:       envId,
			Payload:             string(payloadJson[:]),
			Error:               err.Error(),
		})

		// We got a nil response with nil error, this should never happen
		return err
	}

	if odcErr := rep.GetError(); odcErr != nil {
		err = fmt.Errorf("code %d from ODC: %s", odcErr.GetCode(), utils.TruncateString(odcErr.GetMsg(), ODC_ERROR_MAX_LENGTH))

		payload["odcResponse"] = &rep
		payloadJson, _ = json.Marshal(payload)
		the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
			Name:                call.GetName(),
			OperationName:       call.Func,
			OperationStatus:     pb.OpStatus_DONE_ERROR,
			OperationStep:       "perform ODC call: Status",
			OperationStepStatus: pb.OpStatus_DONE_ERROR,
			EnvironmentId:       envId,
			Payload:             string(payloadJson[:]),
			Error:               err.Error(),
		})
	}
	if replyStatus := rep.GetStatus(); replyStatus != odcpb.ReplyStatus_SUCCESS {
		err = fmt.Errorf("status %s from ODC", replyStatus.String())

		payload["odcResponse"] = &rep
		payloadJson, _ = json.Marshal(payload)
		the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
			Name:                call.GetName(),
			OperationName:       call.Func,
			OperationStatus:     pb.OpStatus_DONE_ERROR,
			OperationStep:       "perform ODC call: Status",
			OperationStepStatus: pb.OpStatus_DONE_ERROR,
			EnvironmentId:       envId,
			Payload:             string(payloadJson[:]),
			Error:               err.Error(),
		})

		return err
	}

	payload["odcResponse"] = &rep
	payloadJson, _ = json.Marshal(payload)
	the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
		Name:                call.GetName(),
		OperationName:       call.Func,
		OperationStatus:     pb.OpStatus_ONGOING,
		OperationStep:       "perform ODC call: Status",
		OperationStepStatus: pb.OpStatus_DONE_OK,
		EnvironmentId:       envId,
		Payload:             string(payloadJson[:]),
	})

	log.WithField("partition", envId).
		WithFields(logrus.Fields{
			"odcCall":     "Status",
			"odcMsg":      rep.GetMsg(),
			"odcStatus":   rep.GetStatus().String(),
			"odcExectime": rep.GetExectime(),
		}).
		Debug("call to ODC complete: odc.GetStatus")

	partitionIdsKnownToOdc := make([]string, len(rep.GetPartitions()))
	for i, v := range rep.GetPartitions() {
		partitionIdsKnownToOdc[i] = v.Partitionid
	}
	log.WithField("partition", envId).
		Debugf("partitions known to ODC: %s", strings.Join(partitionIdsKnownToOdc, ", "))

	knownEnvs := environment.ManagerInstance().Ids()
	partitionsToClean := make(map[string]struct{})
	for _, odcPartition := range partitionIdsKnownToOdc {
		isOrphan := true
		for _, knownEnv := range knownEnvs {
			if odcPartition == knownEnv.String() { // found a matching env
				isOrphan = false
				break
			}
		}
		if isOrphan { // no env was found for the given ODC partition
			partitionsToClean[odcPartition] = struct{}{}
		}
	}

	// The present function can in principle be called with envId = "", if the cleanup is triggered from
	// outside of an active environment.
	// If an envId is passed, we append it to the list of partitions to clean up just in case, otherwise we
	// ignore it.
	if envId != "" {
		partitionsToClean[envId] = struct{}{}
	}

	partitionsToCleanStr := make([]string, len(partitionsToClean))
	i := 0
	for k := range partitionsToClean {
		partitionsToCleanStr[i] = k
		i++
	}
	log.WithField("partition", envId).
		Debugf("partitions about to be cleaned: %s", strings.Join(partitionIdsKnownToOdc, ", "))

	wg := &sync.WaitGroup{}
	wg.Add(len(partitionsToClean))

	// Then the actual cleanup calls begin, in parallel...
	for odcPartitionId := range partitionsToClean {
		go func(odcPartitionId string) {
			defer wg.Done()
			err = doShutdown(ctx, odcClient, arguments, paddingTimeout, odcPartitionId, call) // FIXME make this parallel
			if err != nil {
				log.WithError(printGrpcError(err)).
					WithField("level", infologger.IL_Devel).
					WithField("partition", odcPartitionId).
					Warn("ODC Shutdown call failed")
			}
		}(odcPartitionId)
	}
	wg.Wait()

	if len(partitionsToClean) == 0 {
		the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
			Name:                call.GetName(),
			OperationName:       call.Func,
			OperationStatus:     pb.OpStatus_DONE_OK,
			OperationStep:       "no cleanup necessary",
			OperationStepStatus: pb.OpStatus_DONE_OK,
			EnvironmentId:       envId,
		})
	} else {
		the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
			Name:                call.GetName(),
			OperationName:       call.Func,
			OperationStatus:     pb.OpStatus_DONE_OK,
			OperationStep:       "perform ODC calls for all known partitions: Shutdown",
			OperationStepStatus: pb.OpStatus_DONE_OK,
			EnvironmentId:       envId,
		})
	}
	return nil // We clobber the error because nothing can be done for a failed cleanup
}

func doReset(ctx context.Context, odcClient *RpcClient, arguments map[string]string, paddingTimeout time.Duration, envId string, call *callable.Call) error {
	// RESET
	req := &odcpb.ResetRequest{
		Request: &odcpb.StateRequest{
			Partitionid: envId,
			Path:        "",
			Detailed:    false,
		},
	}
	// We ask this ODC call to complete within our own DEADLINE, minus 1 second
	ctxDeadline, ok := ctx.Deadline()
	if ok {
		req.Request.Timeout = uint32((time.Until(ctxDeadline) - paddingTimeout).Seconds())
	}

	var err error = nil
	var rep *odcpb.StateReply

	payload := map[string]interface{}{
		"odcRequest": &req,
	}
	payloadJson, _ := json.Marshal(payload)
	the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
		Name:                call.GetName(),
		OperationName:       call.Func,
		OperationStatus:     pb.OpStatus_STARTED,
		OperationStep:       "perform ODC call: Reset",
		OperationStepStatus: pb.OpStatus_STARTED,
		EnvironmentId:       envId,
		Payload:             string(payloadJson[:]),
	})

	rep, err = odcClient.Reset(ctx, req, grpc.EmptyCallOption{})
	if err != nil {
		the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
			Name:                call.GetName(),
			OperationName:       call.Func,
			OperationStatus:     pb.OpStatus_DONE_ERROR,
			OperationStep:       "perform ODC call: Reset",
			OperationStepStatus: pb.OpStatus_DONE_ERROR,
			EnvironmentId:       envId,
			Payload:             string(payloadJson[:]),
			Error:               err.Error(),
		})

		return printGrpcError(err)
	}

	if rep == nil || rep.Reply == nil {
		// We got a nil response with nil error, this should never happen
		err = errors.New("nil response error")

		the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
			Name:                call.GetName(),
			OperationName:       call.Func,
			OperationStatus:     pb.OpStatus_DONE_ERROR,
			OperationStep:       "perform ODC call: Reset",
			OperationStepStatus: pb.OpStatus_DONE_ERROR,
			EnvironmentId:       envId,
			Payload:             string(payloadJson[:]),
			Error:               err.Error(),
		})

		return err
	}

	// We nullify rep.Devices and rep.Reply.Hosts because the payload is too large to be included in the outgoing event
	rep.Devices = nil
	rep.Reply.Hosts = nil

	if odcErr := rep.Reply.GetError(); odcErr != nil {
		err = fmt.Errorf("code %d from ODC: %s", odcErr.GetCode(), utils.TruncateString(odcErr.GetMsg(), ODC_ERROR_MAX_LENGTH))

		payload["odcResponse"] = &rep
		payloadJson, _ = json.Marshal(payload)
		the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
			Name:                call.GetName(),
			OperationName:       call.Func,
			OperationStatus:     pb.OpStatus_DONE_ERROR,
			OperationStep:       "perform ODC call: Reset",
			OperationStepStatus: pb.OpStatus_DONE_ERROR,
			EnvironmentId:       envId,
			Payload:             string(payloadJson[:]),
			Error:               err.Error(),
		})

		return err
	}
	if replyStatus := rep.Reply.Status; replyStatus != odcpb.ReplyStatus_SUCCESS {
		err = fmt.Errorf("status %s from ODC", replyStatus.String())

		payload["odcResponse"] = &rep
		payloadJson, _ = json.Marshal(payload)
		the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
			Name:                call.GetName(),
			OperationName:       call.Func,
			OperationStatus:     pb.OpStatus_DONE_ERROR,
			OperationStep:       "perform ODC call: Reset",
			OperationStepStatus: pb.OpStatus_DONE_ERROR,
			EnvironmentId:       envId,
			Payload:             string(payloadJson[:]),
			Error:               err.Error(),
		})

		return err
	}

	payload["odcResponse"] = &rep
	payloadJson, _ = json.Marshal(payload)
	the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
		Name:                call.GetName(),
		OperationName:       call.Func,
		OperationStatus:     pb.OpStatus_DONE_OK,
		OperationStep:       "perform ODC call: Reset",
		OperationStepStatus: pb.OpStatus_DONE_OK,
		EnvironmentId:       envId,
		Payload:             string(payloadJson[:]),
	})

	log.WithFields(logrus.Fields{
		"odcMsg":       rep.Reply.Msg,
		"odcStatus":    rep.Reply.Status.String(),
		"odcExectime":  rep.Reply.Exectime,
		"partition":    rep.Reply.Partitionid,
		"odcSessionid": rep.Reply.Sessionid,
	}).
		Debug("call to ODC complete: odc.Reset")
	return err
}

func doTerminate(ctx context.Context, odcClient *RpcClient, arguments map[string]string, paddingTimeout time.Duration, envId string, call *callable.Call) error {
	// TERMINATE
	req := &odcpb.TerminateRequest{
		Request: &odcpb.StateRequest{
			Partitionid: envId,
			Path:        "",
			Detailed:    false,
		},
	}
	// We ask this ODC call to complete within our own DEADLINE, minus 1 second
	ctxDeadline, ok := ctx.Deadline()
	if ok {
		req.Request.Timeout = uint32((time.Until(ctxDeadline) - paddingTimeout).Seconds())
	}

	var err error = nil
	var rep *odcpb.StateReply

	payload := map[string]interface{}{
		"odcRequest": &req,
	}
	payloadJson, _ := json.Marshal(payload)
	the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
		Name:                call.GetName(),
		OperationName:       call.Func,
		OperationStatus:     pb.OpStatus_STARTED,
		OperationStep:       "perform ODC call: Terminate",
		OperationStepStatus: pb.OpStatus_STARTED,
		EnvironmentId:       envId,
		Payload:             string(payloadJson[:]),
	})

	rep, err = odcClient.Terminate(ctx, req, grpc.EmptyCallOption{})
	if err != nil {
		the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
			Name:                call.GetName(),
			OperationName:       call.Func,
			OperationStatus:     pb.OpStatus_DONE_ERROR,
			OperationStep:       "perform ODC call: Terminate",
			OperationStepStatus: pb.OpStatus_DONE_ERROR,
			EnvironmentId:       envId,
			Payload:             string(payloadJson[:]),
			Error:               err.Error(),
		})

		return printGrpcError(err)
	}

	if rep == nil || rep.Reply == nil {
		// We got a nil response with nil error, this should never happen
		err = errors.New("nil response error")

		the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
			Name:                call.GetName(),
			OperationName:       call.Func,
			OperationStatus:     pb.OpStatus_DONE_ERROR,
			OperationStep:       "perform ODC call: Terminate",
			OperationStepStatus: pb.OpStatus_DONE_ERROR,
			EnvironmentId:       envId,
			Payload:             string(payloadJson[:]),
			Error:               err.Error(),
		})

		return err
	}

	// We nullify rep.Devices and rep.Reply.Hosts because the payload is too large to be included in the outgoing event
	rep.Devices = nil
	rep.Reply.Hosts = nil

	if odcErr := rep.Reply.GetError(); odcErr != nil {
		err = fmt.Errorf("code %d from ODC: %s", odcErr.GetCode(), utils.TruncateString(odcErr.GetMsg(), ODC_ERROR_MAX_LENGTH))

		payload["odcResponse"] = &rep
		payloadJson, _ = json.Marshal(payload)
		the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
			Name:                call.GetName(),
			OperationName:       call.Func,
			OperationStatus:     pb.OpStatus_DONE_ERROR,
			OperationStep:       "perform ODC call: Terminate",
			OperationStepStatus: pb.OpStatus_DONE_ERROR,
			EnvironmentId:       envId,
			Payload:             string(payloadJson[:]),
			Error:               err.Error(),
		})

		return err
	}
	if replyStatus := rep.Reply.Status; replyStatus != odcpb.ReplyStatus_SUCCESS {
		err = fmt.Errorf("status %s from ODC", replyStatus.String())

		payload["odcResponse"] = &rep
		payloadJson, _ = json.Marshal(payload)
		the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
			Name:                call.GetName(),
			OperationName:       call.Func,
			OperationStatus:     pb.OpStatus_DONE_ERROR,
			OperationStep:       "perform ODC call: Terminate",
			OperationStepStatus: pb.OpStatus_DONE_ERROR,
			EnvironmentId:       envId,
			Payload:             string(payloadJson[:]),
			Error:               err.Error(),
		})

		return err
	}

	payload["odcResponse"] = &rep
	payloadJson, _ = json.Marshal(payload)
	the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
		Name:                call.GetName(),
		OperationName:       call.Func,
		OperationStatus:     pb.OpStatus_DONE_OK,
		OperationStep:       "perform ODC call: Terminate",
		OperationStepStatus: pb.OpStatus_DONE_OK,
		EnvironmentId:       envId,
		Payload:             string(payloadJson[:]),
	})

	log.WithFields(logrus.Fields{
		"odcMsg":       rep.Reply.Msg,
		"odcStatus":    rep.Reply.Status.String(),
		"odcExectime":  rep.Reply.Exectime,
		"partition":    rep.Reply.Partitionid,
		"odcSessionid": rep.Reply.Sessionid,
	}).
		Debug("call to ODC complete: odc.Terminate")
	return err
}

func doShutdown(ctx context.Context, odcClient *RpcClient, arguments map[string]string, paddingTimeout time.Duration, envId string, call *callable.Call) error {
	// SHUTDOWN
	shutdownRequest := &odcpb.ShutdownRequest{
		Partitionid: envId,
	}
	// We ask this ODC call to complete within our own DEADLINE, minus 1 second
	ctxDeadline, ok := ctx.Deadline()
	if ok {
		shutdownRequest.Timeout = uint32((time.Until(ctxDeadline) - paddingTimeout).Seconds())
	}

	var err error = nil
	var shutdownResponse *odcpb.GeneralReply

	payload := map[string]interface{}{
		"odcRequest": &shutdownRequest,
	}
	payloadJson, _ := json.Marshal(payload)
	the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
		Name:                call.GetName(),
		OperationName:       call.Func,
		OperationStatus:     pb.OpStatus_STARTED,
		OperationStep:       "perform ODC call: Shutdown",
		OperationStepStatus: pb.OpStatus_STARTED,
		EnvironmentId:       envId,
		Payload:             string(payloadJson[:]),
	})

	shutdownResponse, err = odcClient.Shutdown(ctx, shutdownRequest, grpc.EmptyCallOption{})
	if err != nil {
		the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
			Name:                call.GetName(),
			OperationName:       call.Func,
			OperationStatus:     pb.OpStatus_DONE_ERROR,
			OperationStep:       "perform ODC call: Shutdown",
			OperationStepStatus: pb.OpStatus_DONE_ERROR,
			EnvironmentId:       envId,
			Payload:             string(payloadJson[:]),
			Error:               err.Error(),
		})

		return printGrpcError(err)
	}

	if shutdownResponse == nil {
		// We got a nil response with nil error, this should never happen
		err = errors.New("nil response error")

		the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
			Name:                call.GetName(),
			OperationName:       call.Func,
			OperationStatus:     pb.OpStatus_DONE_ERROR,
			OperationStep:       "perform ODC call: Shutdown",
			OperationStepStatus: pb.OpStatus_DONE_ERROR,
			EnvironmentId:       envId,
			Payload:             string(payloadJson[:]),
			Error:               err.Error(),
		})

		return err
	}

	// We nullify rep.Hosts because the payload is too large to be included in the outgoing event
	shutdownResponse.Hosts = nil

	if odcErr := shutdownResponse.GetError(); odcErr != nil {
		err = fmt.Errorf("code %d from ODC: %s", odcErr.GetCode(), utils.TruncateString(odcErr.GetMsg(), ODC_ERROR_MAX_LENGTH))

		payload["odcResponse"] = &shutdownResponse
		payloadJson, _ = json.Marshal(payload)
		the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
			Name:                call.GetName(),
			OperationName:       call.Func,
			OperationStatus:     pb.OpStatus_DONE_ERROR,
			OperationStep:       "perform ODC call: Shutdown",
			OperationStepStatus: pb.OpStatus_DONE_ERROR,
			EnvironmentId:       envId,
			Payload:             string(payloadJson[:]),
			Error:               err.Error(),
		})

		return err
	}
	if replyStatus := shutdownResponse.Status; replyStatus != odcpb.ReplyStatus_SUCCESS {
		err = fmt.Errorf("status %s from ODC", replyStatus.String())

		payload["odcResponse"] = &shutdownResponse
		payloadJson, _ = json.Marshal(payload)
		the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
			Name:                call.GetName(),
			OperationName:       call.Func,
			OperationStatus:     pb.OpStatus_DONE_ERROR,
			OperationStep:       "perform ODC call: Shutdown",
			OperationStepStatus: pb.OpStatus_DONE_ERROR,
			EnvironmentId:       envId,
			Payload:             string(payloadJson[:]),
			Error:               err.Error(),
		})

		return err
	}

	payload["odcResponse"] = &shutdownResponse
	payloadJson, _ = json.Marshal(payload)
	the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
		Name:                call.GetName(),
		OperationName:       call.Func,
		OperationStatus:     pb.OpStatus_DONE_OK,
		OperationStep:       "perform ODC call: Shutdown",
		OperationStepStatus: pb.OpStatus_DONE_OK,
		EnvironmentId:       envId,
		Payload:             string(payloadJson[:]),
	})

	log.WithFields(logrus.Fields{
		"odcMsg":       shutdownResponse.Msg,
		"odcStatus":    shutdownResponse.Status.String(),
		"odcExectime":  shutdownResponse.Exectime,
		"partition":    shutdownResponse.Partitionid,
		"odcSessionid": shutdownResponse.Sessionid,
	}).
		Debug("call to ODC complete: odc.Shutdown")
	return err
}

func handleRun(ctx context.Context, odcClient *RpcClient, isManualXml bool, arguments map[string]string, paddingTimeout time.Duration, envId string, call *callable.Call) error {
	defer utils.TimeTrackFunction(time.Now(), log.WithPrefix("odcclient"))
	if envId == "" {
		return errors.New("cannot proceed with empty environment id")
	}

	log.Trace("BEGIN handleRun")
	defer log.Trace("END handleRun")

	// RUN request, includes INITIALIZE+SUBMIT+ACTIVATE
	var (
		topology, script, plugin, resources, extractTopoResourcesS string
		extractTopoResources                                       bool
	)
	exists := false

	topology, exists = arguments["topology"]
	if isManualXml && (!exists || len(topology) == 0) {
		return errors.New("empty topology received")
	}
	script, exists = arguments["script"]
	if !isManualXml && (!exists || len(script) == 0) {
		return errors.New("empty script received")
	}
	extractTopoResourcesS, exists = arguments["extractTopoResources"]
	if exists && len(extractTopoResourcesS) > 0 {
		var err error
		extractTopoResources, err = strconv.ParseBool(extractTopoResourcesS)
		if err != nil {
			return errors.New("invalid extractTopoResources value received")
		}
	}

	// absence of plugin and resources is only a problem if we don't extract resources from topology
	plugin, exists = arguments["plugin"]
	if !extractTopoResources && (!exists || len(plugin) == 0) {
		return errors.New("empty plugin received")
	}
	resources, exists = arguments["resources"]
	if !extractTopoResources && (!exists || len(resources) == 0) {
		return errors.New("empty resources received")
	}

	var runRequest *odcpb.RunRequest
	if extractTopoResources {
		runRequest = &odcpb.RunRequest{
			Partitionid:          envId,
			ExtractTopoResources: extractTopoResources,
		}
	} else {
		runRequest = &odcpb.RunRequest{
			Partitionid: envId,
			Plugin:      plugin,
			Resources:   resources,
		}
	}

	// We ask this ODC call to complete within our own DEADLINE, minus 1 second
	ctxDeadline, ok := ctx.Deadline()
	if ok {
		runRequest.Timeout = uint32((time.Until(ctxDeadline) - paddingTimeout).Seconds())
	}

	if isManualXml {
		runRequest.Topology = topology
	} else {
		runRequest.Script = script
	}

	var err error = nil
	var runResponse *odcpb.GeneralReply

	payload := map[string]interface{}{
		"odcRequest": &runRequest,
	}
	payloadJson, _ := json.Marshal(payload)
	the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
		Name:                call.GetName(),
		OperationName:       call.Func,
		OperationStatus:     pb.OpStatus_STARTED,
		OperationStep:       "perform ODC call: Run",
		OperationStepStatus: pb.OpStatus_STARTED,
		EnvironmentId:       envId,
		Payload:             string(payloadJson[:]),
	})

	runResponse, err = odcClient.Run(ctx, runRequest, grpc.EmptyCallOption{})
	if err != nil {
		the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
			Name:                call.GetName(),
			OperationName:       call.Func,
			OperationStatus:     pb.OpStatus_DONE_ERROR,
			OperationStep:       "perform ODC call: Run",
			OperationStepStatus: pb.OpStatus_DONE_ERROR,
			EnvironmentId:       envId,
			Payload:             string(payloadJson[:]),
			Error:               err.Error(),
		})

		return printGrpcError(err)
	}

	if runResponse == nil {
		errMsg := "nil response error"

		the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
			Name:                call.GetName(),
			OperationName:       call.Func,
			OperationStatus:     pb.OpStatus_DONE_ERROR,
			OperationStep:       "perform ODC call: Run",
			OperationStepStatus: pb.OpStatus_DONE_ERROR,
			EnvironmentId:       envId,
			Payload:             string(payloadJson[:]),
			Error:               err.Error(),
		})

		// We got a nil response with nil error, this should never happen
		return errors.New(errMsg)
	}

	// We nullify runResponse.Hosts because the payload is too large to be included in the outgoing event
	runResponse.Hosts = nil

	if odcErr := runResponse.GetError(); odcErr != nil {
		err = fmt.Errorf("code %d from ODC: %s", odcErr.GetCode(), utils.TruncateString(odcErr.GetMsg(), ODC_ERROR_MAX_LENGTH))
	}
	if replyStatus := runResponse.Status; replyStatus != odcpb.ReplyStatus_SUCCESS {
		payload["odcResponse"] = &runResponse
		payloadJson, _ = json.Marshal(payload)

		errMsg := ""
		if err != nil {
			errMsg = err.Error()
		}
		the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
			Name:                call.GetName(),
			OperationName:       call.Func,
			OperationStatus:     pb.OpStatus_DONE_ERROR,
			OperationStep:       "perform ODC call: Run",
			OperationStepStatus: pb.OpStatus_DONE_ERROR,
			EnvironmentId:       envId,
			Payload:             string(payloadJson[:]),
			Error:               errMsg,
		})

		return fmt.Errorf("status %s from ODC with error %w", replyStatus.String(), err)
	}
	log.WithFields(logrus.Fields{
		"odcMsg":       runResponse.Msg,
		"odcStatus":    runResponse.Status.String(),
		"odcExectime":  runResponse.Exectime,
		"partition":    runResponse.Partitionid,
		"odcSessionid": runResponse.Sessionid,
	}).
		Debug("call to ODC complete: odc.Run")

	payload["odcResponse"] = &runResponse
	payloadJson, _ = json.Marshal(payload)

	the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
		Name:                call.GetName(),
		OperationName:       call.Func,
		OperationStatus:     pb.OpStatus_DONE_OK,
		OperationStep:       "perform ODC call: Run",
		OperationStepStatus: pb.OpStatus_DONE_OK,
		EnvironmentId:       envId,
		Payload:             string(payloadJson[:]),
	})

	return err
}

func handleConfigure(ctx context.Context, odcClient *RpcClient, arguments map[string]string, paddingTimeout time.Duration, envId string, call *callable.Call) error {
	defer utils.TimeTrackFunction(time.Now(), log.WithPrefix("odcclient"))
	if envId == "" {
		return errors.New("cannot proceed with empty environment id")
	}

	var err error = nil

	// SetProperties before CONFIGURE
	setPropertiesRequest := &odcpb.SetPropertiesRequest{
		Partitionid: envId,
		Path:        "",
		Properties:  make([]*odcpb.Property, len(arguments)),
	}
	// We ask this ODC call to complete within our own DEADLINE, minus 1 second
	ctxDeadline, ok := ctx.Deadline()
	if ok {
		setPropertiesRequest.Timeout = uint32((time.Until(ctxDeadline) - paddingTimeout).Seconds())
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

	log.WithField("partition", envId).Debugf("preparing call odc.SetProperties")

	payload := map[string]interface{}{
		"odcRequest": &setPropertiesRequest,
	}
	payloadJson, _ := json.Marshal(payload)
	the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
		Name:                call.GetName(),
		OperationName:       call.Func,
		OperationStatus:     pb.OpStatus_STARTED,
		OperationStep:       "perform ODC call: SetProperties",
		OperationStepStatus: pb.OpStatus_STARTED,
		EnvironmentId:       envId,
		Payload:             string(payloadJson[:]),
	})

	var setPropertiesResponse *odcpb.GeneralReply
	setPropertiesResponse, err = odcClient.SetProperties(ctx, setPropertiesRequest, grpc.EmptyCallOption{})
	if err != nil {
		log.WithField("partition", envId).
			WithError(err).
			Debugf("finished call odc.SetProperties with ERROR")

		the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
			Name:                call.GetName(),
			OperationName:       call.Func,
			OperationStatus:     pb.OpStatus_DONE_ERROR,
			OperationStep:       "perform ODC call: SetProperties",
			OperationStepStatus: pb.OpStatus_DONE_ERROR,
			EnvironmentId:       envId,
			Payload:             string(payloadJson[:]),
			Error:               err.Error(),
		})

		return printGrpcError(err)
	}

	if setPropertiesResponse == nil {
		err = fmt.Errorf("nil response error")
		log.WithField("partition", envId).WithError(err).
			Debugf("finished call odc.SetProperties, ERROR nil response")

		the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
			Name:                call.GetName(),
			OperationName:       call.Func,
			OperationStatus:     pb.OpStatus_DONE_ERROR,
			OperationStep:       "perform ODC call: SetProperties",
			OperationStepStatus: pb.OpStatus_DONE_ERROR,
			EnvironmentId:       envId,
			Payload:             string(payloadJson[:]),
			Error:               err.Error(),
		})

		// We got a nil response with nil error, this should never happen
		return err
	}

	// We nullify setPropertiesResponse.Hosts because the payload is too large to be included in the outgoing event
	setPropertiesResponse.Hosts = nil

	if odcErr := setPropertiesResponse.GetError(); odcErr != nil {
		log.WithField("partition", envId).
			WithError(err).
			Debugf("finished call odc.SetProperties, ERROR in response payload")

		err = fmt.Errorf("code %d from ODC: %s", odcErr.GetCode(), utils.TruncateString(odcErr.GetMsg(), ODC_ERROR_MAX_LENGTH))

		payload["odcResponse"] = &setPropertiesResponse
		payloadJson, _ = json.Marshal(payload)
		the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
			Name:                call.GetName(),
			OperationName:       call.Func,
			OperationStatus:     pb.OpStatus_DONE_ERROR,
			OperationStep:       "perform ODC call: SetProperties",
			OperationStepStatus: pb.OpStatus_DONE_ERROR,
			EnvironmentId:       envId,
			Payload:             string(payloadJson[:]),
			Error:               err.Error(),
		})

		return err
	}
	if replyStatus := setPropertiesResponse.Status; replyStatus != odcpb.ReplyStatus_SUCCESS {
		log.WithField("partition", envId).
			WithError(err).
			Debugf("finished call odc.SetProperties, bad status in response payload")

		err = fmt.Errorf("status %s from ODC", replyStatus.String())

		payload["odcResponse"] = &setPropertiesResponse
		payloadJson, _ = json.Marshal(payload)
		the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
			Name:                call.GetName(),
			OperationName:       call.Func,
			OperationStatus:     pb.OpStatus_DONE_ERROR,
			OperationStep:       "perform ODC call: SetProperties",
			OperationStepStatus: pb.OpStatus_DONE_ERROR,
			EnvironmentId:       envId,
			Payload:             string(payloadJson[:]),
			Error:               err.Error(),
		})

		return err
	}

	log.WithField("partition", envId).Debugf("finished call odc.SetProperties with SUCCESS")

	payload["odcResponse"] = &setPropertiesResponse
	payloadJson, _ = json.Marshal(payload)
	the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
		Name:                call.GetName(),
		OperationName:       call.Func,
		OperationStatus:     pb.OpStatus_ONGOING,
		OperationStep:       "perform ODC call: SetProperties",
		OperationStepStatus: pb.OpStatus_DONE_OK,
		EnvironmentId:       envId,
		Payload:             string(payloadJson[:]),
	})

	log.WithFields(logrus.Fields{
		"odcMsg":       setPropertiesResponse.Msg,
		"odcStatus":    setPropertiesResponse.Status.String(),
		"odcExectime":  setPropertiesResponse.Exectime,
		"partition":    setPropertiesResponse.Partitionid,
		"odcSessionid": setPropertiesResponse.Sessionid,
	}).
		Debug("call to ODC complete: odc.SetProperties")

	// CONFIGURE
	configureRequest := &odcpb.ConfigureRequest{
		Request: &odcpb.StateRequest{
			Partitionid: envId,
			Path:        "",
			Detailed:    false,
		},
	}
	// We ask this ODC call to complete within our own DEADLINE, minus 1 second
	ctxDeadline, ok = ctx.Deadline()
	if ok {
		configureRequest.Request.Timeout = uint32((time.Until(ctxDeadline) - paddingTimeout).Seconds())
	}

	payload = map[string]interface{}{
		"odcRequest": &configureRequest,
	}
	payloadJson, _ = json.Marshal(payload)
	the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
		Name:                call.GetName(),
		OperationName:       call.Func,
		OperationStatus:     pb.OpStatus_ONGOING,
		OperationStep:       "perform ODC call: Configure",
		OperationStepStatus: pb.OpStatus_STARTED,
		EnvironmentId:       envId,
		Payload:             string(payloadJson[:]),
	})

	var configureResponse *odcpb.StateReply
	configureResponse, err = odcClient.Configure(ctx, configureRequest, grpc.EmptyCallOption{})
	if err != nil {
		the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
			Name:                call.GetName(),
			OperationName:       call.Func,
			OperationStatus:     pb.OpStatus_DONE_ERROR,
			OperationStep:       "perform ODC call: Configure",
			OperationStepStatus: pb.OpStatus_DONE_ERROR,
			EnvironmentId:       envId,
			Payload:             string(payloadJson[:]),
			Error:               err.Error(),
		})

		return printGrpcError(err)
	}

	if configureResponse == nil || configureResponse.Reply == nil {
		err = fmt.Errorf("nil response error")

		the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
			Name:                call.GetName(),
			OperationName:       call.Func,
			OperationStatus:     pb.OpStatus_DONE_ERROR,
			OperationStep:       "perform ODC call: Configure",
			OperationStepStatus: pb.OpStatus_DONE_ERROR,
			EnvironmentId:       envId,
			Payload:             string(payloadJson[:]),
			Error:               err.Error(),
		})

		// We got a nil response with nil error, this should never happen
		return err
	}

	// We nullify configureResponse.Devices and configureResponse.Reply.Hosts because the payload is too large to be included in the outgoing event
	configureResponse.Devices = nil
	configureResponse.Reply.Hosts = nil

	if odcErr := configureResponse.Reply.GetError(); odcErr != nil {
		err = fmt.Errorf("code %d from ODC: %s", odcErr.GetCode(), utils.TruncateString(odcErr.GetMsg(), ODC_ERROR_MAX_LENGTH))

		payload["odcResponse"] = &configureResponse
		payloadJson, _ = json.Marshal(payload)
		the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
			Name:                call.GetName(),
			OperationName:       call.Func,
			OperationStatus:     pb.OpStatus_DONE_ERROR,
			OperationStep:       "perform ODC call: Configure",
			OperationStepStatus: pb.OpStatus_DONE_ERROR,
			EnvironmentId:       envId,
			Payload:             string(payloadJson[:]),
			Error:               err.Error(),
		})

		return err
	}
	if replyStatus := configureResponse.Reply.Status; replyStatus != odcpb.ReplyStatus_SUCCESS {
		err = fmt.Errorf("status %s from ODC", replyStatus.String())

		payload["odcResponse"] = &configureResponse
		payloadJson, _ = json.Marshal(payload)
		the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
			Name:                call.GetName(),
			OperationName:       call.Func,
			OperationStatus:     pb.OpStatus_DONE_ERROR,
			OperationStep:       "perform ODC call: Configure",
			OperationStepStatus: pb.OpStatus_DONE_ERROR,
			EnvironmentId:       envId,
			Payload:             string(payloadJson[:]),
			Error:               err.Error(),
		})

		return err
	}

	payload["odcResponse"] = &configureResponse
	payloadJson, _ = json.Marshal(payload)
	the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
		Name:                call.GetName(),
		OperationName:       call.Func,
		OperationStatus:     pb.OpStatus_DONE_OK,
		OperationStep:       "perform ODC call: Configure",
		OperationStepStatus: pb.OpStatus_DONE_OK,
		EnvironmentId:       envId,
		Payload:             string(payloadJson[:]),
	})

	log.WithFields(logrus.Fields{
		"odcMsg":       configureResponse.Reply.Msg,
		"odcStatus":    configureResponse.Reply.Status.String(),
		"odcExectime":  configureResponse.Reply.Exectime,
		"partition":    configureResponse.Reply.Partitionid,
		"odcSessionid": configureResponse.Reply.Sessionid,
	}).
		Debug("call to ODC complete: odc.Configure")
	return err
}

func printGrpcError(err error) error {
	grpcStatus, ok := status.FromError(err)
	if ok {
		log.WithFields(logrus.Fields{
			"code":    grpcStatus.Code().String(),
			"message": grpcStatus.Message(),
			"details": grpcStatus.Details(),
			"error":   grpcStatus.Err().Error(),
		}).
			Trace("ODC call error")
		err = fmt.Errorf("ODC returned %s: %s", grpcStatus.Code().String(), grpcStatus.Message())
	} else {
		if err == nil {
			err = errors.New("nil gRPC status")
		} else {
			err = fmt.Errorf("invalid gRPC status: %w", err)
		}
		log.WithField("error", err.Error()).
			Trace("ODC call error")
	}
	return err
}
