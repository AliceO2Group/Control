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

// Package transitioner defines the Transitioner interface, as well as
// its implementations in order to translate between internal O² state
// machine states and events, and the states and events of some other
// process state machine (such as the FairMQ Device state machine).
package transitioner

import (
	"github.com/AliceO2Group/Control/common/controlmode"
	"github.com/AliceO2Group/Control/common/logger"
	"github.com/sirupsen/logrus"
)

var log = logger.New(logrus.StandardLogger(), "executorcontrol")

type EventInfo struct {
	Evt  string
	Src  string
	Dst  string
	Args map[string]string
}

type Transitioner interface {
	Commit(evt string, src string, dst string, args map[string]string) (finalState string, err error)
	FromDeviceState(state string) string
}

func NewTransitioner(cm controlmode.ControlMode, transitionFunc DoTransitionFunc) Transitioner {
	switch cm {
	case controlmode.FAIRMQ:
		return NewFairMQTransitioner(transitionFunc)
	case controlmode.BASIC:
		fallthrough
	case controlmode.HOOK:
		fallthrough
	case controlmode.DIRECT:
		fallthrough
	default:
		return NewDirectTransitioner(transitionFunc)
	}
}

type DoTransitionFunc func(ei EventInfo) (newState string, err error)
