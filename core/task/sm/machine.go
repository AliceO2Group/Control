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

// Package sm provides state machine functionality for task lifecycle management,
// including state transitions and event handling.
package sm

type Transition struct {
	Evt  Event
	Src  State
	Dst  State
	Args EventArgs
}

type Event string

const (
	CONFIGURE = Event("CONFIGURE")
	RESET     = Event("RESET")
	START     = Event("START")
	STOP      = Event("STOP")
	EXIT      = Event("EXIT")
	GO_ERROR  = Event("GO_ERROR")
	RECOVER   = Event("RECOVER")
)

func (e Event) String() string {
	return string(e)
}

type EventArgs map[string]string
