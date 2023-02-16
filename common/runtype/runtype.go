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

//go:generate go run github.com/dmarkham/enumer -type=RunType -yaml -json -text -transform=upper -output=runtype_strings.go

package runtype

// NOTE: make sure the enum values include and match those in RunType in dcs.pb.go and apricot.proto
// NOTE: this run type list is replicated in AliceO2 repo in
// https://github.com/AliceO2Group/AliceO2/blob/dev/DataFormats/Parameters/include/DataFormatsParameters/ECSDataAdapters.h
// Inform Ruben when the list is updated.
// NOTE: this run type list is replicated in Bookkeeping upstream, common.proto
// Inform George/Martin when the list is updated.
type RunType int

const (
	NONE RunType = iota
	PHYSICS
	TECHNICAL
	PEDESTAL
	PULSER
	LASER
	CALIBRATION_ITHR_TUNING
	CALIBRATION_VCASN_TUNING
	CALIBRATION_THR_SCAN
	CALIBRATION_DIGITAL_SCAN
	CALIBRATION_ANALOG_SCAN
	CALIBRATION_FHR
	CALIBRATION_ALPIDE_SCAN
	CALIBRATION // no correspondence with DCS
	COSMICS     // no correspondence with DCS
	SYNTHETIC   // no correspondence with DCS
	NOISE
	CALIBRATION_PULSE_LENGTH
	CALIBRATION_VRESETD
)
