/*
 * === This file is part of octl <https://github.com/teo/octl> ===
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


package environment

import (
	"errors"
)

type Transition interface {
	eventName() string
	check() error
	do(*Environment) error
}

// CONFIGURE
type ConfigureTransition struct {
	roleman			*RoleManager
	addRoles		[]string
	removeRoles		[]string
	reconfigureAll	bool
}

func (ConfigureTransition) eventName() string {
	return "CONFIGURE"
}

func (t ConfigureTransition) check() (err error) {
	if t.roleman == nil {
		err = errors.New("cannot configure environment with nil roleman")
	}
	return
}

func (t ConfigureTransition) do(env *Environment) (err error) {
	if env == nil {
		return errors.New("cannot transition in NIL environment")
	}

	// First we free the relevant roles, if any
	if len(t.removeRoles) != 0 {
		rolesThatStay := env.roles[:0]
		rolesToRelease := make([]string, 0)

		for _, role := range env.roles {
			for _, removeRole := range t.removeRoles {
				if role == removeRole {
					rolesToRelease = append(rolesToRelease, role)
					break
				}
				rolesThatStay = append(rolesThatStay, role)
			}
		}
		err = t.roleman.ReleaseRoles(env.Id().Array(), rolesToRelease)
		if err != nil {
			return
		}
		env.roles = rolesThatStay
	}
	// IDEA: instead of passing around m.state or roleman, pass around one or more of
	// roleman's channels. This way roleman could potentially be lockless, and we just pipe
	// him a list of rolenames to remove/add, or even a function or a struct that does so.
	// This struct would implement an interface of the type of his channel, and he could
	// use type assertion to check whether he needs to add, remove or do something else.

	// Alright, so now we have freed some roles (if required).
	// We proceed by deduplicating and attempting an acquire.
	if len(t.addRoles) != 0 {
		rolesToAcquire := make([]string, 0)

		for _, addRole := range t.addRoles {
			for _, role := range env.roles {
				if role == addRole {
					break
				}
				rolesToAcquire = append(rolesToAcquire, addRole)
			}
		}
		err = t.roleman.AcquireRoles(env.Id().Array(), rolesToAcquire)
		if err != nil {
			return
		}
		env.roles = append(env.roles, rolesToAcquire...)
	}

	// Finally, we configure.
	if t.reconfigureAll {
		err = t.roleman.ConfigureRoles(env.Id().Array(), env.roles)
		if err != nil {
			return
		}
	}

	return
}
