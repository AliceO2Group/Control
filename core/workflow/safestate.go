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
	"github.com/AliceO2Group/Control/core/task/sm"

	"sync"
)

// SafeState is a thread-safe structure that holds the state of a role.
type SafeState struct {
	mu    sync.RWMutex
	state sm.State
}

func aggregateState(roles []Role) (s sm.State) {
	s = sm.INVARIANT
	if len(roles) == 0 {
		return
	}
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
		s = s.X(c.GetState())
	}
	return
}

func (t *SafeState) merge(s sm.State, r Role) {
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
	case s == sm.MIXED && t.state != sm.ERROR:
		t.state = sm.MIXED
		return
	case s == sm.ERROR:
		t.state = sm.ERROR
		return
	default:
		allRoles := r.GetRoles()
		t.state = aggregateState(allRoles)
	}
}

func (t *SafeState) get() sm.State {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.state
}
