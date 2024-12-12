/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2020-2022 CERN and copyright holders of ALICE O².
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

package template

import (
	"encoding/json"
	"fmt"
	"github.com/AliceO2Group/Control/common/logger/infologger"
	texttemplate "text/template"
	"time"

	"github.com/AliceO2Group/Control/common/utils"
	"github.com/AliceO2Group/Control/configuration/componentcfg"
)

func getConfigLegacy(confSvc ConfigurationService, varStack map[string]string, path string) string {
	defer utils.TimeTrack(time.Now(), "GetConfigLegacy", log.WithPrefix("template"))
	query, err := componentcfg.NewQuery(path)
	if err != nil {
		return fmt.Sprintf("{\"error\":\"%s\"}", err.Error())
	}

	payload, err := confSvc.GetComponentConfiguration(query)
	if err != nil {
		log.WithError(err).
			WithField("path", query.Path()).
			Warn("failed to get component configuration")
		return fmt.Sprintf("{\"error\":\"%s\"}", err.Error())
	}

	fields := Fields{WrapPointer(&payload)}
	err = fields.Execute(confSvc, query.Path(), varStack, nil, nil, make(map[string]texttemplate.Template), nil)
	log.WithField("level", infologger.IL_Devel).Debug(varStack)
	log.WithField("level", infologger.IL_Devel).Debug(payload)
	return payload
}

func getConfig(confSvc ConfigurationService, varStack map[string]string, path string) string {
	defer utils.TimeTrack(time.Now(), "GetConfig", log.WithPrefix("template"))
	query, err := componentcfg.NewQuery(path)
	if err != nil {
		return fmt.Sprintf("{\"error\":\"%s\"}", err.Error())
	}

	payload, err := confSvc.GetAndProcessComponentConfiguration(query, varStack)
	if err != nil {
		return fmt.Sprintf("{\"error\":\"%s\"}", err.Error())
	}

	log.WithField("level", infologger.IL_Devel).Debug(varStack)
	log.WithField("level", infologger.IL_Devel).Debug(payload)
	return payload
}

func resolveConfig(confSvc ConfigurationService, query *componentcfg.Query) string {
	defer utils.TimeTrack(time.Now(), "ResolveConfig", log.WithPrefix("template"))
	var resolved *componentcfg.Query
	resolved, err := confSvc.ResolveComponentQuery(query)
	if err != nil {
		return fmt.Sprintf("{\"error\":\"%s\"}", err.Error())
	}

	return resolved.Raw()
}

func resolveConfigPath(confSvc ConfigurationService, path string) string {
	defer utils.TimeTrack(time.Now(), "ResolveConfigPath", log.WithPrefix("template"))
	query, err := componentcfg.NewQuery(path)
	if err != nil {
		return fmt.Sprintf("{\"error\":\"%s\"}", err.Error())
	}

	var resolved *componentcfg.Query
	resolved, err = confSvc.ResolveComponentQuery(query)
	if err != nil {
		return fmt.Sprintf("{\"error\":\"%s\"}", err.Error())
	}

	return resolved.Raw()
}

func detectorForHost(confSvc ConfigurationService, hostname string) string {
	payload, err := confSvc.GetDetectorForHost(hostname)
	if err != nil {
		return fmt.Sprintf("{\"error\":\"%s\"}", err.Error())
	}
	return payload
}

func detectorsForHosts(confSvc ConfigurationService, hosts string) string {
	defer utils.TimeTrack(time.Now(), "DetectorsForHosts", log.WithPrefix("template"))
	hostsSlice := make([]string, 0)
	// first we convert the incoming string treated as JSON list into a []string
	bytes := []byte(hosts)
	err := json.Unmarshal(bytes, &hostsSlice)
	if err != nil {
		return fmt.Sprintf("{\"error\":\"DetectorsForHosts function: %s\"}", err.Error())
	}

	payload, err := confSvc.GetDetectorsForHosts(hostsSlice)
	if err != nil {
		return fmt.Sprintf("{\"error\":\"%s\"}", err.Error())
	}

	// and back to JSON list for the active detectors slice
	bytes, err = json.Marshal(payload)
	if err != nil {
		return fmt.Sprintf("{\"error\":\"DetectorsForHosts function: %s\"}", err.Error())
	}
	outString := string(bytes)

	return outString
}

func cruCardsForHost(confSvc ConfigurationService, hostname string) string {
	defer utils.TimeTrack(time.Now(), "CRUCardsForHost", log.WithPrefix("template"))
	payload, err := confSvc.GetCRUCardsForHost(hostname)
	if err != nil {
		return fmt.Sprintf("[\"error: %s\"]", err.Error())
	}
	return payload
}

func endpointsForCruCard(confSvc ConfigurationService, hostname string, cardSerial string) string {
	defer utils.TimeTrack(time.Now(), "EndpointsForCRUCard", log.WithPrefix("template"))
	payload, err := confSvc.GetEndpointsForCRUCard(hostname, cardSerial)
	if err != nil {
		return fmt.Sprintf("{\"error\":\"%s\"}", err.Error())
	}
	return payload
}

func getRuntimeConfig(confSvc ConfigurationService, component string, key string) string {
	defer utils.TimeTrack(time.Now(), "GetRuntimeConfig", log.WithPrefix("template"))

	payload, err := confSvc.GetRuntimeEntry(component, key)
	if err != nil {
		return fmt.Sprintf("{\"error\":\"%s\"}", err.Error())
	}

	return payload
}

func setRuntimeConfig(confSvc ConfigurationService, component, key, value string) string {
	defer utils.TimeTrack(time.Now(), "SetRuntimeConfig", log.WithPrefix("template"))

	err := confSvc.SetRuntimeEntry(component, key, value)
	if err != nil {
		return fmt.Sprintf("{\"error\":\"%s\"}", err.Error())
	}

	return value
}
