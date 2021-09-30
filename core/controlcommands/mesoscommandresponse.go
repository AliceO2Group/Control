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

	"github.com/rs/xid"
)

type MesosCommandResponse interface {
	GetCommandName() string
	GetCommandId() xid.ID
	GetResponseSenders() []MesosCommandTarget
	IsMultiResponse() bool
	Err() error
	Errors() map[MesosCommandTarget]error
}

type MesosCommandResponseBase struct {
	CommandName     string               `json:"name"`
	CommandId       xid.ID               `json:"id"`
	ErrorString     string               `json:"error"`
	MessageType     string               `json:"_messageType"`
	ResponseSenders []MesosCommandTarget `json:"-"`
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
		MessageType:       "MesosCommandResponse",
	}
}

func (m *MesosCommandResponseBase) GetCommandName() string {
	if m != nil {
		return m.CommandName
	}
	return ""
}

func (m *MesosCommandResponseBase) GetCommandId() xid.ID {
	if m != nil {
		return m.CommandId
	}
	return xid.NilID()
}

func (m *MesosCommandResponseBase) IsMultiResponse() bool {
	return false
}

func (m *MesosCommandResponseBase) GetResponseSenders() []MesosCommandTarget {
	if m != nil {
		return m.ResponseSenders
	}
	return nil
}

func (m *MesosCommandResponseBase) Err() error {
	if m != nil {
		if len(m.ErrorString) > 0 {
			return errors.New(m.ErrorString)
		}
		return nil
	}
	return errors.New("nil response")
}

// dummy implementation for single-origin responses which defaults to Err()
func (m *MesosCommandResponseBase) Errors() map[MesosCommandTarget]error {
	errMap := make(map[MesosCommandTarget]error)
	mct := MesosCommandTarget{}
	if len(m.ResponseSenders) > 0 {
		mct = m.ResponseSenders[0]
	}

	if m != nil {
		errMap[mct] = errors.New(m.ErrorString)
		return errMap
	}
	errMap[mct] = errors.New("nil response")
	return errMap
}
