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
	"sync"
	"github.com/AliceO2Group/Control/common/logger"
	"github.com/sirupsen/logrus"
	"errors"
	"fmt"
	"strings"
)

const QUEUE_SIZE = 1024

var log = logger.New(logrus.StandardLogger(), "cmdq")

type queueEntry struct {
	cmd      MesosCommand
	callback chan<- Response
}

type SendCommandFunc func(command MesosCommand, receiver MesosCommandReceiver) (*SingleResponse, error)

type CommandQueue struct {
	sync.Mutex

	q        chan queueEntry
	SendFunc SendCommandFunc
}

func (m *CommandQueue) Enqueue(cmd MesosCommand, callback chan<- Response) error {
	m.Lock()
	defer m.Unlock()

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

func (m* CommandQueue) Start() {
	m.q = make(chan queueEntry, QUEUE_SIZE)
	go func() {
		for {
			select {
			case entry, more := <-m.q:
				if !more {  // if the channel is closed, we bail
					return
				}
				response, err := m.commit(entry.cmd)

				log.Debug(response.Error())
				log.Debug(err.Error())

				entry.callback <- response
			}
		}
	}()
}

func (m *CommandQueue) Stop() {
	close(m.q)
}

func (m *CommandQueue) commit(command MesosCommand) (response *MultiResponse, err error) {
	if m == nil {
		return nil, errors.New("command queue is nil")
	}

	// Parallel for
	type empty struct{}
	errorList := make([]error, 0)
	semaphore := make(chan empty, len(command.receivers()))

	for _, rec := range command.receivers() {
		go func(receiver MesosCommandReceiver) {
			res, err := m.SendFunc(command, receiver)
			response.responses[receiver] = res
			errorList = append(errorList, err)
			semaphore <- empty{}
		}(rec)
	}
	// Wait for goroutines to finish
	for i := 0; i < len(command.receivers()); i++ {
		<- semaphore
	}
	close(semaphore)

	if len(errorList) != 0 {
		err = errors.New(strings.Join(func() (out []string) {
			for i, e := range errorList {
				out = append(out, fmt.Sprintf("[%d] %s", i, e.Error()))
			}
			return
		}(), "\n"))
		return
	}

	return response, nil
}
/*
 * Hook up all of the above from roleman.ConfigureRoles.
 * Hook up roleman.ConfigureRoles to be called from the Control GUI gRPC server.
 * Figure out where to get the context from, for scheduler.SendCommand.
 */