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
	"strconv"
	"strings"

	"github.com/AliceO2Group/Control/core/controlcommands"
)

type Inbound struct {
	channel
	Addressing       AddressFormat             `yaml:"addressing"` //default: tcp
}

/*
FairMQ inbound channel property map example:
chans.data1.0.address       = tcp://*:5555                                                                                                                                                                                                                                                                                                                                                                                                         <string>      [provided]
chans.data1.0.method        = bind                                                                                                                                                                                                                                                                                                                                                                                                                 <string>      [provided]
chans.data1.0.rateLogging   = 0                                                                                                                                                                                                                                                                                                                                                                                                                    <int>         [provided]
chans.data1.0.rcvBufSize    = 1000                                                                                                                                                                                                                                                                                                                                                                                                                 <int>         [provided]
chans.data1.0.rcvKernelSize = 0                                                                                                                                                                                                                                                                                                                                                                                                                    <int>         [provided]
chans.data1.0.sndBufSize    = 1000                                                                                                                                                                                                                                                                                                                                                                                                                 <int>         [provided]
chans.data1.0.sndKernelSize = 0                                                                                                                                                                                                                                                                                                                                                                                                                    <int>         [provided]
chans.data1.0.transport     = default                                                                              <string>      [provided]
chans.data1.0.type          = push                                                                                 <string>      [provided]
chans.data1.numSockets      = 1
 */

func (inbound *Inbound) ToFMQMap(endpoint Endpoint) (pm controlcommands.PropertyMap) {
	return inbound.buildFMQMap(endpoint.ToBoundEndpoint().GetAddress(), endpoint.GetTransport())
}

func (inbound *Inbound) buildFMQMap(address string, transport TransportType) (pm controlcommands.PropertyMap) {
	pm = make(controlcommands.PropertyMap)
	const chans = "chans"
	chName := inbound.Name
	// We assume one socket per channel, so this must always be set
	pm[strings.Join([]string{chans, chName, "numSockets"}, ".")] = "1"
	prefix := strings.Join([]string{chans, chName, "0"}, ".")

	chanProps := controlcommands.PropertyMap{
		"address": address,
		"method": "bind",
		"rateLogging": strconv.Itoa(inbound.RateLogging),
		"rcvBufSize": strconv.Itoa(inbound.RcvBufSize),
		"rcvKernelSize": "0", //NOTE: hardcoded
		"sndBufSize": strconv.Itoa(inbound.SndBufSize),
		"sndKernelSize": "0", //NOTE: hardcoded
		"transport": transport.String(),
		"type": inbound.Type.String(),
	}

	for k, v := range chanProps {
		pm[prefix + "." + k] = v
	}
	return
}