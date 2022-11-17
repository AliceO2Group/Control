/*
 * === This file is part of ALICE O² ===
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

package workflow

import (
	"github.com/AliceO2Group/Control/core/task"
	"sync"
)

type SafeState struct {
	mu    sync.RWMutex
	state task.State
}

// //////////////
// CHECK HERE //
// //////////////
// Aggregate the state of multiple tasks using the "task/state.go" X function
func aggregateState(roles []Role) (state task.State) {
	if len(roles) == 0 {
		state = task.INVARIANT
		return
	}
	state = roles[0].GetState()
	if len(roles) > 1 {
		for _, c := range roles[1:] {
			if state == task.MIXED {
				return
			}
			if state == task.ERROR {
				return
			}
			state = state.X(c.GetState())
		}
	}
	return
}

// //////////////
// CHECK HERE //
// //////////////
func (t *SafeState) merge(s task.State, r Role) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.state == s {
		return
	}

	_, isTaskRole := r.(*taskRole)
	_, isCallRole := r.(*callRole)
	if isTaskRole || isCallRole { // no aggregation, we just update
		t.state = s
		return
	}

	switch {
	case s == task.MIXED:
		t.state = task.MIXED
		return
	case s == task.ERROR:
		t.state = task.ERROR
		return
	default:
		allRoles := r.GetRoles()
		t.state = aggregateState(allRoles)
	}
}

func (t *SafeState) get() task.State {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.state
}
