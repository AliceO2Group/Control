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

func aggregateState(roles []Role) (state task.State) {
	state = task.INVARIANT
	if len(roles) == 0 {
		return
	}
	var rolesToCheck = make([]Role, 0)
	for _, c := range roles {
		taskR, isTaskRole := c.(*taskRole)
		callR, isCallRole := c.(*callRole)
		if isTaskRole {
			if !taskR.Critical {
				continue
			}
		} else if isCallRole {
			if !callR.Critical {
				continue
			}
		}
		state = state.X(c.GetState())
		if state == task.MIXED {
			rolesToCheck = append(rolesToCheck, c)
		}
		if state == task.ERROR {
			rolesToCheck = append(rolesToCheck, c)
		}
	}
	if len(rolesToCheck) > 0 {
		checkedState := state
		for _, c := range rolesToCheck {
			checkedState = checkedState.X(c.GetState())
		}
		state = checkedState
	}
	return
}

func (t *SafeState) merge(s task.State, r Role) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.state == s {
		return
	}

	_, isTaskRole := r.(*taskRole)
	_, isCallRole := r.(*callRole)
	if isTaskRole || isCallRole {
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
