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

package environment

import (
	"fmt"
	"strings"
	"github.com/pborman/uuid"
)

type RoleError interface {
	error
	GetRoleName() string
}

type RolesError interface {
	error
	GetRoleNames() []string
}

type roleErrorBase struct {
	roleName string
}
func (r roleErrorBase) GetRoleName() string {
	return r.roleName
}

type rolesErrorBase struct {
	roleNames []string
}
func (r rolesErrorBase) GetRoleNames() []string {
	return r.roleNames
}

type GenericRoleError struct {
	roleErrorBase
	message string
}
func (r GenericRoleError) Error() string {
	return fmt.Sprintf("role %s error: %s", r.roleName, r.message)
}

type GenericRolesError struct {
	rolesErrorBase
	message string
}
func (r GenericRolesError) Error() string {
	return fmt.Sprintf("roles [%s] error: %s", strings.Join(r.roleNames, ", "), r.message)
}

type RolesDeploymentError rolesErrorBase
func (r RolesDeploymentError) Error() string {
	return fmt.Sprintf("deployment failed for roles [%s]", r.roleNames)
}

type RoleAlreadyReleasedError roleErrorBase
func (r RoleAlreadyReleasedError) Error() string {
	return fmt.Sprintf("role %s already released", r.roleName)
}

type RoleNotFoundError roleErrorBase
func (r RoleNotFoundError) Error() string {
	return fmt.Sprintf("role %s not found", r.roleName)
}

type RoleLockedError struct {
	roleErrorBase
	envId uuid.Array
}
func (r RoleLockedError) Error() string {
	return fmt.Sprintf("role %s is locked by environment %s", r.roleName, r.envId)
}
func (r RoleLockedError) EnvironmentId() uuid.Array {
	return r.envId
}