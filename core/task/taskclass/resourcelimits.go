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

// Package taskclass provides task class definitions and resource management
// for different types of tasks in the O² Control system.
package taskclass

import "strconv"

type ResourceLimits struct {
	Cpu    *float64 `yaml:"cpu"`
	Memory *float64 `yaml:"memory"`
}

func (rw *ResourceLimits) UnmarshalYAML(unmarshal func(interface{}) error) (err error) {
	type _resourceLimits struct {
		Cpu    *string `yaml:"cpu"`
		Memory *string `yaml:"memory"`
	}
	aux := _resourceLimits{}
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
	return
}
