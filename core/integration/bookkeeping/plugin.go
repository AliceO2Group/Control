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
	"github.com/AliceO2Group/Control/core/environment"
	"github.com/AliceO2Group/Control/core/integration"
	"github.com/AliceO2Group/Control/core/workflow/callable"
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
	p.bookkeepingClient = Instance()
	if p.bookkeepingClient == nil {
		return fmt.Errorf("failed to connect to Bookkeeping service on %s", viper.GetString("bookkeepingBaseUri"))
	}
	log.Debug("Bookkeeping plugin initialized")
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
	// Run related Bookkeeping functions
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

		rnString := strconv.FormatUint(uint64(runNumber), 10)

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

		p.bookkeepingClient.CreateLog(env.GetVarsAsString(), fmt.Sprintf("Log for run %s and environment %s", rnString, env.Id().String()), rnString, -1)
		return
	}
	updateRunFunc := func(runNumber64 int64, state string) (out string) { // must formally return string even when we return nothing
		p.bookkeepingClient.UpdateRun(int32(runNumber64), state, time.Now(), time.Now())
		return
	}
	stack["UpdateRunStart"] = func() (out string) {
		var err error
		callFailedStr := "Bookkeeping UpdateRunStart call failed"

		rn := varStack["run_number"]
		runNumber64, err := strconv.ParseInt(rn, 10, 32)
		if err != nil {
			log.WithField("partition", envId).
				WithError(err).
				Error("cannot acquire run number for Bookkeeping UpdateRunStart")
		}

		log.WithField("partition", envId).
			WithField("level", infologger.IL_Ops).
			WithField("runNumber", runNumber64).
			Infof("performing Bookkeeping UpdateRunStart")

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

		parsedEnvId, err := uid.FromString(envId)
		envMan := environment.ManagerInstance()
		env, err := envMan.Environment(parsedEnvId)
		envState := env.CurrentState()
		if envState != "RUNNING" {
			return updateRunFunc(runNumber64, "bad")
		} else {
			return
		}
	}
	stack["UpdateRunStop"] = func() (out string) {
		var err error
		callFailedStr := "Bookkeeping UpdateRunStop call failed"

		rn := varStack["run_number"]
		runNumber64, err := strconv.ParseInt(rn, 10, 32)
		if err != nil {
			log.WithField("partition", envId).
				WithError(err).
				Error("cannot acquire run number for Bookkeeping UpdateRunStop")
		}

		log.WithField("partition", envId).
			WithField("level", infologger.IL_Ops).
			WithField("runNumber", runNumber64).
			Infof("performing Bookkeeping UpdateRunStop")

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

		parsedEnvId, err := uid.FromString(envId)
		envMan := environment.ManagerInstance()
		env, err := envMan.Environment(parsedEnvId)
		envState := env.CurrentState()
		if envState != "CONFIGURED" {
			return updateRunFunc(runNumber64, "bad")
		} else {
			return updateRunFunc(runNumber64, "good")
		}
	}
	// Environment related Bookkeeping functions
	stack["CreateEnv"] = func() (out string) {
		var err error
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

		parsedEnvId, err := uid.FromString(envId)
		envMan := environment.ManagerInstance()
		env, err := envMan.Environment(parsedEnvId)
		envState := env.CurrentState()
		if envState == "STANDBY" {
			p.bookkeepingClient.CreateEnvironment(env.Id().String(), time.Now(), envState, "success: the environment is in STANDBY state after creation")
		} else {
			p.bookkeepingClient.CreateEnvironment(env.Id().String(), time.Now(), envState, "error: the environment is in "+envState+" state after creation")
		}
		return
	}
	updateEnvFunc := func(envId string, toredownAt time.Time, status string, statusMessage string) (out string) {
		p.bookkeepingClient.UpdateEnvironment(envId, toredownAt, status, statusMessage)
		return
	}
	stack["UpdateDeployEnv"] = func() (out string) {
		var err error
		callFailedStr := "Bookkeeping UpdateDeployEnv call failed"

		if p.bookkeepingClient == nil {
			err = fmt.Errorf("Bookkeeping plugin not initialized, UpdateDeployEnv impossible")

			log.WithError(err).
				WithField("level", infologger.IL_Support).
				WithField("endpoint", viper.GetString("bookkeepingBaseUri")).
				WithField("partition", envId).
				WithField("call", "UpdateDeployEnv").
				Error("Bookkeeping UpdateDeployEnv error")

			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr

			return
		}

		parsedEnvId, err := uid.FromString(envId)
		envMan := environment.ManagerInstance()
		env, err := envMan.Environment(parsedEnvId)
		envState := env.CurrentState()
		if envState == "DEPLOYED" {
			return updateEnvFunc(env.Id().String(), time.Time{}, envState, "success: the environment is in DEPLOYED state after DEPLOY transition")
		} else {
			return updateEnvFunc(env.Id().String(), time.Time{}, envState, "error: the environment is in "+envState+" state after DEPLOY transition")
		}
	}
	stack["UpdateEnvConfigure"] = func() (out string) {
		var err error
		callFailedStr := "Bookkeeping UpdateEnvConfigure call failed"

		if p.bookkeepingClient == nil {
			err = fmt.Errorf("Bookkeeping plugin not initialized, UpdateEnvConfigure impossible")

			log.WithError(err).
				WithField("level", infologger.IL_Support).
				WithField("endpoint", viper.GetString("bookkeepingBaseUri")).
				WithField("partition", envId).
				WithField("call", "UpdateEnvConfigure").
				Error("Bookkeeping UpdateEnvConfigure error")

			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr

			return
		}

		parsedEnvId, err := uid.FromString(envId)
		envMan := environment.ManagerInstance()
		env, err := envMan.Environment(parsedEnvId)
		envState := env.CurrentState()
		if envState == "CONFIGURED" {
			return updateEnvFunc(env.Id().String(), time.Time{}, envState, "success: the environment is in CONFIGURED state after CONFIGURE transition")
		} else {
			return updateEnvFunc(env.Id().String(), time.Time{}, envState, "error: the environment is in "+envState+" state after CONFIGURE transition")
		}
	}
	stack["UpdateEnvReset"] = func() (out string) {
		var err error
		callFailedStr := "Bookkeeping UpdateEnvReset call failed"

		if p.bookkeepingClient == nil {
			err = fmt.Errorf("Bookkeeping plugin not initialized, UpdateEnvReset impossible")

			log.WithError(err).
				WithField("level", infologger.IL_Support).
				WithField("endpoint", viper.GetString("bookkeepingBaseUri")).
				WithField("partition", envId).
				WithField("call", "UpdateEnvReset").
				Error("Bookkeeping UpdateEnvReset error")

			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr

			return
		}

		parsedEnvId, err := uid.FromString(envId)
		envMan := environment.ManagerInstance()
		env, err := envMan.Environment(parsedEnvId)
		envState := env.CurrentState()
		if envState == "DEPLOYED" {
			return updateEnvFunc(env.Id().String(), time.Time{}, envState, "success: the environment is in DEPLOYED state after RESET transition")
		} else {
			return updateEnvFunc(env.Id().String(), time.Time{}, envState, "error: the environment is in "+envState+" state after RESET transition")
		}
	}
	stack["UpdateEnvStart"] = func() (out string) {
		var err error
		callFailedStr := "Bookkeeping UpdateEnvStart call failed"

		if p.bookkeepingClient == nil {
			err = fmt.Errorf("Bookkeeping plugin not initialized, UpdateEnvStart impossible")

			log.WithError(err).
				WithField("level", infologger.IL_Support).
				WithField("endpoint", viper.GetString("bookkeepingBaseUri")).
				WithField("partition", envId).
				WithField("call", "UpdateEnvStart").
				Error("Bookkeeping UpdateEnvStart error")

			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr

			return
		}

		parsedEnvId, err := uid.FromString(envId)
		envMan := environment.ManagerInstance()
		env, err := envMan.Environment(parsedEnvId)
		envState := env.CurrentState()
		if envState == "RUNNING" {
			return updateEnvFunc(env.Id().String(), time.Time{}, envState, "success: the environment is in RUNNING state after START_ACTIVITY transition")
		} else {
			return updateEnvFunc(env.Id().String(), time.Time{}, envState, "error: the environment is in "+envState+" state after START_ACTIVITY transition")
		}
	}
	stack["UpdateEnvStop"] = func() (out string) {
		var err error
		callFailedStr := "Bookkeeping UpdateEnvStop call failed"

		if p.bookkeepingClient == nil {
			err = fmt.Errorf("Bookkeeping plugin not initialized, UpdateEnvStop impossible")

			log.WithError(err).
				WithField("level", infologger.IL_Support).
				WithField("endpoint", viper.GetString("bookkeepingBaseUri")).
				WithField("partition", envId).
				WithField("call", "UpdateEnvStop").
				Error("Bookkeeping UpdateEnvStop error")

			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr

			return
		}

		parsedEnvId, err := uid.FromString(envId)
		envMan := environment.ManagerInstance()
		env, err := envMan.Environment(parsedEnvId)
		envState := env.CurrentState()
		if envState == "CONFIGURED" {
			return updateEnvFunc(env.Id().String(), time.Time{}, envState, "success: the environment is in CONFIGURED state after STOP_ACTIVITY transition")
		} else {
			return updateEnvFunc(env.Id().String(), time.Time{}, envState, "error: the environment is in "+envState+" state after STOP_ACTIVITY transition")
		}
	}
	stack["UpdateEnvExit"] = func() (out string) {
		var err error
		callFailedStr := "Bookkeeping UpdateEnvStop call failed"

		if p.bookkeepingClient == nil {
			err = fmt.Errorf("Bookkeeping plugin not initialized, UpdateEnvExit impossible")

			log.WithError(err).
				WithField("level", infologger.IL_Support).
				WithField("endpoint", viper.GetString("bookkeepingBaseUri")).
				WithField("partition", envId).
				WithField("call", "UpdateEnvExit").
				Error("Bookkeeping UpdateEnvExit error")

			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr

			return
		}

		parsedEnvId, err := uid.FromString(envId)
		envMan := environment.ManagerInstance()
		env, err := envMan.Environment(parsedEnvId)
		envState := env.CurrentState()
		if envState == "DONE" {
			return updateEnvFunc(env.Id().String(), time.Now(), envState, "success: the environment is in DONE state after EXIT transition")
		} else {
			return updateEnvFunc(env.Id().String(), time.Now(), envState, "error: the environment is in "+envState+" state after EXIT transition")
		}
	}
	stack["UpdateEnvGoError"] = func() (out string) {
		var err error
		callFailedStr := "Bookkeeping UpdateEnvGoError call failed"

		if p.bookkeepingClient == nil {
			err = fmt.Errorf("Bookkeeping plugin not initialized, UpdateEnvGoError impossible")

			log.WithError(err).
				WithField("level", infologger.IL_Support).
				WithField("endpoint", viper.GetString("bookkeepingBaseUri")).
				WithField("partition", envId).
				WithField("call", "UpdateEnvGoError").
				Error("Bookkeeping UpdateEnvGoError error")

			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr

			return
		}

		parsedEnvId, err := uid.FromString(envId)
		envMan := environment.ManagerInstance()
		env, err := envMan.Environment(parsedEnvId)
		envState := env.CurrentState()
		if envState == "ERROR" {
			return updateEnvFunc(env.Id().String(), time.Time{}, envState, "success: the environment is in ERROR state after GO_ERROR transition")
		} else {
			return updateEnvFunc(env.Id().String(), time.Time{}, envState, "error: the environment is in "+envState+" state after GO_ERROR transition")
		}
	}
	stack["UpdateEnvRecover"] = func() (out string) {
		var err error
		callFailedStr := "Bookkeeping UpdateEnvRecover call failed"

		if p.bookkeepingClient == nil {
			err = fmt.Errorf("Bookkeeping plugin not initialized, UpdateEnvRecover impossible")

			log.WithError(err).
				WithField("level", infologger.IL_Support).
				WithField("endpoint", viper.GetString("bookkeepingBaseUri")).
				WithField("partition", envId).
				WithField("call", "UpdateEnvRecover").
				Error("Bookkeeping UpdateEnvRecover error")

			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr

			return
		}

		parsedEnvId, err := uid.FromString(envId)
		envMan := environment.ManagerInstance()
		env, err := envMan.Environment(parsedEnvId)
		envState := env.CurrentState()
		if envState == "DEPLOYED" {
			return updateEnvFunc(env.Id().String(), time.Time{}, envState, "success: the environment is in DEPLOYED state after RECOVER transition")
		} else {
			return updateEnvFunc(env.Id().String(), time.Time{}, envState, "error: the environment is in "+envState+" state after RECOVER transition")
		}
	}

	return
}

func (p *Plugin) Destroy() error {
	return nil
}
