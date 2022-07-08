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
	"errors"
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
		detectors := strings.Join(env.GetActiveDetectors().StringList(), ",")
		//odcTopology, _ := env.Workflow().GetVars().Get("odc_topology_fullname")
		//lhcPeriod := env.GetKV("", "lhc_period")

		err = p.bookkeepingClient.CreateRun(env.Id().String(), len(env.GetActiveDetectors()), 0, len(flps), int32(runNumber), env.GetRunType().String(), ddEnabled, dcsEnabled, epnEnabled, odcTopology, detectors)
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
			log.WithField("runNumber", runNumber).
				WithField("partition", envId).
				Debug("CreateRun call successful")
		}

		for _, flp := range flps {
			err = p.bookkeepingClient.CreateFlp(flp, flp, int32(runNumber))
			if err != nil {
				log.WithError(err).
					WithField("flp", flp).
					WithField("runNumber", runNumber).
					WithField("partition", envId).
					WithField("call", "CreateFlp").
					Error("Bookkeeping API CreateFlp error")

				call.VarStack["__call_error_reason"] = err.Error()
				call.VarStack["__call_error"] = callFailedStr
				return
			}
		}
		log.WithField("runNumber", runNumber).
			WithField("partition", envId).
			Debug("CreateFlp call done")

		err = p.bookkeepingClient.CreateLog(env.GetVarsAsString(), fmt.Sprintf("Log for run %s and environment %s", rnString, env.Id().String()), rnString, -1)
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
	updateRunFunc := func(runNumber64 int64, state string, timeO2Start time.Time, timeO2End time.Time, timeTrgStart time.Time, timeTrgEnd time.Time) (out string) {
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
		tfbMode := env.GetKV("", "tfb_dd_mode")
		//odcTopology, _ := env.Workflow().GetVars().Get("odc_topology_fullname")
		//lhcPeriod := env.GetKV("", "lhc_period")
		err = p.bookkeepingClient.UpdateRun(int32(runNumber64), state, timeO2Start, timeO2End, timeTrgStart, timeTrgEnd, trgGlobalRunEnabled, trgEnabled, pdpConfig, pdpTopology, tfbMode /*, odcTopology, lhcPeriod */)
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
			log.WithField("runNumber", runNumber64).
				WithField("state", state).
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

		return updateRunFunc(runNumber64, "test", time.Now(), time.Time{}, time.Now(), time.Time{})
	}
	stack["UpdateRunStop"] = func() (out string) {
		callFailedStr := "Bookkeeping UpdateRunStop call failed"

		rn := varStack["run_number"]
		runNumber64, err := strconv.ParseInt(rn, 10, 32)
		if (strings.Contains(trigger, "DESTROY") || strings.Contains(trigger, "GO_ERROR")) && (rn == "" || runNumber64 == 0) {
			log.WithField("partition", envId).
				Debug("cannot update run, run number is 0 in UpdateRunStop")
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

		return updateRunFunc(runNumber64, "test", time.Time{}, time.Now(), time.Time{}, time.Now())
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

		envState := env.CurrentState()
		if envState == "STANDBY" || envState == "DEPLOYED" {
			err = p.bookkeepingClient.CreateEnvironment(env.Id().String(), time.Now(), envState, "success: the environment is in "+envState+" state after creation")
		} else {
			err = p.bookkeepingClient.CreateEnvironment(env.Id().String(), time.Now(), envState, "error: the environment is in "+envState+" state after creation")
		}
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
		err := p.bookkeepingClient.UpdateEnvironment(envId, toredownAt, status, statusMessage)
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
