/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2021 CERN and copyright holders of ALICE O².
 * Author: Kostas Alexopoulos <kostas.alexopoulos@cern.ch>
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

package repos

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"io/ioutil"
)

func IsFilePublicWorkflow(filePath string) bool {
	yamlFile, err := ioutil.ReadFile(filePath)
	if err != nil { return false }
	return IsPublicWorkflow(yamlFile)
}

func IsPublicWorkflow(yamlFile []byte) bool {
	var nameNode struct {
		Node yaml.Node `yaml:"name"`
	}
	err := yaml.Unmarshal(yamlFile, &nameNode)
	if err != nil { return false }
	return nameNode.Node.Tag == "!public"
}

func ParseWorkflowPublicVariableInfo(fileName string) (map[string]VarSpec, error) {
	yamlFile, err := ioutil.ReadFile(fileName)
	if err != nil { return nil, err }

	nodes := make(map[string]yaml.Node)
	err = yaml.Unmarshal(yamlFile, &nodes)
	if err != nil { return nil, err }

	workflowVarInfo := make(map[string]VarSpec)
	for k, v := range nodes {
		if err = parseYamlPublicVars(&AuxNode{k, &v }, &workflowVarInfo); err != nil {
			return nil, err
		}
	}

	return workflowVarInfo, nil
}

// VarSpec is the type of struct into which public variable information from workflows may be parsed
type VarSpec struct {
	DefaultValue string `yaml:"value"`
	VarType string `yaml:"type"`
	Label string `yaml:"label"`
	Description string `yaml:"description"`
	UiWidgetHint string `yaml:"widget"`
	Panel string `yaml:"panel" `
	AllowedValues []string `yaml:"values"`
}

// AuxNode Use an auxiliary node struct that also carries its parent name
type AuxNode struct {
	parentName string
	node *yaml.Node
}

func parseYamlPublicVars(auxNode *AuxNode, workflowVarInfo *map[string]VarSpec) error {
	node := auxNode.node

	// Recursion stops if node is nil, or isn't a mapping or a sequence node
	if node == nil ||
		node.Kind != yaml.MappingNode && node.Kind != yaml.SequenceNode {
		return nil
	}

	if node.Kind == yaml.SequenceNode { // If it's a sequence node, continue searching for a map within it
		for _, v := range node.Content {
			err := parseYamlPublicVars(&AuxNode{"", v}, workflowVarInfo)
			if err != nil { return err}
		}
	} else if node.Kind == yaml.MappingNode { // If it's a mapping node, iterate through it
		// We do this decoding to have a sane key -> node map
		// otherwise, we get two yaml nodes for a single element
		// with the first one holding the name and the second one holding the tag
		m := make(map[string]yaml.Node)
		err := node.Decode(&m)
		if err != nil { return err }

		parentName := auxNode.parentName
		// Search within the node contents for a "!public" mapping node,
		// which is also the ancestor of a "defaults" or "vars" parent
		for k, v := range m {
			var varSpec VarSpec

			if (parentName == "defaults" || parentName == "vars") && v.Kind == yaml.MappingNode && v.Tag == "!public" {
				err = v.Decode(&varSpec)
				if err != nil { fmt.Println(err); continue }

				// If the key already exists we have come upon a duplicate!
				if _, exists := (*workflowVarInfo)[k]; exists {
					duplicateError := fmt.Errorf("duplicate public variable \"%s\" parsed, input workflow file invalid", k)
					return duplicateError
				}

				// Update the map
				(*workflowVarInfo)[k] = varSpec
			} else {
				err = parseYamlPublicVars(&AuxNode{k, &v}, workflowVarInfo)
				if err != nil {  return err }
			}
		}
	}
	return nil
}