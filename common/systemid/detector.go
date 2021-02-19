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


package systemid

// Source: https://alice-notes.web.cern.ch/system/files/notes/public/1052/2020-07-15-2020-07-07-O2_Report_Identification_of_sources_for_ALICE_data.pdf
// System ID mapping on page 3


// FIXME: use stringer/enumer on this one
const (

	// 1
	// 2
	TPC = 3
	TRD = 4
	TOF = 5
	HMP = 6
	PHS = 7
	CPV = 8
	// 9
	MCH = 10
	// 11-14
	ZDC = 15
	// 16
	TRG = 17
	EMC = 18
	TST = 19
	// 20-31
	ITS = 32
	FDD = 33
	FT0 = 34
	FV0 = 35
	MFT = 36
	MID = 37
	DCS = 38
	FOC = 39

	NIL = 255
)
