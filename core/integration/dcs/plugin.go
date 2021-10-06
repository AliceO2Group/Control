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

//go:generate protoc --go_out=. --go-grpc_out=require_unimplemented_servers=false:. protos/dcs.proto

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
	"time"

	"github.com/AliceO2Group/Control/common/runtype"
	"github.com/AliceO2Group/Control/common/utils/uid"
	"github.com/AliceO2Group/Control/core/integration"
	dcspb "github.com/AliceO2Group/Control/core/integration/dcs/protos"
	"github.com/AliceO2Group/Control/core/workflow/callable"
	"github.com/imdario/mergo"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
)

const (
	DCS_DIAL_TIMEOUT = 2 * time.Hour
	DCS_GENERAL_OP_TIMEOUT = 45 * time.Second
)

type Plugin struct {
	dcsHost        string
	dcsPort        int

	dcsClient      *RpcClient

	pendingEORs    map[string /*envId*/]int64
}

func NewPlugin(endpoint string) integration.Plugin {
	u, err := url.Parse(endpoint)
	if err != nil {
		log.WithField("endpoint", endpoint).
			WithError(err).
			Error("bad service endpoint")
		return nil
	}

	portNumber, _ := strconv.Atoi(u.Port())

	return &Plugin{
		dcsHost:   u.Hostname(),
		dcsPort:   portNumber,
		dcsClient: nil,
		pendingEORs: make(map[string]int64),
	}
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

func (p *Plugin) GetData(environmentIds []uid.ID) string {
	if p == nil || p.dcsClient == nil {
		return ""
	}

	partitionStates := make(map[string]string)

	for _, envId := range environmentIds {
		if _, ok := p.pendingEORs[envId.String()]; ok {
			partitionStates[envId.String()] = "SOR SUCCESSFUL"
		}
	}

	out, err := json.Marshal(partitionStates)
	if err != nil {
		return ""
	}
	return string(out[:])
}

func (p *Plugin) Init(instanceId string) error {
	if p.dcsClient == nil {
		cxt, cancel := context.WithCancel(context.Background())
		p.dcsClient = NewClient(cxt, cancel, viper.GetString("dcsServiceEndpoint"))
		if p.dcsClient == nil {
			return fmt.Errorf("failed to connect to DCS service on %s", viper.GetString("dcsServiceEndpoint"))
		}

		in := &dcspb.SubscriptionRequest{InstanceId: instanceId}
		evStream, err := p.dcsClient.Subscribe(context.Background(), in, grpc.EmptyCallOption{})
		if err != nil {
			return fmt.Errorf("failed to subscribe to DCS client on %s", viper.GetString("dcsServiceEndpoint"))
		}
		go func() {
			for {
				ev, err := evStream.Recv()
				if err == io.EOF {
					break
				}
				if err != nil {
					log.WithError(err).Error("bad event from DCS service")
				}
				log.WithField("event", ev.String()).Debug("received DCS event")
			}
		}()
	}
	if p.dcsClient == nil {
		return fmt.Errorf("failed to start DCS client on %s", viper.GetString("dcsServiceEndpoint"))
	}
	log.Debug("DCS plugin initialized")
	return nil
}

func (p *Plugin) ObjectStack(data interface{}) (stack map[string]interface{}) {
	call, ok := data.(*callable.Call)
	if !ok {
		return
	}
	varStack := call.VarStack
	envId, ok := varStack["environment_id"]
	if !ok {
		log.Error("cannot acquire environment ID")
		return
	}

	stack = make(map[string]interface{})
	stack["StartOfRun"] = func() (out string) {	// must formally return string even when we return nothing
		log.WithField("partition", envId).
			Debug("performing DCS SOR")

		parameters, ok := varStack["dcs_sor_parameters"]
		if !ok {
			log.WithField("partition", envId).
				Debug("no DCS SOR parameters set")
			parameters = "{}"
		}

		argMap := make(map[string]string)
		bytes := []byte(parameters)
		err := json.Unmarshal(bytes, &argMap)
		if err != nil {
			log.WithField("partition", envId).
				WithError(err).
				Error("error processing DCS SOR parameters")
			return
		}

		rn := varStack["run_number"]
		var runNumber64 int64
		runNumber64, err = strconv.ParseInt(rn, 10, 32)
		if err != nil {
			log.WithField("partition", envId).
				WithError(err).
				Error("cannot acquire run number for DCS SOR")
		}

		rt := dcspb.RunType_TECHNICAL
		runTypeS, ok := varStack["run_type"]
		if ok {
			// a detector is defined in the var stack
			// so we convert from the provided string to the correct enum value in common/runtype
			intRt, err := runtype.RunTypeString(runTypeS)
			if err == nil {
				// the runType was correctly matched to the common/runtype enum, but since the DCS enum is
				// kept compatible, we can directly convert the runtype.RunType to a dcspb.RunType enum value
				rt = dcspb.RunType(intRt)
			}
		}

		dcsDetectorsParam, ok := varStack["dcs_detectors"]
		if !ok {
			log.WithField("partition", envId).
				Debug("empty DCS detectors list provided")
			dcsDetectorsParam = "[\"NULL_DETECTOR\"]"
		}

		detectors, err := p.parseDetectors(dcsDetectorsParam)
		if err != nil {
			return
		}

		// Preparing the per-detector request payload
		in := dcspb.SorRequest{
			RunType:   rt,
			RunNumber: int32(runNumber64),
			Detectors: make([]*dcspb.DetectorOperationRequest, len(detectors)),
		}
		for i, det := range detectors {
			perDetectorParameters, ok := varStack[strings.ToLower(det.String()) + "_dcs_sor_parameters"]
			if !ok {
				log.WithField("partition", envId).
					Debug("empty DCS detectors list provided")
				perDetectorParameters = "{}"
			}
			detectorArgMap := make(map[string]string)
			bytes := []byte(perDetectorParameters)
			err = json.Unmarshal(bytes, &detectorArgMap)
			if err != nil {
				log.WithError(err).
					WithField("detector", det.String()).
					WithField("partition", envId).
					Errorf("error processing DCS SOR parameters for detector %s", det.String())
				return
			}

			// Per-detector parameters override any general dcs_sor_parameters
			err = mergo.Merge(&detectorArgMap, argMap)
			if err != nil {
				log.WithError(err).
					WithField("detector", det.String()).
					WithField("partition", envId).
					Errorf("error building parameter map for detector %s", det.String())
				return
			}

			in.Detectors[i] = &dcspb.DetectorOperationRequest{
				Detector:        det,
				ExtraParameters: detectorArgMap,
			}
		}

		if p.dcsClient == nil {
			log.WithError(fmt.Errorf("DCS plugin not initialized")).
				WithField("endpoint", viper.GetString("dcsServiceEndpoint")).
				WithField("partition", envId).
				Error("failed to perform DCS SOR")
			return
		}
		if p.dcsClient.GetConnState() != connectivity.Ready {
			log.WithError(fmt.Errorf("DCS client connection not available")).
				WithField("endpoint", viper.GetString("dcsServiceEndpoint")).
				WithField("partition", envId).
				Error("failed to perform DCS SOR")
			return
		}

		var stream dcspb.Configurator_StartOfRunClient
		timeout := callable.AcquireTimeout(DCS_GENERAL_OP_TIMEOUT, varStack, "SOR", envId)
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		stream, err = p.dcsClient.StartOfRun(ctx, &in, grpc.EmptyCallOption{})
		if err != nil {
			log.WithError(err).
				WithField("endpoint", viper.GetString("dcsServiceEndpoint")).
				WithField("partition", envId).
				Error("failed to perform DCS SOR")
		}
		var dcsEvent *dcspb.RunEvent
		for {
			dcsEvent, err = stream.Recv()
			if err == io.EOF {
				log.WithField("partition", envId).
					Debug("DCS SOR event stream EOF, closed")
				break // no more data
			}
			if err != nil || dcsEvent == nil {
				if dcsEvent == nil {
					log.WithField("partition", envId).
						Warn("nil DCS event received")
					err = errors.New("nil DCS event")
				}
				log.WithError(err).WithField("partition", envId).
					Warn("bad DCS event received")
				break
			}

			if dcsEvent.GetState() == dcspb.DetectorState_SOR_FAILURE {
				log.WithField("event", dcsEvent).
					WithField("detector", dcsEvent.GetDetector().String()).
					WithField("partition", envId).
					Warn("DCS SOR failure")
				return
			}
			if dcsEvent.GetState() == dcspb.DetectorState_RUN_OK && dcsEvent.GetDetector() == dcspb.Detector_DCS {
				log.WithField("event", dcsEvent).
					WithField("partition", envId).
					Debug("DCS SOR success")
				envId, ok := varStack["environment_id"]
				if !ok {
					break
				}
				p.pendingEORs[envId] = runNumber64
				break
			}
			log.WithField("event", dcsEvent).
				WithField("partition", envId).
				Debug("incoming DCS SOR event")
		}
		return
	}
	eorFunc := func(runNumber int64) (out string) { // must formally return string even when we return nothing
		log.WithField("partition", envId).Debug("performing DCS EOR")

		parameters, ok := varStack["dcs_eor_parameters"]
		if !ok {
			log.WithField("partition", envId).Debug("no DCS EOR parameters set")
			parameters = "{}"
		}

		argMap := make(map[string]string)
		bytes := []byte(parameters)
		err := json.Unmarshal(bytes, &argMap)
		if err != nil {
			log.WithError(err).
				WithField("partition", envId).
				Error("error processing DCS EOR parameters")
			return
		}

		dcsDetectorsParam, ok := varStack["dcs_detectors"]
		if !ok {
			log.WithField("partition", envId).
				Debug("empty DCS detectors list provided")
			dcsDetectorsParam = "[\"NULL_DETECTOR\"]"
		}

		detectors, err := p.parseDetectors(dcsDetectorsParam)
		if err != nil {
			return
		}

		// Preparing the per-detector request payload
		in := dcspb.EorRequest{
			RunNumber: int32(runNumber),
			Detectors: make([]*dcspb.DetectorOperationRequest, len(detectors)),
		}
		for i, det := range detectors {
			perDetectorParameters, ok := varStack[strings.ToLower(det.String()) + "_dcs_eor_parameters"]
			if !ok {
				log.WithField("partition", envId).
					Debug("empty DCS detectors list provided")
				perDetectorParameters = "{}"
			}
			detectorArgMap := make(map[string]string)
			bytes := []byte(perDetectorParameters)
			err = json.Unmarshal(bytes, &detectorArgMap)
			if err != nil {
				log.WithError(err).
					WithField("detector", det.String()).
					WithField("partition", envId).
					Errorf("error processing DCS EOR parameters for detector %s", det.String())
				return
			}

			// Per-detector parameters override any general dcs_sor_parameters
			err = mergo.Merge(&detectorArgMap, argMap)
			if err != nil {
				log.WithError(err).
					WithField("detector", det.String()).
					WithField("partition", envId).
					Errorf("error building parameter map for detector %s", det.String())
				return
			}

			in.Detectors[i] = &dcspb.DetectorOperationRequest{
				Detector:        det,
				ExtraParameters: detectorArgMap,
			}
		}

		if p.dcsClient == nil {
			log.WithError(fmt.Errorf("DCS plugin not initialized")).
				WithField("endpoint", viper.GetString("dcsServiceEndpoint")).
				WithField("partition", envId).
				Error("failed to perform DCS EOR")
			return
		}
		if p.dcsClient.GetConnState() != connectivity.Ready {
			log.WithError(fmt.Errorf("DCS client connection not available")).
				WithField("endpoint", viper.GetString("dcsServiceEndpoint")).
				WithField("partition", envId).
				Error("failed to perform DCS EOR")
			return
		}

		var stream dcspb.Configurator_EndOfRunClient
		timeout := callable.AcquireTimeout(DCS_GENERAL_OP_TIMEOUT, varStack, "EOR", envId)
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		stream, err = p.dcsClient.EndOfRun(ctx, &in, grpc.EmptyCallOption{})
		if err != nil {
			log.WithError(err).
				WithField("endpoint", viper.GetString("dcsServiceEndpoint")).
				WithField("partition", envId).
				Error("failed to perform DCS EOR")
		}
		var dcsEvent *dcspb.RunEvent
		for {
			dcsEvent, err = stream.Recv()
			if err == io.EOF {
				log.WithField("partition", envId).
					Debug("DCS EOR event stream EOF, closed")
				break // no more data
			}
			if err != nil || dcsEvent == nil {
				if dcsEvent == nil {
					log.WithField("partition", envId).
						Warn("nil DCS event received")
					err = errors.New("nil DCS event")
				}
				log.WithError(err).Warn("bad DCS event received")
				break
			}

			if dcsEvent.GetState() == dcspb.DetectorState_EOR_FAILURE {
				log.WithField("event", dcsEvent).
					WithField("detector", dcsEvent.GetDetector().String()).
					WithField("partition", envId).
					Warn("DCS EOR failure")
				return
			}
			if dcsEvent.GetState() == dcspb.DetectorState_RUN_OK && dcsEvent.GetDetector() == dcspb.Detector_DCS {
				log.WithField("event", dcsEvent).
					WithField("partition", envId).
					Debug("DCS EOR success")
				envId, ok := varStack["environment_id"]
				if !ok {
					break
				}
				delete(p.pendingEORs, envId)
				break
			}

			log.WithField("event", dcsEvent).
				WithField("partition", envId).
				Debug("incoming DCS EOR event")
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
				Warn("no environment_id found for DCS cleanup")
			return
		}

		runNumber, ok := p.pendingEORs[envId]
		if !ok {
			log.WithField("partition", envId).
				Debug("DCS cleanup: nothing to do")
			return
		}

		log.WithField("runNumber", runNumber).
			WithField("partition", envId).
			Debug("pending DCS EOR found, performing cleanup")
		return eorFunc(runNumber)
	}

	return
}

func (p *Plugin) parseDetectors(dcsDetectorsParam string) (detectors []dcspb.Detector, err error) {
	detectorsSlice := make([]string, 0)
	bytes := []byte(dcsDetectorsParam)
	err = json.Unmarshal(bytes, &detectorsSlice)
	if err != nil {
		log.WithError(err).Error("error processing DCS detectors list")
		return
	}

	// Now we process the stringSlice into a slice of detector enum values
	detectors = make([]dcspb.Detector, len(detectorsSlice))
	for i, det := range detectorsSlice {
		intDet, ok := dcspb.Detector_value[det]
		if !ok {
			err = fmt.Errorf("invalid DCS detector %s", det)
			log.WithError(err).Error("bad DCS detector entry")
			return
		}

		// detector string correctly matched to DCS enum
		detectors[i] = dcspb.Detector(intDet)
	}
	return
}

func (p *Plugin) Destroy() error {
	return p.dcsClient.Close()
}
