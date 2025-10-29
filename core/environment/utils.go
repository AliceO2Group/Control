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
	"errors"
	"fmt"
	"github.com/AliceO2Group/Control/common/logger/infologger"
	pb "github.com/AliceO2Group/Control/common/protos"
	"github.com/AliceO2Group/Control/core/task"
	"github.com/AliceO2Group/Control/core/task/sm"
	"github.com/AliceO2Group/Control/core/workflow"
	"github.com/looplab/fsm"
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
		_, err := fmt.Fprintf(b, "%s=\"%s\"\n", k, m[k])
		if err != nil {
			log.WithField(infologger.Level, infologger.IL_Devel).Errorf("Error formatting or buffering string for key %s: %v", k, err)
		}
	}
	return b.String()
}

func NewEnvGoErrorEvent(env *Environment, err string) *pb.Ev_EnvironmentEvent {
	return &pb.Ev_EnvironmentEvent{
		EnvironmentId:        env.GetId().String(),
		State:                env.Sm.Current(),
		RunNumber:            env.GetCurrentRunNumber(),
		Error:                err,
		Message:              "a critical error occurred, GO_ERROR transition imminent",
		LastRequestUser:      env.GetLastRequestUser(),
		WorkflowTemplateInfo: env.GetWorkflowInfo(),
	}
}

func newCriticalTasksErrorMessage(env *Environment) string {
	criticalTasksInError := env.workflow.GetTasks().Filtered(func(t *task.Task) bool {
		return t.GetTraits().Critical && t.GetState() == sm.ERROR
	})

	if len(criticalTasksInError) == 0 {
		return "no critical tasks in ERROR"
	} else if len(criticalTasksInError) == 1 {
		t := criticalTasksInError[0]
		name := t.GetName()

		// if available, we prefer role name, because it does not have a long hash for JIT-generated DPL tasks
		role, ok := t.GetParentRole().(workflow.Role)
		if ok {
			name = role.GetName()
		}
		return fmt.Sprintf("critical task '%s' on host '%s' transitioned to ERROR", name, t.GetHostname())
	} else {
		return fmt.Sprintf("%d critical tasks transitioned to ERROR, could not determine the first one to fail", len(criticalTasksInError))
	}
}

func handleFailedGoError(err error, env *Environment) {
	var invalidEventErr *fsm.InvalidEventError
	if errors.As(err, &invalidEventErr) {
		// this case can occur if the environment is in either:
		// - ERROR (env already transitioned to ERROR for another reason)
		// - DONE (an error might have occurred during teardown, but it's already over, no point in spreading panic)
		log.WithError(invalidEventErr).
			WithField("partition", env.Id().String()).
			WithField("run", env.currentRunNumber).
			WithField("state", env.CurrentState()).
			WithField(infologger.Level, infologger.IL_Support).
			Warn("did not perform GO_ERROR transition")
	} else {
		// in principle this should never happen, so we log it accordingly and force the ERROR state just in case
		log.WithError(err).
			WithField("partition", env.Id().String()).
			WithField("run", env.currentRunNumber).
			WithField("state", env.CurrentState()).
			WithField(infologger.Level, infologger.IL_Ops).
			Error("could not perform GO_ERROR transition due to unexpected error, forcing...")
		env.setState("ERROR")
	}
}
