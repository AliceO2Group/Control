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

package core

import (
	"fmt"
	"github.com/AliceO2Group/Control/core/repos"
	"sync"

	"github.com/AliceO2Group/Control/common"
	"github.com/AliceO2Group/Control/core/protos"
	"github.com/AliceO2Group/Control/core/task/channel"

	"github.com/AliceO2Group/Control/core/task"
	"github.com/AliceO2Group/Control/core/workflow"
)

const MESOS_AGENT_PORT = 5051

func commandInfoToPbCommandInfo(c *common.TaskCommandInfo) (pci *pb.CommandInfo) {
	if c == nil {
		return
	}
	pci = &pb.CommandInfo{
		Env: c.GetEnv(),
		Shell: c.GetShell(),
		Value: c.GetValue(),
		Arguments: c.GetArguments(),
		User: c.GetUser(),
	}
	return
}

func inboundChannelsToPbChannels(chs []channel.Inbound) (pchs []*pb.ChannelInfo) {
	if chs == nil {
		return
	}
	pchs = make([]*pb.ChannelInfo, len(chs))
	for i, c := range chs {
		pchs[i] = inboundChannelToPbChannel(c)
	}
	return
}

func inboundChannelToPbChannel(ch channel.Inbound) (pch *pb.ChannelInfo) {
	pch = &pb.ChannelInfo{
		Name: ch.Name,
		Type: ch.Type.String(),
	}
	return
}

func outboundChannelsToPbChannels(chs []channel.Outbound) (pchs []*pb.ChannelInfo) {
	if chs == nil {
		return
	}
	pchs = make([]*pb.ChannelInfo, len(chs))
	for i, c := range chs {
		pchs[i] = outboundChannelToPbChannel(c)
	}
	return
}

func outboundChannelToPbChannel(ch channel.Outbound) (pch *pb.ChannelInfo) {
	pch = &pb.ChannelInfo{
		Name: ch.Name,
		Type: ch.Type.String(),
		Target: ch.Target,
	}
	return
}

func taskToShortTaskInfo(t *task.Task, taskman *task.Manager) (sti *pb.ShortTaskInfo) {
	if t == nil {
		return
	}

	var(
		slaveHost = t.GetHostname()
		slavePort = MESOS_AGENT_PORT
		workDir = "/var/mesos"
		slaveId = t.GetAgentId()
		frameworkId = taskman.GetFrameworkID()
		executorId = t.GetExecutorId()
		containerId = "latest"
	)
	sandboxStdoutUri := fmt.Sprintf("http://%s:%d/files/download?path=%s/slaves/%s/frameworks/%s/executors/%s/runs/%s/stdout",
		slaveHost,
		slavePort,
		workDir,
		slaveId,
		frameworkId,
		executorId,
		containerId)

	sti = &pb.ShortTaskInfo{
		Name:   t.GetName(),
		Locked: t.IsLocked(),
		TaskId: t.GetTaskId(),
		Status: "UNKNOWN",
		State:  "UNKNOWN",
		ClassName: t.GetClassName(),
		DeploymentInfo: &pb.TaskDeploymentInfo{
			Hostname: t.GetHostname(),
			AgentId: t.GetAgentId(),
			OfferId: t.GetOfferId(),
			ExecutorId: t.GetExecutorId(),
		},
		Pid: t.GetTaskPID(),
		SandboxStdout: sandboxStdoutUri,
	}
	parentRole, ok := t.GetParentRole().(workflow.Role)
	if ok && parentRole != nil {
		sti.Status = parentRole.GetStatus().String()
		sti.State = parentRole.GetState().String()
	}
	return
}

func tasksToShortTaskInfos(tasks []*task.Task, taskman *task.Manager) (stis []*pb.ShortTaskInfo) {
	if tasks == nil {
		return
	}
	stis = make([]*pb.ShortTaskInfo, len(tasks))
	for i, t := range tasks {
		shortTaskInfo := taskToShortTaskInfo(t, taskman)
		stis[i] = shortTaskInfo
	}
	return
}

func tasksToTaskIds(tasks []*task.Task) (ss []string) {
	if tasks == nil {
		return
	}
	ss = make([]string, len(tasks))
	for i, t := range tasks {
		ss[i] = t.GetTaskId()
	}
	return
}

func workflowToRoleTree(root workflow.Role) (ri *pb.RoleInfo) {
	if root == nil {
		return
	}
	childRoles := root.GetRoles()
	childRoleInfos := make([]*pb.RoleInfo, len(childRoles))
	for i, cr := range childRoles {
		childRoleInfos[i] = workflowToRoleTree(cr)
	}
	consolidatedVarStack, _ := root.ConsolidatedVarStack()
	ri = &pb.RoleInfo{
		Name:              root.GetName(),
		Status:            root.GetStatus().String(),
		State:             root.GetState().String(),
		FullPath:          root.GetPath(),
		TaskIds:           tasksToTaskIds(root.GetTasks()),
		Roles:             childRoleInfos,
		Defaults:          root.GetDefaults().Raw(),
		Vars:              root.GetVars().Raw(),
		UserVars:          root.GetUserVars().Raw(),
		ConsolidatedStack: consolidatedVarStack,
	}
	return
}

// SafeStreamsMap is a safe map where the key is usually a
// subscriptionID received from the grpc call and as a value
// a channel where get events from the environment 
// and we stream them to the grpc client.
type SafeStreamsMap struct {
	mu sync.RWMutex
	streams map[string]chan *pb.Event
}

func (s *SafeStreamsMap) add(id string, ch chan *pb.Event) {
	s.mu.Lock()
	s.streams[id] = ch
	s.mu.Unlock()
}

func (s *SafeStreamsMap) delete(id string) {
	s.mu.Lock()
	delete(s.streams, id)
	s.mu.Unlock()
}

func (s *SafeStreamsMap) GetChannel(id string) (ch chan *pb.Event, ok bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ch, ok = s.streams[id]
	return
}

func newSafeStreamsMap() SafeStreamsMap {
	return SafeStreamsMap{
		streams: make(map[string]chan *pb.Event),
	}
}

func VarSpecMapToPbVarSpecMap(varSpecMap map[string]repos.VarSpec) map[string]*pb.VarSpecMessage {
	ret := make(map[string]*pb.VarSpecMessage)
	var vsm *pb.VarSpecMessage
	for k, v := range varSpecMap {
		vsm = &pb.VarSpecMessage {
			DefaultValue: v.DefaultValue,
			Type: convertVarTypeStringToEnum(v.VarType),
			Label: v.Label,
			Description: v.Description,
			UiWidgetHint: convertUiWidgetHintStringToEnum(v.UiWidgetHint),
			Panel: v.Panel,
			AllowedValues: v.AllowedValues,
		}
		ret[k] = vsm
	}
	return ret
}

func convertUiWidgetHintStringToEnum(hint string) pb.VarSpecMessage_UiWidget {
	switch hint {
	case "slider":
		return 1
	case "listBox":
		return 2
	case "dropDownBox":
		return 3
	case "comboBox":
		return 4
	default:
		return 0 // "editBox
	}
}

func convertVarTypeStringToEnum(varType string) pb.VarSpecMessage_Type {
	switch varType {
	case "bool":
		return 0
	case "number":
		return 1
	case "list":
		return 3
	case "map":
		return 4
	default:
		return 2 // "string"
	}
}