/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2019 CERN and copyright holders of ALICE O².
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

package executable

import (
	"github.com/AliceO2Group/Control/core/controlcommands"
	"github.com/AliceO2Group/Control/executor/executorcmd"
)

type BasicTask struct {
	taskBase
}

func (b *BasicTask) Launch() error {
	panic("implement me")
}

func (b *BasicTask) Kill() error {
	panic("implement me")
}

func (b *BasicTask) Transition(transition *executorcmd.ExecutorCommand_Transition) *controlcommands.MesosCommandResponse_Transition {
	panic("implement me")
}

func (b *BasicTask) UnmarshalTransition([]byte) (*executorcmd.ExecutorCommand_Transition, error) {
	panic("implement me")
}

type HookTask struct {
	taskBase
}

