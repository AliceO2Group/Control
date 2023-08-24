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

package testplugin

import (
	"strconv"
	"time"

	"github.com/AliceO2Group/Control/common/logger"
	"github.com/AliceO2Group/Control/common/logger/infologger"
	"github.com/AliceO2Group/Control/common/utils/uid"
	"github.com/AliceO2Group/Control/core/integration"
	"github.com/AliceO2Group/Control/core/workflow/callable"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var log = logger.New(logrus.StandardLogger(), "testplugin")

const (
	TESTPLUGIN_GENERAL_OP_TIMEOUT = 3 * time.Second
)

type Plugin struct {
}

func NewPlugin(endpoint string) integration.Plugin {
	log.WithField("endpoint", endpoint).
		Info("initializing test plugin, DO NOT USE IN PRODUCTION")

	return &Plugin{}
}

func (p *Plugin) GetName() string {
	return "testplugin"
}

func (p *Plugin) GetPrettyName() string {
	return "Test plugin"
}

func (p *Plugin) GetEndpoint() string {
	return viper.GetString("testPluginEndpoint")
}

func (p *Plugin) GetConnectionState() string {
	return "READY"
}

func (p *Plugin) GetData(_ []any) string {
	return ""
}

func (p *Plugin) GetEnvironmentsData(_ []uid.ID) map[uid.ID]string {
	return nil
}

func (p *Plugin) GetEnvironmentsShortData(_ []uid.ID) map[uid.ID]string {
	return nil
}

func (p *Plugin) Init(_ string) error {
	log.Debug("Test plugin initialized")
	return nil
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

	message, ok := varStack["testplugin_message"]
	if !ok {
		message = "running testplugin.Noop"
	}

	doFailS, ok := varStack["testplugin_fail"]
	if !ok {
		doFailS = "false"
	}
	doFail, convErr := strconv.ParseBool(doFailS)
	if convErr != nil {
		doFail = false
	}

	stack = make(map[string]interface{})
	stack["Noop"] = func() (out string) { // must formally return string even when we return nothing
		log.WithField("partition", envId).
			WithField("level", infologger.IL_Ops).
			WithField("rolepath", call.GetParentRolePath()).
			WithField("trigger", call.GetTraits().Trigger).
			WithField("await", call.GetTraits().Await).
			Infof("executing testplugin.Noop call: %s", message)

		rn := varStack["run_number"]
		var (
			runNumber64 int64
			err         error
		)
		runNumber64, err = strconv.ParseInt(rn, 10, 32)
		if err != nil {
			runNumber64 = 0
		}

		timeout := callable.AcquireTimeout(TESTPLUGIN_GENERAL_OP_TIMEOUT, varStack, "Noop", envId)
		defer log.WithField("partition", envId).
			WithField("level", infologger.IL_Ops).
			WithField("rolepath", call.GetParentRolePath()).
			WithField("trigger", call.GetTraits().Trigger).
			WithField("await", call.GetTraits().Await).
			WithField("run", runNumber64).
			Infof("executed testplugin.Noop call in %s", timeout)

		time.Sleep(timeout)

		return
	}
	stack["Test"] = func() (out string) { // must formally return string even when we return nothing
		log.WithField("partition", envId).
			WithField("level", infologger.IL_Ops).
			WithField("rolepath", call.GetParentRolePath()).
			WithField("trigger", call.GetTraits().Trigger).
			WithField("await", call.GetTraits().Await).
			Infof("executing testplugin.Test call: %s", message)

		rn := varStack["run_number"]
		var (
			runNumber64 int64
			err         error
		)
		runNumber64, err = strconv.ParseInt(rn, 10, 32)
		if err != nil {
			runNumber64 = 0
		}

		timeout := callable.AcquireTimeout(TESTPLUGIN_GENERAL_OP_TIMEOUT, varStack, "Test", envId)
		defer log.WithField("partition", envId).
			WithField("level", infologger.IL_Ops).
			WithField("rolepath", call.GetParentRolePath()).
			WithField("trigger", call.GetTraits().Trigger).
			WithField("await", call.GetTraits().Await).
			WithField("run", runNumber64).
			Infof("executed testplugin.Test call in %s", timeout)

		time.Sleep(timeout)
		if doFail {
			call.VarStack["__call_error"] = "error triggered in testplugin.Test call"
		}

		return
	}

	return
}

func (p *Plugin) Destroy() error {
	return nil
}
