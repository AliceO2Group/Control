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
    "gopkg.in/yaml.v3"
    "strings"
)

func Graft(root yaml.Node, path string, toAdd []byte) (out Role, err error)  {
    roleToAdd, err := LoadWorkflow(toAdd)

    destinationNode := navigateNode(root, path)
    destinationNode.Content = append(destinationNode.Content, navigateNode(root, path))
    destinationNode.Content = append(destinationNode.Content, roleToAdd.Content[0].Content...)
    destinationNode.Tag = "!!seq"

    var result aggregatorRole
    err = root.Decode(&result)
    if err != nil{
        return nil, err
    }

    out = &result

    return out, nil
}

func navigateNode(root yaml.Node, path string) (out *yaml.Node) {
    steps := strings.Split(path, PATH_SEPARATOR)

    for _, step := range steps {
        out = iterateNode(&root, step)
    }

    return out
}

func iterateNode(node *yaml.Node, identifier string) *yaml.Node {
    for _, n := range node.Content {
        if n.Value == identifier {
            return node
        }
        if len(n.Content) > 0 {
            acNode := iterateNode(n, identifier)
            if acNode != nil {
                return acNode
            }
        }
    }
    return nil
}
