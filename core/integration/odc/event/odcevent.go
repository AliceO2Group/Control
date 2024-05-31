/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2023 CERN and copyright holders of ALICE O².
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

package event

import (
	"github.com/AliceO2Group/Control/common/event"
	"github.com/AliceO2Group/Control/common/utils/uid"
)

type OdcPartitionStateChangeEvent struct {
	event.IntegratedServiceEventBase
	EnvironmentId uid.ID `json:"serviceName"`
	State         string `json:"state"`
	EcsState      string `json:"ecsState"`
}

func (e *OdcPartitionStateChangeEvent) GetName() string {
	return "ODC_PARTITION_STATE_CHANGE_EVENT"
}

func (e *OdcPartitionStateChangeEvent) GetEnvironmentId() uid.ID {
	if e == nil {
		return ""
	}
	return e.EnvironmentId
}

func (e *OdcPartitionStateChangeEvent) GetState() string {
	if e == nil {
		return ""
	}
	return e.State
}
