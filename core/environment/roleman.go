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
	"sync"
	"github.com/pborman/uuid"
	"github.com/AliceO2Group/Control/configuration"
	"errors"
	"fmt"
	"strings"
	"github.com/mesos/mesos-go/api/v1/lib"
	"github.com/AliceO2Group/Control/core/controlcommands"
)

type RoleDeploymentError string
func (r RoleDeploymentError) Error() string {
	return string(r)
}

type RoleManager struct {
	mu                 sync.Mutex
	roleClasses        map[string]*RoleClass
	roster             Roles

	cfgman             configuration.Configuration
	resourceOffersDone <-chan Roles
	rolesToDeploy      chan<- map[string]RoleCfg
	reviveOffersTrg    chan struct{}
	cq                 *controlcommands.CommandQueue
	envman             *EnvManager
}

func NewRoleManager(envman *EnvManager,
					cfgman configuration.Configuration,
                    resourceOffersDone <-chan Roles,
                    rolesToDeploy chan<- map[string]RoleCfg,
                    reviveOffersTrg chan struct{},
                    cq *controlcommands.CommandQueue) *RoleManager {
	envman.roleman = &RoleManager{
		roleClasses: make(map[string]*RoleClass),
		roster:      make(Roles),
		cfgman:      cfgman,
		resourceOffersDone: resourceOffersDone,
		rolesToDeploy: rolesToDeploy,
		reviveOffersTrg: reviveOffersTrg,
		cq:          cq,
		envman:      envman,
	}
	return envman.roleman
}

// RoleForMesosOffer accepts a Mesos offer and a RoleCfg and returns a newly
// constructed Role.
// This function should only be called by the Mesos scheduler controller when
// matching role requests with offers (matchRoles).
// The new role is not assigned to an environment and comes without a roleClass
// function, as those two are filled out later on by RoleManager.AcquireRoles.
func (m* RoleManager) RoleForMesosOffer(offer *mesos.Offer, roleCfg *RoleCfg) (role *Role) {
	role = &Role{
		name:          roleCfg.Name,
		roleClassName: roleCfg.RoleClass,
		configuration: *roleCfg,
		hostname:      offer.Hostname,
		agentId:       offer.AgentID.Value,
		offerId:       offer.ID.Value,
		taskId:        uuid.NewUUID().String(),
		executorId:    uuid.NewUUID().String(),
		envId:         nil,
		roleClass:     nil,
	}
	role.roleClass = func() *RoleClass {
		return m.GetRoleClass(role.roleClassName)
	}
	return
}

func (m *RoleManager) RefreshRoleClasses() (err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	roleClassesMap, err := m.cfgman.GetRecursive("o2/roleclasses")
	if err != nil {
		return
	}
	if roleClassesMap == nil {
		err = errors.New("no base roles found in configuration")
		return
	}

	cfgErr := errors.New("bad configuration for some roleClasses")

	errs := make([]string, 0)
	for k, v := range roleClassesMap {
		if !v.IsMap() {
			err = cfgErr
			continue
		}
		vMap := v.Map()
		roleClass, err := roleClassFromConfiguration(k, vMap)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}

		m.roleClasses[k] = roleClass
	}
	if len(errs) > 0 {
		err = errors.New(fmt.Sprintf("%s, errors: %s",
			cfgErr.Error(), strings.Join(errs, "; ")))
	}
	return
}

func (m *RoleManager) AcquireRoles(envId uuid.Array, roleNames []string) (err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	/*
	Here's what's gonna happen:
	1) get configuration for all roleNames
	2) check if any of them are already in Roster, whether they are already locked in an environment,
	   and whether their config is not stale
		2a) if at least one is found that's locked in another env, bail
		2b) for each of them in Roster with stale config, teardown and remove from Roster
		2b) for each of them not in Roster (including the ones we just stopped), add to rolesToRun
	3) teardown the roles in rolesToTeardown
	4) start the roles in rolesToRun
	5) ensure that all of them are in a CONFIGURED state now
	*/

	rolesToRun := make(map[string]RoleCfg, 0)
	rolesToTeardown := make([]string, 0)
	rolesAlreadyRunning := make([]string, 0)
	for _, roleName := range roleNames {
		cfgMap, err := m.cfgman.GetRecursive(fmt.Sprintf("o2/roles/%s", roleName))
		if err != nil {
			return errors.New(fmt.Sprintf("cannot fetch role configuration: %s", err.Error()))
		}

		roleCfg, err := roleCfgFromConfiguration(roleName, cfgMap)
		if err != nil {
			return errors.New(fmt.Sprintf("bad role configuration for %s: %s", roleName, err.Error()))
		}

		if m.roster.Contains(roleName) {
			if m.roster[roleName].IsLocked() {
				return errors.New(fmt.Sprintf("role %s already in use, cannot configure environment %s", roleName, envId.String()))
			}

			if m.roster[roleName].configuration.Equals(roleCfg) {
				rolesAlreadyRunning = append(rolesAlreadyRunning, roleName)
			} else {
				// ↑ the RoleCfg has changed, we need to teardown
				//   and restart
				rolesToTeardown = append(rolesToTeardown, roleName)
				rolesToRun[roleName] = *roleCfg
			}
		} else {
			rolesToRun[roleName] = *roleCfg
		}
	}

	if len(rolesToTeardown) > 0 {
		err = m.TeardownRoles(rolesToTeardown)
		if err != nil {
			return errors.New(fmt.Sprintf("cannot restart roles: %s", err.Error()))
		}
	}

	deploymentSuccess := true // hopefully

	deployedRoles := make(Roles, 0)
	if len(rolesToRun) > 0 {
		// Teardown done, now we
		// 4a) ask Mesos to revive offers and block until done
		// 4b) ask Mesos to run the required roles - if any.

		m.reviveOffersTrg <- struct{}{} // signal scheduler to revive offers
		<- m.reviveOffersTrg            // we only continue when it's done

		m.rolesToDeploy <- rolesToRun // blocks until received
		log.WithField("environmentId", envId).
			Debug("scheduler should have received request to deploy")

		// IDEA: a flps mesos-role assigned to all mesos agents on flp hosts, and then a static
		//       reservation for that mesos-role on behalf of our scheduler

		deployedRoles = <- m.resourceOffersDone
		log.WithField("roles", deployedRoles).
			Debug("resourceOffers is done, new roles running")

		if len(deployedRoles) != len(rolesToRun) {
			// ↑ Not all roles could be deployed. We cannot proceed
			//   with running this environment, but we keep the roles
			//   running since they might be useful in the future.
			deploymentSuccess = false
		}
	}

	if deploymentSuccess {
		// ↑ means all the required processes are now running, and we
		//   are ready to update the envId
		for _, role := range deployedRoles {
			role.envId = new(uuid.UUID)
			*role.envId = envId.UUID()
			// Ensure everything is filled out properly
			if !role.IsLocked() {
				log.WithField("role", role.name).Warning("cannot lock newly deployed role")
				deploymentSuccess = false
			}
		}
	}
	if !deploymentSuccess {
		// While all the required roles are running, for some reason we
		// can't lock some of them, so we must roll back and keep them
		// unlocked in the roster.
		for _, role := range deployedRoles {
			role.envId = nil
		}
		err = RoleDeploymentError("cannot deploy required roles")
	}

	// Finally, we write to the roster. Point of no return!
	for _, role := range deployedRoles {
		m.roster[role.name] = role
	}
	if deploymentSuccess {
		for _, roleName := range rolesAlreadyRunning {
			*m.roster[roleName].envId = envId.UUID()
		}
	}

	return
}

func (m *RoleManager) TeardownRoles(roleNames []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	//TODO: implement

	return nil
}

func (m *RoleManager) ReleaseRoles(envId uuid.Array, roleNames []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	//TODO: implement

	return nil
}

func (m *RoleManager) ConfigureRoles(envId uuid.Array, roleNames []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	notify := make(chan controlcommands.Response)
	receivers := make([]controlcommands.MesosCommandReceiver, 0)
	for _, roleName := range roleNames {
		role := m.roster[roleName]
		if role == nil {
			return errors.New(fmt.Sprintf("cannot configure role %s: not in roster", roleName))
		}
		if !role.IsLocked() {
			return errors.New(fmt.Sprintf("role %s is not locked, cannot send control commands", role.GetName()))
		}
		receivers = append(receivers, controlcommands.MesosCommandReceiver{
			AgentId: mesos.AgentID{
				Value: role.GetAgentId(),
			},
			ExecutorId: mesos.ExecutorID{
				Value: role.GetExecutorId(),
			},
		})
	}

	env, err := m.envman.Environment(envId.UUID())
	if err != nil {
		return err
	}
	src := env.CurrentState()
	event := "CONFIGURE"
	dest := "CONFIGURED"
	args := make(map[string]string)

	// FIXME: fetch configuration from Consul here and put it in args

	cmd := controlcommands.NewMesosCommand_Transition(receivers, src, event, dest, args)
	m.cq.Enqueue(cmd, notify)

	response := <- notify
	close(notify)

	errText := response.Error()
	if len(strings.TrimSpace(errText)) != 0 {
		return errors.New(response.Error())
	}

	// FIXME: improve error handling ↑

	return nil
}

func (m *RoleManager) GetRoleClass(name string) (b *RoleClass) {
	if m == nil {
		return
	}
	b = m.roleClasses[name]
	return
}

func (m *RoleManager) RoleCount() int {
	if m == nil {
		return -1
	}
	return len(m.roster)
}

func (m *RoleManager) GetRoles() Roles {
	if m == nil {
		return nil
	}
	return m.roster
}

type Roles map[string]*Role

func (m Roles) Contains(roleName string) bool {
	for k, _ := range m {
		if k == roleName {
			return true
		}
	}
	return false
}

func (m Roles) RolesForBase(roleClassName string) (roles Roles) {
	return m.Filtered(func(k *string, v *Role) bool {
		if v == nil {
			return false
		}
		return v.roleClassName == roleClassName
	})
}

func (m Roles) Filtered(filterFunc func(*string, *Role) bool) (roles Roles) {
	if m == nil {
		return nil
	}
	roles = make(Roles)
	for i, j := range m {
		if filterFunc(&i, j) {
			roles[i] = j
		}
	}
	return roles
}
