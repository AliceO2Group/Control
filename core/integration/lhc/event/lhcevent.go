/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2025 CERN and copyright holders of ALICE O².
 * Author: Piotr Konopka <pkonopka@cern.ch>
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

package event

import (
	"github.com/AliceO2Group/Control/common/event"
	commonpb "github.com/AliceO2Group/Control/common/protos"
)

// BeamInfo mirrors (a subset of) the information described in the proto draft.
type BeamInfo struct {
	StableBeamsStart  int64             `json:"stableBeamsStart,omitempty"`
	StableBeamsEnd    int64             `json:"stableBeamsEnd,omitempty"`
	FillNumber        int32             `json:"fillNumber,omitempty"`
	FillingSchemeName string            `json:"fillingSchemeName,omitempty"`
	BeamType          string            `json:"beamType,omitempty"`
	BeamMode          commonpb.BeamMode `json:"beamMode,omitempty"`
}

type LhcStateChangeEvent struct {
	event.IntegratedServiceEventBase
	BeamInfo BeamInfo
}

func (e *LhcStateChangeEvent) GetName() string {
	return "LHC_STATE_CHANGE_EVENT"
}

func (e *LhcStateChangeEvent) GetBeamInfo() BeamInfo {
	if e == nil {
		return BeamInfo{}
	}
	return e.BeamInfo
}
