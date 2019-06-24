/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2018 CERN and copyright holders of ALICE O².
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

package workflow

import (
	"github.com/AliceO2Group/Control/configuration"
	"github.com/AliceO2Group/Control/core/repos"
	"github.com/AliceO2Group/Control/core/the"

	//"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	//"strings"
)

// FIXME: workflowPath should be of type configuration.Path, not string
func Load(cfg configuration.ROSource, workflowPath string, parent Updatable) (workflow Role, err error) {
	var yamlDoc []byte

	reposInstance := the.RepoManager()

	var resolvedWorkflowPath string
	var workflowRepo *repos.Repo
	resolvedWorkflowPath, workflowRepo, err = reposInstance.GetWorkflow(workflowPath) //Will fail if repo unknown
	if err != nil {
		return
	}

	yamlDoc, err = ioutil.ReadFile(resolvedWorkflowPath)
	if err != nil {
		return
	}


	root := new(aggregatorRole)
	root.parent = parent
	err = yaml.Unmarshal(yamlDoc, root)
	if err != nil {
		return nil, err
	}
	if parent != nil {
		root.parent = parent
	}

	workflow = root
	workflow.ProcessTemplates(workflowRepo)
	//pp.Println(workflow)

	return
}