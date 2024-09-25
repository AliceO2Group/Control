/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2021-2024 CERN and copyright holders of ALICE O².
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

//go:generate protoc --go_out=. --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative --go-grpc_out=require_unimplemented_servers=false:. protos/dcs.proto

package dcs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"dario.cat/mergo"
	"github.com/AliceO2Group/Control/common/event/topic"
	"github.com/AliceO2Group/Control/common/logger/infologger"
	pb "github.com/AliceO2Group/Control/common/protos"
	"github.com/AliceO2Group/Control/common/runtype"
	"github.com/AliceO2Group/Control/common/utils/uid"
	"github.com/AliceO2Group/Control/core/environment"
	"github.com/AliceO2Group/Control/core/integration"
	dcspb "github.com/AliceO2Group/Control/core/integration/dcs/protos"
	"github.com/AliceO2Group/Control/core/the"
	"github.com/AliceO2Group/Control/core/workflow/callable"
	"github.com/jinzhu/copier"
	"github.com/spf13/viper"
	"golang.org/x/exp/maps"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
)

const (
	DCS_DIAL_TIMEOUT       = 2 * time.Hour
	DCS_GENERAL_OP_TIMEOUT = 45 * time.Second
	DCS_TIME_FORMAT        = "2006-01-02 15:04:05.000"
	TOPIC                  = topic.IntegratedService + topic.Separator + "dcs"
)

type Plugin struct {
	dcsHost string
	dcsPort int

	dcsClient   *RpcClient
	pendingEORs map[string] /*envId*/ int64

	detectorMap   DCSDetectorInfoMap
	detectorMapMu sync.RWMutex
}

type DCSDetectors []dcspb.Detector

type DCSDetectorOpAvailabilityMap map[dcspb.Detector]dcspb.DetectorState

type DCSDetectorInfoMap map[dcspb.Detector]*dcspb.DetectorInfo

type ECSDetectorInfoMap map[string]ECSDetectorInfo

func NewPlugin(endpoint string) integration.Plugin {
	u, err := url.Parse(endpoint)
	if err != nil {
		log.WithField("endpoint", endpoint).
			WithError(err).
			Error("bad service endpoint")
		return nil
	}

	portNumber, _ := strconv.Atoi(u.Port())

	newPlugin := &Plugin{
		dcsHost:     u.Hostname(),
		dcsPort:     portNumber,
		dcsClient:   nil,
		pendingEORs: make(map[string]int64),
		detectorMap: make(DCSDetectorInfoMap),
	}
	return newPlugin
}

func (p *Plugin) GetName() string {
	return "dcs"
}

func (p *Plugin) GetPrettyName() string {
	return "DCS"
}

func (p *Plugin) GetEndpoint() string {
	return viper.GetString("dcsServiceEndpoint")
}

func (p *Plugin) GetConnectionState() string {
	if p == nil || p.dcsClient == nil {
		return "UNKNOWN"
	}
	return p.dcsClient.conn.GetState().String()
}

func (p *Plugin) GetData(_ []any) string {
	if p == nil || p.dcsClient == nil {
		return ""
	}

	environmentIds := environment.ManagerInstance().Ids()

	outMap := make(map[string]interface{})
	outMap["partitions"] = p.pendingEorsForEnvs(environmentIds)

	p.detectorMapMu.RLock()
	outMap["detectors"] = p.detectorMap.ToEcsDetectors()
	p.detectorMapMu.RUnlock()

	out, err := json.Marshal(outMap)
	if err != nil {
		return ""
	}
	return string(out[:])
}

func (p *Plugin) GetEnvironmentsData(environmentIds []uid.ID) map[uid.ID]string {
	if p == nil || p.dcsClient == nil {
		return nil
	}

	envMan := environment.ManagerInstance()
	pendingEors := p.pendingEorsForEnvs(environmentIds)

	out := make(map[uid.ID]string)

	detectorMap := p.detectorMap.ToEcsDetectors()
	for _, envId := range environmentIds {
		env, err := envMan.Environment(envId)
		if err != nil {
			log.WithField("partition", envId).
				WithError(err).
				Error("DCS client cannot acquire environment")
			continue
		}

		includedDetectors := env.GetActiveDetectors().StringList()
		includedDetectorsMap := detectorMap.Filtered(includedDetectors)

		pi := PartitionInfo{
			Detectors: includedDetectorsMap,
		}
		if pendingEorStatus, pendingEorExists := pendingEors[envId]; pendingEorExists {
			pi.SorSuccessful = pendingEorStatus
		}

		marshalled, err := json.Marshal(pi)
		if err != nil {
			continue
		}
		out[envId] = string(marshalled[:])
	}

	return out
}

func (p *Plugin) GetEnvironmentsShortData(environmentIds []uid.ID) map[uid.ID]string {
	return nil
}

func (p *Plugin) pendingEorsForEnvs(envIds []uid.ID) map[uid.ID]bool {
	out := make(map[uid.ID]bool)
	for _, envId := range envIds {
		_, pendingEorExists := p.pendingEORs[envId.String()]
		out[envId] = pendingEorExists
	}
	return out
}

func (p *Plugin) updateLastKnownDetectorStates(detectorMatrix []*dcspb.DetectorInfo) {
	p.detectorMapMu.Lock()
	defer p.detectorMapMu.Unlock()

	for _, detInfo := range detectorMatrix {
		dcsDet := detInfo.GetDetector()
		if _, ok := p.detectorMap[dcsDet]; !ok {
			p.detectorMap[dcsDet] = detInfo
		} else {
			// If we're getting a PFR or SOR availability information within the State field of an incoming STATE_CHANGE_EVENT,
			// before processing it as any other state change, we need to update the availability fields
			if detInfo.State == dcspb.DetectorState_PFR_AVAILABLE || detInfo.State == dcspb.DetectorState_PFR_UNAVAILABLE {
				p.detectorMap[dcsDet].PfrAvailability = detInfo.State
			}
			if detInfo.State == dcspb.DetectorState_SOR_AVAILABLE || detInfo.State == dcspb.DetectorState_SOR_UNAVAILABLE {
				p.detectorMap[dcsDet].SorAvailability = detInfo.State
			}

			// if we're getting a STATE_CHANGE event with any non-null state
			if detInfo.State != dcspb.DetectorState_NULL_STATE {
				p.detectorMap[dcsDet].State = detInfo.State
				timestamp, err := time.Parse(DCS_TIME_FORMAT, detInfo.Timestamp)
				if err == nil {
					p.detectorMap[dcsDet].Timestamp = fmt.Sprintf("%d", timestamp.UnixMilli())
				}
			}
		}
	}
}

func (p *Plugin) updateDetectorOpAvailabilities(detectorMatrix []*dcspb.DetectorInfo) {
	p.detectorMapMu.Lock()
	defer p.detectorMapMu.Unlock()

	pfrAvailabilityChangedDetectors := map[dcspb.Detector]struct{}{}
	sorAvailabilityChangedDetectors := map[dcspb.Detector]struct{}{}

	for _, detInfo := range detectorMatrix {
		dcsDet := detInfo.GetDetector()
		if _, ok := p.detectorMap[dcsDet]; !ok {
			pfrAvailabilityChangedDetectors[dcsDet] = struct{}{}
			sorAvailabilityChangedDetectors[dcsDet] = struct{}{}

			p.detectorMap[dcsDet] = detInfo
		} else {
			if p.detectorMap[dcsDet].PfrAvailability != detInfo.PfrAvailability {
				pfrAvailabilityChangedDetectors[dcsDet] = struct{}{}
			}
			if p.detectorMap[dcsDet].SorAvailability != detInfo.SorAvailability {
				sorAvailabilityChangedDetectors[dcsDet] = struct{}{}
			}
			p.detectorMap[dcsDet].PfrAvailability = detInfo.PfrAvailability
			p.detectorMap[dcsDet].SorAvailability = detInfo.SorAvailability
			timestamp, err := time.Parse(DCS_TIME_FORMAT, detInfo.Timestamp)
			if err == nil {
				p.detectorMap[dcsDet].Timestamp = fmt.Sprintf("%d", timestamp.UnixMilli())
			}
		}
	}

	// build payload for event
	payload := map[string]interface{}{
		"detectors": p.detectorMap.ToEcsDetectors(),
		"changed": map[string]interface{}{
			"pfrAvailability": DCSDetectors(maps.Keys(pfrAvailabilityChangedDetectors)).EcsDetectorsSlice(),
			"sorAvailability": DCSDetectors(maps.Keys(sorAvailabilityChangedDetectors)).EcsDetectorsSlice(),
		},
	}

	payloadJson, _ := json.Marshal(payload)
	the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
		Name:    "dcs.updateDetectorOpAvailabilities",
		Payload: string(payloadJson[:]),
	})
}

func (p *Plugin) Init(instanceId string) error {
	if p.dcsClient == nil {
		cxt, cancel := context.WithCancel(context.Background())
		p.dcsClient = NewClient(cxt, cancel, viper.GetString("dcsServiceEndpoint"))
		if p.dcsClient == nil {
			return fmt.Errorf("failed to connect to DCS service on %s", viper.GetString("dcsServiceEndpoint"))
		}

		in := &dcspb.SubscriptionRequest{
			InstanceId: instanceId,
		}
		evStream, err := p.dcsClient.Subscribe(context.Background(), in, grpc.EmptyCallOption{})
		if err != nil {
			return fmt.Errorf("failed to subscribe to DCS service on %s, possible network issue or DCS gateway malfunction", viper.GetString("dcsServiceEndpoint"))
		}
		go func() {
			for {
				for {
					if evStream == nil {
						break
					}
					ev, streamErr := evStream.Recv()
					if streamErr == io.EOF {
						log.Info("unexpected EOF from DCS service, possible DCS gateway malfunction")
						break
					}

					if streamErr != nil {
						log.WithError(streamErr).
							Error("stream error or bad event from DCS service, dropping stream")
						time.Sleep(3 * time.Second)
						break
					}

					if ev != nil && ev.Eventtype == dcspb.EventType_HEARTBEAT {
						log.Trace("received DCS heartbeat event")
						if dm := ev.GetDetectorMatrix(); len(dm) > 0 {
							p.updateDetectorOpAvailabilities(dm)
						}
						continue
					}

					if ev != nil && ev.Eventtype == dcspb.EventType_STATE_CHANGE_EVENT {
						log.Trace("received DCS state change event")
						if dm := ev.GetDetectorMatrix(); len(dm) > 0 {
							p.updateLastKnownDetectorStates(dm)
						}
						continue
					}

					log.WithField("event", ev.String()).
						Debug("received DCS event")
				}

				log.WithField("endpoint", viper.GetString("dcsServiceEndpoint")).
					Info("DCS stream dropped, attempting reconnect")

				evStream, err = p.dcsClient.Subscribe(context.Background(), in, grpc.EmptyCallOption{})
				if err != nil {
					log.WithField("endpoint", viper.GetString("dcsServiceEndpoint")).
						WithError(err).
						Warnf("failed to resubscribe to DCS service, possible network issue or DCS gateway malfunction")
					time.Sleep(3 * time.Second)
				} else {
					log.WithField("endpoint", viper.GetString("dcsServiceEndpoint")).
						WithField("level", infologger.IL_Support).
						Info("successfully resubscribed to DCS service")
				}
			}
		}()
	}
	if p.dcsClient == nil {
		return fmt.Errorf("failed to start DCS client on %s", viper.GetString("dcsServiceEndpoint"))
	}
	log.Debug("DCS plugin initialized")
	return nil
}

func (p *Plugin) ObjectStack(_ map[string]string, _ map[string]string) (stack map[string]interface{}) {
	stack = make(map[string]interface{})
	return stack
}

func (p *Plugin) getDetectorsPfrAvailability(dcsDetectors DCSDetectors) DCSDetectorOpAvailabilityMap {
	p.detectorMapMu.RLock()
	defer p.detectorMapMu.RUnlock()

	availabilityMap := make(DCSDetectorOpAvailabilityMap)

	for _, dcsDet := range dcsDetectors {
		if dcsDet == dcspb.Detector_DCS {
			continue
		}
		availabilityMap[dcsDet] = dcspb.DetectorState_NULL_STATE

		if _, contains := p.detectorMap[dcsDet]; contains {
			availabilityMap[dcsDet] = p.detectorMap[dcsDet].PfrAvailability
		}
	}

	return availabilityMap
}

func (p *Plugin) getDetectorsSorAvailability(dcsDetectors DCSDetectors) DCSDetectorOpAvailabilityMap {
	p.detectorMapMu.RLock()
	defer p.detectorMapMu.RUnlock()

	availabilityMap := make(DCSDetectorOpAvailabilityMap)

	for _, dcsDet := range dcsDetectors {
		if dcsDet == dcspb.Detector_DCS {
			continue
		}
		availabilityMap[dcsDet] = dcspb.DetectorState_NULL_STATE

		if _, contains := p.detectorMap[dcsDet]; contains {
			availabilityMap[dcsDet] = p.detectorMap[dcsDet].SorAvailability
		}
	}

	return availabilityMap
}

func (p *Plugin) CallStack(data interface{}) (stack map[string]interface{}) {
	call, callOk := data.(*callable.Call)
	if !callOk {
		return
	}
	varStack := call.VarStack
	envId, envOk := varStack["environment_id"]
	if !envOk {
		log.WithField("call", "PrepareForRun").
			Error("cannot acquire environment ID")
		return
	}

	stack = make(map[string]interface{})
	stack["PrepareForRun"] = func() (out string) { // must formally return string even when we return nothing
		var err error
		callFailedStr := "DCS PrepareForRun call failed"

		dcsDetectorsParam, ok := varStack["dcs_detectors"]
		if !ok {
			log.WithField("partition", envId).
				WithField("call", "PrepareForRun").
				Debug("empty DCS detectors list provided")
			dcsDetectorsParam = "[\"NULL_DETECTOR\"]"
		}

		dcsDetectors, err := p.parseDetectors(dcsDetectorsParam)
		if err != nil {
			log.WithError(err).
				WithField("level", infologger.IL_Ops).
				WithField("partition", envId).
				WithField("call", "PrepareForRun").
				Error("DCS error")

			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr

			return
		}

		// We acquire a grace period during which we hope that DCS will become compatible with the operation.
		// During this period we'll keep checking our internal state for op compatibility as reported by DCS at 1Hz,
		// and if we don't get a compatible state within the grace period, we declare the operation failed.
		pfrGracePeriod := time.Duration(0)
		pfrGracePeriodS, ok := varStack["dcs_pfr_grace_period"]
		if ok {
			pfrGracePeriod, err = time.ParseDuration(pfrGracePeriodS)
			if err != nil {
				log.WithError(err).
					WithField("level", infologger.IL_Ops).
					WithField("partition", envId).
					WithField("call", "PrepareForRun").
					Warnf("cannot parse DCS PFR grace period, assuming 0 seconds")
			}
		} else {
			log.WithField("level", infologger.IL_Ops).
				WithField("partition", envId).
				WithField("call", "PrepareForRun").
				Info("DCS PFR grace period not set, defaulting to 0 seconds")
		}

		payload := map[string]interface{}{
			"detectors": dcsDetectors.EcsDetectorsSlice(),
		}
		payloadJson, _ := json.Marshal(payload)

		the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
			Name:                call.GetName(),
			OperationName:       call.Func,
			OperationStatus:     pb.OpStatus_STARTED,
			OperationStep:       "acquire detectors availability",
			OperationStepStatus: pb.OpStatus_STARTED,
			EnvironmentId:       envId,
			Payload:             string(payloadJson[:]),
		})

		pfrGraceTimeout := time.Now().Add(pfrGracePeriod)
		isCompatibleWithOperation := false

		knownDetectorStates := p.getDetectorsPfrAvailability(dcsDetectors)
		isCompatibleWithOperation, err = knownDetectorStates.compatibleWithDCSOperation(dcspb.DetectorState_PFR_AVAILABLE)

		for {
			if isCompatibleWithOperation {
				break
			} else {
				log.WithField("level", infologger.IL_Ops).
					WithField("partition", envId).
					WithField("call", "PrepareForRun").
					WithField("grace_period", pfrGracePeriod.String()).
					WithField("remaining_grace_period", pfrGraceTimeout.Sub(time.Now()).String()).
					Infof("waiting for DCS operation readiness: %s", err.Error())
				time.Sleep(1 * time.Second)
			}

			if time.Now().Before(pfrGraceTimeout) {
				knownDetectorStates = p.getDetectorsPfrAvailability(dcsDetectors)
				isCompatibleWithOperation, err = knownDetectorStates.compatibleWithDCSOperation(dcspb.DetectorState_PFR_AVAILABLE)
			} else {
				break
			}
		}

		if !isCompatibleWithOperation {
			log.WithError(err).
				WithField("level", infologger.IL_Ops).
				WithField("partition", envId).
				WithField("call", "PrepareForRun").
				Error("DCS error")

			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr

			the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
				Name:                call.GetName(),
				OperationName:       call.Func,
				OperationStatus:     pb.OpStatus_DONE_ERROR,
				OperationStep:       "acquire detectors availability",
				OperationStepStatus: pb.OpStatus_DONE_ERROR,
				EnvironmentId:       envId,
				Payload:             string(payloadJson[:]),
				Error:               err.Error(),
			})

			return
		} else if isCompatibleWithOperation && err != nil {
			log.WithField("level", infologger.IL_Ops).
				WithField("partition", envId).
				WithField("call", "PrepareForRun").
				Warnf("cannot determine PFR readiness: %s", err.Error())
		}

		payload["detectorsReadiness"] = knownDetectorStates.EcsDetectorsMap()
		payloadJson, _ = json.Marshal(payload)

		the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
			Name:                call.GetName(),
			OperationName:       call.Func,
			OperationStatus:     pb.OpStatus_ONGOING,
			OperationStep:       "acquire detectors availability",
			OperationStepStatus: pb.OpStatus_DONE_OK,
			EnvironmentId:       envId,
			Payload:             string(payloadJson[:]),
		})

		// By now the DCS must be in a compatible state, so we proceed with gathering params for the operation

		log.WithField("partition", envId).
			WithField("level", infologger.IL_Ops).
			Infof("performing DCS PFR for detectors: %s", strings.Join(dcsDetectors.EcsDetectorsSlice(), " "))

		parameters, ok := varStack["dcs_sor_parameters"]
		if !ok {
			log.WithField("partition", envId).
				WithField("call", "PrepareForRun").
				Debug("no DCS SOR parameters set")
			parameters = "{}"
		}

		argMap := make(map[string]string)
		bytes := []byte(parameters)
		err = json.Unmarshal(bytes, &argMap)
		if err != nil {
			err = fmt.Errorf("error processing DCS SOR parameters: %w", err)
			log.WithError(err).
				WithField("partition", envId).
				WithField("level", infologger.IL_Ops).
				WithField("call", "PrepareForRun").
				Error("DCS error")

			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr

			return
		}

		rt := dcspb.RunType_TECHNICAL
		runTypeS, ok := varStack["run_type"]
		if ok {
			// a detector is defined in the var stack
			// so we convert from the provided string to the correct enum value in common/runtype
			intRt, err := runtype.RunTypeString(runTypeS)
			if err == nil {
				if intRt == runtype.COSMICS {
					// special case for COSMICS runs: there is no DCS run type for it, so we send PHYSICS
					rt = dcspb.RunType_PHYSICS
				} else {
					// the runType was correctly matched to the common/runtype enum, but since the DCS enum is
					// kept compatible, we can directly convert the runtype.RunType to a dcspb.RunType enum value
					rt = dcspb.RunType(intRt)
				}
			}
		}

		// Preparing the per-detector request payload
		in := dcspb.PfrRequest{
			RunType:     rt,
			PartitionId: envId,
			Detectors:   make([]*dcspb.DetectorOperationRequest, len(dcsDetectors)),
		}
		for i, dcsDet := range dcsDetectors {
			ecsDet := dcsToEcsDetector(dcsDet)
			perDetectorParameters, okParam := varStack[strings.ToLower(ecsDet)+"_dcs_sor_parameters"]
			if !okParam {
				log.WithField("partition", envId).
					WithField("call", "PrepareForRun").
					Debug("empty DCS detectors list provided")
				perDetectorParameters = "{}"
			}
			detectorArgMap := make(map[string]string)
			bytes = []byte(perDetectorParameters)
			err = json.Unmarshal(bytes, &detectorArgMap)
			if err != nil {
				err = fmt.Errorf("error processing %s DCS SOR parameter map: %w", ecsDet, err)

				log.WithError(err).
					WithField("level", infologger.IL_Ops).
					WithField("partition", envId).
					WithField("call", "PrepareForRun").
					WithField("detector", ecsDet).
					Error("DCS error")

				call.VarStack["__call_error_reason"] = err.Error()
				call.VarStack["__call_error"] = callFailedStr

				return
			}

			// Per-detector parameters override any general dcs_sor_parameters
			err = mergo.Merge(&detectorArgMap, argMap)
			if err != nil {
				err = fmt.Errorf("error processing %s DCS SOR general parameters override: %w", dcsDet.String(), err)

				log.WithError(err).
					WithField("level", infologger.IL_Ops).
					WithField("partition", envId).
					WithField("call", "PrepareForRun").
					WithField("detector", ecsDet).
					Error("DCS error")

				call.VarStack["__call_error_reason"] = err.Error()
				call.VarStack["__call_error"] = callFailedStr

				return
			}

			// We parse the consolidated per-detector payload for any defaultable parameters with value "default"
			detectorArgMap = resolveDefaults(detectorArgMap, varStack, ecsDet,
				log.WithField("partition", envId).
					WithField("call", "PrepareForRun").
					WithField("detector", ecsDet))

			in.Detectors[i] = &dcspb.DetectorOperationRequest{
				Detector:        dcsDet,
				ExtraParameters: detectorArgMap,
			}
		}

		if p.dcsClient == nil {
			err = fmt.Errorf("DCS plugin not initialized, PrepareForRun impossible")

			log.WithError(err).
				WithField("level", infologger.IL_Ops).
				WithField("endpoint", viper.GetString("dcsServiceEndpoint")).
				WithField("partition", envId).
				WithField("call", "PrepareForRun").
				Error("DCS error")

			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr

			return
		}

		if p.dcsClient.GetConnState() == connectivity.Shutdown {
			err = fmt.Errorf("DCS client connection not available, PrepareForRun impossible")

			log.WithError(err).
				WithField("level", infologger.IL_Ops).
				WithField("endpoint", viper.GetString("dcsServiceEndpoint")).
				WithField("partition", envId).
				WithField("call", "PrepareForRun").
				Error("DCS error")

			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr

			return
		}

		var stream dcspb.Configurator_StartOfRunClient
		timeout := callable.AcquireTimeout(DCS_GENERAL_OP_TIMEOUT, varStack, "PFR", envId)
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		detectorStatusMap := make(map[dcspb.Detector]dcspb.DetectorState)
		for _, v := range dcsDetectors {
			detectorStatusMap[v] = dcspb.DetectorState_NULL_STATE
		}

		the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
			Name:                call.GetName(),
			OperationName:       call.Func,
			OperationStatus:     pb.OpStatus_ONGOING,
			OperationStep:       "perform DCS call: PrepareForRun",
			OperationStepStatus: pb.OpStatus_STARTED,
			EnvironmentId:       envId,
			Payload:             string(payloadJson[:]),
		})

		// Point of no return
		// The gRPC call below is expected to return immediately, with any actual responses arriving subsequently via
		// the response stream.
		// Regardless of DCS SOR success or failure, once the StartOfRun call returns, an EndOfRun **must** be enqueued
		// for later, either during STOP_ACTIVITY or cleanup.
		stream, err = p.dcsClient.PrepareForRun(ctx, &in, grpc.EmptyCallOption{})
		if err != nil {
			log.WithError(err).
				WithField("level", infologger.IL_Ops).
				WithField("endpoint", viper.GetString("dcsServiceEndpoint")).
				WithField("partition", envId).
				WithField("call", "PrepareForRun").
				Error("DCS error")

			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr

			the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
				Name:                call.GetName(),
				OperationName:       call.Func,
				OperationStatus:     pb.OpStatus_DONE_ERROR,
				OperationStep:       "perform DCS call: PrepareForRun",
				OperationStepStatus: pb.OpStatus_DONE_ERROR,
				EnvironmentId:       envId,
				Payload:             string(payloadJson[:]),
				Error:               err.Error(),
			})

			return
		}

		var dcsEvent *dcspb.RunEvent
		for {
			if ctx.Err() != nil {
				err = fmt.Errorf("DCS PrepareForRun context timed out (%s), any future DCS events are ignored", timeout.String())

				the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
					Name:                call.GetName(),
					OperationName:       call.Func,
					OperationStatus:     pb.OpStatus_ONGOING,
					OperationStep:       "perform DCS call: PrepareForRun",
					OperationStepStatus: pb.OpStatus_DONE_TIMEOUT,
					EnvironmentId:       envId,
					Payload:             string(payloadJson[:]),
					Error:               err.Error(),
				})

				break
			}
			dcsEvent, err = stream.Recv()
			if errors.Is(err, io.EOF) { // correct stream termination
				logMsg := "DCS PFR event stream was closed from the DCS side (EOF)"
				log.WithField("partition", envId).
					Debug(logMsg)

				the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
					Name:                call.GetName(),
					OperationName:       call.Func,
					OperationStatus:     pb.OpStatus_ONGOING,
					OperationStep:       "perform DCS call: PrepareForRun",
					OperationStepStatus: pb.OpStatus_DONE_ERROR,
					EnvironmentId:       envId,
					Payload:             string(payloadJson[:]),
					Error:               logMsg,
				})

				break // no more data
			}
			if errors.Is(err, context.DeadlineExceeded) {
				log.WithError(err).
					WithField("partition", envId).
					WithField("timeout", timeout.String()).
					Debug("DCS PFR timed out")
				err = fmt.Errorf("DCS PFR timed out after %s: %w", timeout.String(), err)

				the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
					Name:                call.GetName(),
					OperationName:       call.Func,
					OperationStatus:     pb.OpStatus_ONGOING,
					OperationStep:       "perform DCS call: PrepareForRun",
					OperationStepStatus: pb.OpStatus_DONE_TIMEOUT,
					EnvironmentId:       envId,
					Payload:             string(payloadJson[:]),
					Error:               err.Error(),
				})

				break
			}
			if err != nil { // stream termination in case of general error
				logMsg := "bad DCS PFR event received, any future DCS events are ignored"
				log.WithError(err).
					WithField("partition", envId).
					Warn(logMsg)

				the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
					Name:                call.GetName(),
					OperationName:       call.Func,
					OperationStatus:     pb.OpStatus_ONGOING,
					OperationStep:       "perform DCS call: PrepareForRun",
					OperationStepStatus: pb.OpStatus_DONE_ERROR,
					EnvironmentId:       envId,
					Payload:             string(payloadJson[:]),
					Error:               logMsg,
				})

				break
			}
			if dcsEvent == nil {
				log.WithField("partition", envId).
					Warn("nil DCS PFR event received, skipping to next DCS event")
				continue
			}

			if dcsEvent.GetState() == dcspb.DetectorState_SOR_FAILURE {
				ecsDet := dcsToEcsDetector(dcsEvent.GetDetector())

				logErr := fmt.Errorf("%s PFR failure reported by DCS", ecsDet)
				if err != nil {
					logErr = fmt.Errorf("%v : %v", err, logErr)
				}
				log.WithError(logErr).
					WithField("event", dcsEvent).
					WithField("detector", ecsDet).
					WithField("level", infologger.IL_Ops).
					WithField("endpoint", viper.GetString("dcsServiceEndpoint")).
					WithField("partition", envId).
					WithField("call", "PrepareForRun").
					Error("DCS error")

				call.VarStack["__call_error_reason"] = logErr.Error()
				call.VarStack["__call_error"] = callFailedStr

				payload["detector"] = ecsDet
				payload["dcsEvent"] = dcsEvent
				payloadJson, _ = json.Marshal(payload)
				the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
					Name:                call.GetName(),
					OperationName:       call.Func,
					OperationStatus:     pb.OpStatus_DONE_ERROR,
					OperationStep:       "perform DCS call: PrepareForRun",
					OperationStepStatus: pb.OpStatus_DONE_ERROR,
					EnvironmentId:       envId,
					Payload:             string(payloadJson[:]),
					Error:               logErr.Error(),
				})

				return
			}

			if dcsEvent.GetState() == dcspb.DetectorState_PFR_UNAVAILABLE {
				ecsDet := dcsToEcsDetector(dcsEvent.GetDetector())

				logErr := fmt.Errorf("%s PFR unavailable reported by DCS", ecsDet)
				if err != nil {
					logErr = fmt.Errorf("%v : %v", err, logErr)
				}
				log.WithError(logErr).
					WithField("event", dcsEvent).
					WithField("detector", ecsDet).
					WithField("level", infologger.IL_Ops).
					WithField("endpoint", viper.GetString("dcsServiceEndpoint")).
					WithField("partition", envId).
					WithField("call", "PrepareForRun").
					Error("DCS error")

				call.VarStack["__call_error_reason"] = logErr.Error()
				call.VarStack["__call_error"] = callFailedStr

				payload["detector"] = ecsDet
				payload["dcsEvent"] = dcsEvent
				payloadJson, _ = json.Marshal(payload)
				the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
					Name:                call.GetName(),
					OperationName:       call.Func,
					OperationStatus:     pb.OpStatus_DONE_ERROR,
					OperationStep:       "perform DCS call: PrepareForRun",
					OperationStepStatus: pb.OpStatus_DONE_ERROR,
					EnvironmentId:       envId,
					Payload:             string(payloadJson[:]),
					Error:               logErr.Error(),
				})

				return
			}

			if dcsEvent.GetState() == dcspb.DetectorState_TIMEOUT {
				ecsDet := dcsToEcsDetector(dcsEvent.GetDetector())

				logErr := fmt.Errorf("%s PFR timeout reported by DCS", ecsDet)
				if err != nil {
					logErr = fmt.Errorf("%v : %v", err, logErr)
				}
				log.WithError(logErr).
					WithField("event", dcsEvent).
					WithField("detector", ecsDet).
					WithField("level", infologger.IL_Ops).
					WithField("endpoint", viper.GetString("dcsServiceEndpoint")).
					WithField("partition", envId).
					WithField("call", "PrepareForRun").
					Error("DCS error")

				call.VarStack["__call_error_reason"] = logErr.Error()
				call.VarStack["__call_error"] = callFailedStr

				payload["detector"] = ecsDet
				payload["dcsEvent"] = dcsEvent
				payloadJson, _ = json.Marshal(payload)
				the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
					Name:                call.GetName(),
					OperationName:       call.Func,
					OperationStatus:     pb.OpStatus_DONE_TIMEOUT,
					OperationStep:       "perform DCS call: PrepareForRun",
					OperationStepStatus: pb.OpStatus_DONE_TIMEOUT,
					EnvironmentId:       envId,
					Payload:             string(payloadJson[:]),
					Error:               logErr.Error(),
				})

				return
			}

			detectorStatusMap[dcsEvent.GetDetector()] = dcsEvent.GetState()

			if dcsEvent.GetState() == dcspb.DetectorState_RUN_OK {
				if dcsEvent.GetDetector() == dcspb.Detector_DCS {
					log.WithField("event", dcsEvent).
						WithField("partition", envId).
						WithField("level", infologger.IL_Support).
						Debug("DCS PFR completed successfully")

					detPayload := map[string]interface{}{}
					_ = copier.Copy(&detPayload, payload)
					detPayload["dcsEvent"] = dcsEvent
					detPayloadJson, _ := json.Marshal(detPayload)

					the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
						Name:                call.GetName(),
						OperationName:       call.Func,
						OperationStatus:     pb.OpStatus_ONGOING,
						OperationStep:       "perform DCS call: PrepareForRun",
						OperationStepStatus: pb.OpStatus_ONGOING,
						EnvironmentId:       envId,
						Payload:             string(detPayloadJson[:]),
					})

					break
				} else {
					ecsDet := dcsToEcsDetector(dcsEvent.GetDetector())
					log.WithField("partition", envId).
						WithField("detector", ecsDet).
						Debugf("DCS PFR for %s: received status %s", ecsDet, dcsEvent.GetState().String())

					detPayload := map[string]interface{}{}
					_ = copier.Copy(&detPayload, payload)
					detPayload["detector"] = ecsDet
					detPayload["state"] = dcspb.DetectorState_name[int32(dcsEvent.GetState())]
					detPayload["dcsEvent"] = dcsEvent
					detPayloadJson, _ := json.Marshal(detPayload)

					the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
						Name:                call.GetName(),
						OperationName:       call.Func,
						OperationStatus:     pb.OpStatus_ONGOING,
						OperationStep:       "perform DCS call: PrepareForRun",
						OperationStepStatus: pb.OpStatus_ONGOING,
						EnvironmentId:       envId,
						Payload:             string(detPayloadJson[:]),
					})

				}
			}
			if dcsEvent.GetState() == dcspb.DetectorState_RUN_OK {
				log.WithField("event", dcsEvent).
					WithField("partition", envId).
					WithField("level", infologger.IL_Support).
					Info("ALIECS PFR operation : completed DCS PFR for ")
			} else {
				log.WithField("event", dcsEvent).
					WithField("partition", envId).
					WithField("level", infologger.IL_Devel).
					Info("ALIECS PFR operation : processing DCS PFR for ")

				ecsDet := dcsToEcsDetector(dcsEvent.GetDetector())
				detPayload := map[string]interface{}{}
				_ = copier.Copy(&detPayload, payload)
				detPayload["detector"] = ecsDet
				detPayload["state"] = dcspb.DetectorState_name[int32(dcsEvent.GetState())]
				detPayload["dcsEvent"] = dcsEvent
				detPayloadJson, _ := json.Marshal(detPayload)

				the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
					Name:                call.GetName(),
					OperationName:       call.Func,
					OperationStatus:     pb.OpStatus_ONGOING,
					OperationStep:       "perform DCS call: PrepareForRun",
					OperationStepStatus: pb.OpStatus_ONGOING,
					EnvironmentId:       envId,
					Payload:             string(detPayloadJson[:]),
				})
			}

		}

		dcsFailedEcsDetectors := make([]string, 0)
		dcsopOk := true
		for _, v := range dcsDetectors {
			if detectorStatusMap[v] != dcspb.DetectorState_RUN_OK {
				dcsopOk = false
				dcsFailedEcsDetectors = append(dcsFailedEcsDetectors, dcsToEcsDetector(v))
			}
		}
		if dcsopOk {
			the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
				Name:                call.GetName(),
				OperationName:       call.Func,
				OperationStatus:     pb.OpStatus_DONE_OK,
				OperationStep:       "perform DCS call: PrepareForRun",
				OperationStepStatus: pb.OpStatus_DONE_OK,
				EnvironmentId:       envId,
				Payload:             string(payloadJson[:]),
			})
		} else {
			logErr := fmt.Errorf("PFR failed for %s", strings.Join(dcsFailedEcsDetectors, ", "))
			if err != nil {
				if errors.Is(err, io.EOF) {
					err = fmt.Errorf("DCS PFR stream unexpectedly terminated from DCS side before completion: %w", err)
				}
				logErr = fmt.Errorf("%v : %v", err, logErr)
			}

			log.WithError(logErr).
				WithField("level", infologger.IL_Ops).
				WithField("endpoint", viper.GetString("dcsServiceEndpoint")).
				WithField("partition", envId).
				WithField("call", "PrepareForRun").
				Error("DCS error")

			call.VarStack["__call_error_reason"] = logErr.Error()
			call.VarStack["__call_error"] = callFailedStr

			payload["failedDetectors"] = dcsFailedEcsDetectors
			payloadJson, _ = json.Marshal(payload)

			the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
				Name:                call.GetName(),
				OperationName:       call.Func,
				OperationStatus:     pb.OpStatus_DONE_ERROR,
				OperationStep:       "perform DCS call: PrepareForRun",
				OperationStepStatus: pb.OpStatus_DONE_ERROR,
				EnvironmentId:       envId,
				Payload:             string(payloadJson[:]),
				Error:               logErr.Error(),
			})
		}
		return
	}
	stack["StartOfRun"] = func() (out string) { // must formally return string even when we return nothing
		var err error
		callFailedStr := "DCS StartOfRun call failed"

		rn := varStack["run_number"]
		var runNumber64 int64
		runNumber64, err = strconv.ParseInt(rn, 10, 32)
		if err != nil {
			log.WithField("partition", envId).
				WithField("call", "StartOfRun").
				WithError(err).
				Error("cannot acquire run number for DCS SOR")
		}

		dcsDetectorsParam, ok := varStack["dcs_detectors"]
		if !ok {
			log.WithField("partition", envId).
				WithField("call", "StartOfRun").
				WithField("run", runNumber64).
				Debug("empty DCS detectors list provided")
			dcsDetectorsParam = "[\"NULL_DETECTOR\"]"
		}

		dcsDetectors, err := p.parseDetectors(dcsDetectorsParam)
		if err != nil {
			log.WithError(err).
				WithField("level", infologger.IL_Ops).
				WithField("partition", envId).
				WithField("call", "StartOfRun").
				Error("DCS error")

			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr

			return
		}

		// We acquire a grace period during which we hope that DCS will become compatible with the operation.
		// During this period we'll keep checking our internal state for op compatibility as reported by DCS at 1Hz,
		// and if we don't get a compatible state within the grace period, we declare the operation failed.
		sorGracePeriod := time.Duration(0)
		sorGracePeriodS, ok := varStack["dcs_sor_grace_period"]
		if ok {
			sorGracePeriod, err = time.ParseDuration(sorGracePeriodS)
			if err != nil {
				log.WithError(err).
					WithField("level", infologger.IL_Ops).
					WithField("partition", envId).
					WithField("call", "StartOfRun").
					Warnf("cannot parse DCS SOR grace period, assuming 0 seconds")
			}
		} else {
			log.WithField("level", infologger.IL_Ops).
				WithField("partition", envId).
				WithField("call", "StartOfRun").
				Info("DCS SOR grace period not set, defaulting to 0 seconds")
		}

		payload := map[string]interface{}{
			"detectors": dcsDetectors.EcsDetectorsSlice(),
			"runNumber": runNumber64,
		}
		payloadJson, _ := json.Marshal(payload)

		the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
			Name:                call.GetName(),
			OperationName:       call.Func,
			OperationStatus:     pb.OpStatus_STARTED,
			OperationStep:       "acquire detectors availability",
			OperationStepStatus: pb.OpStatus_STARTED,
			EnvironmentId:       envId,
			Payload:             string(payloadJson[:]),
		})

		sorGraceTimeout := time.Now().Add(sorGracePeriod)
		isCompatibleWithOperation := false

		knownDetectorStates := p.getDetectorsSorAvailability(dcsDetectors)
		isCompatibleWithOperation, err = knownDetectorStates.compatibleWithDCSOperation(dcspb.DetectorState_SOR_AVAILABLE)

		for {
			if isCompatibleWithOperation {
				break
			} else {
				log.WithField("level", infologger.IL_Ops).
					WithField("partition", envId).
					WithField("call", "StartOfRun").
					WithField("grace_period", sorGracePeriod.String()).
					WithField("remaining_grace_period", sorGraceTimeout.Sub(time.Now()).String()).
					Infof("waiting for DCS operation readiness: %s", err.Error())
				time.Sleep(1 * time.Second)
			}

			if time.Now().Before(sorGraceTimeout) {
				knownDetectorStates = p.getDetectorsSorAvailability(dcsDetectors)
				isCompatibleWithOperation, err = knownDetectorStates.compatibleWithDCSOperation(dcspb.DetectorState_SOR_AVAILABLE)
			} else {
				break
			}
		}

		if !isCompatibleWithOperation {
			log.WithError(err).
				WithField("level", infologger.IL_Ops).
				WithField("partition", envId).
				WithField("call", "StartOfRun").
				Error("DCS error")

			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr

			the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
				Name:                call.GetName(),
				OperationName:       call.Func,
				OperationStatus:     pb.OpStatus_DONE_ERROR,
				OperationStep:       "acquire detectors availability",
				OperationStepStatus: pb.OpStatus_DONE_ERROR,
				EnvironmentId:       envId,
				Payload:             string(payloadJson[:]),
				Error:               err.Error(),
			})

			return
		} else if isCompatibleWithOperation && err != nil {
			log.WithField("level", infologger.IL_Ops).
				WithField("partition", envId).
				WithField("call", "StartOfRun").
				Warnf("cannot determine SOR readiness: %s", err.Error())
		}

		payload["detectorsReadiness"] = knownDetectorStates.EcsDetectorsMap()
		payloadJson, _ = json.Marshal(payload)

		the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
			Name:                call.GetName(),
			OperationName:       call.Func,
			OperationStatus:     pb.OpStatus_ONGOING,
			OperationStep:       "acquire detectors availability",
			OperationStepStatus: pb.OpStatus_DONE_OK,
			EnvironmentId:       envId,
			Payload:             string(payloadJson[:]),
		})

		// By now the DCS must be in a compatible state, so we proceed with gathering params for the operation

		log.WithField("partition", envId).
			WithField("level", infologger.IL_Ops).
			WithField("run", runNumber64).
			Infof("performing DCS SOR for detectors: %s", strings.Join(dcsDetectors.EcsDetectorsSlice(), " "))

		parameters, ok := varStack["dcs_sor_parameters"]
		if !ok {
			log.WithField("partition", envId).
				WithField("call", "StartOfRun").
				Debug("no DCS SOR parameters set")
			parameters = "{}"
		}

		argMap := make(map[string]string)
		bytes := []byte(parameters)
		err = json.Unmarshal(bytes, &argMap)
		if err != nil {
			err = fmt.Errorf("error processing DCS SOR parameters: %w", err)
			log.WithError(err).
				WithField("partition", envId).
				WithField("level", infologger.IL_Ops).
				WithField("call", "StartOfRun").
				Error("DCS error")

			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr

			return
		}

		rt := dcspb.RunType_TECHNICAL
		runTypeS, ok := varStack["run_type"]
		if ok {
			// a detector is defined in the var stack
			// so we convert from the provided string to the correct enum value in common/runtype
			intRt, err := runtype.RunTypeString(runTypeS)
			if err == nil {
				if intRt == runtype.COSMICS {
					// special case for COSMICS runs: there is no DCS run type for it, so we send PHYSICS
					rt = dcspb.RunType_PHYSICS
				} else {
					// the runType was correctly matched to the common/runtype enum, but since the DCS enum is
					// kept compatible, we can directly convert the runtype.RunType to a dcspb.RunType enum value
					rt = dcspb.RunType(intRt)
				}
			}
		}

		// Preparing the per-detector request payload
		in := dcspb.SorRequest{
			RunType:   rt,
			RunNumber: int32(runNumber64),
			Detectors: make([]*dcspb.DetectorOperationRequest, len(dcsDetectors)),
		}
		for i, dcsDet := range dcsDetectors {
			ecsDet := dcsToEcsDetector(dcsDet)
			perDetectorParameters, okParam := varStack[strings.ToLower(ecsDet)+"_dcs_sor_parameters"]
			if !okParam {
				log.WithField("partition", envId).
					WithField("run", runNumber64).
					WithField("call", "StartOfRun").
					Debug("empty DCS detectors list provided")
				perDetectorParameters = "{}"
			}
			detectorArgMap := make(map[string]string)
			bytes := []byte(perDetectorParameters)
			err = json.Unmarshal(bytes, &detectorArgMap)
			if err != nil {
				err = fmt.Errorf("error processing %s DCS SOR parameter map: %w", ecsDet, err)

				log.WithError(err).
					WithField("level", infologger.IL_Ops).
					WithField("partition", envId).
					WithField("call", "StartOfRun").
					WithField("detector", ecsDet).
					WithField("run", runNumber64).
					Error("DCS error")

				call.VarStack["__call_error_reason"] = err.Error()
				call.VarStack["__call_error"] = callFailedStr

				return
			}

			// Per-detector parameters override any general dcs_sor_parameters
			err = mergo.Merge(&detectorArgMap, argMap)
			if err != nil {
				err = fmt.Errorf("error processing %s DCS SOR general parameters override: %w", dcsDet.String(), err)

				log.WithError(err).
					WithField("level", infologger.IL_Ops).
					WithField("partition", envId).
					WithField("call", "StartOfRun").
					WithField("detector", ecsDet).
					WithField("run", runNumber64).
					Error("DCS error")

				call.VarStack["__call_error_reason"] = err.Error()
				call.VarStack["__call_error"] = callFailedStr

				return
			}

			// We parse the consolidated per-detector payload for any defaultable parameters with value "default"
			detectorArgMap = resolveDefaults(detectorArgMap, varStack, ecsDet,
				log.WithField("partition", envId).
					WithField("call", "StartOfRun").
					WithField("detector", ecsDet).
					WithField("run", runNumber64))

			in.Detectors[i] = &dcspb.DetectorOperationRequest{
				Detector:        dcsDet,
				ExtraParameters: detectorArgMap,
			}
		}

		if p.dcsClient == nil {
			err = fmt.Errorf("DCS plugin not initialized, StartOfRun impossible")

			log.WithError(err).
				WithField("level", infologger.IL_Ops).
				WithField("endpoint", viper.GetString("dcsServiceEndpoint")).
				WithField("run", runNumber64).
				WithField("partition", envId).
				WithField("call", "StartOfRun").
				Error("DCS error")

			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr

			return
		}

		if p.dcsClient.GetConnState() == connectivity.Shutdown {
			err = fmt.Errorf("DCS client connection not available, StartOfRun impossible")

			log.WithError(err).
				WithField("level", infologger.IL_Ops).
				WithField("endpoint", viper.GetString("dcsServiceEndpoint")).
				WithField("run", runNumber64).
				WithField("partition", envId).
				WithField("call", "StartOfRun").
				Error("DCS error")

			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr

			return
		}

		var stream dcspb.Configurator_StartOfRunClient
		timeout := callable.AcquireTimeout(DCS_GENERAL_OP_TIMEOUT, varStack, "SOR", envId)
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		detectorStatusMap := make(map[dcspb.Detector]dcspb.DetectorState)
		for _, v := range dcsDetectors {
			detectorStatusMap[v] = dcspb.DetectorState_NULL_STATE
		}

		the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
			Name:                call.GetName(),
			OperationName:       call.Func,
			OperationStatus:     pb.OpStatus_ONGOING,
			OperationStep:       "perform DCS call: StartOfRun",
			OperationStepStatus: pb.OpStatus_STARTED,
			EnvironmentId:       envId,
			Payload:             string(payloadJson[:]),
		})

		// Point of no return
		// The gRPC call below is expected to return immediately, with any actual responses arriving subsequently via
		// the response stream.
		// Regardless of DCS SOR success or failure, once the StartOfRun call returns, an EndOfRun **must** be enqueued
		// for later, either during STOP_ACTIVITY or cleanup.
		stream, err = p.dcsClient.StartOfRun(ctx, &in, grpc.EmptyCallOption{})
		if err != nil {
			log.WithError(err).
				WithField("level", infologger.IL_Ops).
				WithField("endpoint", viper.GetString("dcsServiceEndpoint")).
				WithField("run", runNumber64).
				WithField("partition", envId).
				WithField("call", "StartOfRun").
				Error("DCS error")

			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr

			the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
				Name:                call.GetName(),
				OperationName:       call.Func,
				OperationStatus:     pb.OpStatus_DONE_ERROR,
				OperationStep:       "perform DCS call: StartOfRun",
				OperationStepStatus: pb.OpStatus_DONE_ERROR,
				EnvironmentId:       envId,
				Payload:             string(payloadJson[:]),
				Error:               err.Error(),
			})

			return
		}
		p.pendingEORs[envId] = runNumber64 // make sure the corresponding EOR runs sooner or later

		var dcsEvent *dcspb.RunEvent
		for {
			if ctx.Err() != nil {
				err = fmt.Errorf("DCS StartOfRun context timed out (%s), any future DCS events are ignored", timeout.String())

				the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
					Name:                call.GetName(),
					OperationName:       call.Func,
					OperationStatus:     pb.OpStatus_ONGOING,
					OperationStep:       "perform DCS call: StartOfRun",
					OperationStepStatus: pb.OpStatus_DONE_TIMEOUT,
					EnvironmentId:       envId,
					Payload:             string(payloadJson[:]),
					Error:               err.Error(),
				})

				break
			}
			dcsEvent, err = stream.Recv()
			if errors.Is(err, io.EOF) { // correct stream termination
				logMsg := "DCS SOR event stream was closed from the DCS side (EOF)"
				log.WithField("partition", envId).
					WithField("run", runNumber64).
					Debug(logMsg)

				break // no more data
			}
			if errors.Is(err, context.DeadlineExceeded) {
				log.WithError(err).
					WithField("partition", envId).
					WithField("run", runNumber64).
					WithField("timeout", timeout.String()).
					Debug("DCS SOR timed out")
				err = fmt.Errorf("DCS SOR timed out after %s: %w", timeout.String(), err)

				the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
					Name:                call.GetName(),
					OperationName:       call.Func,
					OperationStatus:     pb.OpStatus_ONGOING,
					OperationStep:       "perform DCS call: StartOfRun",
					OperationStepStatus: pb.OpStatus_DONE_TIMEOUT,
					EnvironmentId:       envId,
					Payload:             string(payloadJson[:]),
					Error:               err.Error(),
				})

				break
			}
			if err != nil { // stream termination in case of general error
				logMsg := "bad DCS SOR event received, any future DCS events are ignored"
				log.WithError(err).
					WithField("partition", envId).
					WithField("run", runNumber64).
					Warn(logMsg)

				the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
					Name:                call.GetName(),
					OperationName:       call.Func,
					OperationStatus:     pb.OpStatus_ONGOING,
					OperationStep:       "perform DCS call: StartOfRun",
					OperationStepStatus: pb.OpStatus_DONE_ERROR,
					EnvironmentId:       envId,
					Payload:             string(payloadJson[:]),
					Error:               logMsg,
				})

				break
			}
			if dcsEvent == nil {
				log.WithField("partition", envId).
					WithField("run", runNumber64).
					Warn("nil DCS SOR event received, skipping to next DCS event")
				continue
			}

			if dcsEvent.GetState() == dcspb.DetectorState_SOR_FAILURE {
				ecsDet := dcsToEcsDetector(dcsEvent.GetDetector())

				logErr := fmt.Errorf("%s SOR failure reported by DCS", ecsDet)
				if err != nil {
					logErr = fmt.Errorf("%v : %v", err, logErr)
				}
				log.WithError(logErr).
					WithField("event", dcsEvent).
					WithField("detector", ecsDet).
					WithField("level", infologger.IL_Ops).
					WithField("endpoint", viper.GetString("dcsServiceEndpoint")).
					WithField("run", runNumber64).
					WithField("partition", envId).
					WithField("call", "StartOfRun").
					Error("DCS error")

				call.VarStack["__call_error_reason"] = logErr.Error()
				call.VarStack["__call_error"] = callFailedStr

				payload["detector"] = ecsDet
				payload["dcsEvent"] = dcsEvent
				payloadJson, _ = json.Marshal(payload)
				the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
					Name:                call.GetName(),
					OperationName:       call.Func,
					OperationStatus:     pb.OpStatus_DONE_ERROR,
					OperationStep:       "perform DCS call: StartOfRun",
					OperationStepStatus: pb.OpStatus_DONE_ERROR,
					EnvironmentId:       envId,
					Payload:             string(payloadJson[:]),
					Error:               logErr.Error(),
				})

				return
			}

			if dcsEvent.GetState() == dcspb.DetectorState_SOR_UNAVAILABLE {
				ecsDet := dcsToEcsDetector(dcsEvent.GetDetector())

				logErr := fmt.Errorf("%s SOR unavailable reported by DCS", ecsDet)
				if err != nil {
					logErr = fmt.Errorf("%v : %v", err, logErr)
				}
				log.WithError(logErr).
					WithField("event", dcsEvent).
					WithField("detector", ecsDet).
					WithField("level", infologger.IL_Ops).
					WithField("endpoint", viper.GetString("dcsServiceEndpoint")).
					WithField("run", runNumber64).
					WithField("partition", envId).
					WithField("call", "StartOfRun").
					Error("DCS error")

				call.VarStack["__call_error_reason"] = logErr.Error()
				call.VarStack["__call_error"] = callFailedStr

				payload["detector"] = ecsDet
				payload["dcsEvent"] = dcsEvent
				payloadJson, _ = json.Marshal(payload)
				the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
					Name:                call.GetName(),
					OperationName:       call.Func,
					OperationStatus:     pb.OpStatus_DONE_ERROR,
					OperationStep:       "perform DCS call: StartOfRun",
					OperationStepStatus: pb.OpStatus_DONE_ERROR,
					EnvironmentId:       envId,
					Payload:             string(payloadJson[:]),
					Error:               logErr.Error(),
				})

				return
			}

			if dcsEvent.GetState() == dcspb.DetectorState_TIMEOUT {
				ecsDet := dcsToEcsDetector(dcsEvent.GetDetector())

				logErr := fmt.Errorf("%s SOR timeout reported by DCS", ecsDet)
				if err != nil {
					logErr = fmt.Errorf("%v : %v", err, logErr)
				}
				log.WithError(logErr).
					WithField("event", dcsEvent).
					WithField("detector", ecsDet).
					WithField("level", infologger.IL_Ops).
					WithField("endpoint", viper.GetString("dcsServiceEndpoint")).
					WithField("run", runNumber64).
					WithField("partition", envId).
					WithField("call", "StartOfRun").
					Error("DCS error")

				call.VarStack["__call_error_reason"] = logErr.Error()
				call.VarStack["__call_error"] = callFailedStr

				payload["detector"] = ecsDet
				payload["dcsEvent"] = dcsEvent
				payloadJson, _ = json.Marshal(payload)
				the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
					Name:                call.GetName(),
					OperationName:       call.Func,
					OperationStatus:     pb.OpStatus_DONE_TIMEOUT,
					OperationStep:       "perform DCS call: StartOfRun",
					OperationStepStatus: pb.OpStatus_DONE_TIMEOUT,
					EnvironmentId:       envId,
					Payload:             string(payloadJson[:]),
					Error:               logErr.Error(),
				})

				return
			}

			detectorStatusMap[dcsEvent.GetDetector()] = dcsEvent.GetState()

			if dcsEvent.GetState() == dcspb.DetectorState_RUN_OK {
				if dcsEvent.GetDetector() == dcspb.Detector_DCS {
					log.WithField("event", dcsEvent).
						WithField("partition", envId).
						WithField("run", runNumber64).
						WithField("level", infologger.IL_Support).
						Debug("DCS SOR completed successfully")
					p.pendingEORs[envId] = runNumber64

					detPayload := map[string]interface{}{}
					_ = copier.Copy(&detPayload, payload)
					detPayload["dcsEvent"] = dcsEvent
					detPayloadJson, _ := json.Marshal(detPayload)

					the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
						Name:                call.GetName(),
						OperationName:       call.Func,
						OperationStatus:     pb.OpStatus_ONGOING,
						OperationStep:       "perform DCS call: StartOfRun",
						OperationStepStatus: pb.OpStatus_ONGOING,
						EnvironmentId:       envId,
						Payload:             string(detPayloadJson[:]),
					})

					break
				} else {
					ecsDet := dcsToEcsDetector(dcsEvent.GetDetector())
					log.WithField("partition", envId).
						WithField("run", runNumber64).
						WithField("detector", ecsDet).
						Debugf("DCS SOR for %s: received status %s", ecsDet, dcsEvent.GetState().String())

					detPayload := map[string]interface{}{}
					_ = copier.Copy(&detPayload, payload)
					detPayload["detector"] = ecsDet
					detPayload["dcsEvent"] = dcsEvent
					detPayloadJson, _ := json.Marshal(detPayload)

					the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
						Name:                call.GetName(),
						OperationName:       call.Func,
						OperationStatus:     pb.OpStatus_ONGOING,
						OperationStep:       "perform DCS call: StartOfRun",
						OperationStepStatus: pb.OpStatus_ONGOING,
						EnvironmentId:       envId,
						Payload:             string(detPayloadJson[:]),
					})

				}
			}
			if dcsEvent.GetState() == dcspb.DetectorState_RUN_OK {
				log.WithField("event", dcsEvent).
					WithField("partition", envId).
					WithField("level", infologger.IL_Support).
					WithField("run", runNumber64).
					Info("ALIECS SOR operation : completed DCS SOR for ")
			} else {
				log.WithField("event", dcsEvent).
					WithField("partition", envId).
					WithField("level", infologger.IL_Devel).
					WithField("run", runNumber64).
					Info("ALIECS SOR operation : processing DCS SOR for ")
			}

		}

		dcsFailedEcsDetectors := make([]string, 0)
		dcsopOk := true
		for _, v := range dcsDetectors {
			if detectorStatusMap[v] != dcspb.DetectorState_RUN_OK {
				dcsopOk = false
				dcsFailedEcsDetectors = append(dcsFailedEcsDetectors, dcsToEcsDetector(v))
			}
		}
		if dcsopOk {
			p.pendingEORs[envId] = runNumber64

			the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
				Name:                call.GetName(),
				OperationName:       call.Func,
				OperationStatus:     pb.OpStatus_DONE_OK,
				OperationStep:       "perform DCS call: StartOfRun",
				OperationStepStatus: pb.OpStatus_DONE_OK,
				EnvironmentId:       envId,
				Payload:             string(payloadJson[:]),
			})
		} else {
			logErr := fmt.Errorf("SOR failed for %s, DCS EOR will run anyway for this run", strings.Join(dcsFailedEcsDetectors, ", "))
			if err != nil {
				if errors.Is(err, io.EOF) {
					err = fmt.Errorf("DCS SOR stream unexpectedly terminated from DCS side before completion: %w", err)
				}
				logErr = fmt.Errorf("%v : %v", err, logErr)
			}

			log.WithError(logErr).
				WithField("level", infologger.IL_Ops).
				WithField("endpoint", viper.GetString("dcsServiceEndpoint")).
				WithField("run", runNumber64).
				WithField("partition", envId).
				WithField("call", "StartOfRun").
				Error("DCS error")

			call.VarStack["__call_error_reason"] = logErr.Error()
			call.VarStack["__call_error"] = callFailedStr

			payload["failedDetectors"] = dcsFailedEcsDetectors
			payloadJson, _ = json.Marshal(payload)

			the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
				Name:                call.GetName(),
				OperationName:       call.Func,
				OperationStatus:     pb.OpStatus_DONE_ERROR,
				OperationStep:       "perform DCS call: StartOfRun",
				OperationStepStatus: pb.OpStatus_DONE_ERROR,
				EnvironmentId:       envId,
				Payload:             string(payloadJson[:]),
				Error:               logErr.Error(),
			})
		}
		return
	}
	eorFunc := func(runNumber64 int64) (out string) { // must formally return string even when we return nothing
		callFailedStr := "DCS EndOfRun call failed"

		dcsDetectorsParam, ok := varStack["dcs_detectors"]
		if !ok {
			log.WithField("partition", envId).
				WithField("run", runNumber64).
				Debug("empty DCS detectors list provided")
			dcsDetectorsParam = "[\"NULL_DETECTOR\"]"
		}

		dcsDetectors, err := p.parseDetectors(dcsDetectorsParam)
		if err != nil {
			log.WithError(err).
				WithField("level", infologger.IL_Ops).
				WithField("partition", envId).
				WithField("call", "EndOfRun").
				Error("DCS error")

			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr

			return
		}

		log.WithField("partition", envId).
			WithField("level", infologger.IL_Ops).
			WithField("run", runNumber64).
			Infof("performing DCS EOR for detectors: %s", strings.Join(dcsDetectors.EcsDetectorsSlice(), " "))

		parameters, ok := varStack["dcs_eor_parameters"]
		if !ok {
			log.WithField("partition", envId).
				Debug("no DCS EOR parameters set")
			parameters = "{}"
		}

		argMap := make(map[string]string)
		bytes := []byte(parameters)
		err = json.Unmarshal(bytes, &argMap)
		if err != nil {
			err = fmt.Errorf("error processing DCS EOR parameters: %w", err)

			log.WithError(err).
				WithField("level", infologger.IL_Ops).
				WithField("partition", envId).
				WithField("call", "EndOfRun").
				Error("DCS error")

			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr

			return
		}

		// Preparing the per-detector request payload
		in := dcspb.EorRequest{
			RunNumber: int32(runNumber64),
			Detectors: make([]*dcspb.DetectorOperationRequest, len(dcsDetectors)),
		}
		for i, dcsDet := range dcsDetectors {
			ecsDet := dcsToEcsDetector(dcsDet)
			perDetectorParameters, okParam := varStack[strings.ToLower(ecsDet)+"_dcs_eor_parameters"]
			if !okParam {
				log.WithField("partition", envId).
					WithField("run", runNumber64).
					Debug("empty DCS detectors list provided")
				perDetectorParameters = "{}"
			}
			detectorArgMap := make(map[string]string)
			bytes := []byte(perDetectorParameters)
			err = json.Unmarshal(bytes, &detectorArgMap)
			if err != nil {
				err = fmt.Errorf("error processing %s DCS EOR parameter map: %w", dcsDet.String(), err)

				log.WithError(err).
					WithField("level", infologger.IL_Ops).
					WithField("partition", envId).
					WithField("call", "EndOfRun").
					WithField("detector", ecsDet).
					WithField("run", runNumber64).
					Error("DCS error")

				call.VarStack["__call_error_reason"] = err.Error()
				call.VarStack["__call_error"] = callFailedStr

				return
			}

			// Per-detector parameters override any general dcs_eor_parameters
			err = mergo.Merge(&detectorArgMap, argMap)
			if err != nil {
				err = fmt.Errorf("error processing %s DCS EOR general parameters override: %w", dcsDet.String(), err)

				log.WithError(err).
					WithField("level", infologger.IL_Ops).
					WithField("partition", envId).
					WithField("call", "EndOfRun").
					WithField("detector", ecsDet).
					WithField("run", runNumber64).
					Error("DCS error")

				call.VarStack["__call_error_reason"] = err.Error()
				call.VarStack["__call_error"] = callFailedStr

				return
			}

			in.Detectors[i] = &dcspb.DetectorOperationRequest{
				Detector:        dcsDet,
				ExtraParameters: detectorArgMap,
			}
		}

		if p.dcsClient == nil {
			err = fmt.Errorf("DCS plugin not initialized, EndOfRun impossible")

			log.WithError(err).
				WithField("level", infologger.IL_Ops).
				WithField("endpoint", viper.GetString("dcsServiceEndpoint")).
				WithField("run", runNumber64).
				WithField("partition", envId).
				WithField("call", "EndOfRun").
				Error("DCS error")

			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr

			return
		}
		if p.dcsClient.GetConnState() == connectivity.Shutdown {
			err = fmt.Errorf("DCS client connection not available, EndOfRun impossible")

			log.WithError(err).
				WithField("level", infologger.IL_Ops).
				WithField("endpoint", viper.GetString("dcsServiceEndpoint")).
				WithField("run", runNumber64).
				WithField("partition", envId).
				WithField("call", "EndOfRun").
				Error("DCS error")

			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr

			return
		}

		var stream dcspb.Configurator_EndOfRunClient
		timeout := callable.AcquireTimeout(DCS_GENERAL_OP_TIMEOUT, varStack, "EOR", envId)
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		payload := map[string]interface{}{
			"detectors": dcsDetectors.EcsDetectorsSlice(),
			"runNumber": runNumber64,
		}
		payloadJson, _ := json.Marshal(payload)

		the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
			Name:                call.GetName(),
			OperationName:       call.Func,
			OperationStatus:     pb.OpStatus_STARTED,
			OperationStep:       "perform DCS call: EndOfRun",
			OperationStepStatus: pb.OpStatus_STARTED,
			EnvironmentId:       envId,
			Payload:             string(payloadJson[:]),
		})

		// Point of no return
		// The gRPC call below is expected to return immediately, with any actual responses arriving subsequently via
		// the response stream.
		// Regardless of DCS EOR success or failure, it must run once and only once, therefore if this call returns
		// a nil error, we immediately dequeue the pending EOR.
		stream, err = p.dcsClient.EndOfRun(ctx, &in, grpc.EmptyCallOption{})
		if err != nil {
			log.WithError(err).
				WithField("level", infologger.IL_Ops).
				WithField("endpoint", viper.GetString("dcsServiceEndpoint")).
				WithField("run", runNumber64).
				WithField("partition", envId).
				WithField("call", "EndOfRun").
				Error("DCS error")

			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr

			the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
				Name:                call.GetName(),
				OperationName:       call.Func,
				OperationStatus:     pb.OpStatus_DONE_ERROR,
				OperationStep:       "perform DCS call: EndOfRun",
				OperationStepStatus: pb.OpStatus_DONE_ERROR,
				EnvironmentId:       envId,
				Payload:             string(payloadJson[:]),
				Error:               err.Error(),
			})

			return
		}
		delete(p.pendingEORs, envId) // make sure this EOR never runs again

		log.WithField("level", infologger.IL_Ops).
			WithField("endpoint", viper.GetString("dcsServiceEndpoint")).
			WithField("run", runNumber64).
			WithField("partition", envId).
			WithField("call", "EndOfRun").
			Debug("DCS EndOfRun returned stream, awaiting responses (DCS cleanup will not run for this environment)")

		detectorStatusMap := make(map[dcspb.Detector]dcspb.DetectorState)
		for _, v := range dcsDetectors {
			detectorStatusMap[v] = dcspb.DetectorState_NULL_STATE
		}

		var dcsEvent *dcspb.RunEvent
		for {
			if ctx.Err() != nil {
				err = fmt.Errorf("DCS EndOfRun context timed out (%s), any future DCS events are ignored", timeout.String())

				the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
					Name:                call.GetName(),
					OperationName:       call.Func,
					OperationStatus:     pb.OpStatus_ONGOING,
					OperationStep:       "perform DCS call: EndOfRun",
					OperationStepStatus: pb.OpStatus_DONE_TIMEOUT,
					EnvironmentId:       envId,
					Payload:             string(payloadJson[:]),
					Error:               err.Error(),
				})

				break
			}
			dcsEvent, err = stream.Recv()
			if errors.Is(err, io.EOF) { // correct stream termination
				logMsg := "DCS EOR event stream was closed from the DCS side (EOF)"
				log.WithField("partition", envId).
					WithField("run", runNumber64).
					Debug(logMsg)

				the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
					Name:                call.GetName(),
					OperationName:       call.Func,
					OperationStatus:     pb.OpStatus_ONGOING,
					OperationStep:       "perform DCS call: EndOfRun",
					OperationStepStatus: pb.OpStatus_DONE_ERROR,
					EnvironmentId:       envId,
					Payload:             string(payloadJson[:]),
					Error:               logMsg,
				})

				break // no more data
			}
			if errors.Is(err, context.DeadlineExceeded) {
				log.WithError(err).
					WithField("partition", envId).
					WithField("run", runNumber64).
					WithField("timeout", timeout.String()).
					Debug("DCS EOR timed out")
				err = fmt.Errorf("DCS EOR timed out after %s: %w", timeout.String(), err)

				the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
					Name:                call.GetName(),
					OperationName:       call.Func,
					OperationStatus:     pb.OpStatus_ONGOING,
					OperationStep:       "perform DCS call: EndOfRun",
					OperationStepStatus: pb.OpStatus_DONE_TIMEOUT,
					EnvironmentId:       envId,
					Payload:             string(payloadJson[:]),
					Error:               err.Error(),
				})

				break
			}
			if err != nil { // stream termination in case of general error
				logMsg := "bad DCS EOR event received, any future DCS events are ignored"
				log.WithError(err).
					WithField("partition", envId).
					WithField("run", runNumber64).
					Warn(logMsg)

				the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
					Name:                call.GetName(),
					OperationName:       call.Func,
					OperationStatus:     pb.OpStatus_ONGOING,
					OperationStep:       "perform DCS call: EndOfRun",
					OperationStepStatus: pb.OpStatus_DONE_ERROR,
					EnvironmentId:       envId,
					Payload:             string(payloadJson[:]),
					Error:               logMsg,
				})

				break
			}
			if dcsEvent == nil {
				log.WithField("partition", envId).
					WithField("run", runNumber64).
					Warn("nil DCS EOR event received, skipping to next DCS event")
				continue
			}

			if dcsEvent.GetState() == dcspb.DetectorState_EOR_FAILURE {
				ecsDet := dcsToEcsDetector(dcsEvent.GetDetector())

				logErr := fmt.Errorf("%s EOR failure reported by DCS", ecsDet)
				if err != nil {
					if errors.Is(err, io.EOF) {
						err = fmt.Errorf("DCS EOR stream unexpectedly terminated from DCS side before completion: %w", err)
					}
					logErr = fmt.Errorf("%v : %v", err, logErr)
				}
				log.WithError(logErr).
					WithField("event", dcsEvent).
					WithField("detector", ecsDet).
					WithField("level", infologger.IL_Ops).
					WithField("endpoint", viper.GetString("dcsServiceEndpoint")).
					WithField("run", runNumber64).
					WithField("partition", envId).
					WithField("call", "EndOfRun").
					Error("DCS error")

				call.VarStack["__call_error_reason"] = logErr.Error()
				call.VarStack["__call_error"] = callFailedStr

				payload["detector"] = ecsDet
				payload["dcsEvent"] = dcsEvent
				payloadJson, _ = json.Marshal(payload)
				the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
					Name:                call.GetName(),
					OperationName:       call.Func,
					OperationStatus:     pb.OpStatus_DONE_ERROR,
					OperationStep:       "perform DCS call: EndOfRun",
					OperationStepStatus: pb.OpStatus_DONE_ERROR,
					EnvironmentId:       envId,
					Payload:             string(payloadJson[:]),
					Error:               logErr.Error(),
				})

				return
			}

			if dcsEvent.GetState() == dcspb.DetectorState_TIMEOUT {
				ecsDet := dcsToEcsDetector(dcsEvent.GetDetector())

				logErr := fmt.Errorf("%s EOR timeout reported by DCS", ecsDet)
				if err != nil {
					logErr = fmt.Errorf("%v : %v", err, logErr)
				}
				log.WithError(logErr).
					WithField("event", dcsEvent).
					WithField("detector", ecsDet).
					WithField("level", infologger.IL_Ops).
					WithField("endpoint", viper.GetString("dcsServiceEndpoint")).
					WithField("run", runNumber64).
					WithField("partition", envId).
					WithField("call", "EndOfRun").
					Error("DCS error")

				call.VarStack["__call_error_reason"] = logErr.Error()
				call.VarStack["__call_error"] = callFailedStr

				payload["detector"] = ecsDet
				payload["dcsEvent"] = dcsEvent
				payloadJson, _ = json.Marshal(payload)
				the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
					Name:                call.GetName(),
					OperationName:       call.Func,
					OperationStatus:     pb.OpStatus_DONE_TIMEOUT,
					OperationStep:       "perform DCS call: EndOfRun",
					OperationStepStatus: pb.OpStatus_DONE_TIMEOUT,
					EnvironmentId:       envId,
					Payload:             string(payloadJson[:]),
					Error:               logErr.Error(),
				})

				return
			}

			detectorStatusMap[dcsEvent.GetDetector()] = dcsEvent.GetState()

			if dcsEvent.GetState() == dcspb.DetectorState_RUN_OK {
				if dcsEvent.GetDetector() == dcspb.Detector_DCS {
					log.WithField("event", dcsEvent).
						WithField("partition", envId).
						WithField("run", runNumber64).
						WithField("level", infologger.IL_Support).
						Debug("DCS EOR completed successfully")
					delete(p.pendingEORs, envId)

					detPayload := map[string]interface{}{}
					_ = copier.Copy(&detPayload, payload)
					detPayload["dcsEvent"] = dcsEvent
					detPayloadJson, _ := json.Marshal(detPayload)

					the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
						Name:                call.GetName(),
						OperationName:       call.Func,
						OperationStatus:     pb.OpStatus_ONGOING,
						OperationStep:       "perform DCS call: EndOfRun",
						OperationStepStatus: pb.OpStatus_ONGOING,
						EnvironmentId:       envId,
						Payload:             string(detPayloadJson[:]),
					})

					break
				} else {
					ecsDet := dcsToEcsDetector(dcsEvent.GetDetector())
					log.WithField("partition", envId).
						WithField("run", runNumber64).
						WithField("detector", dcsEvent.GetDetector().String()).
						Debugf("DCS EOR for %s: received status %s", ecsDet, dcsEvent.GetState().String())

					detPayload := map[string]interface{}{}
					_ = copier.Copy(&detPayload, payload)
					detPayload["detector"] = ecsDet
					detPayload["dcsEvent"] = dcsEvent
					detPayloadJson, _ := json.Marshal(detPayload)

					the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
						Name:                call.GetName(),
						OperationName:       call.Func,
						OperationStatus:     pb.OpStatus_ONGOING,
						OperationStep:       "perform DCS call: EndOfRun",
						OperationStepStatus: pb.OpStatus_ONGOING,
						EnvironmentId:       envId,
						Payload:             string(detPayloadJson[:]),
					})
				}
			}

			if dcsEvent.GetState() == dcspb.DetectorState_RUN_OK {
				log.WithField("event", dcsEvent).
					WithField("partition", envId).
					WithField("level", infologger.IL_Support).
					WithField("run", runNumber64).
					Info("ALIECS EOR operation : completed DCS EOR for ")
			} else {
				log.WithField("event", dcsEvent).
					WithField("partition", envId).
					WithField("level", infologger.IL_Devel).
					WithField("run", runNumber64).
					Info("ALIECS EOR operation : processing DCS EOR for ")
			}
		}

		dcsFailedEcsDetectors := make([]string, 0)
		dcsopOk := true
		for _, v := range dcsDetectors {
			if detectorStatusMap[v] != dcspb.DetectorState_RUN_OK {
				dcsopOk = false
				dcsFailedEcsDetectors = append(dcsFailedEcsDetectors, dcsToEcsDetector(v))
			}
		}
		if dcsopOk {
			delete(p.pendingEORs, envId)

			the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
				Name:                call.GetName(),
				OperationName:       call.Func,
				OperationStatus:     pb.OpStatus_DONE_OK,
				OperationStep:       "perform DCS call: EndOfRun",
				OperationStepStatus: pb.OpStatus_DONE_OK,
				EnvironmentId:       envId,
				Payload:             string(payloadJson[:]),
			})
		} else {
			logErr := fmt.Errorf("EOR failed for %s, DCS EOR will NOT run again for this run", strings.Join(dcsFailedEcsDetectors, ", "))
			if err != nil {
				logErr = fmt.Errorf("%v : %v", err, logErr)
			}

			log.WithError(logErr).
				WithField("level", infologger.IL_Ops).
				WithField("endpoint", viper.GetString("dcsServiceEndpoint")).
				WithField("run", runNumber64).
				WithField("partition", envId).
				WithField("call", "EndOfRun").
				Error("DCS error")

			call.VarStack["__call_error_reason"] = logErr.Error()
			call.VarStack["__call_error"] = callFailedStr

			payload["failedDetectors"] = dcsFailedEcsDetectors
			payloadJson, _ = json.Marshal(payload)

			the.EventWriterWithTopic(TOPIC).WriteEvent(&pb.Ev_IntegratedServiceEvent{
				Name:                call.GetName(),
				OperationName:       call.Func,
				OperationStatus:     pb.OpStatus_DONE_ERROR,
				OperationStep:       "perform DCS call: EndOfRun",
				OperationStepStatus: pb.OpStatus_DONE_ERROR,
				EnvironmentId:       envId,
				Payload:             string(payloadJson[:]),
				Error:               logErr.Error(),
			})
		}
		return
	}
	stack["EndOfRun"] = func() (out string) {
		rn := varStack["run_number"]
		var runNumber64 int64
		var err error
		runNumber64, err = strconv.ParseInt(rn, 10, 32)
		if err != nil {
			log.WithError(err).
				WithField("partition", envId).
				Error("cannot acquire run number for DCS EOR")
		}
		return eorFunc(runNumber64)
	}
	stack["Cleanup"] = func() (out string) {
		envId, ok := varStack["environment_id"]
		if !ok {
			log.WithField("partition", envId).
				WithField("level", infologger.IL_Devel).
				Warn("no environment_id found for DCS cleanup")
			return
		}

		runNumber, ok := p.pendingEORs[envId]
		if !ok {
			log.WithField("partition", envId).
				WithField("level", infologger.IL_Devel).
				Debug("DCS cleanup: nothing to do")
			return
		}

		log.WithField("run", runNumber).
			WithField("partition", envId).
			WithField("level", infologger.IL_Devel).
			WithField("call", "Cleanup").
			Debug("pending DCS EOR found, performing cleanup")

		out = eorFunc(runNumber)
		delete(p.pendingEORs, envId)

		return
	}

	return
}

func (p *Plugin) parseDetectors(dcsDetectorsParam string) (detectors DCSDetectors, err error) {
	detectorsSlice := make([]string, 0)
	bytes := []byte(dcsDetectorsParam)
	err = json.Unmarshal(bytes, &detectorsSlice)
	if err != nil {
		log.WithError(err).
			Error("error processing DCS detectors list")
		return
	}

	// Now we process the stringSlice into a slice of detector enum values
	detectors = make(DCSDetectors, len(detectorsSlice))
	for i, det := range detectorsSlice {
		dcsDetector := dcspb.Detector_NULL_DETECTOR
		dcsDetector, err = ecsToDcsDetector(det)
		if err != nil {
			return
		}

		// detector string correctly matched to DCS enum
		detectors[i] = dcsDetector
	}
	return
}

func (p *Plugin) Destroy() error {
	return p.dcsClient.Close()
}
