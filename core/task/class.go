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
	"github.com/AliceO2Group/Control/common/controlmode"
	"github.com/AliceO2Group/Control/common"
	"github.com/AliceO2Group/Control/core/task/channel"
	"github.com/AliceO2Group/Control/core/task/constraint"
	"github.com/AliceO2Group/Control/core/controlcommands"
)

type TaskClass info

// ↓ We need the roles tree to know *where* to run it and how to *configure* it, but
//   the following information is enough to run the task even with no environment or
//   role info.
type info struct {
	Name        string                  `yaml:"name"`
	Control     struct {
		Mode    controlmode.ControlMode `yaml:"mode"`
	}                                   `yaml:"control"`
	Command     *common.CommandInfo     `yaml:"command"`
	Wants       struct{
		Cpu     *float64                `yaml:"cpu"`
		Memory  *float64                `yaml:"memory"`
		Ports   Ranges                  `yaml:"ports"`
	}                                   `yaml:"wants"`
	Bind        []channel.Inbound       `yaml:"bind"`
	Properties  controlcommands.PropertyMap `yaml:"properties"`
	Constraints []constraint.Constraint `yaml:"constraints"`
}

func (this *info) Equals(other *info) (response bool) {
	if this == nil || other == nil {
		return false
	}
	response = this.Name == other.Name &&
		this.Command.Equals(other.Command) &&
		*this.Wants.Cpu == *other.Wants.Cpu &&
		*this.Wants.Memory == *other.Wants.Memory &&
		this.Wants.Ports.Equals(other.Wants.Ports)
	return
}
