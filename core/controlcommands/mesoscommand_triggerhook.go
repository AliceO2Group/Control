/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2020 CERN and copyright holders of ALICE O².
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

package controlcommands

import "github.com/AliceO2Group/Control/common/utils/uid"

type MesosCommand_TriggerHook struct {
	MesosCommandBase
}

func (m *MesosCommand_TriggerHook) MakeSingleTarget(target MesosCommandTarget) (cmd MesosCommand) {
	if m == nil {
		return
	}
	mc := m.MesosCommandBase.MakeSingleTarget(target)
	mcb, ok := mc.(*MesosCommandBase)
	if !ok {
		return
	}

	cmd = &MesosCommand_TriggerHook{
		MesosCommandBase: *mcb,
	}
	return
}

func NewMesosCommand_TriggerHook(envId uid.ID, receivers []MesosCommandTarget) *MesosCommand_TriggerHook {
	return &MesosCommand_TriggerHook{
		MesosCommandBase: *NewMesosCommand("MesosCommand_TriggerHook", envId, receivers, PropertyMapsMap{}),
	}
}

type MesosCommandResponse_TriggerHook struct {
	MesosCommandResponseBase
	TaskId string `json:"taskId"`
}

func NewMesosCommandResponse_TriggerHook(mesosCommand *MesosCommand_TriggerHook, err error, taskId string) *MesosCommandResponse_TriggerHook {
	return &MesosCommandResponse_TriggerHook{
		MesosCommandResponseBase: *NewMesosCommandResponse(mesosCommand, err),
		TaskId:                   taskId,
	}
}
