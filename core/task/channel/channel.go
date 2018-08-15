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

package channel

import (
	"strings"
	"errors"
)

type channel struct {
	Name        string                  `yaml:"name"`
	Type        ChannelType             `yaml:"type"`
	SndBufSize  int                     `yaml:"sndBufSize"`
	RcvBufSize  int                     `yaml:"rcvBufSize"`
	RateLogging int                     `yaml:"rateLogging"`
}

// TODO: FairMQ has the following channel types:
// push/pull/pub/sub/spub/xsub/pair/req/rep/dealer/router
// Do we need to support them all?
type ChannelType string
const (
	PUSH = ChannelType("push")
	PULL = ChannelType("pull")
	PUB  = ChannelType("pub")
	SUB  = ChannelType("sub")
)

func (ct ChannelType) String() string {
	return string(ct)
}

func (ct *ChannelType) UnmarshalText(b []byte) error {
	str := strings.ToLower(strings.Trim(string(b), `"`))

	switch str {
	case PUSH.String(), PULL.String(), PUB.String(), SUB.String():
		*ct = ChannelType(str)
	default:
		return errors.New("invalid channel type: " + str)
	}

	return nil
}
