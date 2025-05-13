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
	"strconv"
	"strings"
	"sync"

	"github.com/AliceO2Group/Control/core/task"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type SafeStatus struct {
	mu     sync.RWMutex
	status task.Status
}

func reportTraceRoles(roles []Role, status task.Status) {
	if viper.GetBool("veryVerbose") {
		stati := make([]string, len(roles))
		critical := make([]string, len(roles))
		names := make([]string, len(roles))
		for i, role := range roles {
			stati[i] = role.GetStatus().String()
			names[i] = role.GetName()
			if taskR, isTaskRole := role.(*taskRole); isTaskRole {
				critical[i] = strconv.FormatBool(taskR.IsCritical())
			} else if callR, isCallRole := role.(*callRole); isCallRole {
				critical[i] = strconv.FormatBool(callR.IsCritical())
			} else {
				critical[i] = strconv.FormatBool(true)
			}
		}
		log.WithFields(logrus.Fields{
			"statuses":   strings.Join(stati, ", "),
			"critical":   strings.Join(critical, ", "),
			"names":      strings.Join(names, ", "),
			"aggregated": status.String(),
		}).
			Trace("aggregating statuses")
	}
}

// role that are not taskRole or callRole are critical by default
func aggregateStatus(roles []Role) (status task.Status) {
	if len(roles) == 0 {
		status = task.UNDEFINED
		return
	}

	status = task.INVARIANT
	for _, role := range roles {
		if status == task.UNDEFINED {
			break
		}

		if taskR, isTaskRole := role.(*taskRole); isTaskRole {
			if !taskR.IsCritical() {
				continue
			}
		} else if callR, isCallRole := role.(*callRole); isCallRole {
			if !callR.IsCritical() {
				continue
			}
		}
		status = status.X(role.GetStatus())
	}

	reportTraceRoles(roles, status)

	return
}

// TODO: this function is prime candidate for refactoring. The reason being that it mostly ignores status argument
// for merging, moreover it also does not use status of role from argument. Both of these behaivour are counter-intuitive.
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
