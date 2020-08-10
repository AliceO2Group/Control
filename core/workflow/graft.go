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
// role to the role in root where the path specifies..
func Graft(root *yaml.Node, path string, toAdd []byte) (out []byte, err error)  {
    roleToAdd, err := LoadWorkflow(toAdd)
    if err != nil {
        return nil, err
    }

    var parent *yaml.Node

    for _, step := range strings.Split(path, PATH_SEPARATOR) {
        // FIXME: no return value
        _ = iterateNode(root, &parent, step)
    }

    if parent == nil {
        return nil, fmt.Errorf("specified path not found")
    }

    // Not appending to root, only to a copy of root
    parent.Content = append(parent.Content, roleToAdd.Content[0])

    out, err = yaml.Marshal(root)
    if err != nil{
        return nil, err
    }

    return out, nil
}

// When passed a node, iterate through each node of its node.Content array. If found, return that node.
// If not found, call iterateNode on the current node's node.Content array. The goal of this function is to get
// the parent node of where the search string is found.
func iterateNode(node *yaml.Node, parent **yaml.Node,identifier string) (found *yaml.Node) {
    for _, n := range node.Content {
        if n.Value == identifier {
            return n
        }
        if len(n.Content) > 0 {
            if n.Tag == "!!seq" {
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
