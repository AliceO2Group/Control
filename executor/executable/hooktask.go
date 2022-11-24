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

package executable

import (
	"github.com/AliceO2Group/Control/executor/executorcmd/transitioner"
)

type HookTask struct {
	basicTaskBase
}

func (t *HookTask) makeTransitionFunc() transitioner.DoTransitionFunc {
	// NOOP transition function, because hooks don't obey any transition
	return func(ei transitioner.EventInfo) (newState string, err error) {
		log.WithField("partition", t.knownEnvironmentId.String()).
			WithField("detector", t.knownDetector).
			WithField("event", ei.Evt).
			Debug("executor hook task transitioner requesting transition")

		return ei.Dst, nil
	}
}

func (t *HookTask) Launch() error {
	return t.doLaunch(t.makeTransitionFunc())
}

func (t *HookTask) Trigger() error {
	return t.startBasicTask()
}
