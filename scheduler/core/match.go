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

package core

import (
	"github.com/mesos/mesos-go/api/v1/lib"
	"github.com/sirupsen/logrus"
	"errors"
	"github.com/teo/octl/scheduler/core/environment"
	"fmt"
)


// indexOfAttribute accepts as arguments a slice of Mesos attributes and a string with a
// desired attribute name. It returns the index of the attribute with that name, or -1
// if not found in the input slice.
func indexOfAttribute(attributes []mesos.Attribute, attributeName string) (index int) {
	index = -1
	for i, a := range attributes {
		if a.GetName() == attributeName {
			index = i
			return
		}
	}
	return
}

// indexOfOfferForO2Role accepts as arguments a slice of Mesos offer values and a string
// with an O² role name. It returns the index of the offer whose "o2role" attribute
// matches the given role name, or -1 if not found.
func indexOfOfferForO2Role(offers []mesos.Offer, roleCfg *environment.RoleCfg) (index int) {
	index = -1
	for i, offer := range offers {
		if attrIdx := indexOfAttribute(offer.Attributes, "o2role"); attrIdx > -1 {
			if offer.Attributes[attrIdx].GetText().GetValue() == roleCfg.Name {
				// We have a role-offer match by name!
				index = i
				return
			}
		}
	}

	// TODO: when Contstraints are implemented in RoleCfg, this is where we'll implement
	//       different constraint-matching behaviors, e.g. match o2roleclass but not o2role,
	//       or other criteria.
	//       For now, we only do a full o2role match.
	return
}

// matchRoles takes in a map of RoleCfgs and Mesos offers, tries to find a complete
// match between them, and returns a slice of used offers, a slice of unused offers, a
// slice of newly created Roles for the given RoleCfgs, and an error value.
// If err != nil, the other return values are still valid.
func matchRoles(roleman *environment.RoleManager, roleCfgsToDeploy map[string]environment.RoleCfg, offers []mesos.Offer) (
	offersUsed []mesos.Offer, offersLeft []mesos.Offer, rolesDeployed environment.Roles, err error) {
	rolesDeployed = make(environment.Roles)

	offersLeft = make([]mesos.Offer, len(offers))
	copy(offersLeft, offers)

	// for roleCfg in roleCfgsToDeploy:
	// 1) find the first match for o2roleclass and o2role
	//    NOTE: each mesos.Offer machine might have multiple o2roleclass and o2role entries!
	//    TODO: figure out if these entries are JSON-lists inside one item or multiple items

	for roleName, roleCfg := range roleCfgsToDeploy {
		if index := indexOfOfferForO2Role(offers, &roleCfg); index > -1 {
			offer := offersLeft[index]
			rolesDeployed[roleName] = roleman.RoleForMesosOffer(&offer, &roleCfg)
			offersUsed = append(offersUsed, offer)
			// ↑ We are accepting an offer, so we must add it to the accepted list
			// ↓ and we must remove it from the offers list since it's just been claimed.
			offersLeft = append(offersLeft[:index], offersLeft[index + 1:]...)
		} else {
			msg := fmt.Sprintf("offer not found for some O² roles")
			log.WithFields(logrus.Fields{
				"roleName": 		roleName,
				"roleClass":		roleCfg.RoleClass,
			}).Error(msg)
			err = errors.New(msg)
		}
	}
	return
}
