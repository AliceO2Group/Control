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

package controlcommands

import (
	"fmt"
	"sync"
	"time"

	"github.com/AliceO2Group/Control/common/utils"
	"github.com/rs/xid"
	"github.com/sirupsen/logrus"
)

type SendCommandFunc func(command MesosCommand, receiver MesosCommandTarget) error

// Call represents an active request, pending response
type Call struct {
	Request  MesosCommand
	Response MesosCommandResponse
	Done     chan empty
	Error    error
}

func NewCall(cmd MesosCommand) *Call {
	return &Call{
		Request:  cmd,
		Response: nil,
		Done:     make(chan empty),
		Error:    nil,
	}
}

type CallId struct {
	Id     xid.ID
	Target MesosCommandTarget
}

type Servent struct {
	mu      sync.Mutex
	pending map[CallId]*Call

	SendFunc SendCommandFunc
}

func NewServent(commandFunc SendCommandFunc) *Servent {
	return &Servent{
		SendFunc: commandFunc,
		pending:  make(map[CallId]*Call),
	}
}

func (s *Servent) RunCommand(cmd MesosCommand, receiver MesosCommandTarget) (MesosCommandResponse, error) {
	log.Debug("Servent.RunCommand BEGIN")
	defer log.Debug("Servent.RunCommand END")

	defer utils.TimeTrack(time.Now(), fmt.Sprintf("servent.RunCommand %s to target %s", cmd.GetName(), receiver.TaskId.Value),
		log.WithPrefix("servent").WithField("partition", cmd.GetEnvironmentId().String()))

	cmdId := cmd.GetId()
	call := NewCall(cmd)

	callId := CallId{
		Id:     cmdId,
		Target: receiver,
	}

	log.WithPrefix("servent").
		WithField("partition", cmd.GetEnvironmentId().String()).
		Trace("servent mutex locking")
	s.mu.Lock()
	log.WithPrefix("servent").
		WithField("partition", cmd.GetEnvironmentId().String()).
		Trace("servent mutex locked")

	// We append the new call to the pending map, and send the request
	s.pending[callId] = call

	s.mu.Unlock()
	log.WithPrefix("servent").
		WithField("partition", cmd.GetEnvironmentId().String()).
		Trace("servent mutex unlocked")

	log.WithPrefix("servent").
		WithFields(logrus.Fields{
			"name":       cmd.GetName(),
			"partition":  cmd.GetEnvironmentId().String(),
			"id":         cmd.GetId(),
			"agentId":    receiver.AgentId,
			"executorId": receiver.ExecutorId,
		}).
		Trace("calling scheduler SendFunc")

	err := s.SendFunc(cmd, receiver)
	if err != nil {
		log.WithPrefix("servent").
			WithField("partition", cmd.GetEnvironmentId().String()).
			Trace("servent mutex locking")
		s.mu.Lock()
		log.WithPrefix("servent").
			WithField("partition", cmd.GetEnvironmentId().String()).
			Trace("servent mutex locked")

		delete(s.pending, callId)

		s.mu.Unlock()
		log.WithPrefix("servent").
			WithField("partition", cmd.GetEnvironmentId().String()).
			WithError(err).
			Trace("servent mutex unlocked")

		return nil, err
	}

	log.WithPrefix("servent").
		WithField("partition", cmd.GetEnvironmentId().String()).
		WithField("timeout", cmd.GetResponseTimeout()).
		Trace("blocking until response or timeout")
	// Neat, now we block until done||timeout
	select {
	case <-call.Done:
		// By the time we get here, ProcessResponse should have already added a Response to the
		// pending call, and removed it from servent.pending.
	case <-time.After(cmd.GetResponseTimeout()):
		call.Error = fmt.Errorf("%s timed out for task %s", cmd.GetName(), receiver.TaskId.Value)

		log.WithPrefix("servent").
			WithField("partition", cmd.GetEnvironmentId().String()).
			Trace("servent mutex locking")
		s.mu.Lock()
		log.WithPrefix("servent").
			WithField("partition", cmd.GetEnvironmentId().String()).
			Trace("servent mutex locked")

		delete(s.pending, callId)

		s.mu.Unlock()
		log.WithPrefix("servent").
			WithField("partition", cmd.GetEnvironmentId().String()).
			Trace("servent mutex unlocked")
	}

	if call.Error != nil {
		return nil, call.Error
	}
	return call.Response, nil
}

func (s *Servent) ProcessResponse(res MesosCommandResponse, sender MesosCommandTarget) {
	callId := CallId{
		Id:     res.GetCommandId(),
		Target: sender,
	}

	s.mu.Lock()
	call, ok := s.pending[callId]
	delete(s.pending, callId)
	s.mu.Unlock()

	if !ok || call == nil {
		log.WithPrefix("servent").
			WithField("partition", res.GetEnvironmentId().String()).
			WithFields(logrus.Fields{
				"commandName": res.GetCommandName(),
				"commandId":   res.GetCommandId(),
				"agentId":     sender.AgentId,
				"executorId":  sender.ExecutorId,
			}).
			Warning("no pending request found")
		return
	}

	call.Response = res
	call.Done <- empty{}
}
