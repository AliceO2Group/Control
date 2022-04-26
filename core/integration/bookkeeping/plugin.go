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

/*
	//go:generate protoc --go_out=. --go-grpc_out=require_unimplemented_servers=false:. protos/bookkeeping.proto
*/

package bookkeeping

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/AliceO2Group/Control/common/logger/infologger"
	"github.com/AliceO2Group/Control/common/utils/uid"
	"github.com/AliceO2Group/Control/core/controlcommands"
	"github.com/AliceO2Group/Control/core/environment"
	"github.com/AliceO2Group/Control/core/integration"
	"github.com/AliceO2Group/Control/core/workflow/callable"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type Plugin struct {
	bookkeepingHost string
	bookkeepingPort int

	bookkeepingClient *BookkeepingWrapper
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
	if p == nil || p.bookkeepingClient == nil {
		return "UNKNOWN"
	}
	return "READY"
}

func (p *Plugin) GetData(environmentIds []uid.ID) string {
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

func (p *Plugin) Init(instanceId string) error {
	if p.bookkeepingClient == nil {
		p.bookkeepingClient = Instance()
		if p.bookkeepingClient == nil {
			return fmt.Errorf("failed to connect to Bookkeeping service on %s", viper.GetString("bookkeepingBaseUri"))
		}
		log.Debug("Bookkeeping plugin initialized")
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
	stack["StartOfRun"] = func() (out string) { // must formally return string even when we return nothing
		var err error
		callFailedStr := "Bookkeeping StartOfRun call failed"

		rn := varStack["run_number"]
		var runNumber64 int64
		runNumber64, err = strconv.ParseInt(rn, 10, 32)
		if err != nil {
			log.WithField("partition", envId).
				WithError(err).
				Error("cannot acquire run number for Bookkeeping SOR")
		}

		log.WithField("partition", envId).
			WithField("level", infologger.IL_Ops).
			WithField("runNumber", runNumber64).
			Infof("performing Bookkeeping SOR")

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

		parsedEnvId, err := uid.FromString(envId)
		envMan := environment.ManagerInstance()
		env, err := envMan.Environment(parsedEnvId)
		runNumber := env.GetCurrentRunNumber()

		args := controlcommands.PropertyMap{
			"runNumber": strconv.FormatUint(uint64(runNumber), 10),
		}

		flps := env.GetFLPs()
		dd_enabled, _ := strconv.ParseBool(env.GetKV("", "dd_enabled"))
		dcs_enabled, _ := strconv.ParseBool(env.GetKV("", "dcs_enabled"))
		epn_enabled, _ := strconv.ParseBool(env.GetKV("", "epn_enabled"))
		odc_topology := env.GetKV("", "odc_topology")
		detectors := strings.Join(env.GetActiveDetectors().StringList(), ",")

		p.bookkeepingClient.CreateRun(env.Id().String(), len(env.GetActiveDetectors()), 0, len(flps), int32(runNumber), env.GetRunType().String(), time.Now(), time.Now(), dd_enabled, dcs_enabled, epn_enabled, odc_topology, detectors)

		for _, flp := range flps {
			p.bookkeepingClient.CreateFlp(flp, flp, int32(runNumber))
		}

		p.bookkeepingClient.CreateLog(env.GetVarsAsString(), fmt.Sprintf("Log for run %s and environment %s", args["runNumber"], env.Id().String()), args["runNumber"], -1)
		return
	}
	updateFunc := func(runNumber64 int64, state string) (out string) { // must formally return string even when we return nothing
		p.bookkeepingClient.UpdateRun(int32(runNumber64), state, time.Now(), time.Now())
		return
	}
	stack["UpdateRunStart"] = func() (out string) {
		var err error
		callFailedStr := "Bookkeeping UpdateRun call failed"

		rn := varStack["run_number"]
		runNumber64, err := strconv.ParseInt(rn, 10, 32)
		if err != nil {
			log.WithField("partition", envId).
				WithError(err).
				Error("cannot acquire run number for Bookkeeping UpdateRun")
		}

		log.WithField("partition", envId).
			WithField("level", infologger.IL_Ops).
			WithField("runNumber", runNumber64).
			Infof("performing Bookkeeping UpdateRun")

		if p.bookkeepingClient == nil {
			err = fmt.Errorf("Bookkeeping plugin not initialized, UpdateRun impossible")

			log.WithError(err).
				WithField("level", infologger.IL_Support).
				WithField("endpoint", viper.GetString("bookkeepingBaseUri")).
				WithField("runNumber", runNumber64).
				WithField("partition", envId).
				WithField("call", "UpdateRun").
				Error("Bookkeeping UpdateRun error")

			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr

			return
		}

		parsedEnvId, err := uid.FromString(envId)
		envMan := environment.ManagerInstance()
		env, err := envMan.Environment(parsedEnvId)
		envState := env.CurrentState()
		if envState != "RUNNING" {
			return updateFunc(runNumber64, "bad")
		} else {
			return
		}
	}
	stack["UpdateRunStop"] = func() (out string) {
		var err error
		callFailedStr := "Bookkeeping UpdateRun call failed"

		rn := varStack["run_number"]
		runNumber64, err := strconv.ParseInt(rn, 10, 32)
		if err != nil {
			log.WithField("partition", envId).
				WithError(err).
				Error("cannot acquire run number for Bookkeeping UpdateRun")
		}

		log.WithField("partition", envId).
			WithField("level", infologger.IL_Ops).
			WithField("runNumber", runNumber64).
			Infof("performing Bookkeeping UpdateRun")

		if p.bookkeepingClient == nil {
			err = fmt.Errorf("Bookkeeping plugin not initialized, UpdateRun impossible")

			log.WithError(err).
				WithField("level", infologger.IL_Support).
				WithField("endpoint", viper.GetString("bookkeepingBaseUri")).
				WithField("runNumber", runNumber64).
				WithField("partition", envId).
				WithField("call", "UpdateRun").
				Error("Bookkeeping UpdateRun error")

			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr

			return
		}

		parsedEnvId, err := uid.FromString(envId)
		envMan := environment.ManagerInstance()
		env, err := envMan.Environment(parsedEnvId)
		envState := env.CurrentState()
		if envState != "CONFIGURED" {
			return updateFunc(runNumber64, "bad")
		} else {
			return updateFunc(runNumber64, "good")
		}
	}

	return
}

func (p *Plugin) Destroy() error {
	return nil
}
