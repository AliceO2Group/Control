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

package cts

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/AliceO2Group/Control/common/utils/uid"
	"github.com/AliceO2Group/Control/core/integration"
	ctpecspb "github.com/AliceO2Group/Control/core/integration/cts/protos"
	"github.com/AliceO2Group/Control/core/workflow/callable"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
)

const CTS_DIAL_TIMEOUT = 2 * time.Second

type Plugin struct {
	ctsHost        string
	ctsPort        int

	ctsClient      *RpcClient

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
		ctsHost:   u.Hostname(),
		ctsPort:   portNumber,
		ctsClient: nil,
	}
}

func (p *Plugin) GetName() string {
	return "cts"
}

func (p *Plugin) GetPrettyName() string {
	return "Central Trigger System"
}

func (p *Plugin) GetEndpoint() string {
	return viper.GetString("ctsServiceEndpoint")
}

func (p *Plugin) GetConnectionState() string {
	if p == nil || p.ctsClient == nil {
		return "UNKNOWN"
	}
	return p.ctsClient.conn.GetState().String()
}

func (p *Plugin) GetData(environmentIds []uid.ID) string {
	if p == nil || p.ctsClient == nil {
		return ""
	}
	return ""
}

func (p *Plugin) Init(instanceId string) error {
	if p.ctsClient == nil {
		callTimeout := CTS_DIAL_TIMEOUT
		cxt, cancel := context.WithTimeout(context.Background(), callTimeout)
		p.ctsClient = NewClient(cxt, cancel, viper.GetString("ctsServiceEndpoint"))
		if p.ctsClient == nil {
			return fmt.Errorf("failed to connect to CTS service on %s", viper.GetString("ctsServiceEndpoint"))
		}
	}
	if p.ctsClient == nil {
		return fmt.Errorf("failed to start CTS client on %s", viper.GetString("ctsServiceEndpoint"))
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
	stack["RunLoad"] = func() (out string) {	// must formally return string even when we return nothing
		log.Debug("performing run load request")

		parameters, ok := varStack["cts_run_parameters"]
		if !ok {
			log.Debug("no CTS config set, using default configuration")
			parameters = ""
		}
		// TODO (malexis): pass consul key to CTS if avail

		rn := varStack["run_number"]
		var runNumber64 int64
		runNumber64, err := strconv.ParseInt(rn, 10, 32)
		if err != nil {
			log.WithError(err).Error("cannot acquire run number for Run Load")
		}

		ctsDetectorsParam, ok := varStack["cts_detectors"]
		if !ok {
			// "" -all required must be ready
			log.Debug("empty CTS detectors list provided")
			ctsDetectorsParam = ""
		}

		in := ctpecspb.RunLoadRequest{
			Runn:  uint32(runNumber64),
			Detectors:   ctsDetectorsParam,
			Config: parameters,
		}
		if p.ctsClient == nil {
			log.WithError(fmt.Errorf("CTS plugin not initialized")).
				WithField("endpoint", viper.GetString("ctsServiceEndpoint")).
				Error("failed to perform Run Load request")
			return
		}
		if p.ctsClient.GetConnState() != connectivity.Ready {
			log.WithError(fmt.Errorf("CTS client connection not available")).
				WithField("endpoint", viper.GetString("ctsServiceEndpoint")).
				Error("failed to perform Run Load request")
			return
		}

		var response *ctpecspb.RunReply
		response, err = p.ctsClient.RunLoad(context.Background(), &in, grpc.EmptyCallOption{})
		if err != nil {
			log.WithError(err).
				WithField("endpoint", viper.GetString("ctsServiceEndpoint")).
				Error("failed to perform Run Load request")
		}
		if response.Rc != 2 {
			log.WithField("response rc", response.Rc).
				WithField("Message", response.Msg).
				Error("Run Load failed")
		}
		return
	}
	stack["RunStart"] = func() (out string) {	// must formally return string even when we return nothing
		log.Debug("performing CTS run start")

		parameters, ok := varStack["cts_run_parameters"]
		if !ok {
			log.Debug("no CTS config set, using default configuration")
			parameters = ""
		}
		// TODO (malexis): pass consul key to CTS if avail

		rn := varStack["run_number"]
		var runNumber64 int64
		runNumber64, err := strconv.ParseInt(rn, 10, 32)
		if err != nil {
			log.WithError(err).Error("cannot acquire run number for Run Start")
		}

		ctsDetector, ok := varStack["detector"]
		if !ok {
			// "" it is a global run
			log.Debug("Detector for host is not available, starting global run")
			ctsDetector = ""
		}

		in := ctpecspb.RunStartRequest{
			Runn:  uint32(runNumber64),
			Detector:   ctsDetector,
			Config: parameters,
		}

		if p.ctsClient == nil {
			log.WithError(fmt.Errorf("CTS plugin not initialized")).
				WithField("endpoint", viper.GetString("ctsServiceEndpoint")).
				Error("failed to perform Run Start request")
			return
		}
		if p.ctsClient.GetConnState() != connectivity.Ready {
			log.WithError(fmt.Errorf("CTS client connection not available")).
				WithField("endpoint", viper.GetString("ctsServiceEndpoint")).
				Error("failed to perform Run Start request")
			return
		}

		var response *ctpecspb.RunReply

		response, err = p.ctsClient.RunStart(context.Background(), &in, grpc.EmptyCallOption{})
		if err != nil {
			log.WithError(err).
				WithField("endpoint", viper.GetString("ctsServiceEndpoint")).
				Error("failed to perform Run Start request")
		}
		if response.Rc != 0 {
			log.WithField("response rc", response.Rc).
				WithField("Message", response.Msg).
				Error("Run Start failed")
		}

		return
	}
	stack["RunStop"] = func() (out string) {
		rn := varStack["run_number"]
		var runNumber64 int64
		var err error
		runNumber64, err = strconv.ParseInt(rn, 10, 32)
		if err != nil {
			log.WithError(err).Error("cannot acquire run number for CTS Run Stop")
		}

		ctsDetector, ok := varStack["DetectorForHost"]
		if !ok {
			// "" it is a global run
			log.Debug("Detector for host is not available, starting global run")
			ctsDetector = ""
		}

		in := ctpecspb.RunStopRequest{
			Runn:  uint32(runNumber64),
			Detector:   ctsDetector,
		}

		if p.ctsClient == nil {
			log.WithError(fmt.Errorf("CTS plugin not initialized")).
				WithField("endpoint", viper.GetString("ctsServiceEndpoint")).
				Error("failed to perform Run Stop request")
			return
		}
		if p.ctsClient.GetConnState() != connectivity.Ready {
			log.WithError(fmt.Errorf("CTS client connection not available")).
				WithField("endpoint", viper.GetString("ctsServiceEndpoint")).
				Error("failed to perform Run Stop request")
			return
		}

		var response *ctpecspb.RunReply
		response, err = p.ctsClient.RunStop(context.Background(), &in, grpc.EmptyCallOption{})
		if err != nil {
			log.WithError(err).
				WithField("endpoint", viper.GetString("ctsServiceEndpoint")).
				Error("failed to perform Run Stop request")
		}
		if response.Rc == 3 {
			log.WithField("response rc", response.Rc).
				WithField("Message", response.Msg).
				Error("Run Stop failed")
		}
		return
	}

	return
}

func (p *Plugin) Destroy() error {
	return p.ctsClient.Close()
}
