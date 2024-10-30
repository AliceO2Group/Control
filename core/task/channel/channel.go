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
	"errors"
	"strconv"
	"strings"

	"github.com/AliceO2Group/Control/common/logger"
	"github.com/sirupsen/logrus"
)

var log = logger.New(logrus.StandardLogger(), "channel")

type Channel struct {
	Name        string        `yaml:"name"`
	Type        ChannelType   `yaml:"type"`
	SndBufSize  int           `yaml:"sndBufSize"`
	RcvBufSize  int           `yaml:"rcvBufSize"`
	RateLogging string        `yaml:"rateLogging"` //actually an int but we allow templating
	Transport   TransportType `yaml:"transport"`   //default: default
	Target      string        `yaml:"target"`      //default: empty
	// allowed values for `target` field:
	//   outbound channel (mandatory!): ->outbound.go
	//     tcp://host:port
	//     ipc://named-pipe-name
	//     path.to.role:channel_name
	//     global_channel_name
	//   inbound channel (optional!): ->inbound.go
	//     tcp://host:port
	//     ipc://named-pipe-name
	//     <empty> -> automatic port assignment (pre-v0.24 behaviour)
}

func (c *Channel) UnmarshalYAML(unmarshal func(interface{}) error) (err error) {
	type _channel struct {
		Name        string        `yaml:"name"`
		Type        ChannelType   `yaml:"type"`
		SndBufSize  string        `yaml:"sndBufSize"`
		RcvBufSize  string        `yaml:"rcvBufSize"`
		RateLogging string        `yaml:"rateLogging"`
		Transport   TransportType `yaml:"transport"`
		Target      string        `yaml:"target"`
	}
	aux := _channel{}
	err = unmarshal(&aux)
	if err != nil {
		return
	}

	c.Name = aux.Name
	c.Type = aux.Type
	if aux.SndBufSize == "" {
		aux.SndBufSize = "1000"
	}
	if aux.RcvBufSize == "" {
		aux.RcvBufSize = "1000"
	}
	if aux.RateLogging == "" {
		aux.RateLogging = "0"
	}
	if len(aux.Transport) == 0 {
		aux.Transport = DEFAULT
	}

	c.SndBufSize, err = strconv.Atoi(aux.SndBufSize)
	if err != nil {
		return
	}
	c.RcvBufSize, err = strconv.Atoi(aux.RcvBufSize)
	if err != nil {
		return
	}
	c.RateLogging = aux.RateLogging
	c.Transport = aux.Transport
	c.Target = strings.TrimSpace(aux.Target)

	return
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
