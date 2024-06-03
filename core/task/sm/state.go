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

package sm

type State int

const (
	UNKNOWN State = iota
	STANDBY
	CONFIGURED
	RUNNING
	ERROR
	DONE
	MIXED
	INVARIANT
)

var _names = []string{
	"UNKNOWN",
	"STANDBY",
	"CONFIGURED",
	"RUNNING",
	"ERROR",
	"DONE",
	"MIXED",
	"INVARIANT",
}

func (s State) String() string {
	if s > MIXED {
		return "UNKNOWN"
	}
	return _names[s]
}

func StateFromString(s string) State {
	for i, v := range _names {
		if s == v {
			return State(i)
		}
	}
	return UNKNOWN
}

func (s State) X(other State) State {
	if s == other {
		return s
	}
	if s == ERROR || other == ERROR {
		return ERROR
	}
	if s != other {
		if s == INVARIANT {
			return other
		}
		if other == INVARIANT {
			return s
		}
	}
	return MIXED
}
