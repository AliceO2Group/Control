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


package environment

import (
	"errors"
	"github.com/AliceO2Group/Control/core/protos"
	"github.com/AliceO2Group/Control/core/task"
)

type Transition interface {
	eventName() string
	check() error
	do(*Environment) error
}

func MakeTransition(taskman *task.ManagerV2, optype pb.ControlEnvironmentRequest_Optype) Transition {
	switch optype {
	case pb.ControlEnvironmentRequest_CONFIGURE:
		return NewConfigureTransition(taskman, nil, nil, true)
	case pb.ControlEnvironmentRequest_START_ACTIVITY:
		return NewStartActivityTransition(taskman)
	case pb.ControlEnvironmentRequest_STOP_ACTIVITY:
		return NewStopActivityTransition(taskman)
	case pb.ControlEnvironmentRequest_RESET:
		return NewResetTransition(taskman)
	case pb.ControlEnvironmentRequest_GO_ERROR:
		fallthrough
	case pb.ControlEnvironmentRequest_NOOP:
		fallthrough
	default:
		return nil
	}
	return nil
}

type baseTransition struct {
	taskman         *task.ManagerV2
	name            string
}

func (t baseTransition) check() (err error) {
	if t.taskman == nil {
		err = errors.New("cannot configure environment with nil roleman")
	}
	return
}

func (t baseTransition) eventName() string {
	return t.name
}

