/*
* === This file is part of ALICE O² ===
*
* Copyright 2021 CERN and copyright holders of ALICE O².
* Author: Claire Guyot <claire.guyot@cern.ch>
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
//go:generate protoc --go_out=. --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative --go-grpc_out=require_unimplemented_servers=false:. protos/common.proto
//go:generate protoc --go_out=. --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative --go-grpc_out=require_unimplemented_servers=false:. protos/environment.proto
//go:generate protoc --go_out=. --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative --go-grpc_out=require_unimplemented_servers=false:. protos/flp.proto
//go:generate protoc --go_out=. --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative --go-grpc_out=require_unimplemented_servers=false:. protos/log.proto
//go:generate protoc --go_out=. --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative --go-grpc_out=require_unimplemented_servers=false:. protos/run.proto

package bookkeeping

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	bkpb "github.com/AliceO2Group/Control/core/integration/bookkeeping/protos"
	"google.golang.org/grpc"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/AliceO2Group/Control/common/logger/infologger"
	"github.com/AliceO2Group/Control/common/utils/uid"
	"github.com/AliceO2Group/Control/core/environment"
	"github.com/AliceO2Group/Control/core/integration"
	"github.com/AliceO2Group/Control/core/workflow/callable"
	"github.com/spf13/viper"
)

const (
	BKP_SOR_TIMEOUT             = 30 * time.Second
	BKP_EOR_TIMEOUT             = 30 * time.Second
	BKP_DEFAULT_POLLING_TIMEOUT = 30 * time.Second
)

type Plugin struct {
	bookkeepingHost string
	bookkeepingPort int

	bookkeepingClient *RpcClient

	pendingRunStops map[string] /*envId*/ int64
	pendingO2Stops  map[string] /*envId*/ string
	pendingTrgStops map[string] /*envId*/ string
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
		bookkeepingHost:   u.Hostname(),
		bookkeepingPort:   portNumber,
		bookkeepingClient: nil,
		pendingRunStops:   make(map[string]int64),
		pendingO2Stops:    make(map[string]string),
		pendingTrgStops:   make(map[string]string),
	}
}

func (p *Plugin) GetName() string {
	return "bookkeeping"
}

func (p *Plugin) GetPrettyName() string {
	return "Bookkeeping"
}

func (p *Plugin) GetEndpoint() string {
	return viper.GetString("bookkeepingBaseUri")
}

func (p *Plugin) GetConnectionState() string {
	return "READY"
}

func (p *Plugin) GetData(_ []any) string {
	if p == nil || p.bookkeepingClient == nil {
		return ""
	}

	partitionStates := make(map[string]string)

	out, err := json.Marshal(partitionStates)
	if err != nil {
		return ""
	}
	return string(out[:])
}

func (p *Plugin) GetEnvironmentsData(_ []uid.ID) map[uid.ID]string {
	return nil
}

func (p *Plugin) Init(instanceId string) error {
	if p.bookkeepingClient == nil {
		cxt, cancel := context.WithCancel(context.Background())
		p.bookkeepingClient = NewClient(cxt, cancel, p.GetEndpoint())
		if p.bookkeepingClient == nil {
			return fmt.Errorf("failed to connect to Bookkeeping service on %s", p.GetEndpoint())
		}
		log.Debug("DD scheduler plugin initialized")
	}
	return nil
}

func (p *Plugin) ObjectStack(_ map[string]string) (stack map[string]interface{}) {
	stack = make(map[string]interface{})
	return stack
}

func (p *Plugin) CallStack(data interface{}) (stack map[string]interface{}) {
	call, ok := data.(*callable.Call)
	if !ok {
		return
	}
	varStack := call.VarStack
	envId, ok := varStack["environment_id"]
	if !ok {
		err := errors.New("cannot acquire environment ID")
		log.Error(err)

		call.VarStack["__call_error_reason"] = err.Error()
		call.VarStack["__call_error"] = "Bookkeeping plugin Call Stack failed"
		return
	}
	trigger, ok := varStack["__call_trigger"]
	if !ok {
		err := errors.New("cannot acquire trigger from varStack")
		log.WithField("call", call).
			WithField("partition", envId).
			Error(err)

		call.VarStack["__call_error_reason"] = err.Error()
		call.VarStack["__call_error"] = "Bookkeeping plugin Call Stack failed"
		return
	}
	var err error
	parsedEnvId, err := uid.FromString(envId)
	if err != nil {
		if strings.Contains(trigger, "DESTROY") || strings.Contains(trigger, "GO_ERROR") {
			log.WithField("partition", envId).
				Debug("cannot parse environment ID when DESTROY or GO_ERROR transition")
			return
		}
		log.WithError(err).
			WithField("partition", envId).
			Error("cannot parse environment ID")

		call.VarStack["__call_error_reason"] = err.Error()
		call.VarStack["__call_error"] = "Bookkeeping plugin Call Stack failed"
		return
	}
	envMan := environment.ManagerInstance()
	env, err := envMan.Environment(parsedEnvId)
	if err != nil {
		if strings.Contains(trigger, "DESTROY") || strings.Contains(trigger, "GO_ERROR") {
			log.WithField("partition", envId).
				Debug("cannot acquire environment from parsed environment ID when DESTROY or GO_ERROR transition")
			return
		}
		log.WithError(err).
			WithField("partition", envId).
			Error("cannot acquire environment from parsed environment ID")

		call.VarStack["__call_error_reason"] = err.Error()
		call.VarStack["__call_error"] = "Bookkeeping plugin Call Stack failed"
		return
	}

	stack = make(map[string]interface{})
	// Run related Bookkeeping functions
	stack["StartOfRun"] = func() (out string) {
		callFailedStr := "Bookkeeping StartOfRun call failed"

		rn := varStack["run_number"]
		var runNumber64 int64
		runNumber64, err = strconv.ParseInt(rn, 10, 32)
		if err != nil {
			log.WithField("partition", envId).
				WithError(err).
				Error("cannot acquire run number for Bookkeeping SOR")

			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr
			return
		}

		if p.bookkeepingClient == nil {
			err = fmt.Errorf("Bookkeeping plugin not initialized, StartOfRun impossible")

			log.WithError(err).
				WithField("level", infologger.IL_Support).
				WithField("endpoint", viper.GetString("bookkeepingBaseUri")).
				WithField("runNumber", runNumber64).
				WithField("partition", envId).
				WithField("call", "StartOfRun").
				Error("Bookkeeping SOR error")

			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr

			return
		}

		runNumber := env.GetCurrentRunNumber()

		rnString := strconv.FormatUint(uint64(runNumber), 10)

		flps := env.GetFLPs()
		epns, err := strconv.ParseInt(env.GetKV("", "odc_n_epns"), 10, 0)
		if err != nil {
			log.WithError(err).
				WithField("runNumber", runNumber64).
				WithField("partition", envId).
				WithField("call", "StartOfRun").
				Warning("cannot parse number of EPNs")
		}
		ddEnabled, err := strconv.ParseBool(env.GetKV("", "dd_enabled"))
		if err != nil {
			log.WithError(err).
				WithField("runNumber", runNumber64).
				WithField("partition", envId).
				WithField("call", "StartOfRun").
				Warning("cannot parse DD enabled")
		}
		dcsEnabled, err := strconv.ParseBool(env.GetKV("", "dcs_enabled"))
		if err != nil {
			log.WithError(err).
				WithField("runNumber", runNumber64).
				WithField("partition", envId).
				WithField("call", "StartOfRun").
				Warning("cannot parse DCS enabled")
		}
		epnEnabled, err := strconv.ParseBool(env.GetKV("", "epn_enabled"))
		if err != nil {
			log.WithError(err).
				WithField("runNumber", runNumber64).
				WithField("partition", envId).
				WithField("call", "StartOfRun").
				Warning("cannot parse EPN enabled")
		}
		odcTopology := env.GetKV("", "odc_topology")
		odcTopologyFullname, ok := env.Workflow().GetVars().Get("odc_topology_fullname")
		if !ok {
			log.WithField("runNumber", runNumber64).
				WithField("partition", envId).
				WithField("call", "UpdateRun").
				Warning("cannot acquire ODC topology fullname")
		}
		detectors := env.GetActiveDetectors().StringList()
		var detectorsList = make([]bkpb.Detector, len(detectors))
		for _, name := range detectors {
			detectorsList = append(detectorsList, bkpb.Detector(bkpb.Detector_value[name]))
		}

		inRun := bkpb.RunCreationRequest{
			RunNumber:           int32(runNumber64),
			EnvironmentId:       env.Id().String(),
			NDetectors:          int32(len(env.GetActiveDetectors())),
			NEpns:               int32(epns),
			NFlps:               int32(len(flps)),
			RunType:             bkpb.RunType(env.GetRunType()),
			DdFlp:               ddEnabled,
			Dcs:                 dcsEnabled,
			Epn:                 epnEnabled,
			EpnTopology:         odcTopology,
			OdcTopologyFullName: odcTopologyFullname,
			Detectors:           detectorsList,
		}

		timeout := callable.AcquireTimeout(BKP_SOR_TIMEOUT, varStack, "CreateRun", envId)
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		_, err = p.bookkeepingClient.RunServiceClient.Create(ctx, &inRun, grpc.EmptyCallOption{})
		if err != nil {
			log.WithError(err).
				WithField("runNumber", runNumber).
				WithField("partition", envId).
				WithField("call", "CreateRun").
				Error("Bookkeeping API CreateRun error")

			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr
			return
		} else {
			p.pendingRunStops[envId] = runNumber64
			p.pendingO2Stops[envId] = ""
			p.pendingTrgStops[envId] = ""
			log.WithField("runNumber", runNumber).
				WithField("partition", envId).
				Debug("CreateRun call successful")
		}

		var inFlps = bkpb.ManyFlpsCreationRequest{
			Flps: make([]*bkpb.FlpCreationRequest, len(flps)),
		}

		for _, flp := range flps {
			inFlp := &bkpb.FlpCreationRequest{
				Name:      flp,
				HostName:  flp,
				RunNumber: int32(runNumber64),
			}
			inFlps.Flps = append(inFlps.Flps, inFlp)
		}

		timeout = callable.AcquireTimeout(BKP_SOR_TIMEOUT, varStack, "CreateFlp", envId)
		ctx, cancel = context.WithTimeout(context.Background(), timeout)
		defer cancel()
		_, err = p.bookkeepingClient.FlpServiceClient.CreateMany(ctx, &inFlps, grpc.EmptyCallOption{})
		if err != nil {
			log.WithError(err).
				WithField("runNumber", runNumber).
				WithField("partition", envId).
				WithField("call", "CreateFlp").
				Error("Bookkeeping API CreateFlp error")

			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr
			return
		}
		log.WithField("runNumber", runNumber).
			WithField("partition", envId).
			Debug("CreateFlp call done")

		runNumbers := make([]int32, 0)
		runNumbers = append(runNumbers, int32(runNumber64))
		inLog := bkpb.LogCreationRequest{
			RunNumbers:  runNumbers,
			Title:       fmt.Sprintf("Log for run %s and environment %s", rnString, env.Id().String()),
			Text:        env.GetVarsAsString(),
			ParentLogId: -1,
		}

		timeout = callable.AcquireTimeout(BKP_SOR_TIMEOUT, varStack, "CreateLog", envId)
		ctx, cancel = context.WithTimeout(context.Background(), timeout)
		defer cancel()
		_, err = p.bookkeepingClient.LogServiceClient.Create(ctx, &inLog, grpc.EmptyCallOption{})
		if err != nil {
			log.WithError(err).
				WithField("runNumber", runNumber).
				WithField("partition", envId).
				WithField("call", "CreateLog").
				Error("Bookkeeping API CreateLog error")

			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr
			return
		} else {
			log.WithField("runNumber", runNumber).
				WithField("partition", envId).
				Debug("CreateLog call successful")
		}
		return
	}
	updateRunFunc := func(runNumber64 int64, state string, timeO2Start string, timeO2End string, timeTrgStart string, timeTrgEnd string) (out string) {
		callFailedStr := "Bookkeeping UpdateRun call failed"
		trgGlobalRunEnabled, err := strconv.ParseBool(env.GetKV("", "trg_global_run_enabled"))
		if err != nil {
			log.WithError(err).
				WithField("runNumber", runNumber64).
				WithField("partition", envId).
				WithField("call", "UpdateRun").
				Warning("cannot parse TRG global run enabled")
		}
		trgEnabled, err := strconv.ParseBool(env.GetKV("", "trg_enabled"))
		if err != nil {
			log.WithError(err).
				WithField("runNumber", runNumber64).
				WithField("partition", envId).
				WithField("call", "UpdateRun").
				Warning("cannot parse TRG enabled")
		}
		var trg = ""
		if trgEnabled == false {
			trg = "OFF"
		} else {
			if trgGlobalRunEnabled == false {
				trg = "LTU"
			} else {
				trg = "CTP"
			}
		}
		pdpConfig, ok := varStack["pdp_config_option"]
		if !ok {
			log.WithField("runNumber", runNumber64).
				WithField("partition", envId).
				WithField("call", "UpdateRun").
				Warning("cannot acquire PDP workflow configuration mode")
		}
		pdpTopology, ok := varStack["pdp_topology_description_library_file"]
		if !ok {
			log.WithField("runNumber", runNumber64).
				WithField("partition", envId).
				WithField("call", "UpdateRun").
				Warning("cannot acquire PDP topology description library file")
		}
		pdpParameters, ok := varStack["pdp_workflow_parameters"]
		if !ok {
			log.WithField("runNumber", runNumber64).
				WithField("partition", envId).
				WithField("call", "UpdateRun").
				Warning("cannot acquire PDP workflow parameters")
		}
		pdpBeam, ok := varStack["pdp_beam_type"]
		if !ok {
			log.WithField("runNumber", runNumber64).
				WithField("partition", envId).
				WithField("call", "UpdateRun").
				Warning("cannot acquire PDP beam type")
		}
		tfbMode := env.GetKV("", "tfb_dd_mode")
		odcTopologyFullname, ok := env.Workflow().GetVars().Get("odc_topology_fullname")
		if !ok {
			log.WithField("runNumber", runNumber64).
				WithField("partition", envId).
				WithField("call", "UpdateRun").
				Warning("cannot acquire ODC topology fullname")
		}
		lhcPeriod := env.GetKV("", "lhc_period")
		readoutUri, ok := varStack["readout_cfg_uri"]

		var runquality bkpb.RunQuality
		switch state {
		case string(bkpb.RunQuality_RUN_QUALITY_GOOD):
			runquality = bkpb.RunQuality_RUN_QUALITY_GOOD
		case string(bkpb.RunQuality_RUN_QUALITY_BAD):
			runquality = bkpb.RunQuality_RUN_QUALITY_BAD
		default:
			runquality = bkpb.RunQuality_RUN_QUALITY_TEST
		}

		timeO2S, err := strconv.ParseInt(timeO2Start, 10, 64)
		if err != nil {
			log.WithField("runNumber", runNumber64).
				WithField("time", timeO2Start).
				Warning("cannot parse O2 start time")
			timeO2S = -1
		}
		if timeO2Start == "" || timeO2S <= 0 {
			timeO2S = -1
		}
		var timeO2E int64 = -1
		if timeO2End != "" {
			timeO2E, err = strconv.ParseInt(timeO2End, 10, 64)
			if err != nil {
				log.WithField("runNumber", runNumber64).
					WithField("time", timeO2End).
					Warning("cannot parse O2 end time")
				timeO2E = -1
			}
		}
		if timeO2End == "" || timeO2E <= 0 {
			timeO2E = -1
		}
		var timeTrgS int64 = -1
		var timeTrgE int64 = -1
		if trg == "LTU" || trg == "CTP" {
			timeTrgS, err = strconv.ParseInt(timeTrgStart, 10, 64)
			if err != nil {
				log.WithField("runNumber", runNumber64).
					WithField("time", timeTrgStart).
					Warning("cannot parse Trg start time")
				timeTrgS = -1
			}
			if timeTrgStart == "" || timeTrgS <= 0 {
				timeTrgS = -1
			}
			timeTrgE, err = strconv.ParseInt(timeTrgEnd, 10, 64)
			if err != nil {
				log.WithField("runNumber", runNumber64).
					WithField("time", timeTrgEnd).
					Warning("cannot parse Trg end time")
				timeTrgE = -1
			}
			if timeTrgEnd == "" || timeTrgE <= 0 {
				timeTrgE = -1
			}
		}

		inRun := bkpb.RunUpdateRequest{
			RunNumber:                         int32(runNumber64),
			RunQuality:                        runquality,
			TimeO2Start:                       timeO2S,
			TimeO2End:                         timeO2E,
			TimeTrgStart:                      timeTrgS,
			TimeTrgEnd:                        timeTrgE,
			TriggerValue:                      trigger,
			PdpConfigOption:                   pdpConfig,
			PdpTopologyDescriptionLibraryFile: pdpTopology,
			TfbDdMode:                         tfbMode,
			LhcPeriod:                         lhcPeriod,
			OdcTopologyFullName:               odcTopologyFullname,
			PdpWorkflowParameters:             pdpParameters,
			PdpBeamType:                       pdpBeam,
			ReadoutCfgUri:                     readoutUri,
		}

		timeout := callable.AcquireTimeout(BKP_SOR_TIMEOUT, varStack, "UpdateRun", envId)
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		_, err = p.bookkeepingClient.RunServiceClient.Update(ctx, &inRun, grpc.EmptyCallOption{})
		if err != nil {
			log.WithError(err).
				WithField("runNumber", runNumber64).
				WithField("partition", envId).
				WithField("call", "UpdateRun").
				Error("Bookkeeping API UpdateRun error")

			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr
			return
		} else {
			var updatedRun string
			if function, ok := varStack["__call_func"]; ok && strings.Contains(function, "UpdateRunStop") {
				if p.pendingO2Stops[envId] == "" || (trgEnabled && p.pendingTrgStops[envId] == "") {
					updatedRun = "INCOMPLETE"
					log.WithField("runNumber", runNumber64).
						WithField("partition", envId).
						Debug("UpdateRun call: run information incomplete")
				} else {
					updatedRun = "STOPPED"
					delete(p.pendingRunStops, envId)
					delete(p.pendingO2Stops, envId)
					delete(p.pendingTrgStops, envId)
				}
			} else {
				updatedRun = "STARTED"
			}
			log.WithField("runNumber", runNumber64).
				WithField("updated to", updatedRun).
				WithField("partition", envId).
				Debug("UpdateRun call successful")
		}
		return
	}
	stack["UpdateRunStart"] = func() (out string) {
		callFailedStr := "Bookkeeping UpdateRunStart call failed"

		rn := varStack["run_number"]
		runNumber64, err := strconv.ParseInt(rn, 10, 32)
		if err != nil {
			log.WithField("partition", envId).
				WithError(err).
				Error("cannot acquire run number for Bookkeeping UpdateRunStart")

			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr
			return
		}

		if p.bookkeepingClient == nil {
			err = fmt.Errorf("Bookkeeping plugin not initialized, UpdateRunStart impossible")

			log.WithError(err).
				WithField("level", infologger.IL_Support).
				WithField("endpoint", viper.GetString("bookkeepingBaseUri")).
				WithField("runNumber", runNumber64).
				WithField("partition", envId).
				WithField("call", "UpdateRunStart").
				Error("Bookkeeping UpdateRunStart error")

			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr

			return
		}

		O2StartTime := varStack["run_start_time_ms"]
		TrgStartTime := varStack["trg_start_time_ms"]

		return updateRunFunc(runNumber64, "test", O2StartTime, "", TrgStartTime, "")
	}
	stack["UpdateRunStop"] = func() (out string) {
		callFailedStr := "Bookkeeping UpdateRunStop call failed"

		rn := varStack["run_number"]
		runNumber64, err := strconv.ParseInt(rn, 10, 32)
		if (strings.Contains(trigger, "DESTROY") || strings.Contains(trigger, "GO_ERROR")) && (rn == "" || runNumber64 == 0) {
			log.WithField("partition", envId).
				Debug("cannot update run on stop, no run for the environment")
			return
		}
		if err != nil {
			log.WithField("partition", envId).
				WithError(err).
				Error("cannot acquire run number for Bookkeeping UpdateRunStop")

			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr
			return
		}

		if p.bookkeepingClient == nil {
			err = fmt.Errorf("Bookkeeping plugin not initialized, UpdateRunStop impossible")

			log.WithError(err).
				WithField("level", infologger.IL_Support).
				WithField("endpoint", viper.GetString("bookkeepingBaseUri")).
				WithField("runNumber", runNumber64).
				WithField("partition", envId).
				WithField("call", "UpdateRunStop").
				Error("Bookkeeping UpdateRunStop error")

			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr

			return
		}

		O2StartTime := varStack["run_start_time_ms"]
		O2EndTime := varStack["run_end_time_ms"]
		p.pendingO2Stops[envId] = O2EndTime

		TrgStartTime := varStack["trg_start_time_ms"]
		TrgEndTime := varStack["trg_end_time_ms"]
		p.pendingTrgStops[envId] = TrgEndTime

		if _, ok := p.pendingRunStops[envId]; ok {
			return updateRunFunc(runNumber64, "test", O2StartTime, O2EndTime, TrgStartTime, TrgEndTime)
		} else {
			log.WithField("partition", envId).
				Warning("skipping UpdateRun call, no pending run number found")
			return
		}
	}
	// Environment related Bookkeeping functions
	stack["CreateEnv"] = func() (out string) {
		callFailedStr := "Bookkeeping CreateEnv call failed"

		if p.bookkeepingClient == nil {
			err = fmt.Errorf("Bookkeeping plugin not initialized, CreateEnv impossible")

			log.WithError(err).
				WithField("level", infologger.IL_Support).
				WithField("endpoint", viper.GetString("bookkeepingBaseUri")).
				WithField("partition", envId).
				WithField("call", "CreateEnv").
				Error("Bookkeeping CreateEnv error")

			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr

			return
		}

		var statusMessage = ""
		envState := env.CurrentState()
		if envState == "STANDBY" || envState == "DEPLOYED" {
			statusMessage = "success: the environment is in " + envState + " state after creation"
		} else {
			statusMessage = "error: the environment is in " + envState + " state after creation"
		}

		inEnv := bkpb.EnvironmentCreationRequest{
			EnvId:         env.Id().String(),
			CreatedAt:     time.Now().UnixMilli(),
			Status:        envState,
			StatusMessage: statusMessage,
		}

		timeout := callable.AcquireTimeout(BKP_SOR_TIMEOUT, varStack, "UpdateRun", envId)
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		_, err = p.bookkeepingClient.EnvironmentServiceClient.Create(ctx, &inEnv, grpc.EmptyCallOption{})

		if err != nil {
			log.WithError(err).
				WithField("partition", envId).
				WithField("call", "CreateEnvironment").
				Error("Bookkeeping API CreateEnvironment error")

			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr
			return
		} else {
			log.WithField("partition", envId).
				Debug("CreateEnvironment call successful")
		}
		return
	}
	updateEnvFunc := func(envId string, toredownAt time.Time, status string, statusMessage string) (out string) {
		callFailedStr := "Bookkeeping UpdateEnv call failed"

		inEnv := bkpb.EnvironmentUpdateRequest{
			EnvId:         env.Id().String(),
			ToredownAt:    toredownAt.UnixMilli(),
			Status:        status,
			StatusMessage: statusMessage,
		}

		timeout := callable.AcquireTimeout(BKP_SOR_TIMEOUT, varStack, "UpdateRun", envId)
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		_, err = p.bookkeepingClient.EnvironmentServiceClient.Update(ctx, &inEnv, grpc.EmptyCallOption{})
		if err != nil {
			log.WithError(err).
				WithField("partition", envId).
				WithField("call", "UpdateEnvironment").
				Error("Bookkeeping API UpdateEnvironment error")

			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr
			return
		} else {
			log.WithField("partition", envId).
				WithField("state", status).
				Debug("UpdateEnvironment call successful")
		}
		return
	}
	stack["UpdateEnv"] = func() (out string) {
		callFailedStr := "Bookkeeping UpdateEnv call failed"

		if p.bookkeepingClient == nil {
			err = fmt.Errorf("Bookkeeping plugin not initialized, UpdateEnv impossible")

			log.WithError(err).
				WithField("level", infologger.IL_Support).
				WithField("endpoint", viper.GetString("bookkeepingBaseUri")).
				WithField("partition", envId).
				WithField("call", "UpdateEnv").
				Error("Bookkeeping UpdateEnv error")

			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr

			return
		}

		envState := env.CurrentState()

		if strings.Contains(trigger, "DESTROY") {
			envState = "DESTROYED"
			return updateEnvFunc(env.Id().String(), time.Time{}, envState, "the environment is DESTROYED after DESTROY transition")
		}
		if strings.Contains(trigger, "DEPLOY") {
			if envState == "DEPLOYED" {
				return updateEnvFunc(env.Id().String(), time.Time{}, envState, "success: the environment is in DEPLOYED state after DEPLOY transition")
			} else {
				return updateEnvFunc(env.Id().String(), time.Time{}, envState, "error: the environment is in "+envState+" state after DEPLOY transition")
			}
		}
		if strings.Contains(trigger, "CONFIGURE") {
			if envState == "CONFIGURED" {
				return updateEnvFunc(env.Id().String(), time.Time{}, envState, "success: the environment is in CONFIGURED state after CONFIGURE transition")
			} else {
				return updateEnvFunc(env.Id().String(), time.Time{}, envState, "error: the environment is in "+envState+" state after CONFIGURE transition")
			}
		}
		if strings.Contains(trigger, "RESET") {
			if envState == "DEPLOYED" {
				return updateEnvFunc(env.Id().String(), time.Time{}, envState, "success: the environment is in DEPLOYED state after RESET transition")
			} else {
				return updateEnvFunc(env.Id().String(), time.Time{}, envState, "error: the environment is in "+envState+" state after RESET transition")
			}
		}
		if strings.Contains(trigger, "START_ACTIVITY") {
			if envState == "RUNNING" {
				return updateEnvFunc(env.Id().String(), time.Time{}, envState, "success: the environment is in RUNNING state after START_ACTIVITY transition")
			} else {
				return updateEnvFunc(env.Id().String(), time.Time{}, envState, "error: the environment is in "+envState+" state after START_ACTIVITY transition")
			}
		}
		if strings.Contains(trigger, "STOP_ACTIVITY") {
			if envState == "CONFIGURED" {
				return updateEnvFunc(env.Id().String(), time.Time{}, envState, "success: the environment is in CONFIGURED state after STOP_ACTIVITY transition")
			} else {
				return updateEnvFunc(env.Id().String(), time.Time{}, envState, "error: the environment is in "+envState+" state after STOP_ACTIVITY transition")
			}
		}
		if strings.Contains(trigger, "EXIT") {
			if envState == "DONE" {
				return updateEnvFunc(env.Id().String(), time.Now(), envState, "success: the environment is in DONE state after EXIT transition")
			} else {
				return updateEnvFunc(env.Id().String(), time.Now(), envState, "error: the environment is in "+envState+" state after EXIT transition")
			}
		}
		if strings.Contains(trigger, "GO_ERROR") {
			if envState == "ERROR" {
				return updateEnvFunc(env.Id().String(), time.Time{}, envState, "success: the environment is in ERROR state after GO_ERROR transition")
			} else {
				return updateEnvFunc(env.Id().String(), time.Time{}, envState, "error: the environment is in "+envState+" state after GO_ERROR transition")
			}
		}
		if strings.Contains(trigger, "RECOVER") {
			if envState == "DEPLOYED" {
				return updateEnvFunc(env.Id().String(), time.Time{}, envState, "success: the environment is in DEPLOYED state after RECOVER transition")
			} else {
				return updateEnvFunc(env.Id().String(), time.Time{}, envState, "error: the environment is in "+envState+" state after RECOVER transition")
			}
		}
		log.WithField("partition", envId).
			WithField("call", call).
			Error("could not obtain transition in UpdateEnv from trigger: ", trigger)

		call.VarStack["__call_error_reason"] = err.Error()
		call.VarStack["__call_error"] = callFailedStr
		return
	}

	return
}

func (p *Plugin) Destroy() error {
	return nil
}
