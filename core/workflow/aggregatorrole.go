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
	"errors"
	"strings"
	"sync"
	texttemplate "text/template"

	"github.com/AliceO2Group/Control/common/event"
	"github.com/AliceO2Group/Control/configuration/template"
	"github.com/AliceO2Group/Control/core/repos"
	"github.com/AliceO2Group/Control/core/task"
	"github.com/AliceO2Group/Control/core/the"
	"github.com/gobwas/glob"
	"github.com/sirupsen/logrus"
)

type aggregatorRole struct {
	roleBase
	aggregator
}

func (r *aggregatorRole) UnmarshalYAML(unmarshal func(interface{}) error) (err error) {
	// NOTE: see NOTE in roleBase.UnmarshalYAML

	innerRoleBase := roleBase{}
	err = unmarshal(&innerRoleBase)
	if err != nil {
		return
	}

	role := aggregatorRole{
		roleBase: innerRoleBase,
	}
	err = unmarshal(&role.aggregator)
	if err != nil {
		return
	}

	*r = role
	for _, v := range r.Roles {
		v.setParent(r)
	}
	return
}

func (r *aggregatorRole) MarshalYAML() (interface{}, error) {
	aux := make(map[string]interface{})

	auxAggregator, err := r.aggregator.MarshalYAML()
	mapAggregator := auxAggregator.(map[string]interface{})
	for k, v := range mapAggregator {
		aux[k] = v
	}

	auxRoleBase, err := r.roleBase.MarshalYAML()
	mapRoleBase := auxRoleBase.(map[string]interface{})
	for k, v := range mapRoleBase {
		aux[k] = v
	}

	return aux, err
} 

func (r *aggregatorRole) GlobFilter(g glob.Glob) (rs []Role) {
	rs = make([]Role, 0)
	if g.Match(r.GetPath()) {
		rs = append(rs, r)
	}
	for _, chr := range r.Roles {
		chrs := chr.GlobFilter(g)
		if len(chrs) != 0 {
			rs = append(rs, chrs...)
		}
	}
	return
}

func (r *aggregatorRole) ProcessTemplates(workflowRepo repos.IRepo, loadSubworkflow LoadSubworkflowFunc) (err error) {
	if r == nil {
		return errors.New("role tree error when processing templates")
	}

	templSequence := template.Sequence{
		template.STAGE0: template.WrapMapItems(r.Defaults.Raw()),
		template.STAGE1: template.WrapMapItems(r.Vars.Raw()),
		template.STAGE2: template.WrapMapItems(r.UserVars.Raw()),
		template.STAGE3: template.Fields{
			template.WrapPointer(&r.Name),
		},
		template.STAGE4: append(append(
			WrapConstraints(r.Constraints),
			r.wrapBindAndConnectFields()...),
			template.WrapPointer(&r.Enabled)),
	}

	// TODO: push cached templates here
	err = templSequence.Execute(the.ConfSvc(), r.GetPath(), template.VarStack{
		Locals:   r.Locals,
		Defaults: r.Defaults,
		Vars:     r.Vars,
		UserVars: r.UserVars,
	}, r.makeBuildObjectStackFunc(), make(map[string]texttemplate.Template))
	if err != nil {
		return
	}

	// After template processing we write the Locals to Vars in order to make them available to children
	for k, v := range r.Locals {
		r.Vars.Set(k, v)
	}

	r.Enabled = strings.TrimSpace(r.Enabled)

	var wg sync.WaitGroup
	wg.Add(len(r.Roles))

	// Process templates for child roles
	for _, role := range r.Roles {
		go func(role Role) {
			defer wg.Done()
			role.setParent(r)
			err = role.ProcessTemplates(workflowRepo, loadSubworkflow)
			if err != nil {
				return
			}
		}(role)
	}

	wg.Wait()

	// If any child is not Enabled after template resolution,
	// we filter it out of existence
	enabledRoles := make([]Role, 0)
	for _, role := range r.Roles {
		if role.IsEnabled() {
			enabledRoles = append(enabledRoles, role)
		}
	}
	r.Roles = enabledRoles

	return
}

func (r *aggregatorRole) copy() copyable {
	rCopy := aggregatorRole{
		roleBase: *r.roleBase.copy().(*roleBase),
		aggregator: *r.aggregator.copy().(*aggregator),
	}
	for i := 0; i < len(rCopy.Roles); i++ {
		rCopy.Roles[i].setParent(&rCopy)
	}
	return &rCopy
}

func (r *aggregatorRole) setParent(role Updatable) {
	r.parent = role
	r.Defaults.Wrap(role.GetDefaults())
	r.Vars.Wrap(role.GetVars())
	r.UserVars.Wrap(role.GetUserVars())
}

func (r *aggregatorRole) updateStatus(s task.Status) {
	if r == nil {
		return
	}
	log.WithFields(logrus.Fields{
			"child status": s.String(),
			"aggregator status": r.status.get().String(),
			"aggregator role": r.Name,
		}).
		Debug("aggregator role about to merge incoming child status")
	r.status.merge(s, r)
	log.WithField("new status", r.status.get()).Debug("status merged")
	r.SendEvent(&event.RoleEvent{Name: r.Name, Status: r.status.get().String(), RolePath: r.GetPath()})
	if r.parent != nil {
		r.parent.updateStatus(r.status.get())
	}
}

func (r *aggregatorRole) updateState(s task.State) {
	if r == nil {
		return
	}
	log.WithField("role", r.Name).WithField("state", s.String()).Debug("updating state")
	r.state.merge(s, r)
	r.SendEvent(&event.RoleEvent{Name: r.Name, State: r.state.get().String(), RolePath: r.GetPath()})
	if r.parent != nil {
		r.parent.updateState(r.state.get())
	}
}