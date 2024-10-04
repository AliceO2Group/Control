/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2020-2021 CERN and copyright holders of ALICE O².
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

package apricot

import (
	"fmt"
	"net/url"
	"sync"

	"github.com/AliceO2Group/Control/apricot/cacheproxy"
	"github.com/AliceO2Group/Control/apricot/local"
	"github.com/AliceO2Group/Control/apricot/remote"
	"github.com/AliceO2Group/Control/common/logger/infologger"
	"github.com/AliceO2Group/Control/configuration"
	"github.com/spf13/viper"
)

var (
	once     sync.Once
	instance configuration.Service
)

func newService(configUri string) (configuration.Service, error) {
	parsedUri, err := url.Parse(configUri)
	if err != nil {
		return nil, err
	}

	var svc configuration.Service

	switch parsedUri.Scheme {
	case "consul":
		fallthrough
	case "file":
		if viper.GetString("component") == "apricot" {
			log.WithField("configUri", configUri).
				Debug("new embedded apricot instance")
		} else if viper.GetString("component") == "coconut" {
			log.WithField("config_endpoint", configUri).
				Debug("new embedded apricot instance")
		} else {
			log.WithField("configUri", configUri).
				WithField("level", infologger.IL_Support).
				Info("new embedded apricot instance")
		}
		svc, err = local.NewService(configUri)
		if err != nil {
			return svc, err
		}
		if viper.GetBool("configCache") {
			svc, err = cacheproxy.NewService(svc)
		}
		return svc, err
	case "apricot":
		if viper.GetString("component") == "apricot" {
			log.WithField("configUri", configUri).
				Warn("apricot proxy mode")
		} else if viper.GetString("component") == "coconut" {
			log.WithField("config_endpoint", configUri).
				Debug("new apricot client")
		} else {
			log.WithField("configUri", configUri).
				WithField("level", infologger.IL_Support).
				Info("new apricot client")
		}
		svc, err = remote.NewService(configUri)
		if err != nil {
			return svc, err
		}
		if viper.GetBool("configCache") {
			svc, err = cacheproxy.NewService(svc)
		}
		return svc, err
	case "mock":
		svc, err = local.NewService(configUri)
		if err != nil {
			return svc, err
		}
		return svc, err
	default:
		return nil, fmt.Errorf("invalid configuration URI scheme %s", parsedUri.Scheme)
	}
}

func Instance() configuration.Service {
	once.Do(func() {
		var (
			err       error
			configUri string
		)
		if viper.IsSet("config_endpoint") { //coconut
			configUri = viper.GetString("config_endpoint")
		} else if viper.IsSet("configServiceUri") { //core
			configUri = viper.GetString("configServiceUri")
		} else { //apricot
			configUri = viper.GetString("backendUri")
		}
		instance, err = newService(configUri)
		if err != nil {
			log.WithField("level", infologger.IL_Support).WithField("configServiceUri", configUri).Fatal("bad configuration URI")
		}
	})
	return instance
}
