/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2018-2020 CERN and copyright holders of ALICE O².
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

package template

type Field interface {
	Get() string
	Set(value string)
}

type pointerWrapper struct {
	field *string
}

func WrapPointer(field *string) Field {
	return &pointerWrapper{field: field}
}

func (t *pointerWrapper) Get() string {
	return *t.field
}

func (t *pointerWrapper) Set(value string) {
	*t.field = value
}


type mapItemWrapper struct {
	getter func() string
	setter func(value string)
}

func WrapMapItems(items map[string]string) Fields {
	fields := make(Fields, 0)
	for k, _ := range items {
		key := k // we need a local copy for the getter/setter closures
		fields = append(fields, &mapItemWrapper{
			getter: func() string {
				return items[key]
			},
			setter: func(value string) {
				items[key] = value
			},
		})
	}
	return fields
}

func (t *mapItemWrapper) Get() string {
	return t.getter()
}

func (t *mapItemWrapper) Set(value string) {
	t.setter(value)
}
