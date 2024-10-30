/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2020 CERN and copyright holders of ALICE O².
 * Author: Ayaan Zaidi <azaidi@cern.ch>
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
	"gopkg.in/yaml.v3"
	"strings"
)

// Graft takes a root node, a path to a role in root, byte array with an existing role and appends this
// role asa child of the role in root where the path specifies.
func Graft(root *yaml.Node, path string, toAdd []byte, graftedName string) (out []byte, err error) {
	var parent *yaml.Node
	for _, step := range strings.Split(path, PATH_SEPARATOR) {
		_ = iterateNode(root, &parent, step)
	}
	if parent == nil {
		return nil, fmt.Errorf("specified path not found")
	}

	err = appendRole(parent, toAdd)
	if err != nil {
		return nil, err
	}

	if graftedName != "" {
		nameNode := iterateNode(root, &parent, "name")
		for index, node := range parent.Content {
			// Search for the name node and change the Value field for the subsequent node
			if node == nameNode {
				parent.Content[index+1].Value = graftedName
				break
			}
		}
	}

	out, err = yaml.Marshal(root)
	if err != nil {
		return nil, err
	}

	return out, nil
}

// When passed a node, iterate through each node of its node.Content array. If found, return that node.
// If not found, call iterateNode on the current node's node.Content array. The goal of this function is to get
// the parent node of where the search string is found.
func iterateNode(node *yaml.Node, parent **yaml.Node, identifier string) (found *yaml.Node) {
	for _, n := range node.Content {
		if n.Value == identifier {
			return n
		}
		if len(n.Content) > 0 {
			if n.Tag == "!!map" {
				*parent = n
			}
			acNode := iterateNode(n, parent, identifier)
			if acNode != nil {
				return acNode
			}
		}
	}
	return nil
}

// appendRole checks if the yaml.Node passed has a "roles" field. If it exists, the toAdd yaml.Node will
// be appended. If not, appendRole will error out.
func appendRole(parent *yaml.Node, toAdd []byte) (err error) {
	var childRole yaml.Node
	err = yaml.Unmarshal(toAdd, &childRole)
	var auxParent *yaml.Node // dummy value to run iterateNode successfully

	if iterateNode(parent, &auxParent, "roles") != nil {
		for i, v := range parent.Content {
			if v.Value == "roles" {
				// If a !!str yaml.Node with value "roles" found, append toAdd to the next yaml.Node's Content
				parent.Content[i+1].Content = append(parent.Content[i+1].Content, childRole.Content[0])
			}
		}
	} else {
		// error out if current yaml.Node does not have a "roles" field
		return fmt.Errorf("specified node does not have a 'roles' field")
	}
	return
}
