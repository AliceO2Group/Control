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

package task

type Status uint8

const (
	UNDEFINED = iota
	INACTIVE
	PARTIAL
	ACTIVE
	UNDEPLOYABLE
)

var (
	STATUS_PRODUCT = map[Status]map[Status]Status{
		UNDEFINED: {
			UNDEFINED: UNDEFINED,
			INACTIVE:  UNDEFINED,
			PARTIAL:   UNDEFINED,
			ACTIVE:    UNDEFINED,
		},
		INACTIVE: {
			UNDEFINED: UNDEFINED,
			INACTIVE:  INACTIVE,
			PARTIAL:   PARTIAL,
			ACTIVE:    PARTIAL,
		},
		PARTIAL: {
			UNDEFINED: UNDEFINED,
			INACTIVE:  PARTIAL,
			PARTIAL:   PARTIAL,
			ACTIVE:    PARTIAL,
		},
		ACTIVE: {
			UNDEFINED: UNDEFINED,
			INACTIVE:  PARTIAL,
			PARTIAL:   PARTIAL,
			ACTIVE:    ACTIVE,
		},
		UNDEPLOYABLE: {
			UNDEFINED:    UNDEFINED,
			INACTIVE:     UNDEPLOYABLE,
			PARTIAL:      UNDEPLOYABLE,
			ACTIVE:       UNDEPLOYABLE,
			UNDEPLOYABLE: UNDEPLOYABLE,
		},
	}
)

func (s Status) String() string {
	names := []string{
		"UNDEFINED",
		"INACTIVE",
		"PARTIAL",
		"ACTIVE",
		"UNDEPLOYABLE",
	}
	if s > UNDEPLOYABLE {
		return "UNDEFINED"
	}
	return names[s]
}

func (s Status) X(other Status) Status {
	return STATUS_PRODUCT[s][other]
}
