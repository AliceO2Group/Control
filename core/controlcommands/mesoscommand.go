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
	"github.com/AliceO2Group/Control/common/utils/uid"
	"github.com/mesos/mesos-go/api/v1/lib"
	"github.com/rs/xid"

	"time"
)

const (
	defaultResponseTimeout = 90 * time.Second
)

type MesosCommand interface {
	GetName() string
	GetId() xid.ID
	GetEnvironmentId() uid.ID
	IsMultiCmd() bool
	MakeSingleTarget(target MesosCommandTarget) MesosCommand
	IsMutator() bool
	GetResponseTimeout() time.Duration

	targets() []MesosCommandTarget
}

type MesosCommandTarget struct {
	AgentId    mesos.AgentID
	ExecutorId mesos.ExecutorID
	TaskId     mesos.TaskID
}

type PropertyMap map[string]string
type PropertyMapsMap map[MesosCommandTarget]PropertyMap

type MesosCommandBase struct {
	Name            string               `json:"name"`
	Id              xid.ID               `json:"id"`
	EnvironmentId   uid.ID               `json:"environmentId"`
	ResponseTimeout time.Duration        `json:"timeout"`
	Arguments       PropertyMap          `json:"arguments"`
	TargetList      []MesosCommandTarget `json:"targetList"`
	argMap          PropertyMapsMap      `json:"-"`
}

func NewMesosCommand(name string, envId uid.ID, receivers []MesosCommandTarget, argMap PropertyMapsMap) *MesosCommandBase {
	return &MesosCommandBase{
		Name:            name,
		Id:              xid.New(),
		EnvironmentId:   envId,
		ResponseTimeout: defaultResponseTimeout,
		TargetList:      receivers,
		argMap:          argMap,
	}
}

func (m *MesosCommandBase) GetName() string {
	if m != nil {
		return m.Name
	}
	return ""
}

func (m *MesosCommandBase) GetId() xid.ID {
	if m != nil {
		return m.Id
	}
	return xid.NilID()
}

func (m *MesosCommandBase) GetEnvironmentId() uid.ID {
	if m != nil {
		return m.EnvironmentId
	}
	return uid.NilID()
}

func (m *MesosCommandBase) IsMultiCmd() bool {
	if m != nil {
		return len(m.TargetList) > 1
	}
	return false
}

func (m *MesosCommandBase) MakeSingleTarget(receiver MesosCommandTarget) (cmd MesosCommand) {
	if m == nil {
		return
	}
	rcvOk := false
	for _, rcv := range m.TargetList {
		if rcv == receiver {
			rcvOk = true
			break
		}
	}
	if !rcvOk {
		return
	}

	argMap := make(PropertyMapsMap)
	if args, ok := m.argMap[receiver]; ok {
		argMap[receiver] = args
	} else {
		argMap[receiver] = make(PropertyMap)
	}

	cmd = &MesosCommandBase{
		Name:            m.Name,
		Id:              m.Id,
		EnvironmentId:   m.EnvironmentId,
		ResponseTimeout: m.ResponseTimeout,
		TargetList:      []MesosCommandTarget{receiver},
		argMap:          argMap,
		Arguments:       argMap[receiver],
	}
	return
}

func (m *MesosCommandBase) IsMutator() bool {
	return true
}

func (m *MesosCommandBase) GetResponseTimeout() time.Duration {
	if m != nil {
		return m.ResponseTimeout
	}
	return defaultResponseTimeout
}

func (m *MesosCommandBase) targets() []MesosCommandTarget {
	if m != nil {
		return m.TargetList
	}
	return nil
}
