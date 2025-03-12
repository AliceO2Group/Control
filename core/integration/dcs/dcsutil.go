/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2022-2023 CERN and copyright holders of ALICE O².
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

package dcs

import (
	"fmt"
	"github.com/AliceO2Group/Control/common/logger/infologger"
	"github.com/AliceO2Group/Control/core/the"
	"strings"

	dcspb "github.com/AliceO2Group/Control/core/integration/dcs/protos"
	"github.com/sirupsen/logrus"
)

func resolveDefaults(detectorArgMap map[string]string, varStack map[string]string, ecsDetector string, theLog *logrus.Entry) map[string]string {
	// Do we have any default expressions for defaultable values?
	defaultableKeys := []string{"ddl_list"}

	for _, key := range defaultableKeys {
		if defaultableValue, ok := detectorArgMap[key]; ok {
			if strings.TrimSpace(strings.ToLower(defaultableValue)) == "default" { // if one of the defaultable keys has value `default`...
				defaultPayloadKey := strings.ToLower(ecsDetector) + "_default_" + key // e.g. tof_default_ddl_list
				if defaultPayload, ok := varStack[defaultPayloadKey]; ok {
					detectorArgMap[key] = defaultPayload
				} else {
					theLog.Warnf("requested default value for DCS parameter %s but no payload found at key %s: the string 'default' will be sent instead", key, defaultPayloadKey)
				}
			}
		}
	}
	return detectorArgMap
}

func addEnabledLinks(detectorArgMap map[string]string, varStack map[string]string, ecsDetector string, theLog *logrus.Entry) map[string]string {
	sendDetLinks, ok := varStack[strings.ToLower(ecsDetector)+"_dcs_send_enabled_links"]
	if !ok || sendDetLinks != "true" {
		return detectorArgMap
	}
	linkIDs, err := the.ConfSvc().GetAliasedLinkIDsForDetector(ecsDetector, true)
	if err != nil {
		theLog.WithError(err).Errorf("failed to get aliased link IDs for detector '%s'", ecsDetector)
		return detectorArgMap
	}
	linksJoined := strings.Join(linkIDs, ",")
	theLog.WithField(infologger.Level, infologger.IL_Devel).
		Infof("adding enabled link IDs for detector '%s' to DCS payload '%s'", ecsDetector, linksJoined)
	detectorArgMap["ddl_list"] = linksJoined

	return detectorArgMap
}

func ecsToDcsDetector(ecsDetector string) (dcspb.Detector, error) {
	var err error
	det := strings.ToUpper(strings.TrimSpace(ecsDetector))

	// Special case: if TST is present, we send AGD to DCS
	if det == "TST" {
		det = "AGD"
	}

	intDet, ok := dcspb.Detector_value[det]
	if !ok {
		err = fmt.Errorf("invalid DCS detector %s", det)
		log.WithError(err).Error("bad DCS detector entry")
		return dcspb.Detector_NULL_DETECTOR, err
	}

	return dcspb.Detector(intDet), nil
}

func dcsToEcsDetector(dcsDetector dcspb.Detector) string {
	det := dcsDetector.String()
	// Special case: if AGD is present, we send TST to ECS
	if det == "AGD" {
		det = "TST"
	}
	return det
}
