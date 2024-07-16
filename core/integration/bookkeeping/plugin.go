/*
* === This file is part of ALICE O² ===
*
* Copyright 2021-2024 CERN and copyright holders of ALICE O².
* Author: Claire Guyot <claire.guyot@cern.ch>
*         Teo Mrnjavac <teo.m@cern.ch>
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

// Generate protofiles using the .protos imported from Bookkeeping in the Makefile
//go:generate protoc --go_out=. --go_opt=paths=source_relative protos/bkcommon.proto
//go:generate protoc --go_out=. --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative --go-grpc_out=require_unimplemented_servers=false:. protos/environment.proto
//go:generate protoc --go_out=. --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative --go-grpc_out=require_unimplemented_servers=false:. protos/flp.proto
//go:generate protoc --go_out=. --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative --go-grpc_out=require_unimplemented_servers=false:. protos/log.proto
//go:generate protoc --go_out=. --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative --go-grpc_out=require_unimplemented_servers=false:. protos/run.proto
//go:generate protoc --go_out=. --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative --go-grpc_out=require_unimplemented_servers=false:. protos/lhcFill.proto

package bookkeeping

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	bkpb "github.com/AliceO2Group/Control/core/integration/bookkeeping/protos"
	"google.golang.org/grpc"

	"github.com/AliceO2Group/Control/common/logger/infologger"
	"github.com/AliceO2Group/Control/common/utils/uid"
	"github.com/AliceO2Group/Control/core/environment"
	"github.com/AliceO2Group/Control/core/integration"
	"github.com/AliceO2Group/Control/core/workflow/callable"
	"github.com/spf13/viper"
)

const (
	BKP_RUN_TIMEOUT  = 30 * time.Second
	BKP_ENV_TIMEOUT  = 30 * time.Second
	BKP_FILL_TIMEOUT = 30 * time.Second
)

type Plugin struct {
	bookkeepingHost string
	bookkeepingPort int

	// Plugin client connecting to the Bookkeeping gRPC server
	bookkeepingClient *RpcClient

	// Indicators linked to each ongoing environment that give some plugin-specific
	// data about the status of the current run (of that environment)
	missingUpdateRunStarts   map[string] /*envId*/ bool
	missingUpdateRunStartsMu sync.Mutex
	pendingRunStops          map[string] /*envId*/ int64
	pendingRunStopsMu        sync.Mutex
	pendingO2Starts          map[string] /*envId*/ bool
	pendingO2StartsMu        sync.Mutex
	pendingO2Stops           map[string] /*envId*/ bool
	pendingO2StopsMu         sync.Mutex
	pendingTrgStarts         map[string] /*envId*/ bool
	pendingTrgStartsMu       sync.Mutex
	pendingTrgStops          map[string] /*envId*/ bool
	pendingTrgStopsMu        sync.Mutex
}

/**********************************************/
// Plugin and inherited methods Instanciation //
/**********************************************/
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
		bookkeepingHost:        u.Hostname(),
		bookkeepingPort:        portNumber,
		bookkeepingClient:      nil,
		missingUpdateRunStarts: make(map[string]bool),
		pendingRunStops:        make(map[string]int64),
		pendingO2Starts:        make(map[string]bool),
		pendingO2Stops:         make(map[string]bool),
		pendingTrgStarts:       make(map[string]bool),
		pendingTrgStops:        make(map[string]bool),
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

/*******************************************************/
// Utility methods for GetData and GetEnvironmentsData //
/*******************************************************/
func (p *Plugin) getMissingUpdateRunStartsForEnvs(envIds []uid.ID) map[uid.ID]bool {
	p.missingUpdateRunStartsMu.Lock()
	defer p.missingUpdateRunStartsMu.Unlock()

	if p.missingUpdateRunStarts == nil {
		return nil
	}

	out := make(map[uid.ID]bool)

	for _, envId := range envIds {
		if missingStart, ok := p.missingUpdateRunStarts[envId.String()]; ok {
			out[envId] = missingStart
		}
	}
	return out
}

func (p *Plugin) getPendingRunStopsForEnvs(envIds []uid.ID) map[uid.ID]string {
	p.pendingRunStopsMu.Lock()
	defer p.pendingRunStopsMu.Unlock()

	if p.pendingRunStops == nil {
		return nil
	}

	out := make(map[uid.ID]string)

	for _, envId := range envIds {
		if runStop, ok := p.pendingRunStops[envId.String()]; ok {
			out[envId] = string(runStop)
		}
	}
	return out
}

func (p *Plugin) getPendingO2StartsForEnvs(envIds []uid.ID) map[uid.ID]bool {
	p.pendingO2StartsMu.Lock()
	defer p.pendingO2StartsMu.Unlock()

	if p.pendingO2Starts == nil {
		return nil
	}

	out := make(map[uid.ID]bool)

	for _, envId := range envIds {
		if o2Start, ok := p.pendingO2Starts[envId.String()]; ok {
			out[envId] = o2Start
		}
	}
	return out
}

func (p *Plugin) getPendingO2StopsForEnvs(envIds []uid.ID) map[uid.ID]bool {
	p.pendingO2StopsMu.Lock()
	defer p.pendingO2StopsMu.Unlock()

	if p.pendingO2Stops == nil {
		return nil
	}

	out := make(map[uid.ID]bool)

	for _, envId := range envIds {
		if o2Stop, ok := p.pendingO2Stops[envId.String()]; ok {
			out[envId] = o2Stop
		}
	}
	return out
}

func (p *Plugin) getPendingTrgStartsForEnvs(envIds []uid.ID) map[uid.ID]bool {
	p.pendingTrgStartsMu.Lock()
	defer p.pendingTrgStartsMu.Unlock()

	if p.pendingTrgStarts == nil {
		return nil
	}

	out := make(map[uid.ID]bool)

	for _, envId := range envIds {
		if trgStart, ok := p.pendingTrgStarts[envId.String()]; ok {
			out[envId] = trgStart
		}
	}
	return out
}

func (p *Plugin) getPendingTrgStopsForEnvs(envIds []uid.ID) map[uid.ID]bool {
	p.pendingTrgStopsMu.Lock()
	defer p.pendingTrgStopsMu.Unlock()

	if p.pendingTrgStops == nil {
		return nil
	}

	out := make(map[uid.ID]bool)

	for _, envId := range envIds {
		if trgStop, ok := p.pendingTrgStops[envId.String()]; ok {
			out[envId] = trgStop
		}
	}
	return out
}

/*******************************************/
// GetData and GetEnvironmentsData methods //
/*******************************************/

// Make plugin-specific data across all instances available per data field
func (p *Plugin) GetData(_ []any) string {
	if p == nil || p.bookkeepingClient == nil {
		return ""
	}

	envIds := environment.ManagerInstance().Ids()

	outMap := make(map[string]interface{})
	outMap["missingUpdateRunStarts"] = p.getMissingUpdateRunStartsForEnvs(envIds)
	outMap["pendingRunStops"] = p.getPendingRunStopsForEnvs(envIds)
	outMap["pendingO2Starts"] = p.getPendingO2StartsForEnvs(envIds)
	outMap["pendingO2Stops"] = p.getPendingO2StopsForEnvs(envIds)
	outMap["pendingTrgStarts"] = p.getPendingTrgStartsForEnvs(envIds)
	outMap["pendingTrgStops"] = p.getPendingTrgStopsForEnvs(envIds)

	out, err := json.Marshal(outMap)
	if err != nil {
		return ""
	}
	return string(out[:])
}

// Make plugin-specific data across all instances available per environment
func (p *Plugin) GetEnvironmentsData(envIds []uid.ID) map[uid.ID]string {
	if p == nil || p.bookkeepingClient == nil {
		return nil
	}

	inMissingStart := p.getMissingUpdateRunStartsForEnvs(envIds)
	inRunStopMap := p.getPendingRunStopsForEnvs(envIds)
	inO2StartMap := p.getPendingO2StartsForEnvs(envIds)
	inO2StopMap := p.getPendingO2StopsForEnvs(envIds)
	inTrgStartMap := p.getPendingTrgStartsForEnvs(envIds)
	inTrgStopMap := p.getPendingTrgStopsForEnvs(envIds)

	envMap := make(map[string]interface{})
	out := make(map[uid.ID]string)

	for _, envId := range envIds {
		if missingStart, ok := inMissingStart[envId]; ok {
			envMap["missingUpdateRunStart"] = missingStart
		}
		if runStop, ok := inRunStopMap[envId]; ok {
			envMap["runNumber"] = runStop
		}
		if o2Start, ok := inO2StartMap[envId]; ok {
			envMap["pendingO2Start"] = o2Start
		}
		if o2Stop, ok := inO2StopMap[envId]; ok {
			envMap["pendingO2Stop"] = o2Stop
		}
		if trgStart, ok := inTrgStartMap[envId]; ok {
			envMap["pendingTrgStart"] = trgStart
		}
		if trgStop, ok := inTrgStopMap[envId]; ok {
			envMap["pendingTrgStop"] = trgStop
		}
		outJson, err := json.Marshal(envMap)
		if err == nil {
			out[envId] = string(outJson[:])
		}
	}
	return out
}

func (p *Plugin) GetEnvironmentsShortData(envIds []uid.ID) map[uid.ID]string {
	return p.GetEnvironmentsData(envIds)
}

/*************************************/
// Initialize the Bookkeeping client //
/*************************************/
func (p *Plugin) Init(instanceId string) error {
	if p.bookkeepingClient == nil {
		cxt, cancel := context.WithCancel(context.Background())
		p.bookkeepingClient = NewClient(cxt, cancel, p.GetEndpoint())
		if p.bookkeepingClient == nil {
			return fmt.Errorf("failed to connect to Bookkeeping service on %s", p.GetEndpoint())
		}
		log.Debug("Bookkeeping plugin initialized")
	}
	return nil
}

/*************************************/
// ObjectStack and CallStack methods //
/*************************************/

// ObjectStack method
func (p *Plugin) ObjectStack(_ map[string]string, _ map[string]string) (stack map[string]interface{}) {
	stack = make(map[string]interface{})
	return stack
}

// CallStack method, which adds functions to the call stack that can then be called
// under certain conditions via hooks and triggers in the readout-dataflow workflow
func (p *Plugin) CallStack(data interface{}) (stack map[string]interface{}) {

	/*** Set variables that are available to all functions in the call stack ***/
	call, ok := data.(*callable.Call)
	if !ok {
		return
	}
	// The varStack is a map of variables available throughout the core and is a collection
	// of Defaults, Vars and UserVars from the environment, as well as runtime variables set
	// in other parts of AliECS
	varStack := call.VarStack
	envId, ok := varStack["environment_id"]
	if !ok {
		err := errors.New("cannot acquire environment ID")
		log.Error(err)

		// These two lines are necessary to get the error out of the scope of only that function call
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

	/*****************************/
	/*** Run related functions ***/
	/*****************************/

	// StartOfRun function that creates the run in Bookkeeping and initializes it with minimal information
	stack["StartOfRun"] = func() (out string) {

		/*** Get variables that are available at run creation (before START_ACTIVITY) ***/
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
				WithField("run", runNumber64).
				WithField("partition", envId).
				WithField("call", "StartOfRun").
				Error("Bookkeeping plugin SOR error")

			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr

			return
		}

		runNumber := env.GetCurrentRunNumber()
		runNumber32 := int32(runNumber64)

		rnString := strconv.FormatUint(uint64(runNumber), 10)

		flps := env.GetFLPs()

		// The enabled status of CTP Readout and its FLP is governed by the variable ctp_readout_enabled, so it requires
		// special treatment to be included in the list of FLPs
		ctpReadoutEnabled, err := strconv.ParseBool(env.GetKV("", "ctp_readout_enabled"))
		if err != nil {
			log.WithError(err).
				WithField("run", runNumber64).
				WithField("partition", envId).
				WithField("call", "StartOfRun").
				Warning("cannot parse ctp_readout_enabled")
		}
		if ctpReadoutEnabled {
			ctpReadoutFlp := env.GetKV("", "ctp_readout_host")
			if len(ctpReadoutFlp) > 0 {
				flps = append(flps, ctpReadoutFlp)
			}
		}

		epns, err := strconv.ParseInt(env.GetKV("", "odc_n_epns"), 10, 0)
		if err != nil {
			log.WithError(err).
				WithField("run", runNumber64).
				WithField("partition", envId).
				WithField("call", "StartOfRun").
				Warning("cannot parse number of EPNs")
		}
		ddEnabled, err := strconv.ParseBool(env.GetKV("", "dd_enabled"))
		if err != nil {
			log.WithError(err).
				WithField("run", runNumber64).
				WithField("partition", envId).
				WithField("call", "StartOfRun").
				Warning("cannot parse DD enabled")
		}
		dcsEnabled, err := strconv.ParseBool(env.GetKV("", "dcs_enabled"))
		if err != nil {
			log.WithError(err).
				WithField("run", runNumber64).
				WithField("partition", envId).
				WithField("call", "StartOfRun").
				Warning("cannot parse DCS enabled")
		}
		epnEnabled, err := strconv.ParseBool(env.GetKV("", "epn_enabled"))
		if err != nil {
			log.WithError(err).
				WithField("run", runNumber64).
				WithField("partition", envId).
				WithField("call", "StartOfRun").
				Warning("cannot parse EPN enabled")
		}
		odcTopology := env.GetKV("", "odc_topology")
		odcTopologyFullname, ok := env.Workflow().GetVars().Get("odc_topology_fullname")
		if !ok {
			log.WithField("run", runNumber64).
				WithField("partition", envId).
				WithField("call", "UpdateRun").
				Warning("cannot acquire ODC topology fullname")
		}
		detectors := env.GetActiveDetectors().StringList()
		var detectorsList = make([]bkpb.Detector, 0)
		for _, name := range detectors {
			detectorsList = append(detectorsList, bkpb.Detector(bkpb.Detector_value["DETECTOR_"+name]))
		}

		/*** Create and send creation requests to Bookkeeping server ***/

		inRun := bkpb.RunCreationRequest{
			RunNumber:           runNumber32,
			EnvironmentId:       env.Id().String(),
			NDetectors:          int32(len(env.GetActiveDetectors())),
			NEpns:               int32(epns),
			NFlps:               int32(len(flps)),
			RunType:             bkpb.RunType(env.GetRunType()),
			DdFlp:               ddEnabled,
			Dcs:                 dcsEnabled,
			Epn:                 epnEnabled,
			EpnTopology:         odcTopology,
			OdcTopologyFullName: &odcTopologyFullname,
			Detectors:           detectorsList,
		}

		// Send the run creation request
		timeout := callable.AcquireTimeout(BKP_RUN_TIMEOUT, varStack, "CreateRun", envId)
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		_, err1 := p.bookkeepingClient.RunServiceClient.Create(ctx, &inRun, grpc.EmptyCallOption{})
		if err1 != nil {
			// If the run creation request fails, we still send a log creation request
			log.WithError(err1).
				WithField("run", runNumber64).
				WithField("partition", envId).
				WithField("call", "RunServiceClient.Create").
				Error("Bookkeeping API RunServiceClient: Create error")

			runNumbers := make([]int32, 0)
			inLog := bkpb.LogCreationRequest{
				RunNumbers:  runNumbers,
				Title:       fmt.Sprintf("Log for run %s and environment %s", rnString, env.Id().String()),
				Text:        env.GetVarsAsString(),
				ParentLogId: nil,
			}

			timeout = callable.AcquireTimeout(BKP_RUN_TIMEOUT, varStack, "CreateLog", envId)
			ctx, cancel = context.WithTimeout(context.Background(), timeout)
			defer cancel()
			_, err2 := p.bookkeepingClient.LogServiceClient.Create(ctx, &inLog, grpc.EmptyCallOption{})
			if err2 != nil {
				log.WithError(err2).
					WithField("run", runNumber64).
					WithField("partition", envId).
					WithField("call", "LogServiceClient.Create").
					Error("Bookkeeping API LogServiceClient: Create error")
				err = errors.New(err1.Error() + err2.Error())

				call.VarStack["__call_error_reason"] = err.Error()
				call.VarStack["__call_error"] = callFailedStr
				return
			} else {
				log.WithField("run", runNumber64).
					WithField("partition", envId).
					Debug("Bookkeeping API LogServiceClient: Create call successful")
			}
			return
		} else {
			// If the run creation request succeeds, we set the run status indicators
			p.missingUpdateRunStartsMu.Lock()
			p.missingUpdateRunStarts[envId] = true
			p.missingUpdateRunStartsMu.Unlock()
			p.pendingRunStopsMu.Lock()
			p.pendingRunStops[envId] = runNumber64
			p.pendingRunStopsMu.Unlock()
			p.pendingO2StartsMu.Lock()
			p.pendingO2Starts[envId] = true
			p.pendingO2StartsMu.Unlock()
			p.pendingO2StopsMu.Lock()
			p.pendingO2Stops[envId] = true
			p.pendingO2StopsMu.Unlock()
			p.pendingTrgStartsMu.Lock()
			p.pendingTrgStarts[envId] = true
			p.pendingTrgStartsMu.Unlock()
			p.pendingTrgStopsMu.Lock()
			p.pendingTrgStops[envId] = true
			p.pendingTrgStopsMu.Unlock()
			log.WithField("run", runNumber64).
				WithField("partition", envId).
				Debug("Bookkeeping API RunServiceClient: Create call successful")
		}

		runNumbers := make([]int32, 0)
		runNumbers = append(runNumbers, runNumber32)
		inLog := bkpb.LogCreationRequest{
			RunNumbers:  runNumbers,
			Title:       fmt.Sprintf("Log for run %s and environment %s", rnString, env.Id().String()),
			Text:        env.GetVarsAsString(),
			ParentLogId: nil,
		}

		// Send the log creation request
		timeout = callable.AcquireTimeout(BKP_RUN_TIMEOUT, varStack, "CreateLog", envId)
		ctx, cancel = context.WithTimeout(context.Background(), timeout)
		defer cancel()
		_, err = p.bookkeepingClient.LogServiceClient.Create(ctx, &inLog, grpc.EmptyCallOption{})
		if err != nil {
			log.WithError(err).
				WithField("run", runNumber64).
				WithField("partition", envId).
				WithField("call", "LogServiceClient.Create").
				Error("Bookkeeping API LogServiceClient: Create error")

			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr
			return
		} else {
			log.WithField("run", runNumber64).
				WithField("partition", envId).
				Debug("Bookkeeping API LogServiceClient: Create call successful")
		}

		var inFlps = bkpb.ManyFlpsCreationRequest{
			Flps: make([]*bkpb.FlpCreationRequest, len(flps)),
		}

		for _, flp := range flps {
			inFlp := &bkpb.FlpCreationRequest{
				Name:      flp,
				Hostname:  flp,
				RunNumber: &runNumber32,
			}
			inFlps.Flps = append(inFlps.Flps, inFlp)
		}

		// Send the flps creation request
		timeout = callable.AcquireTimeout(BKP_RUN_TIMEOUT, varStack, "CreateFlp", envId)
		ctx, cancel = context.WithTimeout(context.Background(), timeout)
		defer cancel()
		_, err = p.bookkeepingClient.FlpServiceClient.CreateMany(ctx, &inFlps, grpc.EmptyCallOption{})
		if err != nil {
			log.WithError(err).
				WithField("run", runNumber64).
				WithField("partition", envId).
				WithField("call", "FlpServiceClient.CreateMany").
				Error("Bookkeeping API FlpServiceClient: CreateMany error")

			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr
			return
		} else {
			log.WithField("run", runNumber64).
				WithField("partition", envId).
				Debug("Bookkeeping API FlpServiceClient: CreateMany call successful")
		}
		return
	}

	// Update function common to UpdateRunStart and UpdateRunStop
	updateRunFunc := func(runNumber64 int64, state string, timeO2StartInput string, timeO2EndInput string, timeTrgStartInput string, timeTrgEndInput string) (out string) {

		/*** Get variables that are available when the update is called ***/
		callFailedStr := "Bookkeeping UpdateRun call failed"
		trgGlobalRunEnabled, err := strconv.ParseBool(env.GetKV("", "trg_global_run_enabled"))
		if err != nil {
			log.WithError(err).
				WithField("run", runNumber64).
				WithField("partition", envId).
				WithField("call", "UpdateRun").
				Warning("cannot parse TRG global run enabled")
		}
		trgEnabled, err := strconv.ParseBool(env.GetKV("", "trg_enabled"))
		if err != nil {
			log.WithError(err).
				WithField("run", runNumber64).
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
			log.WithField("run", runNumber64).
				WithField("partition", envId).
				WithField("call", "UpdateRun").
				Warning("cannot acquire PDP workflow configuration mode")
		}
		pdpTopology, ok := varStack["pdp_topology_description_library_file"]
		if !ok {
			log.WithField("run", runNumber64).
				WithField("partition", envId).
				WithField("call", "UpdateRun").
				Warning("cannot acquire PDP topology description library file")
		}
		pdpParameters, ok := varStack["pdp_workflow_parameters"]
		if !ok {
			log.WithField("run", runNumber64).
				WithField("partition", envId).
				WithField("call", "UpdateRun").
				Warning("cannot acquire PDP workflow parameters")
		}
		pdpBeam, ok := varStack["pdp_beam_type"]
		if !ok {
			log.WithField("run", runNumber64).
				WithField("partition", envId).
				WithField("call", "UpdateRun").
				Warning("cannot acquire PDP beam type")
		}
		tfbMode := env.GetKV("", "tfb_dd_mode")
		odcTopologyFullname, ok := env.Workflow().GetVars().Get("odc_topology_fullname")
		if !ok {
			log.WithField("run", runNumber64).
				WithField("partition", envId).
				WithField("call", "UpdateRun").
				Warning("cannot acquire ODC topology fullname")
		}
		lhcPeriod := env.GetKV("", "lhc_period")
		readoutUri, ok := varStack["readout_cfg_uri"]

		// O2 timestamps formatting
		var timeO2StartOutput *int64 = nil
		timeO2StartTemp, err := strconv.ParseInt(timeO2StartInput, 10, 64)
		if err != nil {
			log.WithField("run", runNumber64).
				WithField("time", timeO2StartInput).
				Warning("cannot parse O2 start time")
			timeO2StartTemp = -1
		}
		if timeO2StartInput != "" || timeO2StartTemp > 0 {
			timeO2StartOutput = &timeO2StartTemp
		}
		var timeO2EndTemp int64 = -1
		var timeO2EndOutput *int64 = nil
		if timeO2EndInput != "" {
			timeO2EndTemp, err = strconv.ParseInt(timeO2EndInput, 10, 64)
			if err != nil {
				log.WithField("run", runNumber64).
					WithField("time", timeO2EndInput).
					Warning("cannot parse O2 end time")
				timeO2EndTemp = -1
			}
		}
		if timeO2EndInput != "" || timeO2EndTemp > 0 {
			timeO2EndOutput = &timeO2EndTemp
		}

		// Trg timestamps formatting
		var timeTrgStartTemp int64 = -1
		var timeTrgStartOutput *int64 = nil
		var timeTrgEndTemp int64 = -1
		var timeTrgEndOutput *int64 = nil
		if trg == "LTU" || trg == "CTP" {
			timeTrgStartTemp, err = strconv.ParseInt(timeTrgStartInput, 10, 64)
			if err != nil {
				log.WithField("run", runNumber64).
					WithField("time", timeTrgStartInput).
					Warning("cannot parse Trg start time")
				timeTrgStartTemp = -1
			}
			if timeTrgStartInput != "" || timeTrgStartTemp > 0 {
				timeTrgStartOutput = &timeTrgStartTemp
			}
			// If UpdateRunStart was not called and the Trg start time is missing,
			// it is set to the O2 start time.
			p.missingUpdateRunStartsMu.Lock()
			currentMissingUpdateRunStarts := p.missingUpdateRunStarts[envId]
			p.missingUpdateRunStartsMu.Unlock()
			if currentMissingUpdateRunStarts == true && trgEnabled && timeTrgStartOutput == nil {
				p.pendingO2StartsMu.Lock()
				currentPendingO2Starts := p.pendingO2Starts[envId]
				p.pendingO2StartsMu.Unlock()
				if currentPendingO2Starts == false {
					timeTrgStartOutput = timeO2StartOutput
					p.pendingTrgStartsMu.Lock()
					p.pendingTrgStarts[envId] = false
					p.pendingTrgStartsMu.Unlock()
					log.WithField("run", runNumber64).
						WithField("partition", envId).
						Debug("Bookkeeping API RunServiceClient: Update call: completing missing Trg start timestamp after missing UpdateRunStart call")
				} else {
					log.WithField("run", runNumber64).
						WithField("partition", envId).
						Warning("Bookkeeping API RunServiceClient: Update call: run information incomplete, missing O2 start time after missing UpdateRunStart call")
				}
			}
			timeTrgEndTemp, err = strconv.ParseInt(timeTrgEndInput, 10, 64)
			if err != nil {
				log.WithField("run", runNumber64).
					WithField("time", timeTrgEndInput).
					Warning("cannot parse Trg end time")
				timeTrgEndTemp = -1
			}
			if timeTrgEndInput != "" || timeTrgEndTemp > 0 {
				timeTrgEndOutput = &timeTrgEndTemp
			}
		}

		// Create the run update struct
		inRun := bkpb.RunUpdateRequest{
			RunNumber:                         int32(runNumber64),
			TimeO2Start:                       timeO2StartOutput,
			TimeO2End:                         timeO2EndOutput,
			TimeTrgStart:                      timeTrgStartOutput,
			TimeTrgEnd:                        timeTrgEndOutput,
			TriggerValue:                      &trg,
			PdpConfigOption:                   &pdpConfig,
			PdpTopologyDescriptionLibraryFile: &pdpTopology,
			TfbDdMode:                         &tfbMode,
			LhcPeriod:                         &lhcPeriod,
			OdcTopologyFullName:               &odcTopologyFullname,
			PdpWorkflowParameters:             &pdpParameters,
			PdpBeamType:                       &pdpBeam,
			ReadoutCfgUri:                     &readoutUri,
		}

		// If it is an UpdateRunStop call, then SOR timestamps can be updated in the request under certain conditions
		if function, ok := varStack["__call_func"]; ok && strings.Contains(function, "UpdateRunStop") {
			// If O2 start time is not available after UpdateRunStart was missing,
			// we can't update it anymore as we have no way of knowing/approximating it.
			// Else if O2 start time or Trg start time is available, we can update it safely
			// because it means that even if they weren't missing during the UpdateRunStart call,
			// they will be the same and can be overwritten.
			p.missingUpdateRunStartsMu.Lock()
			currentMissingUpdateRunStarts := p.missingUpdateRunStarts[envId]
			p.missingUpdateRunStartsMu.Unlock()
			if currentMissingUpdateRunStarts == true && timeO2StartOutput == nil {
				log.WithField("run", runNumber64).
					WithField("partition", envId).
					Warning("Bookkeeping API RunServiceClient: Update call: run information incomplete, missing SOR timestamps after missing UpdateRunStart call")
			} else if timeO2StartOutput == nil && timeTrgStartOutput != nil {
				inRun = bkpb.RunUpdateRequest{
					RunNumber:                         int32(runNumber64),
					TimeO2End:                         timeO2EndOutput,
					TimeTrgStart:                      timeTrgStartOutput,
					TimeTrgEnd:                        timeTrgEndOutput,
					TriggerValue:                      &trg,
					PdpConfigOption:                   &pdpConfig,
					PdpTopologyDescriptionLibraryFile: &pdpTopology,
					TfbDdMode:                         &tfbMode,
					LhcPeriod:                         &lhcPeriod,
					OdcTopologyFullName:               &odcTopologyFullname,
					PdpWorkflowParameters:             &pdpParameters,
					PdpBeamType:                       &pdpBeam,
					ReadoutCfgUri:                     &readoutUri,
				}
			} else if timeO2StartOutput != nil && timeTrgStartOutput == nil {
				inRun = bkpb.RunUpdateRequest{
					RunNumber:                         int32(runNumber64),
					TimeO2Start:                       timeO2StartOutput,
					TimeO2End:                         timeO2EndOutput,
					TimeTrgEnd:                        timeTrgEndOutput,
					TriggerValue:                      &trg,
					PdpConfigOption:                   &pdpConfig,
					PdpTopologyDescriptionLibraryFile: &pdpTopology,
					TfbDdMode:                         &tfbMode,
					LhcPeriod:                         &lhcPeriod,
					OdcTopologyFullName:               &odcTopologyFullname,
					PdpWorkflowParameters:             &pdpParameters,
					PdpBeamType:                       &pdpBeam,
					ReadoutCfgUri:                     &readoutUri,
				}
			}
		}

		// Send the run update request
		timeout := callable.AcquireTimeout(BKP_RUN_TIMEOUT, varStack, "UpdateRun", envId)
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		_, err = p.bookkeepingClient.RunServiceClient.Update(ctx, &inRun, grpc.EmptyCallOption{})
		if err != nil {
			log.WithError(err).
				WithField("run", runNumber64).
				WithField("partition", envId).
				WithField("call", "RunServiceClient.Update").
				Error("Bookkeeping API RunServiceClient: Update error")

			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr
			return
		} else {
			var updatedRun string
			if function, ok := varStack["__call_func"]; ok && strings.Contains(function, "UpdateRunStop") {
				// If the update is successful and it is an UpdateRunStop call, we check if we are missing
				// EOR timestamps, and if that is the case, we set them to Now() and send a new update request
				p.missingUpdateRunStartsMu.Lock()
				currentMissingUpdateRunStarts := p.missingUpdateRunStarts[envId]
				p.missingUpdateRunStartsMu.Unlock()
				if currentMissingUpdateRunStarts == true {
					defer func() {
						p.pendingO2StartsMu.Lock()
						delete(p.pendingO2Starts, envId)
						p.pendingO2StartsMu.Unlock()
					}()
					defer func() {
						p.pendingTrgStartsMu.Lock()
						delete(p.pendingTrgStarts, envId)
						p.pendingTrgStartsMu.Unlock()
					}()
				}
				defer func() {
					p.pendingRunStopsMu.Lock()
					delete(p.pendingRunStops, envId)
					p.pendingRunStopsMu.Unlock()
				}()
				defer func() {
					p.pendingO2StopsMu.Lock()
					delete(p.pendingO2Stops, envId)
					p.pendingO2StopsMu.Unlock()
				}()
				defer func() {
					p.pendingTrgStopsMu.Lock()
					delete(p.pendingTrgStops, envId)
					p.pendingTrgStopsMu.Unlock()
				}()

				p.pendingO2StopsMu.Lock()
				currentPendingO2Stops := p.pendingO2Stops[envId]
				p.pendingO2StopsMu.Unlock()
				p.pendingTrgStopsMu.Lock()
				currentPendingTrgStops := p.pendingTrgStops[envId]
				p.pendingTrgStopsMu.Unlock()
				if currentPendingO2Stops == true || (trgEnabled && currentPendingTrgStops == true) {
					updatedRun = "INCOMPLETE STOP"
					if currentPendingO2Stops == true {
						timeO2EndTemp = time.Now().UnixMilli()
						timeO2EndOutput = &timeO2EndTemp
						log.WithField("run", runNumber64).
							WithField("partition", envId).
							Warning("Bookkeeping API RunServiceClient: Update call: run information incomplete, missing O2 end time")
					}
					if trgEnabled && currentPendingTrgStops == true {
						timeTrgEndTemp = time.Now().UnixMilli()
						timeTrgEndOutput = &timeO2EndTemp
						log.WithField("run", runNumber64).
							WithField("partition", envId).
							Warning("Bookkeeping API RunServiceClient: Update call: run information incomplete, missing Trg end time")
					}
					inRun = bkpb.RunUpdateRequest{
						RunNumber:  int32(runNumber64),
						TimeO2End:  timeO2EndOutput,
						TimeTrgEnd: timeTrgEndOutput,
					}
					log.WithField("run", runNumber64).
						WithField("partition", envId).
						Debug("Bookkeeping API RunServiceClient: Update call: completing missing EOR timestamps")

					timeout = callable.AcquireTimeout(BKP_RUN_TIMEOUT, varStack, "UpdateRun", envId)
					ctx, cancel = context.WithTimeout(context.Background(), timeout)
					defer cancel()
					_, err = p.bookkeepingClient.RunServiceClient.Update(ctx, &inRun, grpc.EmptyCallOption{})
					if err != nil {
						log.WithError(err).
							WithField("run", runNumber64).
							WithField("partition", envId).
							WithField("call", "RunServiceClient.Update").
							Error("Bookkeeping API RunServiceClient: Update error")

						call.VarStack["__call_error_reason"] = err.Error()
						call.VarStack["__call_error"] = callFailedStr
						return
					}
				} else {
					updatedRun = "STOPPED"
				}
			} else if function, ok := varStack["__call_func"]; ok && strings.Contains(function, "UpdateRunStart") {
				// If the update is successful and it is an UpdateRunStart call, we check if we are missing
				// SOR timestamps, and if that is the case, we set them to Now() and send a new update request
				defer func() {
					p.pendingO2StartsMu.Lock()
					delete(p.pendingO2Starts, envId)
					p.pendingO2StartsMu.Unlock()
				}()
				defer func() {
					p.pendingTrgStartsMu.Lock()
					delete(p.pendingTrgStarts, envId)
					p.pendingTrgStartsMu.Unlock()
				}()
				p.pendingO2StartsMu.Lock()
				currentPendingO2Starts := p.pendingO2Starts[envId]
				p.pendingO2StartsMu.Unlock()
				p.pendingTrgStartsMu.Lock()
				currentPendingTrgStarts := p.pendingTrgStarts[envId]
				p.pendingTrgStartsMu.Unlock()
				if currentPendingO2Starts == true || (trgEnabled && currentPendingTrgStarts == true) {
					updatedRun = "INCOMPLETE START"
					if currentPendingO2Starts == true {
						timeO2StartTemp = time.Now().UnixMilli()
						timeO2StartOutput = &timeO2StartTemp
						log.WithField("run", runNumber64).
							WithField("partition", envId).
							Warning("Bookkeeping API RunServiceClient: Update call: run information incomplete, missing O2 start time")
					}
					if trgEnabled && currentPendingTrgStarts == true {
						timeTrgStartTemp = time.Now().UnixMilli()
						timeTrgStartOutput = &timeO2StartTemp
						log.WithField("run", runNumber64).
							WithField("partition", envId).
							Warning("Bookkeeping API RunServiceClient: Update call: run information incomplete, missing Trg start time")
					}
					inRun = bkpb.RunUpdateRequest{
						RunNumber:    int32(runNumber64),
						TimeO2Start:  timeO2StartOutput,
						TimeTrgStart: timeTrgStartOutput,
					}
					log.WithField("run", runNumber64).
						WithField("partition", envId).
						Debug("Bookkeeping API RunServiceClient: Update call: completing missing SOR timestamps")

					timeout = callable.AcquireTimeout(BKP_RUN_TIMEOUT, varStack, "UpdateRun", envId)
					ctx, cancel = context.WithTimeout(context.Background(), timeout)
					defer cancel()
					_, err = p.bookkeepingClient.RunServiceClient.Update(ctx, &inRun, grpc.EmptyCallOption{})
					if err != nil {
						log.WithError(err).
							WithField("run", runNumber64).
							WithField("partition", envId).
							WithField("call", "RunServiceClient.Update").
							Error("Bookkeeping API RunServiceClient: Update error")

						call.VarStack["__call_error_reason"] = err.Error()
						call.VarStack["__call_error"] = callFailedStr
						return
					}
				} else {
					updatedRun = "STARTED"
				}
			}
			log.WithField("run", runNumber64).
				WithField("updated to", updatedRun).
				WithField("partition", envId).
				Debug("Bookkeeping API RunServiceClient: Update call successful")
		}
		return
	}

	// UpdateRunStart function that updates the run in Bookkeeping with all available information
	stack["UpdateRunStart"] = func() (out string) {

		/*** Get variables that are available at run start update (after START_ACTIVITY) ***/
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
				WithField("run", runNumber64).
				WithField("partition", envId).
				WithField("call", "UpdateRunStart").
				Error("Bookkeeping plugin UpdateRunStart error")

			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr

			return
		}

		O2StartTime := varStack["run_start_time_ms"]
		if O2StartTime != "" {
			p.pendingO2StartsMu.Lock()
			p.pendingO2Starts[envId] = false
			p.pendingO2StartsMu.Unlock()
		}
		TrgStartTime := varStack["trg_start_time_ms"]
		if TrgStartTime != "" {
			p.pendingTrgStartsMu.Lock()
			p.pendingTrgStarts[envId] = false
			p.pendingTrgStartsMu.Unlock()
		}

		p.missingUpdateRunStartsMu.Lock()
		p.missingUpdateRunStarts[envId] = false
		p.missingUpdateRunStartsMu.Unlock()

		return updateRunFunc(runNumber64, "test", O2StartTime, "", TrgStartTime, "")
	}

	// UpdateRunStop function that updates the run in Bookkeeping with all available information and deletes it from the plugin
	stack["UpdateRunStop"] = func() (out string) {

		/*** Get variables that are available at run stop update (after STOP_ACTIVITY, GO_ERROR or DESTROY) ***/
		callFailedStr := "Bookkeeping UpdateRunStop call failed"

		rn := varStack["run_number"]
		if len(rn) == 0 {
			rn = varStack["last_run_number"]
			log.WithField("partition", envId).
				WithField(infologger.Run, rn).
				Debug("run number no longer set, using last known run number for Bookkeeping UpdateRunStop")
		}
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
				WithField("run", runNumber64).
				WithField("partition", envId).
				WithField("call", "UpdateRunStop").
				Error("Bookkeeping plugin UpdateRunStop error")

			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr

			return
		}

		O2StartTime := varStack["run_start_time_ms"]
		if O2StartTime != "" {
			p.pendingO2StartsMu.Lock()
			p.pendingO2Starts[envId] = false
			p.pendingO2StartsMu.Unlock()
		}
		O2EndTime := varStack["run_end_time_ms"]
		if O2EndTime != "" {
			p.pendingO2StopsMu.Lock()
			p.pendingO2Stops[envId] = false
			p.pendingO2StopsMu.Unlock()
		}

		TrgStartTime := varStack["trg_start_time_ms"]
		if TrgStartTime != "" {
			p.pendingTrgStartsMu.Lock()
			p.pendingTrgStarts[envId] = false
			p.pendingTrgStartsMu.Unlock()
		}
		TrgEndTime := varStack["trg_end_time_ms"]
		if TrgEndTime != "" {
			p.pendingTrgStopsMu.Lock()
			p.pendingTrgStops[envId] = false
			p.pendingTrgStopsMu.Unlock()
		}

		p.pendingRunStopsMu.Lock()
		_, hasPendingRunStops := p.pendingRunStops[envId]
		p.pendingRunStopsMu.Unlock()
		if hasPendingRunStops {
			return updateRunFunc(runNumber64, "test", O2StartTime, O2EndTime, TrgStartTime, TrgEndTime)
		} else {
			log.WithField("partition", envId).
				WithField("level", infologger.IL_Devel).
				Debug("skipping UpdateRun call, no pending run number found")
			return
		}
	}

	/*************************************/
	/*** Environment related functions ***/
	/*************************************/

	// CreateEnv function that creates the environment in Bookkeeping
	stack["CreateEnv"] = func() (out string) {

		/*** Get variables that are available at environment creation (before DEPLOY) ***/
		callFailedStr := "Bookkeeping CreateEnv call failed"

		if p.bookkeepingClient == nil {
			err = fmt.Errorf("Bookkeeping plugin not initialized, CreateEnv impossible")

			log.WithError(err).
				WithField("level", infologger.IL_Support).
				WithField("endpoint", viper.GetString("bookkeepingBaseUri")).
				WithField("partition", envId).
				WithField("call", "CreateEnv").
				Error("Bookkeeping plugin CreateEnv error")

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

		/*** Create and send environment creation request to Bookkeeping server ***/

		inEnv := bkpb.EnvironmentCreationRequest{
			Id:            env.Id().String(),
			Status:        &envState,
			StatusMessage: &statusMessage,
		}

		timeout := callable.AcquireTimeout(BKP_ENV_TIMEOUT, varStack, "UpdateRun", envId)
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		response, err := p.bookkeepingClient.EnvironmentServiceClient.Create(ctx, &inEnv, grpc.EmptyCallOption{})
		if response != nil {
			response.StatusMessage = "response not nil"
		}
		if err != nil {
			log.WithError(err).
				WithField("partition", envId).
				WithField("call", "EnvironmentServiceClient.Create").
				Error("Bookkeeping API EnvironmentServiceClient: Create error")

			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr
			return
		} else {
			log.WithField("partition", envId).
				Debug("Bookkeeping API EnvironmentServiceClient: Create call successful")
		}
		return
	}

	// Update function for UpdateEnv
	updateEnvFunc := func(envId string, toredownAt int64, status string, statusMessage string) (out string) {
		callFailedStr := "Bookkeeping UpdateEnv call failed"

		/*** Create and send environment update request to Bookkeeping server ***/

		inEnv := bkpb.EnvironmentUpdateRequest{
			Id:            env.Id().String(),
			ToredownAt:    &toredownAt,
			Status:        &status,
			StatusMessage: &statusMessage,
		}

		timeout := callable.AcquireTimeout(BKP_ENV_TIMEOUT, varStack, "UpdateRun", envId)
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		_, err = p.bookkeepingClient.EnvironmentServiceClient.Update(ctx, &inEnv, grpc.EmptyCallOption{})
		if err != nil {
			log.WithError(err).
				WithField("partition", envId).
				WithField("call", "EnvironmentServiceClient.Update").
				Error("Bookkeeping API EnvironmentServiceClient: Update error")

			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr
			return
		} else {
			log.WithField("partition", envId).
				WithField("state", status).
				Debug("Bookkeeping API EnvironmentServiceClient: Update call successful")
		}
		return
	}

	// UpdateEnv function that updates the environment in Bookkeeping with all available information
	stack["UpdateEnv"] = func() (out string) {
		callFailedStr := "Bookkeeping UpdateEnv call failed"

		if p.bookkeepingClient == nil {
			err = fmt.Errorf("Bookkeeping plugin not initialized, UpdateEnv impossible")

			log.WithError(err).
				WithField("level", infologger.IL_Support).
				WithField("endpoint", viper.GetString("bookkeepingBaseUri")).
				WithField("partition", envId).
				WithField("call", "UpdateEnv").
				Error("Bookkeeping plugin UpdateEnv error")

			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr

			return
		}

		// We update the environment status depending on the current environment state machine status
		envState := env.CurrentState()

		if strings.Contains(trigger, "DESTROY") {
			envState = "DESTROYED"
			return updateEnvFunc(env.Id().String(), time.Time{}.UnixMilli(), envState, "the environment is DESTROYED after DESTROY transition")
		}
		if strings.Contains(trigger, "DEPLOY") {
			if envState == "DEPLOYED" {
				return updateEnvFunc(env.Id().String(), time.Time{}.UnixMilli(), envState, "success: the environment is in DEPLOYED state after DEPLOY transition")
			} else {
				return updateEnvFunc(env.Id().String(), time.Time{}.UnixMilli(), envState, "error: the environment is in "+envState+" state after DEPLOY transition")
			}
		}
		if strings.Contains(trigger, "CONFIGURE") {
			if envState == "CONFIGURED" {
				return updateEnvFunc(env.Id().String(), time.Time{}.UnixMilli(), envState, "success: the environment is in CONFIGURED state after CONFIGURE transition")
			} else {
				return updateEnvFunc(env.Id().String(), time.Time{}.UnixMilli(), envState, "error: the environment is in "+envState+" state after CONFIGURE transition")
			}
		}
		if strings.Contains(trigger, "RESET") {
			if envState == "DEPLOYED" {
				return updateEnvFunc(env.Id().String(), time.Time{}.UnixMilli(), envState, "success: the environment is in DEPLOYED state after RESET transition")
			} else {
				return updateEnvFunc(env.Id().String(), time.Time{}.UnixMilli(), envState, "error: the environment is in "+envState+" state after RESET transition")
			}
		}
		if strings.Contains(trigger, "START_ACTIVITY") {
			if envState == "RUNNING" {
				return updateEnvFunc(env.Id().String(), time.Time{}.UnixMilli(), envState, "success: the environment is in RUNNING state after START_ACTIVITY transition")
			} else {
				return updateEnvFunc(env.Id().String(), time.Time{}.UnixMilli(), envState, "error: the environment is in "+envState+" state after START_ACTIVITY transition")
			}
		}
		if strings.Contains(trigger, "STOP_ACTIVITY") {
			if envState == "CONFIGURED" {
				return updateEnvFunc(env.Id().String(), time.Time{}.UnixMilli(), envState, "success: the environment is in CONFIGURED state after STOP_ACTIVITY transition")
			} else {
				return updateEnvFunc(env.Id().String(), time.Time{}.UnixMilli(), envState, "error: the environment is in "+envState+" state after STOP_ACTIVITY transition")
			}
		}
		if strings.Contains(trigger, "EXIT") {
			if envState == "DONE" {
				return updateEnvFunc(env.Id().String(), time.Now().UnixMilli(), envState, "success: the environment is in DONE state after EXIT transition")
			} else {
				return updateEnvFunc(env.Id().String(), time.Now().UnixMilli(), envState, "error: the environment is in "+envState+" state after EXIT transition")
			}
		}
		if strings.Contains(trigger, "GO_ERROR") {
			if envState == "ERROR" {
				return updateEnvFunc(env.Id().String(), time.Time{}.UnixMilli(), envState, "success: the environment is in ERROR state after GO_ERROR transition")
			} else {
				return updateEnvFunc(env.Id().String(), time.Time{}.UnixMilli(), envState, "error: the environment is in "+envState+" state after GO_ERROR transition")
			}
		}
		if strings.Contains(trigger, "RECOVER") {
			if envState == "DEPLOYED" {
				return updateEnvFunc(env.Id().String(), time.Time{}.UnixMilli(), envState, "success: the environment is in DEPLOYED state after RECOVER transition")
			} else {
				return updateEnvFunc(env.Id().String(), time.Time{}.UnixMilli(), envState, "error: the environment is in "+envState+" state after RECOVER transition")
			}
		}
		log.WithField("partition", envId).
			WithField("call", call).
			Error("could not obtain transition in UpdateEnv from trigger: ", trigger)

		call.VarStack["__call_error_reason"] = err.Error()
		call.VarStack["__call_error"] = callFailedStr
		return
	}

	/*************************************/
	/******** Fill info functions ********/
	/*************************************/

	fetchLHCInfoForRun := func(runNumber int32) (out *bkpb.LHCFill, err error) {
		runFetchRequest := bkpb.RunFetchRequest{RunNumber: runNumber}

		timeout := callable.AcquireTimeout(BKP_FILL_TIMEOUT, varStack, "RetrieveFillInfo", envId)
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		log.WithField("partition", envId).
			WithField("level", infologger.IL_Devel).
			WithField("endpoint", viper.GetString("bookkeepingBaseUri")).
			WithField("call", "RetrieveFillInfo").
			Debugf("requesting BK the run info for run %d", runNumber)
		runWithRelations, err := p.bookkeepingClient.RunServiceClient.Get(ctx, &runFetchRequest, grpc.EmptyCallOption{})
		if err != nil {
			return nil, err
		}
		lhcFill := runWithRelations.GetLhcFill()
		if lhcFill == nil {
			return nil, fmt.Errorf("lhcFill for run %d is nil", runNumber)
		}
		return lhcFill, nil
	}

	fetchLatestLHCInfo := func() (out *bkpb.LHCFill, err error) {
		lhcFillFetchRequest := bkpb.LastLhcFillFetchRequest{}

		timeout := callable.AcquireTimeout(BKP_FILL_TIMEOUT, varStack, "RetrieveFillInfo", envId)
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		log.WithField("partition", envId).
			WithField("level", infologger.IL_Devel).
			WithField("endpoint", viper.GetString("bookkeepingBaseUri")).
			WithField("call", "RetrieveFillInfo").
			Debug("requesting BK the latest LHC fill info")
		LhcFillWithRelations, err := p.bookkeepingClient.LhcFillServiceClient.GetLast(ctx, &lhcFillFetchRequest, grpc.EmptyCallOption{})
		if err != nil {
			return nil, err
		}
		lhcFill := LhcFillWithRelations.GetLhcFill()
		if lhcFill == nil {
			return nil, fmt.Errorf("lhcFill is nil")
		}
		return lhcFill, nil
	}

	propagateLHCInfoToVarStack := func(lhcInfo *bkpb.LHCFill, varStack map[string]string) {
		parentRole, ok := call.GetParentRole().(callable.ParentRole)
		if !ok {
			log.WithField("partition", envId).
				WithField("level", infologger.IL_Devel).
				WithField("endpoint", viper.GetString("bookkeepingBaseUri")).
				WithField("call", "RetrieveFillInfo").
				Errorf("could not access the parent role when trying to propagate the LHC fill info")
			return
		}
		parentRole.SetGlobalRuntimeVar("fill_info_fill_number", string(lhcInfo.FillNumber))
		parentRole.SetGlobalRuntimeVar("fill_info_filling_scheme", lhcInfo.FillingSchemeName)
		parentRole.SetGlobalRuntimeVar("fill_info_beam_type", lhcInfo.BeamType)
		if lhcInfo.StableBeamStart != nil {
			parentRole.SetGlobalRuntimeVar("fill_info_stable_beam_start_ms", strconv.FormatInt(*lhcInfo.StableBeamStart, 10))
		}
		if lhcInfo.StableBeamEnd != nil {
			parentRole.SetGlobalRuntimeVar("fill_info_stable_beam_end_ms", strconv.FormatInt(*lhcInfo.StableBeamEnd, 10))
		}
		log.WithField("partition", envId).
			WithField("level", infologger.IL_Devel).
			WithField("endpoint", viper.GetString("bookkeepingBaseUri")).
			WithField("call", "RetrieveFillInfo").
			Infof("successfully updated the LHC Fill info (number %d, scheme %s, beam type %s)",
				lhcInfo.FillNumber, lhcInfo.FillingSchemeName, lhcInfo.BeamType)
	}
	deleteLHCInfoInVarStack := func(varStack map[string]string) {
		parentRole, ok := call.GetParentRole().(callable.ParentRole)
		if !ok {
			log.WithField("partition", envId).
				WithField("level", infologger.IL_Devel).
				WithField("endpoint", viper.GetString("bookkeepingBaseUri")).
				WithField("call", "RetrieveFillInfo").
				Errorf("could not access the parent role when trying to clean up LHC fill info")
			return
		}
		parentRole.DeleteGlobalRuntimeVar("fill_info_fill_number")
		parentRole.DeleteGlobalRuntimeVar("fill_info_filling_scheme")
		parentRole.DeleteGlobalRuntimeVar("fill_info_beam_type")
		parentRole.DeleteGlobalRuntimeVar("fill_info_stable_beam_start_ms")
		parentRole.DeleteGlobalRuntimeVar("fill_info_stable_beam_end_ms")
	}

	stack["RetrieveFillInfo"] = func() (out string) {
		callFailedStr := "Bookkeeping RetrieveFillInfo call failed"
		if p.bookkeepingClient == nil {
			log.WithError(err).
				WithField("level", infologger.IL_Support).
				WithField("endpoint", viper.GetString("bookkeepingBaseUri")).
				WithField("partition", envId).
				WithField("call", "RetrieveFillInfo").
				Error("bookkeeping plugin RetrieveFillInfo error")
			call.VarStack["__call_error_reason"] = "bookkeeping plugin not initialized, RetrieveFillInfo impossible"
			call.VarStack["__call_error"] = callFailedStr
			return
		}

		// First, we try to get fill info associated to a run.
		// At the time of writing, it is not correctly associated for SYNTHETIC runs,
		// but it might be in the future.
		// Only if there is no fill info associated with a run (e.g. because we ask BK too early during SOR),
		// we ask for the latest LHC fill and will use if the end time of stable beams is not set.
		rn := varStack["run_number"]
		runNumber64, err := strconv.ParseInt(rn, 10, 32)
		if err != nil {
			log.WithField("partition", envId).
				WithField("level", infologger.IL_Devel).
				WithField("endpoint", viper.GetString("bookkeepingBaseUri")).
				WithField("call", "RetrieveFillInfo").
				WithError(err).
				Info("cannot acquire run number for Bookkeeping fill info fetch, perhaps we are not in RUNNING")

			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr
		} else {
			lhcFill, err := fetchLHCInfoForRun(int32(runNumber64))
			if err != nil {
				log.WithField("partition", envId).
					WithField("level", infologger.IL_Devel).
					WithField("endpoint", viper.GetString("bookkeepingBaseUri")).
					WithField("call", "RetrieveFillInfo").
					Infof("could not get LHC fill info associated to run %d, will try to get the latest fill. Details: %s", runNumber64, err.Error())
			} else {
				log.WithField("partition", envId).
					WithField("level", infologger.IL_Devel).
					WithField("endpoint", viper.GetString("bookkeepingBaseUri")).
					WithField("call", "RetrieveFillInfo").
					Infof("received a reply about fill info associated to run %d, filling", runNumber64)
				propagateLHCInfoToVarStack(lhcFill, varStack)
				log.WithField("partition", envId).
					WithField("level", infologger.IL_Devel).
					WithField("endpoint", viper.GetString("bookkeepingBaseUri")).
					WithField("call", "RetrieveFillInfo").
					Infof("successfully updated the LHC Fill info for run %d (number %d, scheme %s, beam type %s)",
						runNumber64, lhcFill.FillNumber, lhcFill.FillingSchemeName, lhcFill.BeamType)
				return
			}
		}

		lhcFill, err := fetchLatestLHCInfo()
		if err != nil {
			log.WithError(err).
				WithField("level", infologger.IL_Support).
				WithField("endpoint", viper.GetString("bookkeepingBaseUri")).
				WithField("partition", envId).
				WithField("call", "RetrieveFillInfo").
				Error("bookkeeping plugin RetrieveFillInfo error")
			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr
		} else if (lhcFill.StableBeamStart != nil && *lhcFill.StableBeamStart != 0) && (lhcFill.StableBeamEnd == nil || *lhcFill.StableBeamEnd == 0) {
			// we enter here only if stable beams started and are not over (stable beams start exists && stable beams end does not exist)
			log.WithField("partition", envId).
				WithField("level", infologger.IL_Devel).
				WithField("endpoint", viper.GetString("bookkeepingBaseUri")).
				WithField("call", "RetrieveFillInfo").
				Debug("received a reply about fill info, filling the var stack")
			propagateLHCInfoToVarStack(lhcFill, varStack)
		} else {
			log.WithField("partition", envId).
				WithField("level", infologger.IL_Devel).
				WithField("endpoint", viper.GetString("bookkeepingBaseUri")).
				WithField("call", "RetrieveFillInfo").
				Debug("received a reply about fill info, but the latest fill is over or stable beams are not started yet, will not read the fill info and will delete any existing")
			deleteLHCInfoInVarStack(varStack)
		}
		return
	}

	return
}

func (p *Plugin) Destroy() error {
	return nil
}
