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
	"github.com/mesos/mesos-go/api/v1/lib"
)

type MesosCommand interface {
	GetName() string
	IsMultiCmd() bool
	IsMutator() bool

	receivers() []MesosCommandReceiver
}

type MesosCommandReceiver struct {
	AgentId      mesos.AgentID
	ExecutorId   mesos.ExecutorID
}

type MesosCommandBase struct {
	Name         string                    `json:"name"`
	rcvList      []MesosCommandReceiver    `json:"-"`
}

func NewMesosCommand(name string, receivers []MesosCommandReceiver) (*MesosCommandBase) {
	return &MesosCommandBase{
		Name: name,
		rcvList: receivers,
	}
}

func (m *MesosCommandBase) GetName() string {
	if m != nil {
		return m.Name
	}
	return ""
}

func (m *MesosCommandBase) IsMultiCmd() bool {
	if m != nil {
		return len(m.rcvList) > 1
	}
	return false
}

func (m *MesosCommandBase) IsMutator() bool {
	return true
}

func (m *MesosCommandBase) receivers() []MesosCommandReceiver {
	if m != nil {
		return m.rcvList
	}
	return nil
}