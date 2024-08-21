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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/AliceO2Group/Control/common/gera"
	"github.com/AliceO2Group/Control/core/repos"
	"github.com/AliceO2Group/Control/core/task"
	"github.com/AliceO2Group/Control/core/task/taskclass"
	"github.com/AliceO2Group/Control/core/the"
	"github.com/k0kubun/pp"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

type LoadSubworkflowFunc func(workflowPathExpr string, parent Updatable) (root *aggregatorRole, workflowRepo repos.IRepo, err error)

// FIXME: workflowPath should be of type configuration.Path, not string
func Load(workflowPath string, parent Updatable, taskManager *task.Manager, userProperties map[string]string, baseConfigStack map[string]string) (workflow Role, err error) {
	repoManager := the.RepoManager()

	var loadSubworkflow LoadSubworkflowFunc = func(workflowPathExpr string, parent Updatable) (root *aggregatorRole, workflowRepo repos.IRepo, err error) {
		var resolvedWorkflowPath string

		resolvedWorkflowPath, workflowRepo, err = repoManager.GetWorkflow(workflowPathExpr) //Will fail if repo unknown
		if err != nil {
			return
		}

		var yamlDoc []byte
		yamlDoc, err = os.ReadFile(resolvedWorkflowPath)
		if err != nil {
			return
		}

		root = new(aggregatorRole)
		root.parent = parent
		err = yaml.Unmarshal(yamlDoc, root)
		if err != nil {
			return nil, nil, err
		}
		if parent != nil {
			root.setParent(parent)
		}
		return
	}

	root, workflowRepo, err := loadSubworkflow(workflowPath, parent)
	if err != nil {
		return nil, err
	}

	var workflowFilename string
	revSlice := strings.Split(workflowPath, "@")
	if len(revSlice) == 2 {
		workflowPath = revSlice[0]
	}
	workflowInfo := strings.Split(workflowPath, "/workflows/")
	if len(workflowInfo) == 1 {
		workflowFilename = workflowInfo[0]

	} else {
		workflowFilename = workflowInfo[1]
	}

	if root.roleBase.Name != workflowFilename {
		err = errors.New("the name of the workflow template file and the name of the workflow don't match")
		return
	}

	workflow = root

	cwd := viper.GetString("coreWorkingDir")

	timestamp := fmt.Sprintf("%f", float64(time.Now().UnixNano())/1e9)
	pp.ColoringEnabled = false
	if viper.GetBool("dumpWorkflows") {
		f, _ := os.Create(filepath.Join(cwd, fmt.Sprintf("wf-unprocessed-%s.json", timestamp)))
		_, _ = pp.Fprintln(f, workflow)
		defer f.Close()
	}

	if !viper.GetBool("concurrentWorkflowTemplateProcessing") {
		log.WithField("workflow", workflowPath).
			Warn("concurrent workflow template aggregator processing is disabled, this will cause a performance hit")
	}
	if !viper.GetBool("concurrentWorkflowTemplateIteratorProcessing") {
		log.WithField("workflow", workflowPath).
			Warn("concurrent workflow template iterator processing is disabled, this will cause a performance hit")
	}
	if !viper.GetBool("concurrentIteratorRoleExpansion") {
		log.WithField("workflow", workflowPath).
			Warn("concurrent iterator role expansion is disabled, this will cause a performance hit")
	}

	err = workflow.ProcessTemplates(workflowRepo, loadSubworkflow, baseConfigStack)
	if err != nil {
		log.WithError(err).Warn("workflow loading failed: template processing error")
		return
	}
	log.WithField("path", workflowPath).Debug("workflow loaded")

	if viper.GetBool("dumpWorkflows") {
		g, _ := os.Create(filepath.Join(cwd, fmt.Sprintf("wf-processed-%s.json", timestamp)))
		_, _ = pp.Fprintln(g, workflow)
		defer g.Close()
	}

	// Update class list
	taskClassesRequired := workflow.GetTaskClasses()
	err = repoManager.EnsureReposPresent(taskClassesRequired)
	if err != nil {
		return
	}
	err = taskManager.RefreshClasses(taskClassesRequired)
	return
}

func LoadDPL(tasks []*taskclass.Class, rootRoleName string, extraVarsMap map[string]string) (workflow Role, err error) {
	root := new(aggregatorRole)

	root.roleBase.Name = rootRoleName
	root.roleBase.Vars = gera.MakeMapWithMap(extraVarsMap)

	for _, taskItem := range tasks {
		SingleTaskRole := taskRole{
			roleBase: roleBase{
				Name:   taskItem.Identifier.Name,
				parent: root,
			},
		}

		SingleTaskRole.Connect = append(SingleTaskRole.Connect, taskItem.Connect...)
		SingleTaskRole.Constraints = append(SingleTaskRole.Constraints, taskItem.Constraints...)
		SingleTaskRole.Bind = append(SingleTaskRole.Bind, taskItem.Bind...)
		SingleTaskRole.Task = task.ClassToTask(taskItem, &SingleTaskRole)

		root.aggregator.Roles = append(root.aggregator.Roles, &SingleTaskRole)
	}

	workflow = root

	// FIXME: either get rid of err or add handling of errors
	return workflow, nil
}

// RoleToYAML exists to avoid exporting aggregatorRole/iteratorRole.
// Simply put, this function is nothing but a wrapper which allows marshalling Roles outside the workflow package
func RoleToYAML(input Role) ([]byte, error) {
	switch input.(type) {
	case *aggregatorRole:
		return yaml.Marshal(input.(*aggregatorRole))
	case *iteratorRole:
		return yaml.Marshal(input.(*iteratorRole))
	default:
		return nil, nil
	}
}

func LoadWorkflow(in []byte) (out yaml.Node, err error) {
	err = yaml.Unmarshal(in, &out)
	if err != nil {
		return
	}

	return out, nil
}
