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

package constraint

import (
	"fmt"
	"strings"

	"github.com/AliceO2Group/Control/common/utils"
	mesos "github.com/mesos/mesos-go/api/v1/lib"
)

type Attributes []mesos.Attribute

func (attrs Attributes) String() string {
	if attrs == nil {
		return "[]"
	}
	strs := make([]string, len(attrs))
	for i, attr := range attrs {
		strs[i] = attr.String()
	}
	return fmt.Sprintf("[%s]", strings.Join(strs, "; "))
}

func (attrs Attributes) Get(attributeName string) (value string, ok bool) {
	for _, a := range attrs {
		if a.Name == attributeName {
			value = a.GetText().GetValue()
			ok = true
			return
		}
	}
	return
}

func (attrs Attributes) Satisfy(cts Constraints) (ok bool) {
	if len(cts) == 0 {
		ok = true
		log.Debug("no constraints to satisfy, defaulting to true")
		return
	}
	if attrs == nil {
		log.Debug("no attributes but non-null constraints, defaulting to false")
		return
	}

	for _, constraint := range cts {
		switch constraint.Operator {
		case Equals:
			var value string
			if value, ok = attrs.Get(constraint.Attribute); ok {
				if strings.Contains(value, ",") {
					values := strings.Split(value, ",")
					if utils.StringSliceContains(values, constraint.Value) {
						ok = true
						continue
					}
				}
				if value == constraint.Value {
					ok = true
					continue
				} else {
					ok = false
					break
				}
			} else { //at least 1 constraint not satisfiable, bailing out
				log.WithField("constraint", constraint.Attribute).Warning("at least 1 constraint not satisfiable (cannot get attribute)")
				ok = false
				break
			}
		default:
			log.WithField("constraint", constraint.Attribute).Warning("unsupported operator, skipping constraint")
			continue
		}
	}
	return
}
