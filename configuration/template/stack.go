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
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/AliceO2Group/Control/core/repos"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	texttemplate "text/template"
	"time"

	"github.com/AliceO2Group/Control/common/utils"
	"github.com/AliceO2Group/Control/common/utils/uid"
	"github.com/AliceO2Group/Control/configuration/componentcfg"
	"github.com/AliceO2Group/Control/core/integration"
	"github.com/sirupsen/logrus"
)

type GetConfigFunc func(string) string
type ConfigAccessFuncs map[string]GetConfigFunc
type ToPtreeFunc func(string, string) string
type MultiVarConfigAccessFuncs map[string]GetMultiVarConfigFunc
type GetMultiVarConfigFunc func(string, string) string

func MakeConfigAccessFuncs(confSvc ConfigurationService, varStack map[string]string) ConfigAccessFuncs {
	return ConfigAccessFuncs{
		"GetConfigLegacy": func(path string) string {
			defer utils.TimeTrack(time.Now(), "GetConfigLegacy", log.WithPrefix("template"))
			query, err := componentcfg.NewQuery(path)
			if err != nil {
				return fmt.Sprintf("{\"error\":\"%s\"}", err.Error())
			}

			payload, err := confSvc.GetComponentConfiguration(query)
			if err != nil {
				log.WithError(err).
					WithField("path", query.Path()).
					Warn("failed to get component configuration")
				return fmt.Sprintf("{\"error\":\"%s\"}", err.Error())
			}

			fields := Fields{WrapPointer(&payload)}
			err = fields.Execute(confSvc, query.Path(), varStack, nil, make(map[string]texttemplate.Template), nil)
			log.Warn(varStack)
			log.Warn(payload)
			return payload
		},
		"GetConfig": func(path string) string {
			defer utils.TimeTrack(time.Now(),"GetConfig", log.WithPrefix("template"))
			query, err := componentcfg.NewQuery(path)
			if err != nil {
				return fmt.Sprintf("{\"error\":\"%s\"}", err.Error())
			}

			payload, err := confSvc.GetAndProcessComponentConfiguration(query, varStack)
			if err != nil {
				return fmt.Sprintf("{\"error\":\"%s\"}", err.Error())
			}

			log.Warn(varStack)
			log.Warn(payload)
			return payload
		},
		"DetectorForHost": func(hostname string) string {
			defer utils.TimeTrack(time.Now(),"DetectorForHost", log.WithPrefix("template"))
			payload, err := confSvc.GetDetectorForHost(hostname)
			if err != nil {
				return fmt.Sprintf("{\"error\":\"%s\"}", err.Error())
			}
			return payload
		},
		"DetectorsForHosts": func(hosts string) string {
			defer utils.TimeTrack(time.Now(),"DetectorsForHosts", log.WithPrefix("template"))
			hostsSlice := make([]string, 0)
			// first we convert the incoming string treated as JSON list into a []string
			bytes := []byte(hosts)
			err := json.Unmarshal(bytes, &hostsSlice)
			if err != nil {
				return fmt.Sprintf("{\"error\":\"DetectorsForHosts function: %s\"}", err.Error())
			}

			payload, err := confSvc.GetDetectorsForHosts(hostsSlice)
			if err != nil {
				return fmt.Sprintf("{\"error\":\"%s\"}", err.Error())
			}

			// and back to JSON list for the active detectors slice
			bytes, err = json.Marshal(payload)
			if err != nil {
				return fmt.Sprintf("{\"error\":\"DetectorsForHosts function: %s\"}", err.Error())
			}
			outString := string(bytes)

			return outString
		},
		"CRUCardsForHost": func(hostname string) string {
			defer utils.TimeTrack(time.Now(),"CRUCardsForHost", log.WithPrefix("template"))
			payload, err := confSvc.GetCRUCardsForHost(hostname)
			if err != nil {
				return fmt.Sprintf("[\"error: %s\"]", err.Error())
			}
			return payload
		},
	}
}

func MakeConfigAndRepoAccessFuncs(confSvc ConfigurationService, varStack map[string]string, workflowRepo repos.IRepo) map[string]interface{} {
	return map[string]interface{} {
		"GenerateDplSubworkflow": func (dplCommand string) string {
			// jitDplGenerate takes a resolved dplCommand as an argument,
			// generates the corresponding tasks and workflow
			// and returns the resolved dplWorkflow as a string
			jitDplGenerate := func(dplCommand string) (dplWorkflow string) {
				const nMaxExpectedQcPayloads = 2
				var metadata string

				// Match any consul URL
				re := regexp.MustCompile(`'consul-json://[^']*'`)
				matches := re.FindAllStringSubmatch(dplCommand, nMaxExpectedQcPayloads)

				// Concatenate the consul LastIndex for each payload in a single string
				for _, match := range matches {
					// Match any key under components
					keyRe := regexp.MustCompile(`components/[^']*`)
					consulKeyMatch := keyRe.FindAllStringSubmatch(match[0], 1)
					consulKey := strings.SplitAfter(consulKeyMatch[0][0], "components/")

					// And query for Consul for its LastIndex
					newQ, err := componentcfg.NewQuery(consulKey[1])
					_, lastIndex, err := confSvc.GetComponentConfigurationWithLastIndex(newQ)
					if err != nil {
						return fmt.Sprintf("JIT failed trying to query qc consul payload %s : " + err.Error(),
							match)
					}
					metadata += strconv.FormatUint(lastIndex, 10)
				}

				var err error

				// Generate a hash out of the concatenation of
				// 1) The full DPL command
				// 2) The LastIndex of each payload
				hash := sha1.New()
				hash.Write([]byte(dplCommand + metadata))
				jitWorkflowName := "jit-" + hex.EncodeToString(hash.Sum(nil))

				// We now have a workflow name made out of a hash that should be unique with respect to
				// 1) DPL command and
				// 2) Consul payload versions
				// Only generate new tasks & workflows if the files don't exist
				// If they exist, hash comparison guarantees validity
				if _, err = os.Stat(filepath.Join(workflowRepo.GetCloneDir(), "workflows", jitWorkflowName+ ".yaml")); err == nil {
					log.Tracef("Workflow %s already exists, skipping DPL creation", jitWorkflowName)
					return jitWorkflowName
				}

				log.Debug("Resolved DPL command: " + dplCommand)

				// TODO: Before executing we need to check that this is a valid dpl command
				// If not, any command may be injected on the aliecs host
				// since this will be run as user `aliecs` it might not pose a problem at this point
				cmdString := dplCommand + " --o2-control " + jitWorkflowName
				dplCmd := exec.Command("bash", "-c", cmdString)

				// execute the DPL command in the repo of the workflow used
				dplCmd.Dir = workflowRepo.GetCloneDir()
				var dplOut []byte
				dplOut, err = dplCmd.CombinedOutput()
				log.Debug("DPL command out: " + string(dplOut))
				if err != nil {
					return fmt.Sprintf("Failed to run DPL command : " + err.Error() + "\nDPL command out : " + string(dplOut))
				}

				return jitWorkflowName
			}

			// Resolve any templates as part of the DPL command
			fields := Fields{WrapPointer(&dplCommand)}
			err := fields.Execute(confSvc, dplCommand, varStack, nil, make(map[string]texttemplate.Template), workflowRepo)
			if err != nil {
				return fmt.Sprintf("JIT failed in template resolution of the dpl_command : " + err.Error())
			}

			return jitDplGenerate(dplCommand)
		},
	}
}

func MakeConfigAccessFuncsMultiVar(confSvc ConfigurationService, varStack map[string]string) MultiVarConfigAccessFuncs {
	return MultiVarConfigAccessFuncs{
		"EndpointsForCRUCard": func(hostname, cardSerial string) string {
			defer utils.TimeTrack(time.Now(), "EndpointsForCRUCard", log.WithPrefix("template"))
			payload, err := confSvc.GetEndpointsForCRUCard(hostname, cardSerial)
			if err != nil {
				return fmt.Sprintf("{\"error\":\"%s\"}", err.Error())
			}
			return payload
		},
		"GetRuntimeConfig": func(component string, key string) string {
			defer utils.TimeTrack(time.Now(), "GetRuntimeConfig", log.WithPrefix("template"))

			payload, err := confSvc.GetRuntimeEntry(component, key)
			if err != nil {
				return fmt.Sprintf("{\"error\":\"%s\"}", err.Error())
			}

			return payload
		},
	}
}

func MakePluginObjectStack(varStack map[string]string) map[string]interface{} {
	return integration.PluginsInstance().ObjectStack(varStack)
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

func MakeStrOperationFuncMap(varStack map[string]string) map[string]interface{} {
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
		"ToUpper": func(in string) (out string) {
			out = strings.ToUpper(in)
			return
		},
		"ToLower": func(in string) (out string) {
			out = strings.ToLower(in)
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
		"NewID": func() (out string) {
			return uid.New().String()
		},
		"PrefixedOverride": func(varname, prefix string) (out string) {
			prefixed, prefixedOk := varStack[prefix + "_" + varname]
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
	}
}