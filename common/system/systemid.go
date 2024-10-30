/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2021 CERN and copyright holders of ALICE O².
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

//go:generate go run github.com/dmarkham/enumer -type=ID -yaml -json -text -transform=upper -output=systemid_strings.go

package system

import "sort"

// Source: https://alice-notes.web.cern.ch/system/files/notes/public/1052/2020-07-15-2020-07-07-O2_Report_Identification_of_sources_for_ALICE_data.pdf
// System ID mapping on page 3

type IDMap map[ID]struct{}

func (m IDMap) StringList() []string {
	list := make([]string, len(m))
	i := 0
	for k := range m {
		list[i] = k.String()
		i++
	}
	sort.Strings(list)
	return list
}

type ID int

const (
	// 1
	// 2
	TPC ID = 3
	TRD ID = 4
	TOF ID = 5
	HMP ID = 6
	PHS ID = 7
	CPV ID = 8
	// 9
	MCH ID = 10
	// 11-14
	ZDC ID = 15
	// 16
	TRG ID = 17
	EMC ID = 18
	TST ID = 19
	// 20-31
	ITS ID = 32
	FDD ID = 33
	FT0 ID = 34
	FV0 ID = 35
	MFT ID = 36
	MID ID = 37
	DCS ID = 38
	FOC ID = 39

	FIT ID = 254 // non-standard mapping: FT0 + FV0 = FIT
	NIL ID = 255
)

// Additional non-standard system IDs
const (
	FLP ID = -1
	EPN ID = -2
	PDP ID = -3
)
