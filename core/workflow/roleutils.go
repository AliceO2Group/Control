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

package workflow

import (
	"github.com/AliceO2Group/Control/configuration/template"
	"github.com/AliceO2Group/Control/core/task/constraint"
)

func WrapConstraints(items constraint.Constraints) template.Fields {
	fields := make(template.Fields, 0)
	for i, _ := range items {
		index := i // we need a local copy for the getter/setter closures
		fields = append(fields, &template.GenericWrapper{
			Getter: func() string {
				return items[index].Value
			},
			Setter: func(value string) {
				items[index].Value = value
			},
		})
	}
	return fields
}
