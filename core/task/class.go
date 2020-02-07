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
	"github.com/AliceO2Group/Control/common"
	"github.com/AliceO2Group/Control/common/controlmode"
	"github.com/AliceO2Group/Control/common/gera"
	"github.com/AliceO2Group/Control/core/task/channel"
	"github.com/AliceO2Group/Control/core/task/constraint"
	"strconv"
)

// ↓ We need the roles tree to know *where* to run it and how to *configure* it, but
//   the following information is enough to run the task even with no environment or
//   role Class.
type Class struct {
	Identifier  taskClassIdentifier		`yaml:"name"`
	Control     struct {
		Mode    controlmode.ControlMode `yaml:"mode"`
	}                                   `yaml:"control"`
	Command     *common.CommandInfo     `yaml:"command"`
	Wants       ResourceWants           `yaml:"wants"`
	Bind        []channel.Inbound       `yaml:"bind"`
	Properties  gera.StringMap          `yaml:"properties"`
	Constraints []constraint.Constraint `yaml:"constraints"`
}

func (c *Class) UnmarshalYAML(unmarshal func(interface{}) error) (err error) {
	// We need to make a fake type to unmarshal into because
	// gera.StringMap is an interface
	type _class struct {
		Identifier  taskClassIdentifier		`yaml:"name"`
		Control     struct {
			Mode    controlmode.ControlMode `yaml:"mode"`
		}                                   `yaml:"control"`
		Command     *common.CommandInfo     `yaml:"command"`
		Wants       ResourceWants           `yaml:"wants"`
		Bind        []channel.Inbound       `yaml:"bind"`
		Properties  map[string]string       `yaml:"properties"`
		Constraints []constraint.Constraint `yaml:"constraints"`
	}
	aux := _class{
		Properties: make(map[string]string),
	}
	err = unmarshal(&aux)
	if err == nil {
		*c = Class{
			Identifier: aux.Identifier,
			Control:    aux.Control,
			Command:    aux.Command,
			Wants:      aux.Wants,
			Bind:       aux.Bind,
			Properties: gera.MakeStringMapWithMap(aux.Properties),
			Constraints:aux.Constraints,
		}
	}
	return

}

type taskClassIdentifier struct {
	repoIdentifier string
	hash           string
	Name           string
}

func (tcID taskClassIdentifier) String() string {
	return fmt.Sprintf("%stasks/%s@%s", tcID.repoIdentifier, tcID.Name, tcID.hash)
}

func (tcID *taskClassIdentifier) UnmarshalYAML(unmarshal func(interface{}) error) (err error) {
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
