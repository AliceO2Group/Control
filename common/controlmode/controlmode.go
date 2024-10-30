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

// Package controlmode contains some enums for switching between
// executor process control modes.
package controlmode

import (
	"errors"
	"fmt"
	"strings"
)

type ControlMode int

const (
	DIRECT ControlMode = iota
	FAIRMQ
	BASIC
	HOOK
)

func (cm ControlMode) String() string {
	switch cm {
	case DIRECT:
		return "direct"
	case FAIRMQ:
		return "fairmq"
	case BASIC:
		return "basic"
	case HOOK:
		return "hook"
	}
	return "direct"
}

func (cm *ControlMode) UnmarshalJSON(b []byte) error {
	return cm.UnmarshalText(b)
}

func (cm *ControlMode) UnmarshalText(b []byte) error {
	str := strings.ToLower(strings.Trim(string(b), `"`))

	switch str {
	case "direct":
		*cm = DIRECT
	case "fairmq":
		*cm = FAIRMQ
	case "basic":
		*cm = BASIC
	case "hook":
		*cm = HOOK
	default:
		*cm = DIRECT
	}

	return nil
}

func (cm *ControlMode) MarshalJSON() (text []byte, err error) {
	text, err = cm.MarshalText()
	return []byte(fmt.Sprintf("\"%s\"", text)), err
}

func (cm *ControlMode) MarshalText() (text []byte, err error) {
	if cm == nil {
		return []byte{}, errors.New("cannot marshal nil ControlMode")
	}

	return []byte(cm.String()), nil
}
