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
	"fmt"
	"strings"

	"github.com/AliceO2Group/Control/common/event"
	"github.com/AliceO2Group/Control/common/gera"
	"github.com/AliceO2Group/Control/common/logger"
	"github.com/AliceO2Group/Control/common/utils/uid"
	"github.com/AliceO2Group/Control/configuration/template"
	"github.com/AliceO2Group/Control/core/task/channel"
	"github.com/AliceO2Group/Control/core/task/sm"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"

	"github.com/AliceO2Group/Control/core/task"
	"github.com/AliceO2Group/Control/core/task/constraint"
)

var log = logger.New(logrus.StandardLogger(), "workflow")

const (
	PATH_SEPARATOR      = "."
	PATH_SEPARATOR_RUNE = '.'
)

type roleBase struct {
	Name        string `yaml:"name"`
	parent      Updatable
	Connect     []channel.Outbound     `yaml:"connect,omitempty"`
	Constraints constraint.Constraints `yaml:"constraints,omitempty"`
	status      SafeStatus
	state       SafeState

	// Defaults, Vars and UserVars are used to store each role's variables.
	// Defaults are the lowest priority, Vars are the second highest, and UserVars are the highest.
	//
	// These use the gera.Map type, which is a wrapper around a map[string]string that provides hierarchical KV store
	// semantics.
	// The gera.Map logic allows us to ensure that defaults are overridden by vars, and vars by userVars throughout the
	// workflow tree, and at the same time that defaults/vars/userVars set in a child role override the relevant values
	// set in its parent role.
	// The way we do this is by ensuring parent-child (Wrap) relationships between all the Default members in the
	// workflow tree, all the Vars members, and all the UserVars members, and then whenever we need to figure out what's
	// the consolidated KV map seen from the point of view of a given role, we Flatten each of these three, and then
	// Wrap and re-Flatten between the flattened defaults, vars and userVars (see ConsolidatedVarStack). This results in
	// a single map, generatable from the POV of any role within the tree.
	Defaults *gera.WrapMap[string, string] `yaml:"defaults,omitempty"`
	Vars     *gera.WrapMap[string, string] `yaml:"vars,omitempty"`
	UserVars *gera.WrapMap[string, string] `yaml:"-"`
	Locals   map[string]string             `yaml:"-"` // only used for passing iterator from template to new role
	Bind     []channel.Inbound             `yaml:"bind,omitempty"`
	Enabled  string                        `yaml:"enabled,omitempty"`
}

// Needed for the yaml package to correctly unmarshal into gera.Map[string, string] those Defaults and Vars entries from
// a workflow template, that have a !public tag to include input widget metadata.
func kvStoreUnmarshalYAMLWithTags(w gera.Map[string, string], unmarshal func(interface{}) error) error {
	nodes := make(map[string]yaml.Node)
	err := unmarshal(&nodes)
	if err == nil {
		m := make(map[string]string)
		for k, v := range nodes {
			if v.Kind == yaml.ScalarNode {
				m[k] = v.Value
			} else if v.Kind == yaml.MappingNode && v.Tag == "!public" {
				type auxType struct {
					Value string
				}
				var aux auxType
				err = v.Decode(&aux)
				if err != nil {
					continue
				}
				m[k] = aux.Value
			}
		}

		wPtr := w.(*gera.WrapMap[string, string])
		*wPtr = *gera.MakeMapWithMap(m)
	} else {
		wPtr := w.(*gera.WrapMap[string, string])
		*wPtr = *gera.MakeMap[string, string]()
	}
	return err
}

func (r *roleBase) IsEnabled() bool {
	// Only valid after ProcessTemplates
	trimmed := strings.ToLower(strings.TrimSpace(r.Enabled))
	return trimmed == "true" || trimmed == "1"
}

func (r *roleBase) SetRuntimeVar(key string, value string) {
	r.UserVars.Set(key, value)
}

func (r *roleBase) SetRuntimeVars(kv map[string]string) {
	for k, v := range kv {
		r.UserVars.Set(k, v)
	}
}

func (r *roleBase) DeleteRuntimeVar(key string) {
	r.UserVars.Del(key)
}

func (r *roleBase) DeleteRuntimeVars(keys []string) {
	for _, k := range keys {
		r.UserVars.Del(k)
	}
}

func (r *roleBase) SetGlobalRuntimeVar(key string, value string) {
	root := r.GetRootRole()
	if root != nil {
		root.SetRuntimeVar(key, value)
	} else {
		r.SetRuntimeVar(key, value)
	}
}

func (r *roleBase) SetGlobalRuntimeVars(kv map[string]string) {
	for k, v := range kv {
		r.SetGlobalRuntimeVar(k, v)
	}
}

func (r *roleBase) DeleteGlobalRuntimeVar(key string) {
	root := r.GetRootRole()
	if root != nil {
		root.DeleteRuntimeVar(key)
	} else {
		r.DeleteRuntimeVar(key)
	}
}

func (r *roleBase) DeleteGlobalRuntimeVars(keys []string) {
	for _, k := range keys {
		r.DeleteGlobalRuntimeVar(k)
	}
}

func (r *roleBase) ConsolidatedVarStack() (varStack map[string]string, err error) {
	// This function is used in task.go to get the parent role's varStack
	var defaults, vars, userVars map[string]string
	defaults, err = r.Defaults.Flattened()
	if err != nil {
		return
	}
	vars, err = r.Vars.Flattened()
	if err != nil {
		return
	}
	userVars, err = r.UserVars.Flattened()
	if err != nil {
		return
	}
	consolidated := gera.MakeMapWithMap(userVars).Wrap(gera.MakeMapWithMap(vars).Wrap(gera.MakeMapWithMap(defaults)))
	varStack, err = consolidated.Flattened()
	if err != nil {
		return
	}
	return
}

func (r *roleBase) ConsolidatedVarMaps() (defaults map[string]string, vars map[string]string, userVars map[string]string, err error) {
	defaults, err = r.Defaults.Flattened()
	if err != nil {
		return
	}
	vars, err = r.Vars.Flattened()
	if err != nil {
		return
	}
	userVars, err = r.UserVars.Flattened()
	return
}

func (r *roleBase) makeBuildObjectStackFunc() template.BuildObjectStackFunc {
	return func(stage template.Stage) map[string]interface{} {
		type wfNode struct {
			Name string
			Path string
		}
		objStack := map[string]interface{}{
			"Parent": func() *wfNode {
				parentRole := r.GetParentRole()
				if parentRole != nil {
					return &wfNode{
						Name: parentRole.GetName(),
						Path: parentRole.GetPath(),
					}
				}
				return nil
			},
			"Up": func(levels int) *wfNode {
				type _parentRole interface {
					GetParent() Updatable
					GetPath() string
				}

				if levels <= 0 {
					return nil
				}
				var p _parentRole = r
				for i := 0; i < levels; i++ {
					p = p.GetParent()
					if p == nil {
						log.WithFields(logrus.Fields{"error": "role has no ancestor", "role": r.GetPath()}).Error("workflow configuration error")
						return nil
					}
				}
				if pr, ok := p.(Role); ok {
					return &wfNode{
						Name: pr.GetName(),
						Path: pr.GetPath(),
					}
				}
				return nil
			},
		}
		if stage > 3 { // varStack and object ready
			objStack["This"] = func() *wfNode {
				return &wfNode{
					Name: r.GetName(),
					Path: r.GetPath(),
				}
			}
		}

		return objStack
	}
}

func (r *roleBase) CollectOutboundChannels() (channels []channel.Outbound) {
	if r.parent == nil {
		channels = make([]channel.Outbound, 0)
	} else {
		channels = channel.MergeOutbound(r.Connect, r.parent.CollectOutboundChannels())
	}

	return
}

func (r *roleBase) UnmarshalYAML(unmarshal func(interface{}) error) (err error) {
	// NOTE: the local type alias is a necessary step, otherwise unmarshal(&role) would
	//       recurse back to this function forever
	type _roleBase roleBase
	role := _roleBase{
		Defaults: gera.MakeMap[string, string]().WithUnmarshalYAML(kvStoreUnmarshalYAMLWithTags),
		Vars:     gera.MakeMap[string, string]().WithUnmarshalYAML(kvStoreUnmarshalYAMLWithTags),
		UserVars: gera.MakeMap[string, string]().WithUnmarshalYAML(kvStoreUnmarshalYAMLWithTags),
		Locals:   make(map[string]string),
		status:   SafeStatus{status: task.INACTIVE},
		state:    SafeState{state: sm.STANDBY},
		Enabled:  "true",
	}
	err = unmarshal(&role)
	if err == nil {
		*r = roleBase(role)
	}
	return
}

func (r *roleBase) MarshalYAML() (interface{}, error) {
	aux := make(map[string]interface{})
	aux["name"] = r.Name

	if r.Connect != nil {
		aux["connect"] = r.Connect
	}
	if r.Constraints != nil {
		aux["constraints"] = r.Constraints
	}
	if r.Defaults.Raw() != nil {
		aux["defaults"] = r.Defaults.RawCopy()
	}
	if r.Vars.Raw() != nil {
		aux["vars"] = r.Vars.RawCopy()
	}
	if r.Bind != nil {
		aux["bind"] = r.Bind
	}
	if r.Enabled != "" {
		aux["enabled"] = r.Enabled
	}

	return aux, nil
}

func (r *roleBase) wrapBindAndConnectFields() template.Fields {
	fields := make(template.Fields, len(r.Connect))
	// first we populate the connect fields, because we know we'll have 1 fields entry
	// for each connect block
	for i := range r.Connect {
		index := i // always keep a local copy for the getter/setter closures
		fields[index] = template.WrapGeneric(
			func() string {
				return r.Connect[index].Target
			},
			func(value string) {
				r.Connect[index].Target = value
			},
		)
	}
	// then we add any Global channel alias declarations for bind fields, we don't know
	// how many in advance because not all bind blocks have Global aliases
	for i := range r.Bind {
		index := i // always keep a local copy for the getter/setter closures
		if len(r.Bind[index].Global) == 0 {
			continue
		}

		fields = append(fields, template.WrapGeneric(
			func() string {
				return r.Bind[index].Global
			},
			func(value string) {
				r.Bind[index].Global = value
			},
		))
	}

	return fields
}

func (r *roleBase) copy() copyable {
	rCopy := roleBase{
		Name:        r.Name,
		parent:      r.parent,
		Defaults:    r.Defaults.Copy().(*gera.WrapMap[string, string]),
		Vars:        r.Vars.Copy().(*gera.WrapMap[string, string]),
		UserVars:    r.UserVars.Copy().(*gera.WrapMap[string, string]),
		Locals:      make(map[string]string),
		Connect:     make([]channel.Outbound, len(r.Connect)),
		Constraints: make(constraint.Constraints, len(r.Constraints)),
		status:      r.status,
		state:       r.state,
		Bind:        make([]channel.Inbound, len(r.Bind)),
		Enabled:     r.Enabled,
	}

	copied := copy(rCopy.Connect, r.Connect)
	if copied != len(r.Connect) {
		log.WithField("role", r.GetPath()).
			WithError(fmt.Errorf("slice copy copied %d items, %d expected", copied, len(r.Connect))).
			Error("role copy error")
	}

	copied = copy(rCopy.Constraints, r.Constraints)
	if copied != len(r.Constraints) {
		log.WithField("role", r.GetPath()).
			WithError(fmt.Errorf("slice copy copied %d items, %d expected", copied, len(r.Constraints))).
			Error("role copy error")
	}

	copied = copy(rCopy.Bind, r.Bind)
	if copied != len(r.Bind) {
		log.WithField("role", r.GetPath()).
			WithError(fmt.Errorf("slice copy copied %d items, %d expected", copied, len(r.Bind))).
			Error("role copy error")
	}

	return &rCopy
}

func (r *roleBase) GetParent() Updatable {
	if r == nil {
		return nil
	}
	parentUpdatable, ok := r.parent.(Updatable)
	if ok {
		return parentUpdatable
	}
	return nil
}

func (r *roleBase) GetParentRole() Role {
	if r == nil {
		return nil
	}
	parentRole, ok := r.parent.(Role)
	if ok {
		return parentRole
	}
	return nil
}

func (r *roleBase) GetRootRole() Role {
	if r == nil {
		return nil
	}

	current := r.GetParentRole()
	for current != nil {
		par := current.GetParentRole()
		if par != nil {
			current = par
		} else {
			break
		}
	}
	return current
}

func (r *roleBase) GetName() string {
	if r == nil {
		return ""
	}
	return r.Name
}

func (r *roleBase) GetEnvironmentId() uid.ID {
	if r.parent == nil {
		return uid.NilID()
	}
	return r.parent.GetEnvironmentId()
}

func (r *roleBase) SendEvent(ev event.Event) {
	if r.parent == nil {
		return
	}
	r.parent.SendEvent(ev)
}

func (r *roleBase) GetPath() string {
	if r == nil {
		return ""
	}
	if r.parent == nil {
		return r.Name
	}

	parentPath := r.parent.GetPath()
	if len(parentPath) > 0 {
		return parentPath + PATH_SEPARATOR + r.Name
	}

	return r.Name
}

func (r *roleBase) GetStatus() task.Status {
	if r == nil {
		return task.UNDEFINED
	}
	return r.status.get()
}

func (r *roleBase) GetState() sm.State {
	if r == nil {
		return sm.UNKNOWN
	}
	return r.state.get()
}

func (r *roleBase) getConstraints() (cts constraint.Constraints) {
	if r == nil {
		return
	}

	if r.Constraints == nil {
		cts = make(constraint.Constraints, 0)
	} else {
		cts = make(constraint.Constraints, len(r.Constraints))
		copy(cts, r.Constraints)
	}

	if r.parent == nil {
		return
	}
	if parentRole := r.GetParentRole(); parentRole != nil {
		cts = cts.MergeParent(parentRole.getConstraints())
	}

	return
}

func (r *roleBase) GetDefaults() gera.Map[string, string] {
	if r == nil {
		return nil
	}
	return r.Defaults
}

func (r *roleBase) GetVars() gera.Map[string, string] {
	if r == nil {
		return nil
	}
	return r.Vars
}

func (r *roleBase) GetUserVars() gera.Map[string, string] {
	if r == nil {
		return nil
	}
	return r.UserVars
}

func (r *roleBase) CollectInboundChannels() (channels []channel.Inbound) {
	if r.parent == nil {
		channels = make([]channel.Inbound, 0)
	} else {
		channels = channel.MergeInbound(r.Bind, r.parent.CollectInboundChannels())
	}
	return
}

func (r *roleBase) GetCurrentRunNumber() uint32 {
	if r == nil {
		return 0
	}
	if r.parent == nil {
		return 0
	}
	return r.parent.GetCurrentRunNumber()
}
