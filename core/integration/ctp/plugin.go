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

package ctp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/AliceO2Group/Control/common/utils/uid"
	"github.com/AliceO2Group/Control/core/integration"
	ctpecspb "github.com/AliceO2Group/Control/core/integration/ctp/protos"
	"github.com/AliceO2Group/Control/core/workflow/callable"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
)

const CTP_DIAL_TIMEOUT = 2 * time.Second

type Plugin struct {
	ctpHost string
	ctpPort int

	ctpClient *RpcClient
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
		ctpHost:   u.Hostname(),
		ctpPort:   portNumber,
		ctpClient: nil,
	}
}

func (p *Plugin) GetName() string {
	return "ctp"
}

func (p *Plugin) GetPrettyName() string {
	return "Central Trigger Processor"
}

func (p *Plugin) GetEndpoint() string {
	return viper.GetString("ctpServiceEndpoint")
}

func (p *Plugin) GetConnectionState() string {
	if p == nil || p.ctpClient == nil {
		return "UNKNOWN"
	}
	return p.ctpClient.conn.GetState().String()
}

func (p *Plugin) GetData(environmentIds []uid.ID) string {
	if p == nil || p.ctpClient == nil {
		return ""
	}
	return ""
}

func (p *Plugin) Init(instanceId string) error {
	if p.ctpClient == nil {
		callTimeout := CTP_DIAL_TIMEOUT
		cxt, cancel := context.WithTimeout(context.Background(), callTimeout)
		p.ctpClient = NewClient(cxt, cancel, viper.GetString("ctpServiceEndpoint"))
		if p.ctpClient == nil {
			return fmt.Errorf("failed to connect to CTP service on %s", viper.GetString("ctpServiceEndpoint"))
		}
	}
	if p.ctpClient == nil {
		return fmt.Errorf("failed to start CTP client on %s", viper.GetString("ctpServiceEndpoint"))
	}
	return nil
}

func (p *Plugin) ObjectStack(data interface{}) (stack map[string]interface{}) {
	call, ok := data.(*callable.Call)
	if !ok {
		return
	}
	varStack := call.VarStack
	stack = make(map[string]interface{})
	// global runs only
	stack["RunLoad"] = func() (out string) { // must formally return string even when we return nothing
		log.Debug("performing CTP Run load Request")

		globalConfig, ok := varStack["ctp_global_config"]
		log.WithField("gloablConfig",globalConfig).Debug("not a CTP Global Run, continuing with CTP Run Start")
		if !ok {
			log.Debug("no CTP Global config set")
			globalConfig = ""
		}
		// TODO (malexis): pass consul key to CTP if avail

		rn := varStack["run_number"]
		var runNumber64 int64
		runNumber64, err := strconv.ParseInt(rn, 10, 32)
		if err != nil {
			log.WithError(err).Error("cannot acquire run number for Run Load")
		}

		ctpDetectorsParam, ok := varStack["ctp_detectors"]
		if !ok {
			// "" -all required must be ready
			log.Debug("empty CTP detectors list provided")
			ctpDetectorsParam = ""
		}

		detectors, err := p.parseDetectors(ctpDetectorsParam)
		if err != nil {
			return
		}

		// standalone run
		if len(strings.Split(detectors, " ")) < 2 && varStack["ctp_global_run_enabled"] == "false" {
			// we do not load any run cause it is standalone
			log.Debug("not a CTP Global Run, continuing with CTP Run Start")
			return
		}

		in := ctpecspb.RunLoadRequest{
			Runn:      uint32(runNumber64),
			Detectors: detectors,
			Config:    globalConfig,
		}
		if p.ctpClient == nil {
			log.WithError(fmt.Errorf("CTP plugin not initialized")).
				WithField("endpoint", viper.GetString("ctpServiceEndpoint")).
				Error("failed to perform Run Load request")
			return
		}
		if p.ctpClient.GetConnState() != connectivity.Ready {
			log.WithError(fmt.Errorf("CTP client connection not available")).
				WithField("endpoint", viper.GetString("ctpServiceEndpoint")).
				Error("failed to perform Run Load request")
			return
		}

		var response *ctpecspb.RunReply
		response, err = p.ctpClient.RunLoad(context.Background(), &in, grpc.EmptyCallOption{})
		if err != nil {
			log.WithError(err).
				WithField("endpoint", viper.GetString("ctpServiceEndpoint")).
				Error("failed to perform Run Load request")
		}
		if response != nil {
			if response.Rc != 0 {
				log.WithField("response rc", response.Rc).
					WithField("Message", response.Msg).
					Error("Run Load failed")
			}
		}
		return
	}
	stack["RunStart"] = func() (out string) { // must formally return string even when we return nothing
		log.Debug("performing CTP Run Start")

		runtimeConfig, ok := varStack["ctp_runtime_config"]
		if !ok {
			log.Debug("no CTP config set, using default configuration")
			runtimeConfig = ""
		}

		rn := varStack["run_number"]
		var runNumber64 int64
		runNumber64, err := strconv.ParseInt(rn, 10, 32)
		if err != nil {
			log.WithError(err).Error("cannot acquire run number for Run Start")
		}

		ctpDetectorsParam, ok := varStack["ctp_detectors"]
		if !ok {
			// "" it is a global run
			log.Debug("Detector for host is not available, starting global run")
			ctpDetectorsParam = ""
		}

		detectors, err := p.parseDetectors(ctpDetectorsParam)
		if err != nil {
			return
		}

		// if global run then start with empty string in detectors
		if len(strings.Split(detectors, " ")) >= 2 || varStack["ctp_global_run_enabled"] == "true" {
			// global run detectors ""
			detectors = ""
		}

		in := ctpecspb.RunStartRequest{
			Runn:     uint32(runNumber64),
			Detector: detectors,
			Config:   runtimeConfig,
		}

		if p.ctpClient == nil {
			log.WithError(fmt.Errorf("CTP plugin not initialized")).
				WithField("endpoint", viper.GetString("ctpServiceEndpoint")).
				Error("failed to perform Run Start request")
			return
		}
		if p.ctpClient.GetConnState() != connectivity.Ready {
			log.WithError(fmt.Errorf("CTP client connection not available")).
				WithField("endpoint", viper.GetString("ctpServiceEndpoint")).
				Error("failed to perform Run Start request")
			return
		}

		var response *ctpecspb.RunReply

		response, err = p.ctpClient.RunStart(context.Background(), &in, grpc.EmptyCallOption{})
		if err != nil {
			log.WithError(err).
				WithField("endpoint", viper.GetString("ctpServiceEndpoint")).
				Error("failed to perform Run Start request")
		}
		if response != nil {
			if response.Rc != 0 {
				log.WithField("response rc", response.Rc).
					WithField("Message", response.Msg).
					Error("Run Start failed")
			}
		}

		return
	}
	stack["RunStop"] = func() (out string) {
		log.Debug("performing CTP Run Stop")

		rn := varStack["run_number"]
		var runNumber64 int64
		var err error
		runNumber64, err = strconv.ParseInt(rn, 10, 32)
		if err != nil {
			log.WithError(err).Error("cannot acquire run number for CTP Run Stop")
		}

		ctpDetectorsParam, ok := varStack["ctp_detectors"]
		if !ok {
			// "" it is a global run
			log.Debug("Detector for host is not available, stoping global run")
			ctpDetectorsParam = ""
		}

		detectors, err := p.parseDetectors(ctpDetectorsParam)
		if err != nil {
			return
		}

		// if global run then start with empty
		if len(strings.Split(detectors, " ")) >= 2 || varStack["ctp_global_run_enabled"] == "true" {
			// global run detectors ""
			detectors = ""
		}

		in := ctpecspb.RunStopRequest{
			Runn:     uint32(runNumber64),
			Detector: detectors,
		}

		if p.ctpClient == nil {
			log.WithError(fmt.Errorf("CTP plugin not initialized")).
				WithField("endpoint", viper.GetString("ctpServiceEndpoint")).
				Error("failed to perform Run Stop request")
			return
		}
		if p.ctpClient.GetConnState() != connectivity.Ready {
			log.WithError(fmt.Errorf("CTP client connection not available")).
				WithField("endpoint", viper.GetString("ctpServiceEndpoint")).
				Error("failed to perform Run Stop request")
			return
		}

		var response *ctpecspb.RunReply
		response, err = p.ctpClient.RunStop(context.Background(), &in, grpc.EmptyCallOption{})
		if err != nil {
			log.WithError(err).
				WithField("endpoint", viper.GetString("ctpServiceEndpoint")).
				Error("failed to perform Run Stop request")
		}
		if response != nil {
			if response.Rc != 0 {
				log.WithField("response rc", response.Rc).
					WithField("Message", response.Msg).
					Error("Run Stop failed")
			}
		}
		return
	}
	stack["RunUnload"] = func() (out string) {
		log.Debug("performing CTP Run Unload")

		rn := varStack["run_number"]
		var runNumber64 int64
		var err error
		runNumber64, err = strconv.ParseInt(rn, 10, 32)
		if err != nil {
			log.WithError(err).Error("cannot acquire run number for CTP Run Stop")
		}

		ctpDetectorsParam, ok := varStack["ctp_detectors"]
		if !ok {
			ctpDetectorsParam = ""
		}

		detectors, err := p.parseDetectors(ctpDetectorsParam)
		if err != nil {
			return
		}

		// if global run then unload
		if len(strings.Split(detectors, " ")) < 2 && varStack["ctp_global_run_enabled"] == "false" {
			log.Debug("not a CTP Global Run, skipping CTP Run Unload")
			return
		}

		in := ctpecspb.RunStopRequest{
			Runn: uint32(runNumber64),
			// "" when unloading global run
			Detector: "",
		}

		if p.ctpClient == nil {
			log.WithError(fmt.Errorf("CTP plugin not initialized")).
				WithField("endpoint", viper.GetString("ctpServiceEndpoint")).
				Error("failed to perform Run Unload request")
			return
		}
		if p.ctpClient.GetConnState() != connectivity.Ready {
			log.WithError(fmt.Errorf("CTP client connection not available")).
				WithField("endpoint", viper.GetString("ctpServiceEndpoint")).
				Error("failed to perform Run Unload request")
			return
		}

		var response *ctpecspb.RunReply
		response, err = p.ctpClient.RunUnload(context.Background(), &in, grpc.EmptyCallOption{})
		if err != nil {
			log.WithError(err).
				WithField("endpoint", viper.GetString("ctpServiceEndpoint")).
				Error("failed to perform Run Unload request")
		}
		if response != nil {
			if response.Rc != 0 {
				log.WithField("response rc", response.Rc).
					WithField("Message", response.Msg).
					Error("Run Unload failed")
			}
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
		log.WithError(err).Error("error processing CTP detectors list")
		return
	}

	detectors = strings.ToLower(strings.Join(detectorsSlice, " "))
	return
}

func (p *Plugin) Destroy() error {
	return p.ctpClient.Close()
}