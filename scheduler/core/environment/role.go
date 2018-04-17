/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2017-2018 CERN and copyright holders of ALICE O².
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
	"github.com/AliceO2Group/Control/common"
	"github.com/AliceO2Group/Control/configuration"
	"strconv"
	"errors"
	"strings"
	"github.com/pborman/uuid"
)


type RoleClass roleInfo

func parsePortRanges(str string) (ranges Ranges, err error) {
	r := make(Ranges, 0)
	if len(strings.TrimSpace(str)) == 0 {
		return
	}

	split := strings.Split(str, ",")
	for _, s := range split {
		trimmed := strings.TrimSpace(s)
		rangeSplit := strings.Split(trimmed, "-")
		if len(rangeSplit) == 1 { // single port range
			var port uint64
			port, err = strconv.ParseUint(rangeSplit[0], 10, 64)
			if err != nil {
				return
			}
			r = append(r, Range{Begin: port, End: port})
			continue
		} else if len(rangeSplit) == 2 { //normal range
			var begin, end uint64
			begin, err = strconv.ParseUint(rangeSplit[0], 10, 64)
			if err != nil {
				return
			}
			end, err = strconv.ParseUint(rangeSplit[0], 10, 64)
			if err != nil {
				return
			}
			r = append(r, Range{Begin: begin, End: end})
			continue
		} else {
			err = errors.New("bad format for roleClass ports range")
			return
		}
	}
	ranges = r
	return
}

func roleClassFromConfiguration(name string, cfgMap configuration.Map) (roleClass *RoleClass, err error) {
	ri, err := roleInfoFromConfiguration(name, cfgMap, true)
	if err != nil {
		return
	}
	roleClass = (*RoleClass)(ri)
	return
}

type Role struct {
	name			string
	roleClassName	string
	configuration	RoleCfg
	hostname		string
	agentId			string
	offerId			string
	taskId			string
	envId			*uuid.UUID

	roleClass        func() *RoleClass
	// ↑ to be filled in by RoleForMesosOffer in RoleManager
}

func (m Role) IsLocked() bool {
	return len(m.hostname) > 0 &&
		   len(m.agentId) > 0 &&
		   len(m.offerId) > 0 &&
		   len(m.taskId) > 0 &&
		   m.envId != nil &&
		   !uuid.Equal(*m.envId, uuid.NIL)
}

func (m *Role) GetName() string {
	if m != nil {
		return m.name
	}
	return ""
}

func (m *Role) GetRoleClassName() string {
	if m != nil {
		return m.roleClassName
	}
	return ""
}

// Returns a consolidated CommandInfo for this Role, based on RoleCfg and
// RoleClass.
func (m Role) GetCommand() (cmd *common.CommandInfo) {
	if roleClass := m.roleClass(); roleClass != nil {
		cmd = roleClass.Command.Copy()
	} else {
		cmd = &common.CommandInfo{}
	}
	if m.configuration.Command != nil {
		if m.configuration.Command != nil {
			cmd.UpdateFrom(m.configuration.Command)
		}

		cmd.Env = append(cmd.Env, m.configuration.CmdExtraEnv...)
		cmd.Arguments = append(cmd.Arguments, m.configuration.CmdExtraArguments...)

		if cmd.Value == nil || len(*cmd.Value) == 0 {
			log.WithField("command", cmd).
				WithField("error", "built nil- or empty-Value CommandInfo object, this cannot possibly be executed by the executor").
				Error("invalid CommandInfo, returning nil")
			return nil
		}
		return
	}
	return
}

func (m *Role) GetWantsCPU() float64 {
	if m != nil {
		if m.configuration.WantsCPU != nil {
			return *m.configuration.WantsCPU
		} else if roleClass := m.roleClass(); roleClass != nil {
			return *roleClass.WantsCPU
		}
	}
	return -1
}

func (m *Role) GetWantsMemory() float64 {
	if m != nil {
		if m.configuration.WantsMemory != nil {
			return *m.configuration.WantsMemory
		} else if roleClass := m.roleClass(); roleClass != nil {
			return *roleClass.WantsMemory
		}
	}
	return -1
}

func (m *Role) GetWantsPorts() Ranges {
	if m != nil {
		if m.configuration.WantsPorts != nil {
			wantsPorts := make(Ranges, len(m.configuration.WantsPorts))
			copy(wantsPorts, m.configuration.WantsPorts)
			return wantsPorts
		} else if roleClass := m.roleClass(); roleClass != nil {
			wantsPorts := make(Ranges, len(roleClass.WantsPorts))
			copy(wantsPorts, roleClass.WantsPorts)
			return wantsPorts
		}
	}
	return nil
}

func (m Role) GetOfferId() string {
	return m.offerId
}

func (m Role) GetTaskId() string {
	return m.taskId
}

func (m Role) GetHostname() string {
	return m.hostname
}