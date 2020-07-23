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
	"github.com/AliceO2Group/Control/common"
	"github.com/AliceO2Group/Control/core/protos"
	"github.com/AliceO2Group/Control/core/task/channel"

	"github.com/AliceO2Group/Control/core/task"
	"github.com/AliceO2Group/Control/core/workflow"
)

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

func taskToShortTaskInfo(t *task.Task) (sti *pb.ShortTaskInfo) {
	if t == nil {
		return
	}
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
		Pid: "UNKNOWN",
	}
	parentRole, ok := t.GetParentRole().(workflow.Role)
	if ok && parentRole != nil {
		sti.Status = parentRole.GetStatus().String()
		sti.State = parentRole.GetState().String()
		sti.Pid = parentRole.GetTaskPID()
	}
	return
}

func tasksToShortTaskInfos(tasks []*task.Task) (stis []*pb.ShortTaskInfo) {
	if tasks == nil {
		return
	}
	stis = make([]*pb.ShortTaskInfo, len(tasks))
	for i, t := range tasks {
		shortTaskInfo := taskToShortTaskInfo(t)
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
	ri = &pb.RoleInfo{
		Name:     root.GetName(),
		Status:   root.GetStatus().String(),
		State:    root.GetState().String(),
		FullPath: root.GetPath(),
		TaskIds:  tasksToTaskIds(root.GetTasks()),
		Roles:    childRoleInfos,
		Defaults: root.GetDefaults().Raw(),
		Vars:     root.GetVars().Raw(),
		UserVars: root.GetUserVars().Raw(),
	}
	return
}
