/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2020 CERN and copyright holders of ALICE O².
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
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	texttemplate "text/template"

	"github.com/AliceO2Group/Control/common/utils"
	"github.com/AliceO2Group/Control/common/utils/uid"
	"github.com/AliceO2Group/Control/core/the"
	"github.com/flosch/pongo2/v4"
	"github.com/osteele/liquid"
	"github.com/sirupsen/logrus"
)

type GetConfigFunc func(string) string
type ConfigAccessFuncs map[string]GetConfigFunc
type ToPtreeFunc func(string, string) string

func MakeConfigAccessFuncs(varStack map[string]string) ConfigAccessFuncs {
	return ConfigAccessFuncs{
		"GetConfig": func(path string) string {
			payload, err := the.ConfSvc().GetComponentConfiguration(path)
			if err != nil {
				log.WithError(err).
					WithField("path", path).
					Warn("failed to get component configuration")
				return fmt.Sprintf("{\"error\":\"%s\"}", err.Error())
			}

			fields := Fields{WrapPointer(&payload)}
			err = fields.Execute(path, varStack, nil, make(map[string]texttemplate.Template))
			log.Warn(payload)
			return payload
		},
		"GetConfigLiquid": func(path string) string {
			payload, err := the.ConfSvc().GetComponentConfiguration(path)
			if err != nil {
				log.WithError(err).
					WithField("path", path).
					Warn("failed to get component configuration")
				return fmt.Sprintf("{\"error\":\"%s\"}", err.Error())
			}

			engine := liquid.NewEngine()
			bindings := make(map[string]interface{})
			for k, v := range varStack {
				bindings[k] = v
			}
			payload, liquidErr := engine.ParseAndRenderString(payload, bindings)
			if liquidErr != nil {
				log.WithError(liquidErr).Warn("template processing error")
			}

			log.Warn(payload)
			return payload
		},
		"GetConfigPongo": func(path string) string {
			payload, err := the.ConfSvc().GetComponentConfiguration(path)
			if err != nil {
				log.WithError(err).
					WithField("path", path).
					Warn("failed to get component configuration")
				return fmt.Sprintf("{\"error\":\"%s\"}", err.Error())
			}

			tpl, err := pongo2.FromString(payload)
			if err != nil {
				return fmt.Sprintf("{\"error\":\"%s\"}", err.Error())
			}

			bindings := make(map[string]interface{})
			for k, v := range varStack {
				bindings[k] = v
			}

			// TODO add a custom pongo2.TemplateLoader for Consul here
			// It should call confsvc.GetComponentConfiguration but without
			// going through the template engine
			payload, err = tpl.Execute(bindings)
			if err != nil {
				return fmt.Sprintf("{\"error\":\"%s\"}", err.Error())
			}

			log.Warn(payload)
			return payload
		},
	}
}


func MakeToPtreeFunc(varStack map[string]string, propMap map[string]string) ToPtreeFunc {
	return func(payload string, syntax string) string {
		// This function is a no-op with respect to the payload, but it stores the payload
		// under a new key which the OCC plugin then processes into a ptree.
		// The payload in the current key is overwritten.
		localPayload := payload
		syntaxLC := strings.ToLower(strings.TrimSpace(syntax))

		if !utils.StringSliceContains([]string{"ini", "json", "xml"}, syntaxLC) {
			err := errors.New("bad ToPtree syntax argument, allowed values: ini, json, xml")
			log.WithError(err).
				WithField("syntax", syntax).
				Warn("failed to generate ptree descriptor")
			localPayload = fmt.Sprintf("{\"error\":\"%s\"}", err.Error())
			syntaxLC = "json"
		}

		ptreeId := fmt.Sprintf("__ptree__:%s:%s", syntaxLC, uid.New().String())
		propMap[ptreeId] = localPayload
		return ptreeId
	}
}

func MakeStrOperationFuncMap() map[string]interface{} {
	return map[string]interface{}{
		"Atoi": func(in string) (out int) {
			var err error
			out, err = strconv.Atoi(in)
			if err != nil {
				log.WithError(err).Warn("error converting string/int in template system")
				return
			}
			return
		},
		"Itoa": func(in int) (out string) {
			out = strconv.Itoa(in)
			return
		},
		"TrimQuotes": func(in string) (out string) {
			out = strings.Trim(in, "\"")
			return
		},
		"TrimSpace": func(in string) (out string) {
			out = strings.TrimSpace(in)
			return
		},
		"FromJson": func(in string) (out interface{}) {
			bytes := []byte(in)
			err := json.Unmarshal(bytes, &out)
			log.WithFields(logrus.Fields{
					"in": in,
					"out": out,
				}).
				Debug("FromJson")
			if err != nil {
				log.WithError(err).Warn("error unmarshaling JSON/YAML in template system")
				return
			}
			return
		},
		"ToJson": func(in interface{}) (out string) {
			bytes, err := json.Marshal(in)
			if err != nil {
				log.WithError(err).Warn("error marshaling JSON/YAML in template system")
				return
			}
			out = string(bytes)
			return
		},
	}
}