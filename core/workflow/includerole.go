/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2021 CERN and copyright holders of ALICE O².
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
	texttemplate "text/template"

	"github.com/AliceO2Group/Control/configuration/template"
	"github.com/AliceO2Group/Control/core/repos"
	"github.com/AliceO2Group/Control/core/task"
	"github.com/AliceO2Group/Control/core/task/sm"
	"github.com/AliceO2Group/Control/core/the"
)

// An includeRole is a delayed aggregatorRole
// It takes an `include` entry, which is then expanded into a full aggregatorRole
// during the ProcessTemplates function.
type includeRole struct {
	aggregatorRole

	Include string `yaml:"include,omitempty"`
}

func (r *includeRole) UnmarshalYAML(unmarshal func(interface{}) error) (err error) {
	// NOTE: see NOTE in roleBase.UnmarshalYAML

	innerRoleBase := roleBase{}
	err = unmarshal(&innerRoleBase)
	if err != nil {
		return
	}

	type auxInclude struct {
		Include string `yaml:"include,omitempty"`
	}
	_auxInclude := auxInclude{}
	err = unmarshal(&_auxInclude)
	if err != nil {
		return
	}

	role := includeRole{
		aggregatorRole: aggregatorRole{
			roleBase:   innerRoleBase,
			aggregator: aggregator{},
		},
		Include: _auxInclude.Include,
	}

	*r = role
	return
}

func (r *includeRole) MarshalYAML() (interface{}, error) {
	return nil, nil
}

func (r *includeRole) ProcessTemplates(workflowRepo repos.IRepo, loadSubworkflow LoadSubworkflowFunc, baseConfigStack map[string]string) (err error) {
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
			template.WrapPointer(&r.Include),
		},
		template.STAGE5: append(append(
			WrapConstraints(r.Constraints),
			r.wrapBindAndConnectFields()...),
			template.WrapPointer(&r.Enabled)),
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
			log.WithField("partition", r.GetEnvironmentId().String()).Trace(err.Error())
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

	if !r.IsEnabled() {
		// Normally it's the parent that checks whether a child is enabled after that child's
		// ProcessTemplates is done, and this is true for the current role as well, but if
		// at this point we already know that the current role is disabled (and will
		// therefore be excluded by its parent), we don't try to resolve the workflow
		// inclusion because there's no reason to do so anyway.
		return nil
	}

	// Common part done, include resolution starts here.
	// An includeRole is essentially a baseRole + `include:` expression. We first need to resolve
	// the expression to obtain a full subworkflow template identifier, i.e. a full repo/wft/branch
	// combo.
	// Once that's done we can load the subworkflow and obtain the root `aggregatorRole` plus a new
	// repos.Repo definition. If a repo or branch is already provided in the subworkflow expression
	// then the returned newWfRepo will reflect this, and any additionally nested includes will
	// default to the repo of their direct parent.
	var subWfRoot *aggregatorRole
	var newWfRepo repos.IRepo
	include := workflowRepo.ResolveSubworkflowTemplateIdentifier(r.Include)
	subWfRoot, newWfRepo, err = loadSubworkflow(include, r)
	if err != nil {
		return err
	}

	// By now the subworkflow is loaded and reparented to this includeRole. This reparenting is
	// needed to ensure the correct gera.Map hierarchies, but now that we replace the
	// composed aggregatorRole with the newly loaded one, we must also fix the reparenting and
	// ensure the loaded name doesn't overwrite the original name of the includeRole.
	parent := r.parent
	name := r.Name
	r.aggregatorRole = *subWfRoot // The previously composed aggregatorRole+roleBase are overwritten here
	r.parent = parent
	r.Name = name

	return r.aggregatorRole.ProcessTemplates(newWfRepo, loadSubworkflow, baseConfigStack)
}

func (r *includeRole) UpdateStatus(s task.Status) {
	r.updateStatus(s)
}

func (r *includeRole) UpdateState(s sm.State) {
	r.updateState(s)
}

func (r *includeRole) copy() copyable {
	rCopy := includeRole{
		aggregatorRole: *r.aggregatorRole.copy().(*aggregatorRole),
		Include:        r.Include,
	}
	rCopy.status = SafeStatus{status: rCopy.GetStatus()}
	rCopy.state = SafeState{state: rCopy.GetState()}
	return &rCopy
}
