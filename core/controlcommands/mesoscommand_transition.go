/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2018 CERN and copyright holders of ALICE O².
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

type MesosCommand_Transition struct {
	MesosCommandBase

	Source        string                `json:"source"`
	Event         string                `json:"event"`
	Destination   string                `json:"destination"`
}

func (m *MesosCommand_Transition) MakeSingleTarget(target MesosCommandTarget) (cmd MesosCommand) {
	if m == nil {
		return
	}
	mc := m.MesosCommandBase.MakeSingleTarget(target)
	mcb, ok := mc.(*MesosCommandBase)
	if !ok {
		return
	}

	cmd = &MesosCommand_Transition{
		MesosCommandBase: *mcb,
		Source:           m.Source,
		Event:            m.Event,
		Destination:      m.Destination,
	}
	return
}

func NewMesosCommand_Transition(receivers []MesosCommandTarget, source string, event string, destination string, arguments PropertyMapsMap) (*MesosCommand_Transition) {
	return &MesosCommand_Transition{
		MesosCommandBase: *NewMesosCommand("MesosCommand_Transition", receivers, arguments),
		Source:           source,
		Event:            event,
		Destination:      destination,
	}
}

type MesosCommandResponse_Transition struct {
	MesosCommandResponseBase

	CurrentState string `json:"state"`
}

func NewMesosCommandResponse_Transition(mesosCommand *MesosCommand_Transition, err error, currentState string) *MesosCommandResponse_Transition {
	return &MesosCommandResponse_Transition{
		MesosCommandResponseBase: *NewMesosCommandResponse(mesosCommand, err),
		CurrentState:             currentState,
	}
}