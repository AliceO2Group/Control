/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2018-2022 CERN and copyright holders of ALICE O².
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

package executor

import (
	"context"

	"github.com/mesos/mesos-go/api/v1/lib/encoding"
	"github.com/mesos/mesos-go/api/v1/lib/executor"
	"github.com/mesos/mesos-go/api/v1/lib/executor/events"
	"github.com/sirupsen/logrus"
)

// nextEventNotify blocks waiting for an incoming event. When an event arrives, it is sent back via
// the eventCh channel, and the function quits.
func nextEventNotify(decoder encoding.Decoder, eventCh chan<- executor.Event, errorCh chan<- error) {
	log.Trace("EVENT LOOP nextEventNotify start")
	var e executor.Event
	var err error
	if err = decoder.Decode(&e); err == nil {
		eventCh <- e
	} else {
		errorCh <- err
	}
}

// eventLoop dispatches incoming events from mesos-agent to the events.Handler (built in buildEventhandler).
func eventLoop(state *internalState, decoder encoding.Decoder, h events.Handler) (err error) {
	log.Trace("listening for events from agent")
	ctx := context.TODO() // dummy context

	// The decoder object is the response obtained from the Mesos slave subscription. We use decoder.Decode
	// to acquire the next event from Mesos. Unfortunately this call blocks, so we rather keep it in a separate
	// goroutine and only pipe single events into the main loop as they come.
	errorCh := make(chan error)          // errors come through here
	eventCh := make(chan executor.Event) // events from Mesos come through here

	// Spawn a goroutine to wait for the first event
	go nextEventNotify(decoder, eventCh, errorCh)

	for err == nil && !state.shouldQuit {
		log.Trace("EVENT LOOP begin new main event loop iteration")
		// housekeeping
		sendFailedTasks(state)

		select {
		case e := <-eventCh:
			log.Trace("EVENT LOOP about to handle event")
			err = h.HandleEvent(ctx, &e)
			// if we get an error here we are stoping the eventloop before triggering another nextEventNotify
			// so it won't get stuck on eventCh, errorCh
			if err != nil {
				break
			}

			// Spawn a goroutine to wait for the next event
			go nextEventNotify(decoder, eventCh, errorCh)
		case err = <-errorCh:
			log.Trace("EVENT LOOP got error")
			// Any error coming through here should immediately destroy the event loop and return
			// control to the Mesos subscription handling.
		case status := <-state.statusCh:
			performStatusUpdate(state, status)
		case message := <-state.messageCh:
			sendOutgoingMessage(state, message)
		}
	}
	return err
}

// sendFailedTasks runs on every iteration of eventLoop to send an UPDATE on any failed tasks to the agent.
func sendFailedTasks(state *internalState) {
	for taskID, status := range state.failedTasks {
		updateErr := update(state, status)
		if updateErr != nil {
			log.WithFields(logrus.Fields{
				"taskId": taskID.Value,
				"error":  updateErr,
			}).
				Error("failed to send status update for task")
		} else {
			// If we have successfully notified Mesos, we clear our list of failed tasks.
			delete(state.failedTasks, taskID)
			// If there aren't any failed and active tasks, we request to shutdown the executor.
			if len(state.failedTasks) == 0 && len(state.activeTasks) == 0 {
				// Originally state.shouldQuit = true but we want to keep the executor running
				log.WithField("executorId", state.executor.ExecutorID).Info("task failure notified, no tasks present on executor")
			}
		}
	}
}
