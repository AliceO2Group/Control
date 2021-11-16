/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2018-2020 CERN and copyright holders of ALICE O².
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

package template

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/AliceO2Group/Control/core/repos"
	"io"
	"strings"
	"text/template"

	"github.com/AliceO2Group/Control/common/gera"
	"github.com/AliceO2Group/Control/common/logger"
	"github.com/AliceO2Group/Control/configuration/componentcfg"
	"github.com/antonmedv/expr"
	"github.com/antonmedv/expr/vm"
	"github.com/sirupsen/logrus"
	"github.com/valyala/fasttemplate"
)

const DEBUG_TEMPLATE_SYSTEM=false

var log = logger.New(logrus.StandardLogger(),"template")

type Sequence map[Stage]Fields
type BuildObjectStackFunc func(stage Stage) map[string]interface{}

type ConfigurationService interface {
	GetComponentConfiguration(query *componentcfg.Query) (payload string, err error)
	GetAndProcessComponentConfiguration(query *componentcfg.Query, varStack map[string]string) (payload string, err error)
	GetDetectorForHost(hostname string) (string, error)
	GetDetectorsForHosts(hosts []string) ([]string, error)
	GetCRUCardsForHost(hostname string) (string, error)
	GetEndpointsForCRUCard(hostname, cardSerial string) (string, error)
	GetRuntimeEntry(component string, key string) (string, error)
	GetEntryWithLastIndex(key string) (string, uint64, error)
}

func (sf Sequence) Execute(confSvc ConfigurationService, parentPath string, varStack VarStack, buildObjectStack BuildObjectStackFunc, stringTemplateCache map[string]template.Template, workflowRepo repos.IRepo) (err error) {
	for i := 0; i < int(_STAGE_MAX); i++ {
		currentStage := Stage(i)

		var stagedStack map[string]string
		stagedStack, err = varStack.consolidated(currentStage)
		if err != nil {
			return
		}

		objectStack := buildObjectStack(currentStage)

		if fields, ok := sf[currentStage]; ok {
			if DEBUG_TEMPLATE_SYSTEM {
				log.WithFields(logrus.Fields{
					"path":  parentPath,
					"stage": currentStage,
					"fields": func() string {
						accumulator := make([]string, len(fields))
						for i, v := range fields {
							accumulator[i] = v.Get()
						}
						return strings.Join(accumulator, ", ")
					}(),
				}).Trace("about to process fields for stage")
			}
			err = fields.Execute(confSvc, parentPath, stagedStack, objectStack, stringTemplateCache, workflowRepo)
			if err != nil {
				log.WithError(err).Errorf("template processing error")
				return
			}
		}
	}
	return
}

type Fields []Field

type Stage int
const (
	// RESOLUTION STAGE ↓      VALUES AVAILABLE ↓
	STAGE0 Stage = iota // parent stack only                         + locals
	STAGE1              // parent stack + defaults                   + locals
	STAGE2              // parent stack + defaults + vars            + locals
	STAGE3              // parent stack + defaults + vars + uservars + locals
	STAGE4              // parent stack + defaults + vars + uservars + locals + full self-object = full stack
	_STAGE_MAX
)

type VarStack struct {
	Locals map[string]string
	Defaults *gera.StringWrapMap
	Vars *gera.StringWrapMap
	UserVars *gera.StringWrapMap
}

func (vs *VarStack) consolidated(stage Stage) (consolidatedStack map[string]string, err error) {
	var defaults, vars, userVars map[string]string
	vars, err = vs.Vars.Flattened()
	if err != nil {
		return
	}
	userVars, err = vs.UserVars.Flattened()
	if err != nil {
		return
	}

	switch stage {
	case STAGE0:
		defaults, err = vs.Defaults.FlattenedParent()
		if err != nil {
			return
		}
		vars, err = vs.Vars.FlattenedParent()
		if err != nil {
			return
		}
		userVars, err = vs.UserVars.FlattenedParent()
		if err != nil {
			return
		}
	case STAGE1:
		defaults, err = vs.Defaults.Flattened()
		if err != nil {
			return
		}
		vars, err = vs.Vars.FlattenedParent()
		if err != nil {
			return
		}
		userVars, err = vs.UserVars.FlattenedParent()
		if err != nil {
			return
		}
	case STAGE2:
		defaults, err = vs.Defaults.Flattened()
		if err != nil {
			return
		}
		vars, err = vs.Vars.Flattened()
		if err != nil {
			return
		}
		userVars, err = vs.UserVars.FlattenedParent()
		if err != nil {
			return
		}
	case STAGE3: fallthrough
	case STAGE4:
		defaults, err = vs.Defaults.Flattened()
		if err != nil {
			return
		}
		vars, err = vs.Vars.Flattened()
		if err != nil {
			return
		}
		userVars, err = vs.UserVars.Flattened()
		if err != nil {
			return
		}
	}

	consolidated := gera.MakeStringMapWithMap(vs.Locals).Wrap(gera.MakeStringMapWithMap(userVars).Wrap(gera.MakeStringMapWithMap(vars).Wrap(gera.MakeStringMapWithMap(defaults))))
	consolidatedStack, err = consolidated.Flattened()
	if err != nil {
		return
	}
	return
}

func (fields Fields) Execute(confSvc ConfigurationService, parentPath string, varStack map[string]string, objStack map[string]interface{}, stringTemplateCache map[string]template.Template, workflowRepo repos.IRepo) (err error) {
	environment := make(map[string]interface{}, len(varStack))
	strOpStack := MakeStrOperationFuncMap(varStack)
	for k, v := range varStack {
		environment[k] = v
	}
	copyMap(objStack, environment) // needed for deep copy e.g. odc.Configure

	for k, v := range strOpStack {
		environment[k] = v
	}

	if workflowRepo != nil {
		repoAccessFuncs := MakeRepoAccessFuncs(confSvc, varStack, workflowRepo)
		for k, v := range repoAccessFuncs {
			environment[k] = v
		}
	}

	configAccessFuncs := MakeConfigAccessFuncs(confSvc, varStack)
	for k, v := range configAccessFuncs {
		environment[k] = v
	}

	multiVarconfigAccessFuncs := MakeConfigAccessFuncsMultiVar(confSvc, varStack)
	for k, v := range multiVarconfigAccessFuncs {
		environment[k] = v
	}

	pluginObjects := MakePluginObjectStack(varStack)
	copyMap(pluginObjects, environment)

	for _, field := range fields {
		buf := new(bytes.Buffer)
		// FIXME: the line below implements the cache
		//tmpl, ok := stringTemplateCache[*str]
		var tmpl *fasttemplate.Template // dummy
		ok := false                     // dummy
		if !ok {
			tmpl, err = fasttemplate.NewTemplate(field.Get(), "{{", "}}")
			if err != nil {
				log.WithError(err).WithField("role", parentPath).Warn("template processing error (bad workflow file)")
				return
			}
		}

		_, err = tmpl.ExecuteFunc(buf, func(w io.Writer, tag string) (i int, err error) {
			var(
				program *vm.Program
				rawOutput interface{}
			)
			program, err = expr.Compile(tag)
			if err != nil {
				return
			}
			rawOutput, err = expr.Run(program, environment)
			if err != nil {
				return
			}

			switch rawOutput.(type) {
			case []interface{}, []string:
				var jsonOutput []byte
				jsonOutput, err = json.Marshal(rawOutput)
				if err != nil {
					return
				}
				return w.Write(jsonOutput)
			default:
				return w.Write([]byte(fmt.Sprintf("%v", rawOutput)))
			}
		})
		if DEBUG_TEMPLATE_SYSTEM {
			log.WithFields(logrus.Fields{
				"path":        parentPath,
				"fieldBefore": field.Get(),
				"fieldAfter":  buf.String(),
				"error":       err,
			}).Trace("processed field for stage")
		}
		if err != nil {
			log.WithError(err).WithField("role", parentPath).Warn("template processing error (bad variable or workflow file)")
			return
		}
		field.Set(buf.String())
	}
	return
}

func copyMap(src map[string]interface{}, dest map[string]interface{}) {
	for k, v := range src {
		vm, ok := v.(map[string]interface{})
		if ok {
			var destk map[string]interface{}
			if _, exists := dest[k]; exists {
				destk, ok = dest[k].(map[string]interface{})
				if !ok {
					destk = make(map[string]interface{})
				}
			} else {
				destk = make(map[string]interface{})
			}
			copyMap(vm, destk)
			dest[k] = destk
		} else {
			dest[k] = v
		}
	}
}
