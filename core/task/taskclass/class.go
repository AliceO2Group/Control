/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2018-2023 CERN and copyright holders of ALICE O².
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

package taskclass

import (
	"fmt"
	"time"

	"github.com/AliceO2Group/Control/common"
	"github.com/AliceO2Group/Control/common/controlmode"
	"github.com/AliceO2Group/Control/common/gera"
	"github.com/AliceO2Group/Control/common/logger"
	"github.com/AliceO2Group/Control/core/task/channel"
	"github.com/AliceO2Group/Control/core/task/constraint"
	"github.com/sirupsen/logrus"
)

var log = logger.New(logrus.StandardLogger(), "taskclass")

type Id struct {
	RepoIdentifier string
	Hash           string
	Name           string
}

func (tcID Id) String() string {
	return fmt.Sprintf("%s/tasks/%s@%s", tcID.RepoIdentifier, tcID.Name, tcID.Hash)
}

func (tcID *Id) UnmarshalYAML(unmarshal func(interface{}) error) (err error) {
	err = unmarshal(&tcID.Name)
	return
}

// ↓ We need the roles tree to know *where* to run it and how to *configure* it, but
//
//	the following information is enough to run the task even with no environment or
//	role Class.
type Class struct {
	Identifier Id                       `yaml:"name"`
	Defaults   gera.Map[string, string] `yaml:"defaults"`
	Vars       gera.Map[string, string] `yaml:"vars"`
	Control    struct {
		Mode controlmode.ControlMode `yaml:"mode"`
	} `yaml:"control"`
	Command          *common.CommandInfo      `yaml:"command"`
	Wants            ResourceWants            `yaml:"wants"`
	Limits           *ResourceLimits          `yaml:"limits"`
	Bind             []channel.Inbound        `yaml:"bind"`
	Properties       gera.Map[string, string] `yaml:"properties"`
	Constraints      []constraint.Constraint  `yaml:"constraints"`
	Connect          []channel.Outbound       `yaml:"connect"`
	UpdatedTimestamp time.Time                `yaml:"-"`
}

func (c *Class) UnmarshalYAML(unmarshal func(interface{}) error) (err error) {
	// We need to make a fake type to unmarshal into because
	// gera.Map is an interface
	type _class struct {
		Identifier Id                `yaml:"name"`
		Defaults   map[string]string `yaml:"defaults"`
		Vars       map[string]string `yaml:"vars"`
		Control    struct {
			Mode controlmode.ControlMode `yaml:"mode"`
		} `yaml:"control"`
		Command     *common.CommandInfo     `yaml:"command"`
		Wants       ResourceWants           `yaml:"wants"`
		Limits      *ResourceLimits         `yaml:"limits"`
		Bind        []channel.Inbound       `yaml:"bind"`
		Properties  map[string]string       `yaml:"properties"`
		Constraints []constraint.Constraint `yaml:"constraints"`
		Connect     []channel.Outbound      `yaml:"connect"`
	}
	aux := _class{
		Defaults:   make(map[string]string),
		Vars:       make(map[string]string),
		Properties: make(map[string]string),
	}
	err = unmarshal(&aux)
	if err == nil {
		for j, ch := range aux.Connect {
			if ch.Target != "" {
				ch.Target = ""
				aux.Connect[j] = ch
				log.Warn("task template outbound channel definition has a target (will be ignored)")
			}
		}
		*c = Class{
			Identifier:       aux.Identifier,
			Defaults:         gera.MakeMapWithMap(aux.Defaults),
			Vars:             gera.MakeMapWithMap(aux.Vars),
			Control:          aux.Control,
			Command:          aux.Command,
			Wants:            aux.Wants,
			Limits:           aux.Limits,
			Bind:             aux.Bind,
			Properties:       gera.MakeMapWithMap(aux.Properties),
			Constraints:      aux.Constraints,
			Connect:          aux.Connect,
			UpdatedTimestamp: time.Now(),
		}
	}
	return

}

func (c *Class) MarshalYAML() (interface{}, error) {
	type _class struct {
		Name     string            `yaml:"name"`
		Defaults map[string]string `yaml:"defaults,omitempty"`
		Vars     map[string]string `yaml:"vars,omitempty"`
		Control  struct {
			Mode string `yaml:"mode"`
		} `yaml:"control"`
		Wants       ResourceWants           `yaml:"wants"`
		Limits      *ResourceLimits         `yaml:"limits,omitempty"`
		Bind        []channel.Inbound       `yaml:"bind,omitempty"`
		Properties  map[string]string       `yaml:"properties,omitempty"`
		Constraints []constraint.Constraint `yaml:"constraints,omitempty"`
		Command     *common.CommandInfo     `yaml:"command"`
	}

	aux := _class{
		Name:        c.Identifier.Name,
		Defaults:    c.Defaults.RawCopy(),
		Vars:        c.Vars.RawCopy(),
		Properties:  c.Properties.RawCopy(),
		Wants:       c.Wants,
		Limits:      c.Limits,
		Bind:        c.Bind,
		Constraints: c.Constraints,
		Command:     c.Command,
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
