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
	"strings"
	"sync"

	"github.com/AliceO2Group/Control/core/task"
	"github.com/sirupsen/logrus"
)

type SafeStatus struct {
	mu     sync.RWMutex
	status task.Status
}

func aggregateStatus(roles []Role) (status task.Status) {
	if len(roles) == 0 {
		status = task.UNDEFINED
		return
	}
	stati := make([]string, len(roles))
	for i, role := range roles {
		stati[i] = role.GetStatus().String()
	}

	status = roles[0].GetStatus()
	if len(roles) > 1 {
		for _, c := range roles[1:] {
			if status == task.UNDEFINED {
				log.WithFields(logrus.Fields{
					"statuses":   strings.Join(stati, ", "),
					"aggregated": status.String(),
				}).
					Trace("aggregating statuses")

				return
			}
			status = status.X(c.GetStatus())
		}
	}
	log.WithFields(logrus.Fields{
		"statuses":   strings.Join(stati, ", "),
		"aggregated": status.String(),
	}).
		Trace("aggregating statuses")

	return
}

func (t *SafeStatus) merge(s task.Status, r Role) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.status == s {
		return
	}

	_, isTaskRole := r.(*taskRole)
	_, isCallRole := r.(*callRole)
	if isTaskRole || isCallRole { // no aggregation, we just update
		t.status = s
		return
	}

	switch {
	case s == task.UNDEFINED: // if we get a new UNDEFINED status, the whole role is UNDEFINED
		t.status = task.UNDEFINED
		return
	default:
		allRoles := r.GetRoles()
		t.status = aggregateStatus(allRoles)
	}
}

func (t *SafeStatus) get() task.Status {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.status
}
