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
	"fmt"
	"strconv"
	"strings"

	"dario.cat/mergo"
	"github.com/AliceO2Group/Control/core/controlcommands"
)

type Inbound struct {
	Channel
	Global     string        `yaml:"global"`
	Addressing AddressFormat `yaml:"addressing"` //default: tcp
	// Addressing is ignored if Target not empty, because it means the WFT/TT is
	// setting a static TCP or IPC bind address.
}

func (inbound *Inbound) UnmarshalYAML(unmarshal func(interface{}) error) (err error) {
	aux := struct {
		Global     string        `yaml:"global"`
		Addressing AddressFormat `yaml:"addressing"` //default: tcp
	}{}
	err = unmarshal(&aux)
	if err != nil {
		return
	}

	ch := Channel{}
	err = unmarshal(&ch)
	if err != nil {
		return
	}
	if len(aux.Addressing) == 0 {
		aux.Addressing = TCP
	}

	inbound.Addressing = aux.Addressing
	inbound.Channel = ch
	inbound.Global = aux.Global
	return
}

func (inbound Inbound) MarshalYAML() (interface{}, error) {
	// Flatten Inbound to have Addressing and Channel elements on the same "level"
	type _inbound struct {
		Name        string        `yaml:"name"`
		Global      string        `yaml:"global"`
		Type        ChannelType   `yaml:"type"`
		SndBufSize  int           `yaml:"sndBufSize,omitempty"`
		RcvBufSize  int           `yaml:"rcvBufSize,omitempty"`
		RateLogging string        `yaml:"rateLogging,omitempty"`
		Transport   TransportType `yaml:"transport"`
		Addressing  AddressFormat `yaml:"addressing"`
	}

	auxInbound := _inbound{
		Name:        inbound.Channel.Name,
		Global:      inbound.Global,
		Type:        inbound.Channel.Type,
		SndBufSize:  inbound.Channel.SndBufSize,
		RcvBufSize:  inbound.Channel.RcvBufSize,
		RateLogging: inbound.RateLogging,
		Transport:   inbound.Channel.Transport,
		Addressing:  inbound.Addressing,
	}

	return auxInbound, nil
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

func (inbound *Inbound) ToFMQMap(bindMap BindMap) (pm controlcommands.PropertyMap, err error) {
	if inbound == nil {
		err = fmt.Errorf("inbound channel object is nil")
		return nil, err
	}

	var address string
	var transport TransportType
	// If an explicit target was provided, we use it.
	// NO MATCHING WILL HAPPEN!
	if strings.HasPrefix(inbound.Target, "tcp://") ||
		strings.HasPrefix(inbound.Target, "ipc://") {
		address = inbound.Target
		transport = inbound.Transport
	} else if len(inbound.Target) != 0 {
		err = fmt.Errorf("inbound channel target %s provided but invalid", inbound.Target)
		log.WithError(err).
			Error("bad inbound channel specification")
		return nil, err
	} else {
		// We ignore any Target spec and proceed with inbound-outbound matching with
		// the provided localBindMap.
		endpoint, ok := bindMap[inbound.Name]
		if !ok {
			err = fmt.Errorf("endpoint for key %s not present in bindMap", inbound.Name)
			log.WithError(err).
				Warn("cannot match endpoint for channel")
			return nil, err
		}
		address = endpoint.ToBoundEndpoint().GetAddress()
		transport = endpoint.GetTransport()
	}

	return inbound.buildFMQMap(address, transport), nil
}

func (inbound *Inbound) buildFMQMap(address string, transport TransportType) (pm controlcommands.PropertyMap) {
	pm = make(controlcommands.PropertyMap)
	const chans = "chans"
	chName := inbound.Name
	// We assume one socket per channel, so this must always be set
	pm[strings.Join([]string{chans, chName, "numSockets"}, ".")] = "1"
	prefix := strings.Join([]string{chans, chName, "0"}, ".")

	chanProps := controlcommands.PropertyMap{
		"address":       address,
		"method":        "bind",
		"autoBind":      "0",
		"rateLogging":   inbound.RateLogging,
		"rcvBufSize":    strconv.Itoa(inbound.RcvBufSize),
		"rcvKernelSize": "0", //NOTE: hardcoded
		"sndBufSize":    strconv.Itoa(inbound.SndBufSize),
		"sndKernelSize": "0", //NOTE: hardcoded
		"transport":     transport.String(),
		"type":          inbound.Type.String(),
	}

	for k, v := range chanProps {
		pm[prefix+"."+k] = v
	}
	return
}

func MergeInbound(hp, lp []Inbound) (channels []Inbound) {
	channels = make([]Inbound, len(hp))
	copy(channels, hp)

	for _, v := range lp {
		updated := false
		for _, pCh := range channels {
			if v.Name == pCh.Name {
				_ = mergo.Merge(&pCh, v)
				updated = true
				break
			}
		}
		if !updated {
			channels = append(channels, v)
		}
	}

	return
}
