/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2022 CERN and copyright holders of ALICE O².
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

// Package trg provides integration with the ALICE trigger system.
package trg

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/AliceO2Group/Control/common/system"
)

type Run struct {
	Cardinality RunCardinality
	RunNumber   uint32
	State       State
	Detectors   []system.ID
	Skip        bool
}

type Runs []Run

type RunCardinality int
type State int

const (
	__RunCardinality_NIL RunCardinality = iota
	CTP_STANDALONE
	CTP_GLOBAL
	__RunCardinality_MAX
)

const (
	__State_NIL State = iota
	CTP_LOADED
	CTP_RUNNING
	__State_MAX
)

// respose examples:
//ecsc>rlist
//rc:4
//G 511707 R     tpc     run511707
//G 511713 R     its, mft, mid, mch, tof, trd, emc, cpv     run511713
//G 511714 R     phs     run511714
//G 511715 R     zdc     run511715
//
//ecsc>rlist
//rc:3
//S   2222 R     hmp
//S   2223 R     fv0
//G   2224 L     its, tpc     run2224

func parseRunList(runCount int, payload string) (runs Runs, err []error) {
	cleanPayload := strings.TrimSpace(payload)
	if len(cleanPayload) == 0 {
		// nothing to do
		return
	}
	lines := strings.Split(strings.TrimSpace(cleanPayload), "\n")
	if len(lines) != runCount {
		err = append(err, fmt.Errorf("cannot parse run count mismatch in payload: %s", payload))
		return
	}
	for i, line := range lines {
		run, parseErr := parseRunLine(line)
		if parseErr != nil {
			err = append(err, fmt.Errorf("cannot parse line %d: %s", i, parseErr.Error()))
			continue
		}
		if !run.Skip {
			runs = append(runs, run)
		}
	}

	return
}

func parseRunLine(line string) (run Run, err error) {
	// lst= lst+"S {:6} R     {:3}\n".format(runn, self.sruns[runn])
	// lst= lst+"G {:6} {:1}     {}     {}\n".format(runn, st, ", ".join(self.gruns[runn].getDets()), self.gruns[runn].pname)
	cols := strings.Fields(line)
	// col 0 -> TRG TYPE
	// col 1 -> RUN NUMBER
	// col 2 -> TRG STATUS
	// col 3 -> comma-seperated list of DETECTORS

	if len(cols) < 4 {
		err = fmt.Errorf("cannot parse run state from line with less than 4 columns: %s", line)
		return
	}

	if cols[0] == "S" {
		run.Cardinality = CTP_STANDALONE
		run.State = CTP_RUNNING
	} else if cols[0] == "G" {
		run.Cardinality = CTP_GLOBAL
		if cols[2] == "R" {
			run.State = CTP_RUNNING
		} else if cols[2] == "L" {
			run.State = CTP_LOADED
		} else {
			run.State = __State_NIL
			err = fmt.Errorf("cannot parse run state from line: %s", line)
			return
		}
	} else {
		run.Cardinality = __RunCardinality_NIL
		err = fmt.Errorf("cannot parse run cardinality from line: %s", line)
		return
	}

	rn64, err := strconv.ParseUint(cols[1], 10, 32)
	run.RunNumber = uint32(rn64)
	if err != nil {
		err = fmt.Errorf("cannot parse run number from line: %s", line)
		return
	}

	run.Detectors = make([]system.ID, 0)

	detectorsSSlice := strings.Split(cols[3], ",")
	for _, item := range detectorsSSlice {
		// don't interfere with daq
		// also daq not in known system ids -> results in error
		if strings.TrimSpace(item) == "daq" {
			run.Skip = true
			continue
		}

		var det system.ID
		det, err = system.IDString(strings.TrimSpace(item))
		if err != nil {
			err = fmt.Errorf("cannot parse detector %s from line: %s", item, line)
			return
		}
		run.Detectors = append(run.Detectors, det)
	}

	return
}
