/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2021 CERN and copyright holders of ALICE O².
 * Author: Miltiadis Alexis <miltiadis.alexis@cern.ch>
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

package environment

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/AliceO2Group/Control/core/the"
	"gopkg.in/yaml.v3"
)

type WorkflowPublicInfo struct {
	IsPublic    bool
	Name        string
	Description string
}

func parseWorkflowPublicInfo(workflowExpr string) (WorkflowPublicInfo, error) {
	repoManager := the.RepoManager()

	resolvedWorkflowPath, _, err := repoManager.GetWorkflow(workflowExpr) //Will fail if repo unknown
	if err != nil {
		return WorkflowPublicInfo{}, err
	}

	yamlFile, err := os.ReadFile(resolvedWorkflowPath)
	if err != nil {
		return WorkflowPublicInfo{}, err
	}

	nodes := make(map[string]yaml.Node)
	err = yaml.Unmarshal(yamlFile, &nodes)
	if err != nil {
		return WorkflowPublicInfo{}, err
	}

	name := nodes["name"].Value

	description := ""
	isPublic := nodes["name"].Tag == "!public"
	if nodes["description"].Tag == "!public" {
		description = nodes["description"].Value
	}

	return WorkflowPublicInfo{IsPublic: isPublic, Name: name, Description: description}, nil
}

func JSONSliceToSlice(payload string) (slice []string, err error) {
	slice = make([]string, 0)
	err = json.Unmarshal([]byte(payload), &slice)
	return
}

func SliceToJSONSlice(slice []string) (payload string, err error) {
	var payloadStr []byte
	payloadStr, err = json.Marshal(slice)
	payload = string(payloadStr)
	return
}

func sortMapToString(m map[string]string) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	b := new(bytes.Buffer)

	for _, k := range keys {
		fmt.Fprintf(b, "%s=\"%s\"\n", k, m[k])
	}
	return b.String()
}
