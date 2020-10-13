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

package uid

import (
	"time"

	"github.com/AliceO2Group/Control/common/logger"
	"github.com/osamingo/indigo"
	"github.com/rs/xid"
	"github.com/sirupsen/logrus"
)

var (
	log = logger.New(logrus.StandardLogger(),"utils")
	uidGen = indigo.New(nil, indigo.StartTime(
		time.Unix(1257894000, 0)),
		indigo.MachineID(func() (uint16, error){return 42, nil}))
)

type ID string

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
		log.Warnf("indigo.NextID() failed with %s, reverting to XID", err)
		return ID(xid.New().String())
	}
	return ID(id)
}
