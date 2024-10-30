/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2020 CERN and copyright holders of ALICE O².
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

package channel

import (
	"errors"
	"strings"
)

type TransportType string

const (
	DEFAULT = TransportType("default")
	ZEROMQ  = TransportType("zeromq")
	NANOMSG = TransportType("nanomsg")
	SHMEM   = TransportType("shmem")
)

func (tr TransportType) String() string {
	return string(tr)
}

func (tr *TransportType) UnmarshalText(b []byte) error {
	str := strings.ToLower(strings.Trim(string(b), `"`))

	switch str {
	case "":
		*tr = DEFAULT
	case DEFAULT.String(), ZEROMQ.String(), NANOMSG.String(), SHMEM.String():
		*tr = TransportType(str)
	default:
		return errors.New("invalid transport type: " + str)
	}

	return nil
}

type AddressFormat string

const (
	TCP = AddressFormat("tcp")
	IPC = AddressFormat("ipc")
)

func (af AddressFormat) String() string {
	return string(af)
}

func (af *AddressFormat) UnmarshalText(b []byte) error {
	str := strings.ToLower(strings.Trim(string(b), `"`))

	switch str {
	case "":
		*af = TCP
	case TCP.String(), IPC.String():
		*af = AddressFormat(str)
	default:
		return errors.New("invalid address format: " + str)
	}

	return nil
}
