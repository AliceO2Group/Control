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

import "github.com/AliceO2Group/Control/core/task/sm"

var (
	fairMqStateMap map[string]sm.State
)

func init() {
	// see https://github.com/FairRootGroup/FairMQ/blob/master/docs/images/device_states.svg
	fairMqStateMap = map[string]sm.State{
		"IDLE":    sm.STANDBY,
		"READY":   sm.CONFIGURED,
		"RUNNING": sm.RUNNING,
		"ERROR":   sm.ERROR,
		"EXITING": sm.DONE,

		"INITIALIZING DEVICE": sm.INVARIANT,
		"INITIALIZED":         sm.INVARIANT,
		"BINDING":             sm.INVARIANT,
		"BOUND":               sm.INVARIANT,
		"CONNECTING":          sm.INVARIANT,
		"DEVICE READY":        sm.INVARIANT,
		"INITIALIZING TASK":   sm.INVARIANT,
		"RESETTING TASK":      sm.INVARIANT,
		"RESETTING DEVICE":    sm.INVARIANT,
		"OK":                  sm.INVARIANT,
		"MIXED":               sm.INVARIANT,
	}
}

func toEcsState(fairMqState string) sm.State {
	if newEcsState, has := fairMqStateMap[fairMqState]; has {
		return newEcsState
	}
	return sm.UNKNOWN
}

func ToEcsState(fairMqState string, previousEcsState sm.State) sm.State {
	if ecsState := toEcsState(fairMqState); ecsState != sm.INVARIANT {
		return ecsState
	} else {
		return previousEcsState
	}
}
