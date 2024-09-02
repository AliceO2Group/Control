/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2020-2022 CERN and copyright holders of ALICE O².
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
	"io/ioutil"
	"slices"
	"strconv"
	"strings"

	"dario.cat/mergo"
	apricotpb "github.com/AliceO2Group/Control/apricot/protos"
	"github.com/AliceO2Group/Control/configuration/componentcfg"
	"github.com/AliceO2Group/Control/core/integration"
	"github.com/AliceO2Group/Control/core/repos"

	"github.com/AliceO2Group/Control/common/utils"
	"github.com/AliceO2Group/Control/common/utils/uid"
	"github.com/sirupsen/logrus"
)

type GetConfigFunc func(string) string
type ConfigAccessFuncs map[string]GetConfigFunc
type ToPtreeFunc func(string, string) string
type MultiVarConfigAccessFuncs map[string]GetMultiVarConfigFunc
type GetMultiVarConfigFunc func(string, string) string

func MakeConfigAccessObject(confSvc ConfigurationService, varStack map[string]string) map[string]interface{} {
	obj := make(map[string]interface{})
	obj["config"] = map[string]interface{}{
		"Get": func(path string) string {
			return getConfig(confSvc, varStack, path)
		},
		"GetLegacy": func(path string) string {
			return getConfigLegacy(confSvc, varStack, path)
		},
		"Resolve": func(component string, runType string, roleName string, entryKey string) string {
			rt, ok := apricotpb.RunType_value[runType]
			if !ok {
				rt = int32(apricotpb.RunType_NULL)
			}
			return resolveConfig(confSvc, &componentcfg.Query{
				Component: component,
				RunType:   apricotpb.RunType(rt),
				RoleName:  roleName,
				EntryKey:  entryKey,
			})
		},
		"ResolvePath": func(path string) string {
			return resolveConfigPath(confSvc, path)
		},
	}
	obj["inventory"] = map[string]interface{}{
		"DetectorForHost": func(hostname string) string {
			return detectorForHost(confSvc, hostname)
		},
		"DetectorsForHosts": func(hosts string) string {
			return detectorsForHosts(confSvc, hosts)
		},
		"CRUCardsForHost": func(hostname string) string {
			return cruCardsForHost(confSvc, hostname)
		},
		"EndpointsForCRUCard": func(hostname, cardSerial string) string {
			return endpointsForCruCard(confSvc, hostname, cardSerial)
		},
	}
	obj["runtime"] = map[string]interface{}{
		"Get": func(component string, key string) string {
			return getRuntimeConfig(confSvc, component, key)
		},
		"Set": func(component string, key string, value string) string {
			return setRuntimeConfig(confSvc, component, key, value)
		},
	}
	return obj
}

// deprecated
func MakeConfigAccessFuncs(confSvc ConfigurationService, varStack map[string]string) ConfigAccessFuncs {
	return ConfigAccessFuncs{
		"GetConfigLegacy": func(path string) string {
			log.WithPrefix("template").Warn("GetConfigLegacy is deprecated, use config.GetLegacy instead")
			return getConfigLegacy(confSvc, varStack, path)
		},
		"GetConfig": func(path string) string {
			log.WithPrefix("template").Warn("GetConfig is deprecated, use config.Get instead")
			return getConfig(confSvc, varStack, path)
		},
		"ResolveConfigPath": func(path string) string {
			log.WithPrefix("template").Warn("ResolveConfigPath is deprecated, use config.ResolvePath instead")
			return resolveConfigPath(confSvc, path)
		},
		"DetectorForHost": func(hostname string) string {
			log.WithPrefix("template").Warn("DetectorForHost is deprecated, use inventory.DetectorForHost instead")
			return detectorForHost(confSvc, hostname)
		},
		"DetectorsForHosts": func(hosts string) string {
			log.WithPrefix("template").Warn("DetectorsForHosts is deprecated, use inventory.DetectorsForHosts instead")
			return detectorsForHosts(confSvc, hosts)
		},
		"CRUCardsForHost": func(hostname string) string {
			log.WithPrefix("template").Warn("CRUCardsForHost is deprecated, use inventory.CRUCardsForHost instead")
			return cruCardsForHost(confSvc, hostname)
		},
	}
}

func MakeConfigAccessFuncsMultiVar(confSvc ConfigurationService, varStack map[string]string) MultiVarConfigAccessFuncs {
	return MultiVarConfigAccessFuncs{
		"EndpointsForCRUCard": func(hostname, cardSerial string) string {
			log.WithPrefix("template").Warn("EndpointsForCRUCard is deprecated, use inventory.EndpointsForCRUCard instead")
			return endpointsForCruCard(confSvc, hostname, cardSerial)
		},
		"GetRuntimeConfig": func(component string, key string) string {
			log.WithPrefix("template").Warn("GetRuntimeConfig is deprecated, use runtime.Get instead")
			return getRuntimeConfig(confSvc, component, key)
		},
	}
}

func MakePluginObjectStack(varStack map[string]string, baseConfigStack map[string]string) map[string]interface{} {
	return integration.PluginsInstance().ObjectStack(varStack, baseConfigStack)
}

func MakeConfigAndRepoAccessFuncs(confSvc ConfigurationService, varStack map[string]string, workflowRepo repos.IRepo) map[string]interface{} {
	return map[string]interface{}{
		"GenerateDplSubworkflow": func(dplCommand string) (string, error) {
			log.WithPrefix("template").Warn("GenerateDplSubworkflow is deprecated, use dpl.Generate instead")
			return generateDplSubworkflow(confSvc, varStack, workflowRepo, dplCommand)
		},
		"GenerateDplSubworkflowFromUri": func(dplCommandUri string) (string, error) {
			log.WithPrefix("template").Warn("GenerateDplSubworkflowFromUri is deprecated, use dpl.GenerateFromUri instead")
			return generateDplSubworkflowFromUri(confSvc, varStack, workflowRepo, dplCommandUri, false)
		},
		"dpl": map[string]interface{}{
			"Generate": func(dplCommand string) (string, error) {
				return generateDplSubworkflow(confSvc, varStack, workflowRepo, dplCommand)
			},
			"GenerateFromUri": func(dplCommandUri string) (string, error) {
				return generateDplSubworkflowFromUri(confSvc, varStack, workflowRepo, dplCommandUri, false)
			},
			"GenerateFromUriOrFallbackToTemplate": func(dplCommandUri string) (string, error) {
				return generateDplSubworkflowFromUri(confSvc, varStack, workflowRepo, dplCommandUri, true)
			},
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

func MakeUtilFuncMap(varStack map[string]string) map[string]interface{} {
	legacy := make(map[string]interface{})
	stringsMap := map[string]interface{}{
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
		"ToUpper": func(in string) (out string) {
			out = strings.ToUpper(in)
			return
		},
		"ToLower": func(in string) (out string) {
			out = strings.ToLower(in)
			return
		},
		"IsTruthy": func(in string) (out bool) {
			toLower := strings.TrimSpace(strings.ToLower(in))
			out = slices.Contains([]string{"true", "yes", "y", "1", "on", "ok"}, toLower)
			return
		},
		"IsFalsy": func(in string) (out bool) {
			toLower := strings.TrimSpace(strings.ToLower(in))
			out = len(toLower) == 0 || slices.Contains([]string{"false", "no", "n", "0", "off", "none"}, toLower)
			return
		},
	}
	_ = mergo.Merge(&legacy, stringsMap)

	jsonMap := map[string]interface{}{
		"Unmarshal": func(in string) (out interface{}) {
			bytes := []byte(in)
			err := json.Unmarshal(bytes, &out)
			log.WithFields(logrus.Fields{
				"in":  in,
				"out": out,
			}).
				Debug("FromJson")
			if err != nil {
				log.WithError(err).Warn("error unmarshaling JSON/YAML in template system")
				return
			}
			return
		},
		"Marshal": func(in interface{}) (out string) {
			bytes, err := json.Marshal(in)
			if err != nil {
				log.WithError(err).Warn("error marshaling JSON/YAML in template system")
				return
			}
			out = string(bytes)
			return
		},
	}
	jsonMap["Deserialize"] = jsonMap["Unmarshal"]
	jsonMap["Serialize"] = jsonMap["Marshal"]

	legacy["FromJson"] = jsonMap["Unmarshal"]
	legacy["ToJson"] = jsonMap["Marshal"]

	uidMap := map[string]interface{}{
		"New": func() (out string) {
			return uid.New().String()
		},
	}
	legacy["NewID"] = uidMap["New"]

	utilMap := map[string]interface{}{
		"PrefixedOverride": func(varname, prefix string) string {
			prefixed, prefixedOk := varStack[prefix+"_"+varname]
			fallback, fallbackOk := varStack[varname]

			// Handle explicit null values
			if prefixedOk && (prefixed == "none" || len(strings.TrimSpace(prefixed)) == 0) {
				prefixedOk = false
			}
			if fallbackOk && (fallback == "none" || len(strings.TrimSpace(fallback)) == 0) {
				fallbackOk = false
			}

			if !prefixedOk {
				if fallbackOk {
					return fallback
				}
				return "" // Neither value exists
			}

			// prefixedOk is true, fallbackOk we don't know & don't care at this point
			return prefixed
		},
		"Dump": func(in, filepath string) string {
			err := ioutil.WriteFile(filepath, []byte(in), 0644)
			if err != nil {
				log.WithError(err).Warn("could not dump variable to file")
			}
			return in
		},
		"Nullable": func(in *string) string {
			if in == nil {
				return ""
			}
			return *in
		},
		"SuffixInRange": func(input string, prefix string, idMinStr string, idMaxStr string) string {
			trimmed := strings.TrimPrefix(input, prefix)
			if input == trimmed {
				return "false"
			}
			id, err := strconv.Atoi(trimmed)
			if err != nil {
				return "false"
			}
			idMin, err := strconv.Atoi(idMinStr)
			if err != nil {
				log.Errorf("Argument idMinStr to SuffixInRange is not a number: %s", idMinStr)
				return "false"
			}
			idMax, err := strconv.Atoi(idMaxStr)
			if err != nil {
				log.Errorf("Argument idMaxStr to SuffixInRange is not a number: %s", idMaxStr)
				return "false"
			}

			if id <= idMax && id >= idMin {
				return "true"
			} else {
				return "false"
			}
		},
	}
	_ = mergo.Merge(&legacy, utilMap)

	legacy["strings"] = stringsMap
	legacy["json"] = jsonMap
	legacy["uid"] = uidMap
	legacy["util"] = utilMap

	return legacy
}
