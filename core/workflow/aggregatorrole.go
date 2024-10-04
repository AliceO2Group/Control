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

	"github.com/AliceO2Group/Control/common/gera"

	"github.com/AliceO2Group/Control/common/event"
	"github.com/AliceO2Group/Control/common/event/topic"
	pb "github.com/AliceO2Group/Control/common/protos"
	"github.com/AliceO2Group/Control/configuration/template"
	"github.com/AliceO2Group/Control/core/repos"
	"github.com/AliceO2Group/Control/core/task"
	"github.com/AliceO2Group/Control/core/task/sm"
	"github.com/AliceO2Group/Control/core/the"
	"github.com/gobwas/glob"
	"github.com/hashicorp/go-multierror"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type aggregatorRole struct {
	roleBase
	aggregator
}

func NewAggregatorRole(name string, roles []Role) (r Role) {
	return &aggregatorRole{
		roleBase: roleBase{
			Name:     name,
			Defaults: gera.MakeMap[string, string](),
			Vars:     gera.MakeMap[string, string](),
			UserVars: gera.MakeMap[string, string](),
		},
		aggregator: aggregator{Roles: roles},
	}
}

func (r *aggregatorRole) UnmarshalYAML(unmarshal func(interface{}) error) (err error) {
	// NOTE: see NOTE in roleBase.UnmarshalYAML

	innerRoleBase := roleBase{
		Defaults: gera.MakeMap[string, string]().WithUnmarshalYAML(kvStoreUnmarshalYAMLWithTags),
		Vars:     gera.MakeMap[string, string]().WithUnmarshalYAML(kvStoreUnmarshalYAMLWithTags),
		UserVars: gera.MakeMap[string, string]().WithUnmarshalYAML(kvStoreUnmarshalYAMLWithTags),
	}
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

func (r *aggregatorRole) ProcessTemplates(workflowRepo repos.IRepo, loadSubworkflow LoadSubworkflowFunc, baseConfigStack map[string]string) (err error) {
	if r == nil {
		return errors.New("role tree error when processing templates")
	}

	templSequence := template.Sequence{
		template.STAGE0: template.Fields{
			template.WrapPointer(&r.Enabled),
		},
		template.STAGE1: template.WrapMapItems(r.Defaults.Raw()),
		template.STAGE2: template.WrapMapItems(r.Vars.Raw()),
		template.STAGE3: template.WrapMapItems(r.UserVars.Raw()),
		template.STAGE4: template.Fields{
			template.WrapPointer(&r.Name),
		},
		template.STAGE5: append(append(
			WrapConstraints(r.Constraints),
			r.wrapBindAndConnectFields()...)),
	}

	// TODO: push cached templates here
	err = templSequence.Execute(
		the.ConfSvc(),
		r.GetPath(),
		template.VarStack{
			Locals:   r.Locals,
			Defaults: r.Defaults,
			Vars:     r.Vars,
			UserVars: r.UserVars,
		},
		r.makeBuildObjectStackFunc(),
		baseConfigStack,
		make(map[string]texttemplate.Template),
		workflowRepo,
		MakeDisabledRoleCallback(r),
	)
	if err != nil {
		var roleDisabledErrorType *template.RoleDisabledError
		if isRoleDisabled := errors.As(err, &roleDisabledErrorType); isRoleDisabled {
			log.WithField("partition", r.GetEnvironmentId().String()).
				Trace(err.Error())
			err = nil // we don't want a disabled role to be considered an error
		} else {
			return
		}
	}

	// After template processing we write the Locals to Vars in order to make them available to children
	for k, v := range r.Locals {
		r.Vars.Set(k, v)
	}

	r.Enabled = strings.TrimSpace(r.Enabled)

	// TREE PRUNING: we don't continue with children if this role is disabled
	if !r.IsEnabled() {
		r.Roles = make([]Role, 0)
	}

	// Process templates for child roles (concurrently + fallback)
	concurrency := viper.GetBool("concurrentWorkflowTemplateProcessing")

	if concurrency {
		var wg sync.WaitGroup
		wg.Add(len(r.Roles))

		var roleErrors *multierror.Error

		// Process templates for child roles
		for roleIdx, _ := range r.Roles {
			go func(roleIdx int) {
				defer wg.Done()
				role := r.Roles[roleIdx]
				role.setParent(r)
				err := role.ProcessTemplates(workflowRepo, loadSubworkflow, baseConfigStack)
				if err != nil {
					roleErrors = multierror.Append(roleErrors, err)
				}
			}(roleIdx)
		}
		wg.Wait()

		err = roleErrors.ErrorOrNil() // only return error if at least one error was accumulated, otherwise nil
		if err != nil {
			return
		}

	} else {
		for _, role := range r.Roles {
			role.setParent(r)
			err = role.ProcessTemplates(workflowRepo, loadSubworkflow, baseConfigStack)
			if err != nil {
				return
			}
		}
	}

	// If any child is not Enabled after template resolution,
	// we filter it out of existence
	enabledRoles := make([]Role, 0)
	for _, role := range r.Roles {
		if role.IsEnabled() {
			enabledRoles = append(enabledRoles, role)
		}
	}
	r.Roles = enabledRoles

	// If there are no roles in the aggregator role, it has no use and should be disabled
	if len(r.Roles) == 0 {
		r.Enabled = "false"
	}

	return
}

func (r *aggregatorRole) IsCritical() bool {
	if r == nil {
		return false
	}
	critical := false
	for _, role := range r.Roles {
		critical = critical || role.IsCritical()
	}
	return critical
}

func (r *aggregatorRole) copy() copyable {
	rCopy := aggregatorRole{
		roleBase:   *r.roleBase.copy().(*roleBase),
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
	oldStatus := r.status.get()
	log.WithFields(logrus.Fields{
		"child status":      s.String(),
		"aggregator status": r.status.get().String(),
		"aggregator role":   r.Name,
		"partition":         r.GetEnvironmentId().String(),
	}).
		Trace("aggregator role about to merge incoming child status")
	r.status.merge(s, r)
	log.WithField("new status", r.status.get()).Trace("status merged")

	if oldStatus != r.status.get() {
		the.EventWriterWithTopic(topic.Role).WriteEvent(&pb.Ev_RoleEvent{
			Name:          r.Name,
			Status:        r.status.get().String(),
			RolePath:      r.GetPath(),
			EnvironmentId: r.GetEnvironmentId().String(),
		})
	}
	r.SendEvent(&event.RoleEvent{Name: r.Name, Status: r.status.get().String(), RolePath: r.GetPath()})
	if r.parent != nil {
		r.parent.updateStatus(r.status.get())
	}
}

func (r *aggregatorRole) updateState(s sm.State) {
	if r == nil {
		return
	}
	oldState := r.state.get()
	r.state.merge(s, r)
	log.WithField("role", r.Name).
		WithField("partition", r.GetEnvironmentId().String()).
		Tracef("updated state to %s upon input state %s", r.state.get().String(), s.String())

	if oldState != r.state.get() {
		the.EventWriterWithTopic(topic.Role).WriteEvent(&pb.Ev_RoleEvent{
			Name:          r.Name,
			State:         r.state.get().String(),
			RolePath:      r.GetPath(),
			EnvironmentId: r.GetEnvironmentId().String(),
		})
	}
	r.SendEvent(&event.RoleEvent{Name: r.Name, State: r.state.get().String(), RolePath: r.GetPath()})
	if r.parent != nil {
		r.parent.updateState(r.state.get())
	}
}
