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

import (
	"fmt"
	"strconv"

	"github.com/AliceO2Group/Control/common"
	"github.com/AliceO2Group/Control/common/controlmode"
	"github.com/AliceO2Group/Control/common/gera"
	"github.com/AliceO2Group/Control/core/task/channel"
	"github.com/AliceO2Group/Control/core/task/constraint"
)

// ↓ We need the roles tree to know *where* to run it and how to *configure* it, but
//   the following information is enough to run the task even with no environment or
//   role Class.
type Class struct {
	Identifier  TaskClassIdentifier		`yaml:"name"`
	Defaults    gera.StringMap          `yaml:"defaults"`
	Control     struct {
		Mode    controlmode.ControlMode `yaml:"mode"`
	}                                   `yaml:"control"`
	Command     *common.CommandInfo     `yaml:"command"`
	Wants       ResourceWants           `yaml:"wants"`
	Bind        []channel.Inbound       `yaml:"bind"`
	Properties  gera.StringMap          `yaml:"properties"`
	Constraints []constraint.Constraint `yaml:"constraints"`
	Connect     []channel.Outbound		`yaml:"connect"`
}

func (c *Class) UnmarshalYAML(unmarshal func(interface{}) error) (err error) {
	// We need to make a fake type to unmarshal into because
	// gera.StringMap is an interface
	type _class struct {
		Identifier  TaskClassIdentifier		`yaml:"name"`
		Defaults    map[string]string       `yaml:"defaults"`
		Control     struct {
			Mode    controlmode.ControlMode `yaml:"mode"`
		}                                   `yaml:"control"`
		Command     *common.CommandInfo     `yaml:"command"`
		Wants       ResourceWants           `yaml:"wants"`
		Bind        []channel.Inbound       `yaml:"bind"`
		Properties  map[string]string       `yaml:"properties"`
		Constraints []constraint.Constraint `yaml:"constraints"`
		Connect     []channel.Outbound		`yaml:"connect"`
	}
	aux := _class{
		Defaults: make(map[string]string),
		Properties: make(map[string]string),
	}
	err = unmarshal(&aux)
	if err == nil {
		for j, ch := range aux.Connect {
			if ch.Target != "" {
				ch.Target = ""
				aux.Connect[j] = ch
				log.Warn("task template should lack target entry for outbound channels")
			}
		}
		*c = Class{
			Identifier: aux.Identifier,
			Defaults:   gera.MakeStringMapWithMap(aux.Defaults),
			Control:    aux.Control,
			Command:    aux.Command,
			Wants:      aux.Wants,
			Bind:       aux.Bind,
			Properties: gera.MakeStringMapWithMap(aux.Properties),
			Constraints:aux.Constraints,
			Connect:    aux.Connect,
		}
	}
	return

}

func (c *Class) MarshalYAML() (interface{}, error) {
	type _bind struct {
		Name        string                `yaml:"name"`
		Type        channel.ChannelType   `yaml:"type"`
		SndBufSize  int                   `yaml:"sndBufSize"`
		RcvBufSize  int                   `yaml:"rcvBufSize"`
		RateLogging string                `yaml:"rateLogging"`
		Transport   channel.TransportType `yaml:"transport"`
		Addressing  channel.AddressFormat `yaml:"addressing"`
    }
    
    type _connect struct {
		Name        string                `yaml:"name"`
		Type        channel.ChannelType   `yaml:"type"`
		SndBufSize  int                   `yaml:"sndBufSize"`
		RcvBufSize  int                   `yaml:"rcvBufSize"`
		RateLogging string                `yaml:"rateLogging"`
		Transport   channel.TransportType `yaml:"transport"`
		Target      string                `yaml:"target,omitempty"`
	}

	type _class struct {
		Name     string                     `yaml:"name"`
		Defaults map[string]string          `yaml:"defaults"`
		Control  struct {
			Mode string                     `yaml:"mode"`
		}                                   `yaml:"control"`
		Wants       ResourceWants           `yaml:"wants"`
		Bind        []_bind                 `yaml:"bind"`
		Connect 	[]_connect              `yaml:"connect"`
		Properties  map[string]string       `yaml:"properties"`
		Constraints []constraint.Constraint `yaml:"constraints,omitempty"`
		Command     *common.CommandInfo     `yaml:"command"`
	}
	aux := _class{
		Name:        c.Identifier.Name,
		Defaults:    c.Defaults.Raw(),
		Properties:  c.Properties.Raw(),
		Wants:       c.Wants,
		Constraints: c.Constraints,
		Command:     c.Command,
	}

	// Flatten connect struct to have channel elements and addressing together
	for _, bind := range c.Bind {
		auxBind := _bind{
			Name:        bind.Name,
			Type:        bind.Type,
			SndBufSize:  bind.SndBufSize,
			RcvBufSize:  bind.RcvBufSize,
			RateLogging: strconv.Itoa(bind.RateLogging),
			Transport:   bind.Transport,
			Addressing:  bind.Addressing,
		}
		aux.Bind = append(aux.Bind, auxBind)
	}

	for _, connect := range c.Connect {
		auxConnect := _connect{
			Name:        connect.Name,
			Type:        connect.Type,
			SndBufSize:  connect.SndBufSize,
			RcvBufSize:  connect.RcvBufSize,
			RateLogging: strconv.Itoa(connect.RateLogging),
			Transport:   connect.Transport,
			Target:      connect.Target,
		}
		aux.Connect = append(aux.Connect, auxConnect)
	}

	if c.Control.Mode == controlmode.FAIRMQ {
		aux.Control.Mode = "fairmq"
	} else if c.Control.Mode == controlmode.BASIC {
		aux.Control.Mode = "basic"
	} else {
		aux.Control.Mode = "direct"
	}

	return aux, nil
}

type TaskClassIdentifier struct {
	repoIdentifier string
	hash           string
	Name           string
}

func (tcID TaskClassIdentifier) String() string {
	return fmt.Sprintf("%stasks/%s@%s", tcID.repoIdentifier, tcID.Name, tcID.hash)
}

func (tcID *TaskClassIdentifier) UnmarshalYAML(unmarshal func(interface{}) error) (err error) {
	err = unmarshal(&tcID.Name)
	return
}

type ResourceWants struct {
	Cpu     *float64                `yaml:"cpu"`
	Memory  *float64                `yaml:"memory"`
	Ports   Ranges                  `yaml:"ports"`
}

func (rw *ResourceWants) UnmarshalYAML(unmarshal func(interface{}) error) (err error) {
	type _resourceWants struct {
		Cpu     *string                 `yaml:"cpu"`
		Memory  *string                 `yaml:"memory"`
		Ports   *string                 `yaml:"ports"`
	}
	aux := _resourceWants{}
	err = unmarshal(&aux)
	if err != nil {
		return
	}

	if aux.Cpu != nil {
		var cpuCount float64
		cpuCount, err = strconv.ParseFloat(*aux.Cpu, 64)
		if err != nil {
			return
		}
		rw.Cpu = &cpuCount
	}
	if aux.Memory != nil {
		var memCount float64
		memCount, err = strconv.ParseFloat(*aux.Memory, 64)
		if err != nil {
			return
		}
		rw.Memory = &memCount
	}
	if aux.Ports != nil {
		var ranges Ranges
		ranges, err = parsePortRanges(*aux.Ports)
		if err != nil {
			return
		}
		rw.Ports = ranges
	}
	return
}

func (c *Class) Equals(other *Class) (response bool) {
	if c == nil || other == nil {
		return false
	}
	response = c.Command.Equals(other.Command) &&
		*c.Wants.Cpu == *other.Wants.Cpu &&
		*c.Wants.Memory == *other.Wants.Memory &&
		c.Wants.Ports.Equals(other.Wants.Ports)
	return
}
