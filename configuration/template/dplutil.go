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
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	texttemplate "text/template"

	"github.com/AliceO2Group/Control/configuration/componentcfg"
	"github.com/AliceO2Group/Control/core/repos"
)

// jitDplGenerate takes a resolved dplCommand as an argument,
// generates the corresponding tasks and workflow
// and returns the resolved dplWorkflow as a string
func jitDplGenerate(confSvc ConfigurationService, varStack map[string]string, workflowRepo repos.IRepo, dplCommand string) (jitWorkflowName string, err error) {
	const nMaxExpectedQcPayloads = 2
	var payloads []string

	// Match any consul URL
	re := regexp.MustCompile(`(consul-json|apricot)://[^ |"\n]*`)
	matches := re.FindAllStringSubmatch(dplCommand, nMaxExpectedQcPayloads)
	matches = append(matches)

	// Gather all the processed configuration payloads from apricot
	for _, match := range matches {
		// Match any key under components
		keyRe := regexp.MustCompile(`components/[^']*`)
		consulKeyMatch := keyRe.FindAllStringSubmatch(match[0], 1)
		consulKey := strings.SplitAfter(consulKeyMatch[0][0], "components/")
		// split between the query and its parameters if there are any
		consulKeyTokens := strings.Split(consulKey[1], "?")

		// Query Apricot for the configuration payload
		query, err := componentcfg.NewQuery(consulKeyTokens[0])
		if err != nil {
			return "", fmt.Errorf("JIT could not create a query out of path '%s'. error: %w", consulKey[1], err)
		}
		// parse parameters if they are present
		queryParams := &componentcfg.QueryParameters{ProcessTemplates: false, VarStack: nil}
		if len(consulKeyTokens) == 2 {
			queryParams, err = componentcfg.NewQueryParameters(consulKeyTokens[1])
			if err != nil {
				return "", fmt.Errorf("JIT could not parse query parameters of path '%s', error: %w", consulKey[1], err)
			}
		}
		var payload string
		if queryParams.ProcessTemplates {
			payload, err = confSvc.GetAndProcessComponentConfiguration(query, queryParams.VarStack)
		} else {
			payload, err = confSvc.GetComponentConfiguration(query)
		}

		if err != nil {
			return "", fmt.Errorf("JIT failed trying to query QC payload '%s', error: %w", match, err)
		}
		payloads = append(payloads, payload)
	}

	// Get the O2 & QualityControl version
	o2VersionCmd := exec.Command("bash", "-c", "rpm -qa o2-O2 o2-QualityControl")
	o2VersionOut, err := o2VersionCmd.Output()
	if err != nil {
		log.Warn("JIT couldn't get O2 / QualityControl version: " + err.Error())
	}

	// Get the env vars necessary for JIT
	jitEnvVars := varStack["jit_env_vars"]

	// Generate a hash out of
	// 1) The full DPL command
	// 2) The O2 + QualityControl package versions
	// 3) The JIT env vars
	// 4) The returned configuration payloads (as separate Write calls to avoid copies of large strings)
	hash := sha1.New()
	hash.Write([]byte(dplCommand + string(o2VersionOut) + jitEnvVars))
	for _, payload := range payloads {
		hash.Write([]byte(payload))
	}
	jitWorkflowName = "jit-" + hex.EncodeToString(hash.Sum(nil))

	// We now have a workflow name made out of a hash that should be unique with respect to
	// 1) DPL command and
	// 2) O2 + QualityControl package versions
	// 3) JIT env vars
	// 4) Configuration payloads returned by Apricot
	// Only generate new tasks & workflows if the files don't exist
	// If they exist, hash comparison guarantees validity
	if _, err = os.Stat(filepath.Join(workflowRepo.GetCloneDir(), "workflows", jitWorkflowName+".yaml")); err == nil {
		log.Tracef("Workflow '%s' already exists, skipping DPL creation", jitWorkflowName)
		return jitWorkflowName, nil
	}

	// TODO: Before executing we need to check that this is a valid dpl command
	// If not, any command may be injected on the aliecs host
	// since this will be run as user `aliecs` it might not pose a problem at this point
	cmdString := "export " + jitEnvVars + " && " + dplCommand + " --o2-control " + jitWorkflowName
	// for some reason the above concatenation may introduce new lines
	cmdString = strings.ReplaceAll(cmdString, "\n", " ")
	log.Trace("Resolved DPL command: " + cmdString)
	dplCmd := exec.Command("bash", "-c", cmdString)

	// execute the DPL command in the repo of the workflow used
	dplCmd.Dir = workflowRepo.GetCloneDir()
	var dplOut []byte
	dplOut, err = dplCmd.CombinedOutput()
	log.Trace("DPL command out: " + string(dplOut))
	if err != nil {
		return "", fmt.Errorf("Failed to run DPL command: %w.\n DPL command out: %s", err, string(dplOut))
	}

	return jitWorkflowName, nil
}

func generateDplSubworkflow(confSvc ConfigurationService, varStack map[string]string, workflowRepo repos.IRepo, dplCommand string) (jitWorkflowName string, err error) {
	if dplCommand == "none" {
		return "", fmt.Errorf("dplCommand is 'none'")
	}

	// Resolve any templates as part of the DPL command
	fields := Fields{WrapPointer(&dplCommand)}
	err = fields.Execute(confSvc, dplCommand, varStack, nil, nil, make(map[string]texttemplate.Template), workflowRepo)
	if err != nil {
		return "", fmt.Errorf("JIT failed in template resolution of the dpl_command: %w", err)
	}

	return jitDplGenerate(confSvc, varStack, workflowRepo, "source /etc/profile.d/o2.sh &&"+dplCommand)
}

func generateDplSubworkflowFromUri(confSvc ConfigurationService, varStack map[string]string, workflowRepo repos.IRepo, dplCommandUri string, fallbackToTemplate bool) (jitWorkflowName string, err error) {
	if dplCommandUri == "none" {
		return "", fmt.Errorf("dplCommand is 'none'")
	}

	dplCommand, err := workflowRepo.GetDplCommand(dplCommandUri)
	if err != nil {
		if fallbackToTemplate {
			// if a file in JIT is missing, it will try to fallback to a standard workflow template in 'workflows/'.
			// effectively, this allows us to have an intermediate switch workflow to select different JIT commands
			// for different nodes.
			log.Debugf("JIT: There is no file 'jit/%s' with a DPL command, falling back the template at 'workflows/%s'", dplCommandUri, dplCommandUri)
			return dplCommandUri, nil
		} else {
			return "", fmt.Errorf("Failed to read DPL command from '%s': %w\n", dplCommandUri, err)
		}
	}

	// Resolve any templates as part of the DPL command
	fields := Fields{WrapPointer(&dplCommand)}
	err = fields.Execute(confSvc, dplCommand, varStack, nil, nil, make(map[string]texttemplate.Template), workflowRepo)
	if err != nil {
		return "", fmt.Errorf("JIT failed in template resolution of the dpl_command: %w", err)
	}

	return jitDplGenerate(confSvc, varStack, workflowRepo, "source /etc/profile.d/o2.sh && "+dplCommand)
}
