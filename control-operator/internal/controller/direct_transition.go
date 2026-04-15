/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2026 CERN and copyright holders of ALICE O².
 * Author: Michal Tichak <michal.tichak@cern.ch>
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

package controller

import (
	"fmt"
)

type State int

const (
	CONFIGURED = iota
	RUNNING
	STANDBY
	ERROR
)

type Transition int

const (
	GO_ERROR = iota
	RECOVER
	CONFIGURE
	RESET
	STOP
	START
	EXIT
)

type FromTo struct {
	from, to State
}

// How to handle EXIT?
var fromStatesToTransition = map[FromTo]Transition{
	{from: CONFIGURED, to: ERROR}:   GO_ERROR,
	{from: CONFIGURED, to: RUNNING}: START,
	{from: CONFIGURED, to: STANDBY}: RESET,
	{from: ERROR, to: STANDBY}:      RECOVER,
	{from: RUNNING, to: CONFIGURED}: STOP,
	{from: RUNNING, to: ERROR}:      GO_ERROR,
	{from: STANDBY, to: CONFIGURED}: CONFIGURE,
}

func FromStatesToTransition(from, to State) (Transition, error) {
	transition, hasValue := fromStatesToTransition[FromTo{from: from, to: to}]
	if !hasValue {
		return 0, fmt.Errorf("failed to find transition from %s, to %s", from, to)
	}
	return transition, nil
}

func (s State) String() string {
	switch s {
	case CONFIGURED:
		return "configured"
	case RUNNING:
		return "running"
	case STANDBY:
		return "standby"
	case ERROR:
		return "error"
	default:
		return fmt.Sprintf("State(%d)", s)
	}
}

func StateFromString(s string) (State, error) {
	switch s {
	case "configured":
		return CONFIGURED, nil
	case "running":
		return RUNNING, nil
	case "standby":
		return STANDBY, nil
	case "error":
		return ERROR, nil
	default:
		return 0, fmt.Errorf("invalid State: %s", s)
	}
}

func (t Transition) String() string {
	switch t {
	case GO_ERROR:
		return "go_error"
	case RECOVER:
		return "recover"
	case CONFIGURE:
		return "configure"
	case RESET:
		return "reset"
	case STOP:
		return "stop"
	case START:
		return "start"
	case EXIT:
		return "exit"
	default:
		return fmt.Sprintf("Transition(%d)", t)
	}
}

func TransitionFromString(s string) (Transition, error) {
	switch s {
	case "go_error":
		return GO_ERROR, nil
	case "recover":
		return RECOVER, nil
	case "configure":
		return CONFIGURE, nil
	case "reset":
		return RESET, nil
	case "stop":
		return STOP, nil
	case "start":
		return START, nil
	case "exit":
		return EXIT, nil
	default:
		return 0, fmt.Errorf("invalid Transition: %s", s)
	}
}
