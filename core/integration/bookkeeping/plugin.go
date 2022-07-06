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
	var err error
	parsedEnvId, err := uid.FromString(envId)
	if err != nil {
		log.WithError(err).
			WithField("partition", envId).
			Error("cannot parse environment ID")
		return
	}
	envMan := environment.ManagerInstance()
	env, err := envMan.Environment(parsedEnvId)
	if err != nil {
		log.WithError(err).
			WithField("partition", envId).
			Error("cannot acquire environment from parsed environment ID")
		return
	}

	stack = make(map[string]interface{})
	// Run related Bookkeeping functions
	stack["StartOfRun"] = func() (out string) {
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
		dd_enabled, _ := strconv.ParseBool(env.GetKV("", "dd_enabled"))
		dcs_enabled, _ := strconv.ParseBool(env.GetKV("", "dcs_enabled"))
		epn_enabled, _ := strconv.ParseBool(env.GetKV("", "epn_enabled"))
		odc_topology := env.GetKV("", "odc_topology")
		detectors := strings.Join(env.GetActiveDetectors().StringList(), ",")
		//odc_topology_fullname, _ := env.Workflow().GetVars().Get("odc_topology_fullname")
		//lhc_period := env.GetKV("", "lhc_period")

		err = p.bookkeepingClient.CreateRun(env.Id().String(), len(env.GetActiveDetectors()), 0, len(flps), int32(runNumber), env.GetRunType().String(), dd_enabled, dcs_enabled, epn_enabled, odc_topology, detectors)
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
		//odc_topology_fullname, _ := env.Workflow().GetVars().Get("odc_topology_fullname")
		trg_global_run_enabled, _ := strconv.ParseBool(env.GetKV("", "trg_global_run_enabled"))
		trg_enabled, _ := strconv.ParseBool(env.GetKV("", "trg_enabled"))
		pdp_config_option, _ := varStack["pdp_config_option"]
		pdp_topology_description_library_file, _ := varStack["pdp_topology_description_library_file"]
		tfb_dd_mode := env.GetKV("", "tfb_dd_mode")
		//lhc_period := env.GetKV("", "lhc_period")
		err := p.bookkeepingClient.UpdateRun(int32(runNumber64), state, timeO2Start, timeO2End, timeTrgStart, timeTrgEnd, trg_global_run_enabled, trg_enabled, pdp_config_option, pdp_topology_description_library_file, tfb_dd_mode /*, odc_topology_fullname, lhc_period */)
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
		var err error
		callFailedStr := "Bookkeeping UpdateRunStart call failed"

		rn := varStack["run_number"]
		runNumber64, err := strconv.ParseInt(rn, 10, 32)
		if err != nil {
			log.WithField("partition", envId).
				WithError(err).
				Error("cannot acquire run number for Bookkeeping UpdateRunStart")
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

		envState := env.CurrentState()
		if envState != "RUNNING" {
			return updateRunFunc(runNumber64, "bad", time.Now(), time.Now(), time.Time{}, time.Time{})
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

		envState := env.CurrentState()
		if envState != "CONFIGURED" {
			return updateRunFunc(runNumber64, "bad", time.Time{}, time.Time{}, time.Now(), time.Now())
		} else {
			return updateRunFunc(runNumber64, "good", time.Time{}, time.Time{}, time.Now(), time.Now())
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
		var err error
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

		trigger, ok := varStack["__call_trigger"]
		if !ok {
			log.WithField("call", call).
				WithField("partition", envId).
				Error("cannot acquire trigger from varStack in UpdateEnv")
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
		return
	}

	return
}

func (p *Plugin) Destroy() error {
	return nil
}
