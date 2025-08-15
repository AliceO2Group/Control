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

// Package port provides port range management functionality for task
// communication, including port range parsing and validation.
package port

import (
	"errors"
	"strconv"
	"strings"
)

type Range struct {
	Begin uint64 `json:"begin" yaml:"begin"`
	End   uint64 `json:"end"   yaml:"end"`
}

/*func (r *Range) UnmarshalYAML(unmarshal func(interface{}) error) (err error) {
	type _range struct {
		Begin string `json:"begin" yaml:"begin"`
		End   string `json:"end"   yaml:"end"`
	}
	aux := _range{}
	err = unmarshal(&aux)
	if err != nil {
		return
	}

	var begin, end uint64
	begin, err = strconv.ParseUint(aux.Begin, 0, 64)
	if err != nil {
		return
	}
	end, err = strconv.ParseUint(aux.End, 0, 64)
	if err != nil {
		return
	}
	r.Begin = begin
	r.End = end
	return
}*/

type Ranges []Range

func (this Ranges) Equals(other Ranges) (response bool) {
	if len(this) != len(other) {
		return false
	}

	response = true
	for i := range this {
		if this[i].Begin == other[i].Begin && this[i].End == other[i].End {
			continue
		}
		response = false
		return
	}
	return
}

func RangesFromExpression(str string) (ranges Ranges, err error) {
	r := make(Ranges, 0)
	if len(strings.TrimSpace(str)) == 0 {
		return
	}

	split := strings.Split(str, ",")
	for _, s := range split {
		trimmed := strings.TrimSpace(s)
		rangeSplit := strings.Split(trimmed, "-")
		if len(rangeSplit) == 1 { // single port range
			var port uint64
			port, err = strconv.ParseUint(rangeSplit[0], 10, 64)
			if err != nil {
				return
			}
			r = append(r, Range{Begin: port, End: port})
			continue
		} else if len(rangeSplit) == 2 { //normal range
			var begin, end uint64
			begin, err = strconv.ParseUint(rangeSplit[0], 10, 64)
			if err != nil {
				return
			}
			end, err = strconv.ParseUint(rangeSplit[0], 10, 64)
			if err != nil {
				return
			}
			r = append(r, Range{Begin: begin, End: end})
			continue
		} else {
			err = errors.New("bad format for roleClass ports range")
			return
		}
	}
	ranges = r
	return
}
