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
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/AliceO2Group/Control/common/logger"
	"github.com/AliceO2Group/Control/common/utils"
	"github.com/sirupsen/logrus"
)

const QUEUE_SIZE = 16384 // upper limit of command queue size

var log = logger.New(logrus.StandardLogger(), "cmdq")

type queueEntry struct {
	cmd      MesosCommand
	callback chan<- MesosCommandResponse
}

type empty struct{}

type CommandQueue struct {
	sync.Mutex

	q       chan queueEntry
	servent *Servent
}

func NewCommandQueue(s *Servent) *CommandQueue {
	return &CommandQueue{
		servent: s,
	}
}

func (m *CommandQueue) Enqueue(cmd MesosCommand, callback chan<- MesosCommandResponse) error {
	select {
	case m.q <- queueEntry{cmd, callback}:
		return nil
	default: // Buffer full!
		err := errors.New("the queue for MESSAGE commands is full")
		log.WithField("error", err.Error()).
			WithField("queueSize", QUEUE_SIZE).
			Error("cannot enqueue control command")
		return err
	}
}

func (m *CommandQueue) Start() {
	m.Lock()
	m.q = make(chan queueEntry, QUEUE_SIZE)
	m.Unlock()

	go func() {
		for {
			select {
			case entry, more := <-m.q:
				if !more { // if the channel is closed, we bail
					return
				}
				m.Lock()
				response, err := m.commit(entry.cmd)
				if err != nil {
					if entry.cmd != nil {
						log.WithError(err).
							Debugf("failed to commit CommandQueue entry %s", entry.cmd.GetName())
					} else {
						log.WithError(err).
							Debug("failed to commit unknown CommandQueue entry")
					}
				}
				if err == nil && response == nil {
					log.Error("nil response")
				}

				entry.callback <- response
				m.Unlock()
			}
		}
	}()
}

func (m *CommandQueue) Stop() {
	m.Lock()
	defer m.Unlock()
	close(m.q)
}

func (m *CommandQueue) commit(command MesosCommand) (response MesosCommandResponse, err error) {
	if m == nil {
		return nil, errors.New("command queue is nil")
	}
	defer utils.TimeTrack(time.Now(), fmt.Sprintf("cmdq.commit %s to %d targets", command.GetName(), len(command.targets())), log.WithPrefix("cmdq"))

	type responseSemaphore struct {
		receiver MesosCommandTarget
		response MesosCommandResponse
		err      error
	}

	// Parallel for
	sendErrorList := make([]error, 0)
	semaphore := make(chan responseSemaphore, len(command.targets()))

	responses := make(map[MesosCommandTarget]MesosCommandResponse)

	log.WithFields(logrus.Fields{
		"name": command.GetName(),
		"id":   command.GetId(),
	}).
		Debug("ready to commit MesosCommand")

	for _, rec := range command.targets() {
		go func(receiver MesosCommandTarget) {
			log.WithFields(logrus.Fields{
				"agentId":    receiver.AgentId,
				"executorId": receiver.ExecutorId,
				"name":       command.GetName(),
			}).
				Trace("sending MesosCommand to target")
			singleCommand := command.MakeSingleTarget(receiver)
			res, err := m.servent.RunCommand(singleCommand, receiver)
			if err != nil {
				log.WithError(err).Warning("MesosCommand send error")

				semaphore <- responseSemaphore{
					receiver: receiver,
					response: res,
					err:      err,
				}
				return
			}

			if res.Err() != nil {
				log.WithFields(logrus.Fields{
					"commandName": res.GetCommandName(),
					"error":       res.Err().Error(),
				}).
					Trace("received MesosCommandResponse with error")
			} else {
				log.WithFields(logrus.Fields{
					"commandName": res.GetCommandName(),
				}).
					Trace("received MesosCommandResponse")
			}

			semaphore <- responseSemaphore{
				receiver: receiver,
				response: res,
			}
		}(rec)
	}
	// Wait for goroutines to finish
	for i := 0; i < len(command.targets()); i++ {
		respSemaphore := <-semaphore
		responses[respSemaphore.receiver] = respSemaphore.response
		if respSemaphore.err != nil {
			sendErrorList = append(sendErrorList, respSemaphore.err)
		}
	}
	close(semaphore)

	log.WithFields(logrus.Fields{}).Debug("responses collected")

	if len(sendErrorList) != 0 {
		err = errors.New(strings.Join(func() (out []string) {
			for i, e := range sendErrorList {
				out = append(out, fmt.Sprintf("[%d] %s", i, e.Error()))
			}
			return
		}(), "\n"))
		return
	}
	response = consolidateResponses(command, responses)

	log.Debug("responses consolidated, CommandQueue commit done")

	return response, nil
}
