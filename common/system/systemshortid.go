/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2022 CERN and copyright holders of ALICE O².
 * Author: Claire Guyot <claire.guyot@cern.ch>
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

//go:generate go run github.com/dmarkham/enumer -type=ShortID -yaml -json -text -transform=upper -output=systemshortid_strings.go

package system

import "sort"

type ShortIDMap map[ShortID]struct{}

func (m ShortIDMap) StringList() []string {
	list := make([]string, len(m))
	i := 0
	for k, _ := range m {
		list[i] = k.String()
		i++
	}
	sort.Strings(list)
	return list
}

type ShortID int
const (
	// 1
	// 2
	T ShortID = 3		// TPC
	R ShortID = 4		// TRD
	O ShortID = 5		// TOF
	H ShortID = 6		// HMP
	P ShortID = 7		// PHS
	C ShortID = 8		// CPV
	// 9
	M ShortID = 10		// MCH
	// 11-14
	Z ShortID = 15		// ZDC
	// 16
	G ShortID = 17		// TRG
	E ShortID = 18		// EMC
	S ShortID = 19		// TST
	// 20-31
	I ShortID = 32		// ITS
	D ShortID = 33		// FDD
	F ShortID = 34		// FT0
	V ShortID = 35		// FV0
	N ShortID = 36		// MFT
	U ShortID = 37		// MID
	J ShortID = 38		// DCS
	W ShortID = 39		// FOC

	K ShortID = 254		// FIT: non-standard mapping: FT0 + FV0 = FIT
	L ShortID = 255		// NIL
)

// Additional non-standard system IDs
const (
	FF ShortID = -1	// FLP
	EE ShortID = -2	// EPN
	PP ShortID = -3	// PDP
)