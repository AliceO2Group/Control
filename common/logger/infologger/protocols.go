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

// Package infologger provides InfoLogger protocol implementation for
// integration with the ALICE InfoLogger logging system.
package infologger

type protoVersion string

const (
	v14 = protoVersion("1.4")
	v13 = protoVersion("1.3")
)

type fieldType string

const (
	ft_String = fieldType("String")
	ft_Number = fieldType("Number")
)

type fieldSpec struct {
	name  string
	ftype fieldType
}

type protoSpec []fieldSpec

var protocols = map[protoVersion]*protoSpec{
	v13: {
		{name: "severity", ftype: ft_String},
		{name: "level", ftype: ft_Number},
		{name: "timestamp", ftype: ft_Number},
		{name: "hostname", ftype: ft_String},
		{name: "rolename", ftype: ft_String},
		{name: "pid", ftype: ft_Number},
		{name: "username", ftype: ft_String},
		{name: "system", ftype: ft_String},
		{name: "facility", ftype: ft_String},
		{name: "detector", ftype: ft_String},
		{name: "partition", ftype: ft_String},
		{name: "dest", ftype: ft_String},
		{name: "run", ftype: ft_Number},
		{name: "errcode", ftype: ft_Number},
		{name: "errline", ftype: ft_Number},
		{name: "errsource", ftype: ft_String},
		{name: "message", ftype: ft_String},
	},
	v14: {
		{name: "severity", ftype: ft_String},
		{name: "level", ftype: ft_Number},
		{name: "timestamp", ftype: ft_Number},
		{name: "hostname", ftype: ft_String},
		{name: "rolename", ftype: ft_String},
		{name: "pid", ftype: ft_Number},
		{name: "username", ftype: ft_String},
		{name: "system", ftype: ft_String},
		{name: "facility", ftype: ft_String},
		{name: "detector", ftype: ft_String},
		{name: "partition", ftype: ft_String},
		{name: "run", ftype: ft_Number},
		{name: "errcode", ftype: ft_Number},
		{name: "errline", ftype: ft_Number},
		{name: "errsource", ftype: ft_String},
		{name: "message", ftype: ft_String},
	},
}
