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

import (
	"errors"

	"github.com/pborman/uuid"
)

type MesosCommandResponse interface {
	GetCommandName() string
	GetCommandId() uuid.Array
	IsMultiResponse() bool
	Err() error
}

type MesosCommandResponseBase struct {
	CommandName string       `json:"name"`
	CommandId   uuid.Array   `json:"id"`
	ErrorString string       `json:"error"`
}

func NewMesosCommandResponse(mesosCommand MesosCommand, err error) (*MesosCommandResponseBase) {
	if mesosCommand == nil {
		log.Debug("trying to create MesosCommandResponseBase for nil MesosCommand, failing miserably")
		return nil
	}

	var errStr string
	if err == nil {
		errStr = ""
	} else {
		errStr = err.Error()
	}

	return &MesosCommandResponseBase{
		CommandName:       mesosCommand.GetName(),
		CommandId:         mesosCommand.GetId(),
		ErrorString:       errStr,
	}
}

func (m *MesosCommandResponseBase) GetCommandName() string {
	if m != nil {
		return m.CommandName
	}
	return ""
}

func (m *MesosCommandResponseBase) GetCommandId() uuid.Array {
	if m != nil {
		return m.CommandId
	}
	return uuid.NIL.Array()
}

func (m *MesosCommandResponseBase) IsMultiResponse() bool {
	return false
}

func (m *MesosCommandResponseBase) Err() error {
	if m != nil {
		return errors.New(m.ErrorString)
	}
	return errors.New("nil response")
}
