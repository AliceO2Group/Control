/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2021 CERN and copyright holders of ALICE O².
 * Author: Kostas Alexopoulos <kostas.alexopoulos@cern.ch>
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
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/AliceO2Group/Control/configuration/template"
	"github.com/AliceO2Group/Control/core/repos"
	"github.com/AliceO2Group/Control/core/task"
	"github.com/AliceO2Group/Control/core/the"
	"os/exec"
	"strings"
	texttemplate "text/template"
)

// A translateRole is a delayed aggregatorRole
// It takes a `translate` entry, which is then translated into a dpl workflow
// and expanded into a full aggregatorRole during the ProcessTemplates function.
type translateRole struct {
	aggregatorRole

	Translate    string                   `yaml:"translate,omitempty"`
}

func (r *translateRole) UnmarshalYAML(unmarshal func(interface{}) error) (err error) {
	// NOTE: see NOTE in roleBase.UnmarshalYAML

	innerRoleBase := roleBase{}
	err = unmarshal(&innerRoleBase)
	if err != nil {
		return
	}

	type auxTranslate struct{
		Translate    string                   `yaml:"translate,omitempty"`
	}
	_auxTranslate:= auxTranslate{}
	err = unmarshal(&_auxTranslate)
	if err != nil {
		return
	}

	role := translateRole{
		aggregatorRole: aggregatorRole{
			roleBase:   innerRoleBase,
			aggregator: aggregator{},
		},
		Translate:        _auxTranslate.Translate,
	}


	*r = role
	return
}

func (r *translateRole) MarshalYAML() (interface{}, error) {
	return nil, nil
}

func (r *translateRole) ProcessTemplates(workflowRepo repos.IRepo, loadSubworkflow LoadSubworkflowFunc) (err error) {

	// TODO: Two "rounds" of templating are necessary here as the initial thing received is {{ dpl_command }} which is *THEN* templated to the actual dpl command payload
	// TODO: Check with Teo on what the best way to do this is
	for i := 0; i < 2; i++ {

		if r == nil {
			return errors.New("role tree error when processing templates")
		}

		templSequence := template.Sequence{
			template.STAGE0: template.WrapMapItems(r.Defaults.Raw()),
			template.STAGE1: template.WrapMapItems(r.Vars.Raw()),
			template.STAGE2: template.WrapMapItems(r.UserVars.Raw()),
			template.STAGE3: template.Fields{
				template.WrapPointer(&r.Name),
				template.WrapPointer(&r.Translate),
			},
			template.STAGE4: append(append(
				WrapConstraints(r.Constraints),
				r.wrapBindAndConnectFields()...),
				template.WrapPointer(&r.Enabled)),
		}

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

		if !r.IsEnabled() {
			// Normally it's the parent that checks whether a child is enabled after that child's
			// ProcessTemplates is done, and this is true for the current role as well, but if
			// at this point we already know that the current role is disabled (and will
			// therefore be excluded by its parent), we don't try to resolve the workflow
			// inclusion because there's no reason to do so anyway.
			return nil
		}
	}

	// JIT requires DPL command as input and returns the ready-to-use workflow name as output
	// JIT takes place after the templated DPL command has been resolved
	jit := func(dplCommand string) (string, error) {
		//start := time.Now().UnixNano() / int64(time.Millisecond)

		hash := sha1.New()
		hash.Write([]byte(r.Translate))
		jitWorkflowName := "jit-" + hex.EncodeToString(hash.Sum(nil))
		// TODO: Check if tasks / workflows already exist?

		log.Trace("Resolved DPL command: " + dplCommand)

		// TODO: Before executing we need to check that this is a valid dpl command
		// If not, any command may be injected on the aliecs host
		// since this will be run as user `aliecs` it might not poes a problem at this point
		cmdString := dplCommand + " --o2-control " + jitWorkflowName
		dplCmd := exec.Command("bash", "-c", cmdString)

		// execute the DPL command in the repo of the workflow used
		dplCmd.Dir = workflowRepo.GetCloneDir()
		var dplOut []byte
		dplOut, err = dplCmd.CombinedOutput()
		log.Trace("DPL command out: " + string(dplOut))
		if err != nil {
			return "", fmt.Errorf("Failed to run DPL command : " + err.Error())
		}

		/*end := time.Now().UnixNano() / int64(time.Millisecond)
		log.Tracef("JIT took %d ms", end-start)*/
		return jitWorkflowName, err
	}

	// Common part done, include resolution starts here.
	// An include Role is essentially a baseRole + `include:` expression. We first need to resolve
	// the expression to obtain a full subworkflow template identifier, i.e. a full repo/wft/branch
	// combo.
	// Once that's done we can load the subworkflow and obtain the root `aggregatorRole` plus a new
	// repos.Repo definition. If a repo or branch is already provided in the subworkflow expression
	// then the returned newWfRepo will reflect this, and any additionally nested includes will
	// default to the repo of their direct parent.
	var subWfRoot *aggregatorRole
	var newWfRepo repos.IRepo

	var jitWorkflowName string
	jitWorkflowName, err = jit(r.Translate)
	translate := workflowRepo.ResolveSubworkflowTemplateIdentifier(jitWorkflowName)
	subWfRoot, newWfRepo, err = loadSubworkflow(translate, r)
	if err != nil {
		return err
	}

	// By now the subworkflow is loaded and reparented to this translateRole. This reparenting is
	// needed to ensure the correct gera.StringMap hierarchies, but now that we replace the
	// composed aggregatorRole with the newly loaded one, we must also fix the reparenting and
	// ensure the loaded name doesn't overwrite the original name of the translateRole.
	parent := r.parent
	name := r.Name
	r.aggregatorRole = *subWfRoot // The previously composed aggregatorRole+roleBase are overwritten here
	r.parent = parent
	r.Name = name

	return r.aggregatorRole.ProcessTemplates(newWfRepo, loadSubworkflow)
}

func (r* translateRole) UpdateStatus(s task.Status) {
	r.updateStatus(s)
}

func (r* translateRole) UpdateState(s task.State) {
	r.updateState(s)
}

func (r *translateRole) copy() copyable {
	rCopy := translateRole{
		aggregatorRole: *r.aggregatorRole.copy().(*aggregatorRole),
		Translate: r.Translate,
	}
	rCopy.status = SafeStatus{status:rCopy.GetStatus()}
	rCopy.state  = SafeState{state:rCopy.GetState()}
	return &rCopy
}
