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

// Package integration provides the plugin system for integrating O² Control
// with external services like DCS, Bookkeeping, ODC, and other ALICE systems.
package integration

import (
	"context"
	"sync"
	"time"

	"github.com/AliceO2Group/Control/common/logger"
	"github.com/AliceO2Group/Control/common/monitoring"
	"github.com/AliceO2Group/Control/common/utils/uid"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var log = logger.New(logrus.StandardLogger(), "integration")

var (
	once     sync.Once
	instance Plugins

	loaderOnce    sync.Once
	pluginLoaders map[string]func() Plugin
)

type Plugins []Plugin

type Plugin interface {
	GetName() string
	GetPrettyName() string
	GetEndpoint() string
	GetConnectionState() string
	GetData(argv []any) string
	GetEnvironmentsData(envIds []uid.ID) map[uid.ID]string
	GetEnvironmentsShortData(envIds []uid.ID) map[uid.ID]string

	Init(instanceId string) error
	CallStack(data interface{}) map[string]interface{}                                                // used in hook call context
	ObjectStack(varStack map[string]string, baseConfigStack map[string]string) map[string]interface{} // all other ProcessTemplates contexts
	Destroy() error
}

type NewFunc func(endpoint string) Plugin

func RegisteredPlugins() map[string]func() Plugin {
	return pluginLoaders
}

func RegisterPlugin(pluginName string, endpointArgumentName string, newFunc NewFunc) {
	loaderOnce.Do(func() {
		pluginLoaders = make(map[string]func() Plugin)
	})
	pluginLoaders[pluginName] = func() Plugin {
		if viper.IsSet(endpointArgumentName) {
			endpoint := viper.GetString(endpointArgumentName)
			return newFunc(endpoint)
		}
		return nil
	}
}

func (p Plugins) InitAll(fid string) {
	wg := &sync.WaitGroup{}
	wg.Add(len(p))
	for _, plugin := range p {
		go func(plugin Plugin) {
			defer wg.Done()
			initErr := plugin.Init(fid)
			if initErr != nil {
				log.WithError(initErr).
					WithField("plugin", plugin.GetName()).
					Error("workflow plugin failed to initialize")
			}
		}(plugin)
	}
	wg.Wait()
}

func (p Plugins) DestroyAll() {
	for _, plugin := range p {
		err := plugin.Destroy()
		if err != nil {
			log.WithError(err).
				WithField("plugin", plugin.GetName()).
				Error("workflow plugin failed to destroy")
		}
	}
}

func (p Plugins) CallStack(data interface{}) (stack map[string]interface{}) {
	stack = make(map[string]interface{})
	for _, plugin := range p {
		s := plugin.CallStack(data)
		stack[plugin.GetName()] = s
	}
	return
}

func (p Plugins) ObjectStack(varStack map[string]string, baseConfigStack map[string]string) (stack map[string]interface{}) {
	stack = make(map[string]interface{})

	// HACK: this is a dummy object+function to allow odc.GenerateEPNTopologyFullname in the root role
	stack["odc"] = map[string]interface{}{
		"GenerateEPNTopologyFullname": func() string {
			return ""
		},
	}

	for _, plugin := range p {
		s := plugin.ObjectStack(varStack, baseConfigStack)
		stack[plugin.GetName()] = s
	}
	return
}

func (p Plugins) GetData(argv []any) (data map[string]string) {
	data = make(map[string]string)
	for _, plugin := range p {
		data[plugin.GetName()] = plugin.GetData(argv)
	}
	return
}

func (p Plugins) GetEnvironmentsData(envIds []uid.ID) (data map[uid.ID]map[string]string) {
	data = make(map[uid.ID]map[string]string)

	// First we query each plugin for environment data of all envIds at once
	pluginEnvData := make(map[ /*plugin*/ string]map[uid.ID]string)
	for _, plugin := range p {
		pluginEnvData[plugin.GetName()] = plugin.GetEnvironmentsData(envIds)
	}

	// Then we invert the nested map to get a map of envId -> plugin -> data
	for _, plugin := range p {
		for envId, pluginData := range pluginEnvData[plugin.GetName()] {
			if _, ok := data[envId]; !ok {
				data[envId] = make(map[string]string)
			}
			data[envId][plugin.GetName()] = pluginData
		}
	}
	return
}

func (p Plugins) GetEnvironmentsShortData(envIds []uid.ID) (data map[uid.ID]map[string]string) {
	data = make(map[uid.ID]map[string]string)

	// First we query each plugin for environment data of all envIds at once
	pluginEnvData := make(map[ /*plugin*/ string]map[uid.ID]string)
	for _, plugin := range p {
		pluginEnvData[plugin.GetName()] = plugin.GetEnvironmentsShortData(envIds)
	}

	// Then we invert the nested map to get a map of envId -> plugin -> data
	for _, plugin := range p {
		for envId, pluginData := range pluginEnvData[plugin.GetName()] {
			if _, ok := data[envId]; !ok {
				data[envId] = make(map[string]string)
			}
			data[envId][plugin.GetName()] = pluginData
		}
	}
	return
}

func PluginsInstance() Plugins {
	once.Do(func() {
		instance = Plugins{}
		pluginList := viper.GetStringSlice("integrationPlugins")

		for _, pluginName := range pluginList {
			if pluginLoaders == nil {
				log.WithField("plugin", pluginName).
					Error("requested plugin unavailable")
				continue
			}
			pluginLoader, ok := pluginLoaders[pluginName]
			if !ok {
				log.WithField("plugin", pluginName).
					Error("requested plugin unavailable")
				continue
			}
			newPlugin := pluginLoader()
			if newPlugin == nil {
				log.WithField("plugin", pluginName).
					Error("plugin loader failed")
				continue
			}
			instance = append(instance, newPlugin)
		}
	})
	return instance
}

// Reset resets the plugin system for testing purposes.
func Reset() {
	once = sync.Once{}
	instance = Plugins{}
	loaderOnce = sync.Once{}
	pluginLoaders = make(map[string]func() Plugin)
}

func ExtractRunTypeOrUndefined(varStack map[string]string) string {
	runType, ok := varStack["run_type"]
	if !ok {
		runType = "undefined"
	}
	return runType
}

func NewContext(envId string, varStack map[string]string, timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(
		monitoring.AddEnvAndRunType(context.Background(),
			envId,
			ExtractRunTypeOrUndefined(varStack),
		),
		timeout)
}

func NewContextEmptyEnvIdRunType(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(
		monitoring.AddEnvAndRunType(context.Background(), "none", "none"),
		timeout)
}
