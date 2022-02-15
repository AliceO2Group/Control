/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2021 CERN and copyright holders of ALICE O².
 * Author: Miltiadis Alexis <miltiadis.alexis@cern.ch>
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

//go:generate protoc --go_out=. --go-grpc_out=require_unimplemented_servers=false:. protos/ctpecs.proto

package trg

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/AliceO2Group/Control/common/logger/infologger"
	"github.com/AliceO2Group/Control/common/utils/uid"
	"github.com/AliceO2Group/Control/core/integration"
	trgecspb "github.com/AliceO2Group/Control/core/integration/trg/protos"
	"github.com/AliceO2Group/Control/core/workflow/callable"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
)

const TRG_DIAL_TIMEOUT = 2 * time.Second

type Plugin struct {
	trgHost        string
	trgPort        int

	trgClient      *RpcClient

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
		trgHost:   u.Hostname(),
		trgPort:   portNumber,
		trgClient: nil,
		pendingEORs: make(map[string]int64),
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

func (p *Plugin) GetData(environmentIds []uid.ID) string {
	if p == nil || p.trgClient == nil {
		return ""
	}
	return ""
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
		log.Error("cannot acquire environment ID")
		return
	}

	stack = make(map[string]interface{})
	// global runs only
	stack["RunLoad"] = func() (out string) { // must formally return string even when we return nothing
		log.WithField("partition", envId).
			WithField("level", infologger.IL_Ops).
			Debug("performing TRG Run load Request")

		globalConfig, ok := varStack["trg_global_config"]
		log.WithField("globalConfig",globalConfig).
			WithField("partition", envId).
			Debug("not a TRG Global Run, continuing with TRG Run Start")
		if !ok {
			log.Debug("no TRG Global config set")
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

		detectors, err := p.parseDetectors(trgDetectorsParam)
		if err != nil {
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

		in := trgecspb.RunLoadRequest{
			Runn:      uint32(runNumber64),
			Detectors: detectors,
			Config:    globalConfig,
		}
		if p.trgClient == nil {
			log.WithError(fmt.Errorf("TRG plugin not initialized")).
				WithField("endpoint", viper.GetString("trgServiceEndpoint")).
				WithField("partition", envId).
				WithField("runNumber", runNumber64).
				Error("failed to perform Run Load request")
			return
		}
		if p.trgClient.GetConnState() != connectivity.Ready {
			log.WithError(fmt.Errorf("TRG client connection not available")).
				WithField("endpoint", viper.GetString("trgServiceEndpoint")).
				WithField("partition", envId).
				WithField("runNumber", runNumber64).
				Error("failed to perform Run Load request")
			return
		}

		var response *trgecspb.RunReply
		response, err = p.trgClient.RunLoad(context.Background(), &in, grpc.EmptyCallOption{})
		if err != nil {
			log.WithError(err).
				WithField("endpoint", viper.GetString("trgServiceEndpoint")).
				WithField("partition", envId).
				WithField("runNumber", runNumber64).
				Error("failed to perform Run Load request")
			return
		}
		if response != nil {
			if response.Rc != 0 {
				log.WithField("response rc", response.Rc).
					WithField("Message", response.Msg).
					WithField("partition", envId).
					WithField("runNumber", runNumber64).
					Error("Run Load failed")
				return
			}
		}
		// runLoad successful, we cache the run number for eventual cleanup
		p.pendingEORs[envId] = runNumber64
		log.WithField("partition", envId).
			WithField("runNumber", runNumber64).
			Debug("TRG RunLoad success")

		return
	}
	stack["RunStart"] = func() (out string) { // must formally return string even when we return nothing
		log.Debug("performing TRG Run Start")

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

		detectors, err := p.parseDetectors(trgDetectorsParam)
		if err != nil {
			return
		}

		// if global run then start with empty string in detectors
		if len(strings.Split(detectors, " ")) >= 2 || varStack["trg_global_run_enabled"] == "true" {
			// global run detectors ""
			detectors = ""
		}

		in := trgecspb.RunStartRequest{
			Runn:     uint32(runNumber64),
			Detector: detectors,
			Config:   runtimeConfig,
		}

		if p.trgClient == nil {
			log.WithError(fmt.Errorf("TRG plugin not initialized")).
				WithField("endpoint", viper.GetString("trgServiceEndpoint")).
				WithField("partition", envId).
				WithField("runNumber", runNumber64).
				Error("failed to perform Run Start request")
			return
		}
		if p.trgClient.GetConnState() != connectivity.Ready {
			log.WithError(fmt.Errorf("TRG client connection not available")).
				WithField("endpoint", viper.GetString("trgServiceEndpoint")).
				WithField("partition", envId).
				WithField("runNumber", runNumber64).
				Error("failed to perform Run Start request")
			return
		}

		var response *trgecspb.RunReply

		response, err = p.trgClient.RunStart(context.Background(), &in, grpc.EmptyCallOption{})
		if err != nil {
			log.WithError(err).
				WithField("endpoint", viper.GetString("trgServiceEndpoint")).
				WithField("partition", envId).
				WithField("runNumber", runNumber64).
				Error("failed to perform Run Start request")
			return
		}
		if response != nil {
			if response.Rc != 0 {
				log.WithField("response rc", response.Rc).
					WithField("Message", response.Msg).
					WithField("partition", envId).
					WithField("runNumber", runNumber64).
					Error("Run Start failed")
				return
			}
		}
		log.WithField("partition", envId).
			WithField("runNumber", runNumber64).
			Debug("TRG RunStart success")

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

		in := trgecspb.RunStopRequest{
			Runn:     uint32(runNumber64),
			Detector: detectors,
		}

		if p.trgClient == nil {
			log.WithError(fmt.Errorf("TRG plugin not initialized")).
				WithField("endpoint", viper.GetString("trgServiceEndpoint")).
				WithField("partition", envId).
				WithField("runNumber", runNumber64).
				Error("failed to perform Run Stop request")
			return
		}
		if p.trgClient.GetConnState() != connectivity.Ready {
			log.WithError(fmt.Errorf("TRG client connection not available")).
				WithField("endpoint", viper.GetString("trgServiceEndpoint")).
				WithField("partition", envId).
				WithField("runNumber", runNumber64).
				Error("failed to perform Run Stop request")
			return
		}

		var response *trgecspb.RunReply
		response, err = p.trgClient.RunStop(context.Background(), &in, grpc.EmptyCallOption{})
		if err != nil {
			log.WithError(err).
				WithField("endpoint", viper.GetString("trgServiceEndpoint")).
				WithField("partition", envId).
				WithField("runNumber", runNumber64).
				Error("failed to perform Run Stop request")
			return
		}
		if response != nil {
			if response.Rc != 0 {
				log.WithField("response rc", response.Rc).
					WithField("Message", response.Msg).
					WithField("partition", envId).
					WithField("runNumber", runNumber64).
					Error("Run Stop failed")
				return
			}
		}
		log.WithField("partition", envId).
			WithField("runNumber", runNumber64).
			Debug("TRG RunStop success")

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

		in := trgecspb.RunStopRequest{
			Runn: uint32(runNumber64),
			// "" when unloading global run
			Detector: "",
		}

		if p.trgClient == nil {
			log.WithError(fmt.Errorf("TRG plugin not initialized")).
				WithField("endpoint", viper.GetString("trgServiceEndpoint")).
				WithField("partition", envId).
				WithField("runNumber", runNumber64).
				Error("failed to perform Run Unload request")
			return
		}
		if p.trgClient.GetConnState() != connectivity.Ready {
			log.WithError(fmt.Errorf("TRG client connection not available")).
				WithField("endpoint", viper.GetString("trgServiceEndpoint")).
				WithField("partition", envId).
				WithField("runNumber", runNumber64).
				Error("failed to perform Run Unload request")
			return
		}

		var response *trgecspb.RunReply
		response, err = p.trgClient.RunUnload(context.Background(), &in, grpc.EmptyCallOption{})
		if err != nil {
			log.WithError(err).
				WithField("endpoint", viper.GetString("trgServiceEndpoint")).
				WithField("partition", envId).
				WithField("runNumber", runNumber64).
				Error("failed to perform Run Unload request")
			return
		}
		if response != nil {
			if response.Rc != 0 {
				log.WithField("response rc", response.Rc).
					WithField("Message", response.Msg).
					WithField("partition", envId).
					WithField("runNumber", runNumber64).
					Error("Run Unload failed")
				return
			}
		}

		// RunUnload successful, we pop the run number from the cache
		delete(p.pendingEORs, envId)
		log.WithField("partition", envId).
			WithField("runNumber", runNumber64).
			Debug("TRG RunUnload success")

		return
	}
	stack["RunStop"] = func() (out string) {
		log.Debug("performing TRG Run Stop")

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
		log.Debug("performing TRG Run Unload")

		rn := varStack["run_number"]
		var runNumber64 int64
		var err error
		runNumber64, err = strconv.ParseInt(rn, 10, 32)
		if err != nil {
			log.WithError(err).
				WithField("partition", envId).
				Error("cannot acquire run number for TRG Run Stop")
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

		runNumber, ok := p.pendingEORs[envId]
		if !ok {
			log.WithField("partition", envId).
				WithField("level", infologger.IL_Devel).
				Debug("TRG cleanup: nothing to do")
			return
		}

		log.WithField("runNumber", runNumber).
			WithField("partition", envId).
			WithField("level", infologger.IL_Devel).
			Debug("pending TRG Stop/Unload found, performing cleanup")

		delete(p.pendingEORs, envId)

		_ = runStopFunc(runNumber)
		return runUnloadFunc(runNumber)
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
	return p.trgClient.Close()
}
