/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2020 CERN and copyright holders of ALICE O².
 * Author: Miltiadis Alexis <miltiadis.alexis@cern.ch>
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

package taskop

import (
	"errors"
	"fmt"
	"strings"
)

type MessageType int

const (
	AcquireTasks MessageType = iota
	ConfigureTasks
	TransitionTasks
	ReleaseTasks
	KillTasks
	TaskStatusMessage
	TaskStateMessage
	Error
)

func (mt MessageType) String() string {
	return [...]string{"AcquireTasks", "ConfigureTasks", "TransitionTasks", "MesosEvent", "ReleaseTasks", "KillTasks", "TaskStatusMessage", "TaskStateMessage", "Error"}[mt]
}

func (mt *MessageType) UnmarshalJSON(b []byte) error {
	return mt.UnmarshalText(b)
}

func (mt *MessageType) UnmarshalText(b []byte) error {
	str := strings.Trim(string(b), `"`)

	switch str {
	case "AcquireTasks":
		*mt = AcquireTasks
	case "ConfigureTasks":
		*mt = ConfigureTasks
	case "TransitionTasks":
		*mt = TransitionTasks
	case "ReleaseTasks":
		*mt = ReleaseTasks
	case "KillTasks":
		*mt = KillTasks
	case "TaskStatusMessage":
		*mt = TaskStatusMessage
	case "TaskStateMessage":
		*mt = TaskStateMessage
	case "Error":
		*mt = Error
	}

	return nil
}

func (mt *MessageType) MarshalJSON() (text []byte, err error) {
	text, err = mt.MarshalText()
	return []byte(fmt.Sprintf("\"%s\"", text)), err
}

func (mt *MessageType) MarshalText() (text []byte, err error) {
	if mt == nil {
		return []byte{}, errors.New("cannot marshal nil MessageType")
	}

	return []byte(mt.String()), nil
}
