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
	"errors"
	"fmt"
	"io"
	"strings"
	"text/template"

	"github.com/AliceO2Group/Control/common/gera"
	"github.com/AliceO2Group/Control/common/logger"
	"github.com/AliceO2Group/Control/configuration/componentcfg"
	"github.com/AliceO2Group/Control/core/repos"
	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
	"github.com/sirupsen/logrus"
	"github.com/valyala/fasttemplate"
)

const DEBUG_TEMPLATE_SYSTEM = false

var log = logger.New(logrus.StandardLogger(), "template")

type Sequence map[Stage]Fields
type BuildObjectStackFunc func(stage Stage) map[string]interface{}
type StageCallbackFunc func(stage Stage, err error) error // called after each stage template processing, if err != nil the sequence bails

type RoleDisabledError struct {
	RolePath string
}

func (e *RoleDisabledError) Error() string {
	return fmt.Sprintf("role %s disabled", e.RolePath)
}

func NullCallback(_ Stage, err error) error {
	return err
}

type ConfigurationService interface {
	GetComponentConfiguration(query *componentcfg.Query) (payload string, err error)
	GetComponentConfigurationWithLastIndex(query *componentcfg.Query) (payload string, lastIndex uint64, err error)
	GetAndProcessComponentConfiguration(query *componentcfg.Query, varStack map[string]string) (payload string, err error)
	ResolveComponentQuery(query *componentcfg.Query) (resolved *componentcfg.Query, err error)

	GetDetectorForHost(hostname string) (string, error)
	GetDetectorsForHosts(hosts []string) ([]string, error)
	GetCRUCardsForHost(hostname string) ([]string, error)
	GetEndpointsForCRUCard(hostname, cardSerial string) ([]string, error)

	GetRuntimeEntry(component string, key string) (string, error)
	SetRuntimeEntry(component string, key string, value string) error
}

func (sf Sequence) Execute(confSvc ConfigurationService,
	parentPath string,
	varStack VarStack,
	buildObjectStack BuildObjectStackFunc,
	baseConfigStack map[string]string,
	stringTemplateCache map[string]template.Template,
	workflowRepo repos.IRepo,
	stageCallback StageCallbackFunc) (err error) {
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
			err = fields.Execute(confSvc, parentPath, stagedStack, objectStack, baseConfigStack, stringTemplateCache, workflowRepo)
			err = stageCallback(currentStage, err)
			if err != nil {
				var roleDisabledErrorType *RoleDisabledError
				if isRoleDisabled := errors.As(err, &roleDisabledErrorType); isRoleDisabled {
					return
				}

				log.WithError(err).
					Errorf("template processing error")
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
	STAGE0 Stage = iota // parent stack only (for enabled)
	STAGE1              // parent stack only                         + locals
	STAGE2              // parent stack + defaults                   + locals
	STAGE3              // parent stack + defaults + vars            + locals
	STAGE4              // parent stack + defaults + vars + uservars + locals
	STAGE5              // parent stack + defaults + vars + uservars + locals + full self-object = full stack
	_STAGE_MAX
)

type VarStack struct {
	Locals   map[string]string
	Defaults *gera.WrapMap[string, string]
	Vars     *gera.WrapMap[string, string]
	UserVars *gera.WrapMap[string, string]
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
		fallthrough
	case STAGE1:
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
	case STAGE2:
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
	case STAGE3:
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
	case STAGE4:
		fallthrough
	case STAGE5:
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
	case _STAGE_MAX:
	}

	consolidated := gera.MakeMapWithMap(vs.Locals).Wrap(gera.MakeMapWithMap(userVars).Wrap(gera.MakeMapWithMap(vars).Wrap(gera.MakeMapWithMap(defaults))))
	consolidatedStack, err = consolidated.Flattened()
	if err != nil {
		return
	}
	return
}

func (fields Fields) Execute(confSvc ConfigurationService, parentPath string, varStack map[string]string, objStack map[string]interface{}, baseConfigStack map[string]string, stringTemplateCache map[string]template.Template, workflowRepo repos.IRepo) (err error) {
	environment := make(map[string]interface{}, len(varStack))
	strOpStack := MakeUtilFuncMap(varStack)
	for k, v := range varStack {
		environment[k] = v
	}
	copyMap(objStack, environment) // needed for deep copy e.g. odc.Configure

	for k, v := range strOpStack {
		environment[k] = v
	}

	if workflowRepo != nil {
		repoAccessFuncs := MakeConfigAndRepoAccessFuncs(confSvc, varStack, workflowRepo)
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

	pluginObjects := MakePluginObjectStack(varStack, baseConfigStack)
	copyMap(pluginObjects, environment)

	configAccessObj := MakeConfigAccessObject(confSvc, varStack)
	copyMap(configAccessObj, environment)

	// We override the "+" operator so that in situations like
	// "string" + <nil>
	// <nil> + "string"
	// <nil> + <nil>
	// we always treat <nil> as string and return a string.
	environment["addNil"] = func(a, b *string) string {
		first, second := "", ""
		if a != nil {
			first = *a
		}
		if b != nil {
			second = *b
		}
		return first + second
	}

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
			var (
				program   *vm.Program
				rawOutput interface{}
			)
			program, err = expr.Compile(tag, []expr.Option{
				expr.Env(environment),
				expr.Operator("+", "addNil"),
			}...)
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
