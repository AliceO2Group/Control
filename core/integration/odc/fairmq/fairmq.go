/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2024 CERN and copyright holders of ALICE O².
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

package fairmq

var (
	fairMqStateMap map[string]string
)

func init() {
	// see https://github.com/FairRootGroup/FairMQ/blob/master/docs/images/device_states.svg
	fairMqStateMap = map[string]string{
		"IDLE":    "STANDBY",
		"READY":   "CONFIGURED",
		"RUNNING": "RUNNING",
		"ERROR":   "ERROR",
		"EXITING": "DONE",

		"INITIALIZING DEVICE":      "STANDBY",
		"INITIALIZED":              "STANDBY",
		"BINDING":                  "STANDBY",
		"BOUND":                    "STANDBY",
		"CONNECTING":               "STANDBY",
		"DEVICE READY CONFIGURING": "STANDBY", // needs special case
		"INITIALIZING TASK":        "STANDBY",

		"RESETTING TASK":         "CONFIGURED",
		"DEVICE READY RESETTING": "CONFIGURED", // needs special case
		"RESETTING DEVICE":       "CONFIGURED",
	}
}

func toEcsState(fairMqState string) string {
	if newEcsState, has := fairMqStateMap[fairMqState]; has {
		return newEcsState
	}
	return "UNKNOWN"
}

func ToEcsState(fairMqState, previousFairMqState string) string {
	odcStateToConvert := fairMqState
	// special case for DEVICE READY
	if odcStateToConvert == "DEVICE READY" {
		if previousFairMqState == "CONNECTING" ||
			previousFairMqState == "BOUND" ||
			previousFairMqState == "BINDING" ||
			previousFairMqState == "INITIALIZED" ||
			previousFairMqState == "INITIALIZING DEVICE" ||
			previousFairMqState == "IDLE" {
			odcStateToConvert += " CONFIGURING"
		} else if previousFairMqState == "RESETTING TASK" ||
			previousFairMqState == "READY" {
			odcStateToConvert += " RESETTING"
		} else {
			return "UNKNOWN"
		}
	}
	return toEcsState(odcStateToConvert)
}
