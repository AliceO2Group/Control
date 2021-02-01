/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2020 CERN and copyright holders of ALICE O².
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

package task

import (
	"sync"
	"github.com/AliceO2Group/Control/core/protos"
)

func taskToShortTaskInfo(t *Task) (sti *pb.ShortTaskInfo) {
	if t == nil {
		return
	}
	sti = &pb.ShortTaskInfo{
		Name:   t.GetName(),
		Locked: t.IsLocked(),
		TaskId: t.GetTaskId(),
		ClassName: t.GetClassName(),
		DeploymentInfo: &pb.TaskDeploymentInfo{
			Hostname: t.GetHostname(),
			AgentId: t.GetAgentId(),
			OfferId: t.GetOfferId(),
			ExecutorId: t.GetExecutorId(),
		},
		Pid: t.GetTaskPID(),
	}
	return
}

func tasksToShortTaskInfos(tasks []*Task) (stis []*pb.ShortTaskInfo) {
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

type safeAcks struct {
	mu       sync.RWMutex
	acks     map[string]chan struct{}
}

func (a *safeAcks) getMap() map[string]chan struct{} {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return a.acks
}

func (a *safeAcks) deleteKey(key string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	delete(a.acks, key)
}

func (a *safeAcks) contains(key string) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()

	_, ok := a.acks[key]
	
	return ok
}


func (a *safeAcks) addAckChannel(key string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	
	a.acks[key] = make(chan struct{})
}

func (a *safeAcks) getValue(key string) (ch chan struct{}, ok bool) {
	a.mu.Lock()
	defer a.mu.Unlock()

	ch, ok = a.acks[key]
	return 
}

func newAcks() *safeAcks {
	return &safeAcks{
		acks: make(map[string]chan struct{}),
	}
}