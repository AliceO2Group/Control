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

type PointerWrapper struct {
	field *string
}

func WrapPointer(field *string) Field {
	return &PointerWrapper{field: field}
}

func (t *PointerWrapper) Get() string {
	return *t.field
}

func (t *PointerWrapper) Set(value string) {
	*t.field = value
}

type GetterFunc func() string
type SetterFunc func(value string)
type GenericWrapper struct {
	Getter GetterFunc
	Setter SetterFunc
}

func (t *GenericWrapper) Get() string {
	return t.Getter()
}

func (t *GenericWrapper) Set(value string) {
	t.Setter(value)
}

func WrapGeneric(getterF GetterFunc, setterF SetterFunc) Field {
	return &GenericWrapper{
		Getter: getterF,
		Setter: setterF,
	}
}

func WrapMapItems(items map[string]string) Fields {
	fields := make(Fields, 0)
	for k := range items {
		key := k // we need a local copy for the Getter/Setter closures
		fields = append(fields, &GenericWrapper{
			Getter: func() string {
				return items[key]
			},
			Setter: func(value string) {
				items[key] = value
			},
		})
	}
	return fields
}

func WrapSliceItems(items []string) Fields {
	fields := make(Fields, 0)
	for i := range items {
		index := i // we need a local copy for the Getter/Setter closures
		fields = append(fields, &GenericWrapper{
			Getter: func() string {
				return items[index]
			},
			Setter: func(value string) {
				items[index] = value
			},
		})
	}
	return fields
}
