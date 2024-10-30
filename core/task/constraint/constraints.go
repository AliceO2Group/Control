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

// Package constraint implements support for predicates on agent
// attributes.
// Constraints notably implement a MergeParent operation, to implement
// override behavior in child Roles.
package constraint

import (
	"fmt"
	"strings"

	"github.com/AliceO2Group/Control/common/logger"
	"github.com/sirupsen/logrus"
)

/*
	TODO:

type Operator func(attribute string, value string) bool

func Equals(attribute string, value string) bool {

}
*/
var log = logger.New(logrus.StandardLogger(), "constraints")

type Constraint struct {
	Attribute string `yaml:"attribute"`
	Value     string `yaml:"value"`
	// TODO: unmarshal this ↓
	Operator Operator
}

type Operator int8

const (
	Equals Operator = 0
)

func (o Operator) String() string {
	switch o {
	case Equals:
		return "EQUALS"
	}
	return ""
}

func (c *Constraint) String() string {
	if c == nil {
		return ""
	}
	return fmt.Sprintf("ATTR:'%s' %s '%s'", c.Attribute, c.Operator.String(), c.Value)
}

type Constraints []Constraint

func (cts Constraints) String() string {
	if cts == nil {
		return "[]"
	}
	strs := make([]string, len(cts))
	for i, ct := range cts {
		strs[i] = ct.String()
	}
	return fmt.Sprintf("[%s]", strings.Join(strs, "; "))
}

func (cts Constraints) MergeParent(parentConstraints Constraints) (merged Constraints) {
	merged = make(Constraints, len(parentConstraints))
	copy(merged, parentConstraints)

	for _, ct := range cts {
		updated := false
		for j, pCt := range merged {
			// If we find that both new and parent have a constraint for the same attribute,
			// then the new one replaces the parent's constraint in the merged Constraints
			if ct.Attribute == pCt.Attribute {
				merged[j] = ct
				updated = true
				break
			}
		}
		if !updated {
			merged = append(merged, ct)
		}
	}
	return
}
