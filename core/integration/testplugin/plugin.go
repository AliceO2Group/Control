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

	"github.com/AliceO2Group/Control/common/logger/infologger"
	"github.com/AliceO2Group/Control/common/utils/uid"
	"github.com/AliceO2Group/Control/core/integration"
	"github.com/AliceO2Group/Control/core/workflow/callable"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

const (
	TESTPLUGIN_GENERAL_OP_TIMEOUT = 15 * time.Second
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

func (p *Plugin) GetData(_ []uid.ID) string {
	return ""
}

func (p *Plugin) Init(_ string) error {
	log.Debug("Test plugin initialized")
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

	message, ok := varStack["testplugin_message"]
	if !ok {
		log.Warn("cannot acquire testplugin message")
		message = "running testplugin.Noop"
		return
	}

	stack = make(map[string]interface{})
	stack["Noop"] = func() (out string) {	// must formally return string even when we return nothing
		log.WithField("partition", envId).
			WithField("level", infologger.IL_Ops).
			WithField("rolepath", call.GetParentRolePath()).
			WithField("trigger", call.GetTraits().Trigger).
			WithField("await", call.GetTraits().Await).
			Infof("executing testplugin.Noop call: %s", message)

		rn := varStack["run_number"]
		var (
			runNumber64 int64
			err error
		)
		runNumber64, err = strconv.ParseInt(rn, 10, 32)
		if err != nil {
			log.WithField("partition", envId).
				WithError(err).
				Error("cannot acquire run number for testplugin.Noop")
		}

		timeout := callable.AcquireTimeout(TESTPLUGIN_GENERAL_OP_TIMEOUT, varStack, "Noop", envId)
		defer log.WithField("partition", envId).
			WithField("level", infologger.IL_Ops).
			WithField("rolepath", call.GetParentRolePath()).
			WithField("trigger", call.GetTraits().Trigger).
			WithField("await", call.GetTraits().Await).
			WithField("runNumber", runNumber64).
			Infof("executed testplugin.Noop call in %s", timeout)

		time.Sleep(timeout)

		return
	}

	return
}

func (p *Plugin) Destroy() error {
	return nil
}
