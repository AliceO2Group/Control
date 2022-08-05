/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2019 CERN and copyright holders of ALICE O².
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

package executable

import (
	"github.com/AliceO2Group/Control/executor/executorcmd/transitioner"
)

type BasicTask struct {
	basicTaskBase
}

func (t *BasicTask) makeTransitionFunc() transitioner.DoTransitionFunc {
	// If it's a basic task role, we make a RUNNING-state based transition function
	// otherwise we process the hooks spec.
	return func(ei transitioner.EventInfo) (newState string, err error) {
		log.WithField("partition", t.knownEnvironmentId.String()).
			WithField("detector", t.knownDetector).
			WithField("event", ei.Evt).
			Debug("executor basic task transitioner requesting transition")

		switch {
		case ei.Src == "CONFIGURED" && ei.Evt == "START" && ei.Dst == "RUNNING":
			err = t.startBasicTask()
			if err != nil {
				return "CONFIGURED", err
			}
			return "RUNNING", err
		case ei.Src == "RUNNING" && ei.Evt == "STOP" && ei.Dst == "CONFIGURED":
			err = t.ensureBasicTaskKilled()
			return "CONFIGURED", err
		default:
			// By default we declare any transition as valid and executed as NOOP
			return ei.Dst, nil
		}
	}
}

func (t *BasicTask) Launch() error {
	return t.doLaunch(t.makeTransitionFunc())
}
