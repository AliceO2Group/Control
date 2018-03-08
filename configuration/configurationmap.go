/*
 * === This file is part of octl <https://github.com/teo/octl> ===
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

package configuration

type Item interface {
	IsValue() bool
	IsMap() bool
	Value() string
	Map() Map
}
type Map	 map[string]Item
type String	 string

func (m Map) IsValue() bool {
	return false
}
func (m Map) IsMap() bool {
	return true
}
func (m Map) Value() string {
	return ""
}
func (m Map) Map() Map {
	return m
}

func (s String) IsValue() bool {
	return true
}
func (s String) IsMap() bool {
	return false
}
func (s String) Value() string {
	return string(s)
}
func (s String) Map() Map {
	return nil
}
