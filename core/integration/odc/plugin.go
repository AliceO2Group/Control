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

//go:generate protoc --go_out=. --go-grpc_out=require_unimplemented_servers=false:. protos/odc.proto

package odc

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
	odc "github.com/AliceO2Group/Control/core/integration/odc/protos"
	"github.com/AliceO2Group/Control/core/workflow/callable"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
)

const ODC_DIAL_TIMEOUT = 2 * time.Second


type Plugin struct {
	odcHost string
	odcPort int

	odcClient *RpcClient
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
		odcHost: u.Hostname(),
		odcPort: portNumber,
		odcClient:   nil,
	}
}

func (p *Plugin) GetName() string {
	return "odc"
}

func (p *Plugin) GetPrettyName() string {
	return "ODC"
}

func (p *Plugin) GetEndpoint() string {
	return viper.GetString("odcEndpoint")
}

func (p *Plugin) GetConnectionState() string {
	if p == nil || p.odcClient == nil {
		return "UNKNOWN"
	}
	return p.odcClient.conn.GetState().String()
}

func (p *Plugin) GetData(environmentIds []uid.ID) string {
	if p == nil || p.odcClient == nil {
		return ""
	}

	partitionStates := make(map[string]string)

	for _, envId := range environmentIds {
		in := odc.StateRequest{
			Partitionid: envId.String(),
			Path:        "",
			Detailed:    false,
		}
		state, err := p.odcClient.GetState(context.Background(), &in, grpc.EmptyCallOption{})
		if err != nil {
			continue
		}
		if state == nil || state.Reply == nil {
			continue
		}
		partitionStates[envId.String()] = state.Reply.State
	}

	out, err := json.Marshal(partitionStates)
	if err != nil {
		return ""
	}
	return string(out[:])
}

func (p *Plugin) Init(_ string) error {
	if p.odcClient == nil {
		cxt, cancel := context.WithCancel(context.Background())
		p.odcClient = NewClient(cxt, cancel, viper.GetString("odcEndpoint"))
		if p.odcClient == nil {
			return fmt.Errorf("failed to connect to ODC service on %s", viper.GetString("ddSchedulerEndpoint"))
		}
		log.Debug("ODC plugin initialized")
	}
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
	stack["Configure"] = func() (out string) {
		var topology, plugin, resources string
		ok := false
		topology, ok = varStack["odc_topology"]
		if !ok {
			log.Error("cannot acquire ODC topology")
			return
		}
		plugin, ok = varStack["odc_plugin"]
		if !ok {
			log.Error("cannot acquire ODC RMS plugin declaration")
			return
		}
		resources, ok = varStack["odc_resources"]
		if !ok {
			log.Error("cannot acquire ODC resources declaration")
			return
		}

		arguments := make(map[string]string)
		arguments["environment_id"] = envId

		// FIXME: this only copies over vars prefixed with "odc_"
		// Figure out a better way!
		for k, v := range varStack {
			if strings.HasPrefix(k, "odc_") {
				arguments[strings.TrimPrefix(k, "odc_")] = v
			}
		}

		err := handleConfigure(context.Background(), p.odcClient, arguments, topology, plugin, resources, envId)
		if err != nil {
			log.WithError(err).Error("ODC error")
		}
		return
	}
	stack["Start"] = func() (out string) {	// must formally return string even when we return nothing
		rn, ok := varStack["run_number"]
		if !ok {
			log.Warn("cannot acquire run number for ODC")
		}

		arguments := make(map[string]string)
		arguments["run_number"] = rn
		arguments["runNumber"] = rn

		err := handleStart(context.Background(), p.odcClient, arguments, envId)
		if err != nil {
			log.WithError(err).Error("ODC error")
		}
		return
	}
	stack["Stop"] = func() (out string) {
		err := handleStop(context.Background(), p.odcClient, nil, envId)
		if err != nil {
			log.WithError(err).Error("ODC error")
		}
		return
	}
	stack["Reset"] = func() (out string) {
		err := handleReset(context.Background(), p.odcClient, nil, envId)
		if err != nil {
			log.WithError(err).Error("ODC error")
		}
		return
	}
	stack["EnsureCleanup"] = func() (out string) {
		err := handleCleanup(context.Background(), p.odcClient, nil, envId)
		if err != nil {
			log.WithError(err).Error("ODC error")
		}
		return
	}

	return
}

func (p *Plugin) Destroy() error {
	return p.odcClient.Close()
}
