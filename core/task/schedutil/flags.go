/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2017 CERN and copyright holders of ALICE O².
 * Author: Teo Mrnjavac <teo.mrnjavac@cern.ch>
 *
 * Portions from examples in <https://github.com/mesos/mesos-go>:
 *     Copyright 2013-2015, Mesosphere, Inc.
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

package schedutil

import (
	"errors"
	"net/url"
	"strings"

	"github.com/mesos/mesos-go/api/v1/lib"
)

var (
	errZeroLengthLabelKey = errors.New("zero-length label key")
)

type URL struct{ url.URL }

func (u *URL) Set(value string) error {
	parsed, err := url.Parse(value)
	if err != nil {
		return err
	}
	u.URL = *parsed
	return nil
}

type Labels []mesos.Label

func (labels *Labels) Set(value string) error {
	set := func(k, v string) {
		var val *string
		if v != "" {
			val = &v
		}
		*labels = append(*labels, mesos.Label{
			Key:   k,
			Value: val,
		})
	}
	e := strings.IndexRune(value, '=')
	c := strings.IndexRune(value, ':')
	if e != -1 && e < c {
		if e == 0 {
			return errZeroLengthLabelKey
		}
		set(value[:e], value[e+1:])
	} else if c != -1 && c < e {
		if c == 0 {
			return errZeroLengthLabelKey
		}
		set(value[:c], value[c+1:])
	} else if e != -1 {
		if e == 0 {
			return errZeroLengthLabelKey
		}
		set(value[:e], value[e+1:])
	} else if c != -1 {
		if c == 0 {
			return errZeroLengthLabelKey
		}
		set(value[:c], value[c+1:])
	} else if value != "" {
		set(value, "")
	}
	return nil
}

func (labels Labels) String() string {
	// super inefficient, but it's only for occassional debugging
	s := ""
	valueString := func(v *string) string {
		if v == nil {
			return ""
		}
		return ":" + *v
	}
	for _, x := range labels {
		if s != "" {
			s += ","
		}
		s += x.Key + valueString(x.Value)
	}
	return s
}
