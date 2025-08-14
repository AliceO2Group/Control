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

// Package uid provides unique identifier generation functionality,
// including machine-specific and time-based ID generation utilities.
package uid

import (
	"encoding/binary"
	"encoding/json"
	"time"

	"github.com/AliceO2Group/Control/common/logger"
	"github.com/denisbrodbeck/machineid"
	"github.com/osamingo/indigo"
	"github.com/pborman/uuid"
	"github.com/rs/xid"
	"github.com/sirupsen/logrus"
)

type ID string

var (
	log    = logger.New(logrus.StandardLogger(), "utils")
	uidGen *indigo.Generator
)

func init() {
	// In order to correctly seed ID generation and ensure that all generated IDs
	// are reasonably unique to a given machine, we need to provide a uint16
	// that represents the current machine.
	// By default, Sonyflake/Indigo attempt to acquire such an ID from the machine's
	// private IP address, but this doesn't always work (e.g. on some Docker
	// instances).
	// So we use denisbrodbeck/machineid to read in the standard machine-id in a
	// cross-platform way. On CentOS and similar Linuxes, this means that we read
	// the file /etc/machine-id.
	// This file contains a standard UUID as string, so we need to parse it into
	// a []byte, and fetch 2 of these bytes to generate the uint16 ID. If this
	// fails, the uint16 ID defaults to 42.
	var machineId uint16 = 42

	id, err := machineid.ID()
	if err == nil {
		parsed := uuid.Parse(id)
		if parsed != nil {
			// The NodeID consists of the last 6 bytes of the full UUID.
			// We use the first 2 bytes of this 6-byte block instead of the full
			// uuid.Array because the first 10 bits are clock-dependent.
			array := parsed.NodeID()
			machineId = binary.BigEndian.Uint16(array[0:2])
		}
	} else {
		id = "<not available>"
	}
	//log.Infof("machine UUID: %s   generator machine ID: %d", id, machineId)

	uidGen = indigo.New(
		nil,
		indigo.StartTime(time.Unix(1257894000, 0)), // Go epoch
		indigo.MachineID(func() (uint16, error) { return machineId, nil }),
	)
}

func (u ID) String() string {
	return string(u)
}

func (u ID) IsNil() bool {
	return len(u) == 0
}

func FromString(s string) (ID, error) {
	_, err := uidGen.Decompose(s)
	if err != nil {
		return "", err
	}
	return ID(s), nil
}

func NilID() ID {
	return ""
}

func New() ID {
	id, err := uidGen.NextID()
	if err != nil {
		//log.Warnf("indigo.NextID() failed with %s, reverting to XID", err)
		return ID(xid.New().String())
	}
	return ID(id)
}

func (u ID) MarshalJSON() ([]byte, error) {
	return json.Marshal(u.String())
}
