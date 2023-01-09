/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2021-2022 CERN and copyright holders of ALICE O².
 * Author: Miltiadis Alexis <miltiadis.alexis@cern.ch>
 *         Teo Mrnjavac <teo.mrnjavac@cern.ch>
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

//go:generate protoc --go_out=. --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative --go-grpc_out=require_unimplemented_servers=false:. protos/ctpecs.proto

package trg

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/AliceO2Group/Control/common/logger/infologger"
	"github.com/AliceO2Group/Control/common/utils/uid"
	"github.com/AliceO2Group/Control/core/environment"
	"github.com/AliceO2Group/Control/core/integration"
	trgpb "github.com/AliceO2Group/Control/core/integration/trg/protos"
	"github.com/AliceO2Group/Control/core/workflow/callable"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
)

const (
	TRG_DIAL_TIMEOUT           = 2 * time.Second
	TRG_POLLING_INTERVAL       = 3 * time.Second
	TRG_RECONCILIATION_TIMEOUT = 5 * time.Second
)

type Plugin struct {
	trgHost string
	trgPort int

	trgClient *RpcClient

	pendingRunStops   map[string] /*envId*/ int64
	pendingRunUnloads map[string] /*envId*/ int64

	cachedStatus           *TrgStatus
	cachedStatusMu         sync.RWMutex
	cachedStatusCancelFunc context.CancelFunc
}

type TrgStatus struct {
	RunCount   int            `json:"runCount,omitempty"`
	Lines      []string       `json:"lines,omitempty"`
	Structured Runs           `json:"structured,omitempty"`
	EnvMap     map[uid.ID]Run `json:"envMap,omitempty"`
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
		trgHost:           u.Hostname(),
		trgPort:           portNumber,
		trgClient:         nil,
		pendingRunStops:   make(map[string]int64),
		pendingRunUnloads: make(map[string]int64),
	}
}

func (p *Plugin) GetName() string {
	return "trg"
}

func (p *Plugin) GetPrettyName() string {
	return "Trigger System"
}

func (p *Plugin) GetEndpoint() string {
	return viper.GetString("trgServiceEndpoint")
}

func (p *Plugin) GetConnectionState() string {
	if p == nil || p.trgClient == nil {
		return "UNKNOWN"
	}
	return p.trgClient.conn.GetState().String()
}

func (p *Plugin) queryRunList() {
	runReply, err := p.trgClient.RunList(context.Background(), &trgpb.Empty{}, grpc.EmptyCallOption{})
	if err != nil {
		err = fmt.Errorf("error querying TRG service at %s: %w", viper.GetString("trgServiceEndpoint"), err)
		log.WithError(err).
			WithField("level", infologger.IL_Devel).
			WithField("endpoint", viper.GetString("trgServiceEndpoint")).
			WithField("call", "RunList").
			Error("TRG error")
	}
	if runReply == nil {
		log.WithField("level", infologger.IL_Support).
			WithField("endpoint", viper.GetString("trgServiceEndpoint")).
			WithField("call", "RunList").
			WithError(fmt.Errorf("TRG RunList response is nil")).
			Error("TRG error")
		runReply = &trgpb.RunReply{}
	}

	structured, errSlice := parseRunList(int(runReply.Rc), runReply.Msg)
	for _, err := range errSlice {
		if err != nil {
			err = fmt.Errorf("error parsing response from TRG service at %s: %w", viper.GetString("trgServiceEndpoint"), err)
			log.WithError(err).
				WithField("level", infologger.IL_Devel).
				WithField("endpoint", viper.GetString("trgServiceEndpoint")).
				WithField("call", "RunList").
				Error("TRG error")
		}
	}
	if structured == nil {
		structured = make(Runs, 0)
	}

	envMap := make(map[uid.ID]Run)
	for envIdS, rn := range p.pendingRunStops {
		var envId uid.ID
		envId, err = uid.FromString(envIdS)
		if err != nil {
			continue
		}

		for _, run := range structured {
			if run.RunNumber == uint32(rn) {
				envMap[envId] = run
			}
		}
	}
	for envIdS, rn := range p.pendingRunUnloads {
		var envId uid.ID
		envId, err = uid.FromString(envIdS)
		if err != nil {
			continue
		}

		for _, run := range structured {
			if run.RunNumber == uint32(rn) {
				envMap[envId] = run
			}
		}
	}

	out := &TrgStatus{
		RunCount:   int(runReply.Rc),
		Lines:      strings.Split(runReply.Msg, "\n"),
		Structured: structured,
		EnvMap:     envMap,
	}

	p.cachedStatusMu.Lock()
	p.cachedStatus = out
	p.cachedStatusMu.Unlock()
}

func (p *Plugin) GetData(_ []any) string {
	if p == nil || p.trgClient == nil {
		return ""
	}

	p.cachedStatusMu.RLock()
	r := p.cachedStatus
	if r == nil {
		p.cachedStatusMu.RUnlock()
		return ""
	}

	out, err := json.Marshal(r)
	p.cachedStatusMu.RUnlock()

	if err != nil {
		return ""
	}
	return string(out[:])
}

func (p *Plugin) GetEnvironmentsData(envIds []uid.ID) map[uid.ID]string {
	if p == nil || p.trgClient == nil {
		return nil
	}

	p.cachedStatusMu.RLock()
	defer p.cachedStatusMu.RUnlock()

	out := make(map[uid.ID]string)

	if p.cachedStatus == nil {
		return nil
	}
	envMap := p.cachedStatus.EnvMap
	for _, envId := range envIds {
		if run, ok := envMap[envId]; !ok {
			runOut, err := json.Marshal(run)
			if err != nil {
				continue
			}
			out[envId] = string(runOut[:])
		}
	}

	return out
}

func (p *Plugin) Init(instanceId string) error {
	if p.trgClient == nil {
		callTimeout := TRG_DIAL_TIMEOUT
		cxt, cancel := context.WithTimeout(context.Background(), callTimeout)
		p.trgClient = NewClient(cxt, cancel, viper.GetString("trgServiceEndpoint"))
		if p.trgClient == nil {
			return fmt.Errorf("failed to connect to TRG service on %s", viper.GetString("trgServiceEndpoint"))
		}
	}
	if p.trgClient == nil {
		return fmt.Errorf("failed to start TRG client on %s", viper.GetString("trgServiceEndpoint"))
	}

	var ctx context.Context
	ctx, p.cachedStatusCancelFunc = context.WithCancel(context.Background())

	trgPollingIntervalStr := viper.GetString("trgPollingInterval")
	trgPollingInterval, err := time.ParseDuration(trgPollingIntervalStr)
	if err != nil {
		trgPollingInterval = TRG_POLLING_INTERVAL
		log.Debugf("TRG plugin cannot acquire polling interval, defaulting to %s", TRG_POLLING_INTERVAL.String())
	}

	// polling
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(trgPollingInterval):
				p.queryRunList()
				p.reconcile()
			}
		}
	}()
	return nil
}

func (p *Plugin) reconcile() {
	envMan := environment.ManagerInstance()
	envIds := envMan.Ids()
	activeEcsRuns := make(map[uint32]struct{})
	for _, envId := range envIds {
		env, err := envMan.Environment(envId)
		if err != nil || env == nil {
			continue
		}
		rn := env.GetCurrentRunNumber()
		if rn != 0 {
			activeEcsRuns[rn] = struct{}{}
		}
	}

	p.cachedStatusMu.RLock()
	if p.cachedStatus != nil && len(p.cachedStatus.Structured) > 0 {
		for _, trgRun := range p.cachedStatus.Structured {
			if _, contains := activeEcsRuns[trgRun.RunNumber]; contains { // if activeEcsRuns contains a RN known to TRG
				continue
			} else { // found TRG run unknown to AliECS
				if p.trgClient == nil {
					continue
				}
				in := trgpb.RunStopRequest{ // applies to both RunStop and RunUnload
					Runn:     trgRun.RunNumber,
					Detector: "",
				}

				if trgRun.State == CTP_RUNNING { // both STANDALONE and GLOBAL

					// Setting the detector here results in a failed RunStop call
					// Returns with RC:12 "Unknown ltud [DETNAME]"
					/*if trgRun.Cardinality == CTP_STANDALONE { // if global run, we send no detector
						if len(trgRun.Detectors) == 1 {
							in.Detector = trgRun.Detectors[0].String()
						}
					}*/

					ctx, _ := context.WithTimeout(context.Background(), TRG_RECONCILIATION_TIMEOUT)
					_, err := p.trgClient.RunStop(ctx, &in, grpc.EmptyCallOption{})
					// TODO: Response's RC should also be checked here
					if err != nil {
						err = fmt.Errorf("TRG reconciliation failure: %w", err)
						log.WithError(err).
							WithField("level", infologger.IL_Support).
							WithField("endpoint", viper.GetString("trgServiceEndpoint")).
							WithField("runNumber", trgRun.RunNumber).
							WithField("call", "RunStop").
							Error("TRG error")
					} else {
						if trgRun.Cardinality == CTP_GLOBAL {
							trgRun.State = CTP_LOADED // must be loaded if GLOBAL && RunStop successful
						} else {
							log.WithField("level", infologger.IL_Support).
								WithField("endpoint", viper.GetString("trgServiceEndpoint")).
								WithField("runNumber", trgRun.RunNumber).
								Info("TRG STANDALONE reconciliation complete")
						}
					}
				}
				if trgRun.State == CTP_LOADED && trgRun.Cardinality == CTP_GLOBAL {
					ctx, _ := context.WithTimeout(context.Background(), TRG_RECONCILIATION_TIMEOUT)
					_, err := p.trgClient.RunUnload(ctx, &in, grpc.EmptyCallOption{})
					if err != nil {
						err = fmt.Errorf("TRG reconciliation failure: %w", err)
						log.WithError(err).
							WithField("level", infologger.IL_Support).
							WithField("endpoint", viper.GetString("trgServiceEndpoint")).
							WithField("runNumber", trgRun.RunNumber).
							WithField("call", "RunUnload").
							Error("TRG error")
					} else {
						log.WithField("level", infologger.IL_Support).
							WithField("endpoint", viper.GetString("trgServiceEndpoint")).
							WithField("runNumber", trgRun.RunNumber).
							Info("TRG GLOBAL reconciliation complete")
					}
				}
			}
		}
	}
	p.cachedStatusMu.RUnlock()

}

func (p *Plugin) ObjectStack(_ map[string]string, _ map[string]string) (stack map[string]interface{}) {
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
		log.Error("cannot acquire environment ID")
		return
	}

	stack = make(map[string]interface{})
	// global runs only
	stack["RunLoad"] = func() (out string) { // must formally return string even when we return nothing
		log.WithField("partition", envId).
			WithField("level", infologger.IL_Ops).
			Info("ALIECS SOR operation : performing TRG Run Load Request")

		parentRole, ok := call.GetParentRole().(callable.ParentRole)
		if ok {
			parentRole.DeleteGlobalRuntimeVar("trg_start_time_ms")
			parentRole.DeleteGlobalRuntimeVar("trg_end_time_ms")
		}

		globalConfig, ok := varStack["trg_global_config"]
		log.WithField("globalConfig", globalConfig).
			WithField("partition", envId).
			Debug("not a TRG Global Run, continuing with TRG Run Start")
		if !ok {
			log.WithField("partition", envId).
				Debug("no TRG Global config set")
			globalConfig = ""
		}
		// TODO (malexis): pass consul key to TRG if avail

		rn := varStack["run_number"]
		var runNumber64 int64
		runNumber64, err := strconv.ParseInt(rn, 10, 32)
		if err != nil {
			log.WithField("partition", envId).
				WithError(err).
				Error("cannot acquire run number for Run Load")
		}

		trgDetectorsParam, ok := varStack["trg_detectors"]
		if !ok {
			// "" -all required must be ready
			log.WithField("partition", envId).
				WithField("runNumber", runNumber64).
				Debug("empty TRG detectors list provided")
			trgDetectorsParam = ""
		}

		callFailedStr := "TRG RunLoad call failed"

		detectors, err := p.parseDetectors(trgDetectorsParam)
		if err != nil {
			log.WithError(err).
				WithField("level", infologger.IL_Support).
				WithField("partition", envId).
				WithField("call", "RunLoad").
				Error("TRG error")
			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr

			return
		}

		// standalone run
		if len(strings.Split(detectors, " ")) < 2 && varStack["trg_global_run_enabled"] == "false" {
			// we do not load any run cause it is standalone
			log.WithField("partition", envId).
				WithField("runNumber", runNumber64).
				Debug("not a TRG Global Run, continuing with TRG Run Start")

			return
		}

		in := trgpb.RunLoadRequest{
			Runn:      uint32(runNumber64),
			Detectors: detectors,
			Config:    globalConfig,
		}
		if p.trgClient == nil {
			err = fmt.Errorf("TRG plugin not initialized, RunLoad impossible")

			log.WithError(err).
				WithField("level", infologger.IL_Support).
				WithField("endpoint", viper.GetString("trgServiceEndpoint")).
				WithField("runNumber", runNumber64).
				WithField("partition", envId).
				WithField("call", "RunLoad").
				Error("TRG error")

			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr

			return
		}

		if p.trgClient.GetConnState() == connectivity.Shutdown {
			err = fmt.Errorf("TRG client connection not available, RunLoad impossible")

			log.WithError(err).
				WithField("level", infologger.IL_Support).
				WithField("endpoint", viper.GetString("trgServiceEndpoint")).
				WithField("runNumber", runNumber64).
				WithField("partition", envId).
				WithField("call", "RunLoad").
				Error("TRG error")

			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr

			return
		}

		var response *trgpb.RunReply
		response, err = p.trgClient.RunLoad(context.Background(), &in, grpc.EmptyCallOption{})
		if err != nil {
			log.WithError(err).
				WithField("level", infologger.IL_Support).
				WithField("endpoint", viper.GetString("trgServiceEndpoint")).
				WithField("runNumber", runNumber64).
				WithField("partition", envId).
				WithField("call", "RunLoad").
				Error("TRG error")

			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr

			return
		}

		if response != nil {
			if response.Rc != 0 {
				err = fmt.Errorf("response code %d from TRG: %s", response.Rc, response.Msg)

				log.WithError(err).
					WithField("level", infologger.IL_Support).
					WithField("endpoint", viper.GetString("trgServiceEndpoint")).
					WithField("runNumber", runNumber64).
					WithField("partition", envId).
					WithField("call", "RunLoad").
					Error("TRG error")

				call.VarStack["__call_error_reason"] = err.Error()
				call.VarStack["__call_error"] = callFailedStr

				return
			}
		}

		// runLoad successful, we cache the run number for eventual cleanup
		p.pendingRunUnloads[envId] = runNumber64
		log.WithField("partition", envId).
			WithField("runNumber", runNumber64).
			Info("ALIECS SOR Operation : TRG RunLoad success")

		return
	}
	stack["RunStart"] = func() (out string) { // must formally return string even when we return nothing
		log.WithField("partition", envId).
			Info("ALIECS SOR operation : performing TRG Run Start")

		runtimeConfig, ok := varStack["trg_runtime_config"]
		if !ok {
			log.WithField("partition", envId).
				Debug("no TRG config set, using default configuration")
			runtimeConfig = ""
		}

		rn := varStack["run_number"]
		var runNumber64 int64
		runNumber64, err := strconv.ParseInt(rn, 10, 32)
		if err != nil {
			log.WithError(err).
				WithField("partition", envId).
				Error("cannot acquire run number for Run Start")
		}

		trgDetectorsParam, ok := varStack["trg_detectors"]
		if !ok {
			// "" it is a global run
			log.WithField("partition", envId).
				WithField("runNumber", runNumber64).
				Debug("Detector for host is not available, starting global run")
			trgDetectorsParam = ""
		}

		callFailedStr := "TRG RunStart call failed"

		detectors, err := p.parseDetectors(trgDetectorsParam)
		if err != nil {
			log.WithError(err).
				WithField("level", infologger.IL_Support).
				WithField("endpoint", viper.GetString("trgServiceEndpoint")).
				WithField("runNumber", runNumber64).
				WithField("partition", envId).
				WithField("call", "RunStart").
				Error("TRG error")

			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr

			return
		}

		// if global run then start with empty string in detectors
		if len(strings.Split(detectors, " ")) >= 2 || varStack["trg_global_run_enabled"] == "true" {
			// global run detectors ""
			detectors = ""
		}

		in := trgpb.RunStartRequest{
			Runn:     uint32(runNumber64),
			Detector: detectors,
			Config:   runtimeConfig,
		}

		if p.trgClient == nil {
			err = fmt.Errorf("TRG plugin not initialized, RunStart impossible")

			log.WithError(err).
				WithField("level", infologger.IL_Support).
				WithField("endpoint", viper.GetString("trgServiceEndpoint")).
				WithField("runNumber", runNumber64).
				WithField("partition", envId).
				WithField("call", "RunStart").
				Error("TRG error")

			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr

			return
		}
		if p.trgClient.GetConnState() == connectivity.Shutdown {
			err = fmt.Errorf("TRG client connection not available, RunStart impossible")

			log.WithError(err).
				WithField("level", infologger.IL_Support).
				WithField("endpoint", viper.GetString("trgServiceEndpoint")).
				WithField("runNumber", runNumber64).
				WithField("partition", envId).
				WithField("call", "RunStart").
				Error("TRG error")

			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr

			return
		}

		var response *trgpb.RunReply

		response, err = p.trgClient.RunStart(context.Background(), &in, grpc.EmptyCallOption{})
		if err != nil {
			log.WithError(err).
				WithField("level", infologger.IL_Support).
				WithField("endpoint", viper.GetString("trgServiceEndpoint")).
				WithField("runNumber", runNumber64).
				WithField("partition", envId).
				WithField("call", "RunStart").
				Error("TRG error")

			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr

			return
		}
		if response != nil {
			if response.Rc != 0 {
				err = fmt.Errorf("response code %d from TRG: %s", response.Rc, response.Msg)

				log.WithError(err).
					WithField("level", infologger.IL_Support).
					WithField("endpoint", viper.GetString("trgServiceEndpoint")).
					WithField("runNumber", runNumber64).
					WithField("partition", envId).
					WithField("call", "RunStart").
					Error("TRG error")

				call.VarStack["__call_error_reason"] = err.Error()
				call.VarStack["__call_error"] = callFailedStr

				return
			}
		}

		// runStart successful, we cache the run number for eventual cleanup
		p.pendingRunStops[envId] = runNumber64
		log.WithField("partition", envId).
			WithField("runNumber", runNumber64).
			Debug("TRG RunStart success")

		trgStartTime := strconv.FormatInt(time.Now().UnixMilli(), 10)
		parentRole, ok := call.GetParentRole().(callable.ParentRole)
		if ok {
			parentRole.SetGlobalRuntimeVar("trg_start_time_ms", trgStartTime)
			parentRole.DeleteGlobalRuntimeVar("trg_end_time_ms")
		}

		return
	}
	runStopFunc := func(runNumber64 int64) (out string) {
		trgDetectorsParam, ok := varStack["trg_detectors"]
		if !ok {
			// "" it is a global run
			log.WithField("partition", envId).
				WithField("runNumber", runNumber64).
				Debug("Detector for host is not available, stoping global run")
			trgDetectorsParam = ""
		}

		detectors, err := p.parseDetectors(trgDetectorsParam)
		if err != nil {
			return
		}

		// if global run then start with empty
		if len(strings.Split(detectors, " ")) >= 2 || varStack["trg_global_run_enabled"] == "true" {
			// global run detectors ""
			detectors = ""
		}

		in := trgpb.RunStopRequest{
			Runn:     uint32(runNumber64),
			Detector: detectors,
		}

		callFailedStr := "TRG RunStop call failed"

		if p.trgClient == nil {
			err = fmt.Errorf("TRG plugin not initialized, RunStop impossible")

			log.WithError(err).
				WithField("level", infologger.IL_Support).
				WithField("endpoint", viper.GetString("trgServiceEndpoint")).
				WithField("runNumber", runNumber64).
				WithField("partition", envId).
				WithField("call", "RunStop").
				Error("TRG error")

			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr

			return
		}
		if p.trgClient.GetConnState() == connectivity.Shutdown {
			err = fmt.Errorf("TRG client connection not available, RunStop impossible")

			log.WithError(err).
				WithField("level", infologger.IL_Support).
				WithField("endpoint", viper.GetString("trgServiceEndpoint")).
				WithField("runNumber", runNumber64).
				WithField("partition", envId).
				WithField("call", "RunStop").
				Error("TRG error")

			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr

			return
		}

		var response *trgpb.RunReply
		response, err = p.trgClient.RunStop(context.Background(), &in, grpc.EmptyCallOption{})
		if err != nil {
			log.WithError(err).
				WithField("level", infologger.IL_Support).
				WithField("endpoint", viper.GetString("trgServiceEndpoint")).
				WithField("runNumber", runNumber64).
				WithField("partition", envId).
				WithField("call", "RunStop").
				Error("TRG error")

			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr

			return
		}
		if response != nil {
			if response.Rc != 0 {
				err = fmt.Errorf("response code %d from TRG: %s", response.Rc, response.Msg)

				log.WithError(err).
					WithField("level", infologger.IL_Support).
					WithField("endpoint", viper.GetString("trgServiceEndpoint")).
					WithField("runNumber", runNumber64).
					WithField("partition", envId).
					WithField("call", "RunStop").
					Error("TRG error")

				call.VarStack["__call_error_reason"] = err.Error()
				call.VarStack["__call_error"] = callFailedStr

				return
			}
		}

		// RunStop successful, we pop the run number from the cache
		delete(p.pendingRunStops, envId)
		log.WithField("partition", envId).
			WithField("runNumber", runNumber64).
			Debug("TRG RunStop success")

		trgEndTime := strconv.FormatInt(time.Now().UnixMilli(), 10)
		parentRole, ok := call.GetParentRole().(callable.ParentRole)
		if ok {
			parentRole.SetGlobalRuntimeVar("trg_end_time_ms", trgEndTime)
		} else {
			log.WithField("partition", envId).
				WithField("runNumber", runNumber64).
				WithField("trgEndTime", trgEndTime).
				Debug("could not get parentRole and set TRG end time")
		}

		return
	}
	runUnloadFunc := func(runNumber64 int64) (out string) {

		trgDetectorsParam, ok := varStack["trg_detectors"]
		if !ok {
			trgDetectorsParam = ""
		}

		detectors, err := p.parseDetectors(trgDetectorsParam)
		if err != nil {
			return
		}

		// if global run then unload
		if len(strings.Split(detectors, " ")) < 2 && varStack["trg_global_run_enabled"] == "false" {
			log.WithField("partition", envId).
				WithField("runNumber", runNumber64).
				Debug("not a TRG Global Run, skipping TRG Run Unload")
			return
		}

		in := trgpb.RunStopRequest{
			Runn: uint32(runNumber64),
			// "" when unloading global run
			Detector: "",
		}

		callFailedStr := "TRG RunUnload call failed"

		if p.trgClient == nil {
			err = fmt.Errorf("TRG plugin not initialized, RunUnload impossible")

			log.WithError(err).
				WithField("level", infologger.IL_Support).
				WithField("endpoint", viper.GetString("trgServiceEndpoint")).
				WithField("runNumber", runNumber64).
				WithField("partition", envId).
				WithField("call", "RunUnload").
				Error("TRG error")

			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr

			return
		}
		if p.trgClient.GetConnState() == connectivity.Shutdown {
			err = fmt.Errorf("TRG client connection not available, RunUnload impossible")

			log.WithError(err).
				WithField("level", infologger.IL_Support).
				WithField("endpoint", viper.GetString("trgServiceEndpoint")).
				WithField("runNumber", runNumber64).
				WithField("partition", envId).
				WithField("call", "RunUnload").
				Error("TRG error")

			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr

			return
		}

		var response *trgpb.RunReply
		response, err = p.trgClient.RunUnload(context.Background(), &in, grpc.EmptyCallOption{})
		if err != nil {
			log.WithError(err).
				WithField("level", infologger.IL_Support).
				WithField("endpoint", viper.GetString("trgServiceEndpoint")).
				WithField("runNumber", runNumber64).
				WithField("partition", envId).
				WithField("call", "RunUnload").
				Error("TRG error")

			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr

			return
		}
		if response != nil {
			if response.Rc != 0 {
				err = fmt.Errorf("response code %d from TRG: %s", response.Rc, response.Msg)

				log.WithError(err).
					WithField("level", infologger.IL_Support).
					WithField("endpoint", viper.GetString("trgServiceEndpoint")).
					WithField("runNumber", runNumber64).
					WithField("partition", envId).
					WithField("call", "RunUnload").
					Error("TRG error")

				call.VarStack["__call_error_reason"] = err.Error()
				call.VarStack["__call_error"] = callFailedStr

				return
			}
		}

		// RunUnload successful, we pop the run number from the cache
		delete(p.pendingRunUnloads, envId)
		log.WithField("partition", envId).
			WithField("runNumber", runNumber64).
			Info("ALICECS EOR operation : TRG RunUnload success")

		return
	}
	stack["RunStop"] = func() (out string) {
		log.WithField("partition", envId).
			//WithField("runNumber", runNumber64).
			Info("ALIECS EOR operation : performing TRG Run Stop ")

		rn := varStack["run_number"]
		var runNumber64 int64
		var err error
		runNumber64, err = strconv.ParseInt(rn, 10, 32)
		if err != nil {
			log.WithError(err).
				WithField("partition", envId).
				Error("cannot acquire run number for TRG Run Stop")
		}

		return runStopFunc(runNumber64)
	}
	stack["RunUnload"] = func() (out string) {
		log.WithField("partition", envId).
			//WithField("runNumber", runNumber64).
			Info("ALIECS EOR operation : performing TRG Run Unload ")

		rn := varStack["run_number"]
		var runNumber64 int64
		var err error
		runNumber64, err = strconv.ParseInt(rn, 10, 32)
		if err != nil {
			log.WithError(err).
				WithField("partition", envId).
				Error("cannot acquire run number for TRG Run Unload")
		}

		return runUnloadFunc(runNumber64)
	}
	stack["Cleanup"] = func() (out string) {
		envId, ok := varStack["environment_id"]
		if !ok {
			log.WithField("partition", envId).
				WithField("level", infologger.IL_Devel).
				Warn("no environment_id found for TRG cleanup")
			return
		}

		// runStop if found pending
		runNumberStop, ok := p.pendingRunStops[envId]
		if ok {
			log.WithField("runNumber", runNumberStop).
				WithField("partition", envId).
				WithField("level", infologger.IL_Devel).
				Debug("pending TRG Stop found, performing cleanup")

			delete(p.pendingRunStops, envId)
			_ = runStopFunc(runNumberStop)

			trgEndTime := strconv.FormatInt(time.Now().UnixMilli(), 10)
			parentRole, ok := call.GetParentRole().(callable.ParentRole)
			if ok {
				parentRole.SetGlobalRuntimeVar("trg_end_time_ms", trgEndTime)
			} else {
				log.WithField("partition", envId).
					WithField("runNumber", runNumberStop).
					WithField("trgEndTime", trgEndTime).
					Debug("could not get parentRole and set TRG end time")
			}
		} else {
			log.WithField("partition", envId).
				WithField("level", infologger.IL_Devel).
				Debug("TRG cleanup: Stop not needed")
		}

		// runUnload if found pending
		runNumberUnload, ok := p.pendingRunUnloads[envId]
		if ok {
			log.WithField("runNumber", runNumberUnload).
				WithField("partition", envId).
				WithField("level", infologger.IL_Devel).
				Debug("pending TRG Unload found, performing cleanup")

			delete(p.pendingRunUnloads, envId)
			_ = runUnloadFunc(runNumberUnload)
		} else {
			log.WithField("partition", envId).
				WithField("level", infologger.IL_Devel).
				Debug("TRG cleanup: Unload not needed")
		}
		return
	}

	return
}

func (p *Plugin) parseDetectors(ctsDetectorsParam string) (detectors string, err error) {
	detectorsSlice := make([]string, 0)
	bytes := []byte(ctsDetectorsParam)
	err = json.Unmarshal(bytes, &detectorsSlice)
	if err != nil {
		log.WithError(err).Error("error processing TRG detectors list")
		return
	}

	detectors = strings.ToLower(strings.Join(detectorsSlice, " "))
	return
}

func (p *Plugin) Destroy() error {
	p.cachedStatusCancelFunc()
	return p.trgClient.Close()
}
