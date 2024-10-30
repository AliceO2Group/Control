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

package core

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/AliceO2Group/Control/core/environment"
)

func signals(state *globalState) {

	// Create channel to receive unix signals
	signal_chan := make(chan os.Signal, 1)

	//Register channel to receive SIGINT and SIGTERM signals
	signal.Notify(signal_chan,
		syscall.SIGINT,
		syscall.SIGTERM)

	// Goroutine executes a blocking receive for signals
	go func() {
		s := <-signal_chan
		manageKillSignals(state)

		// Mesos calls are async.Sleep for 2s to mark tasks as completed.
		time.Sleep(2 * time.Second)
		switch s {
		case syscall.SIGINT:
			os.Exit(130) // 128+2
		case syscall.SIGTERM:
			os.Exit(143) // 128+15
		}
	}()
}

func manageKillSignals(state *globalState) {
	// Get all enviroment ids
	uids := state.environments.Ids()

	for _, uid := range uids {
		// Get enviroment
		env, err := state.environments.Environment(uid)
		if err != nil {
			log.WithPrefix("termination").WithError(err).Error(fmt.Sprintf("cannot find enviroment %s", uid.String()))
		}
		// This might transition to CONFIGURED if needed, of do nothing if we're already there
		if env.CurrentState() == "RUNNING" {

			tasks := env.Workflow().GetTasks()
			// we mark this specific task as ok to STOP
			for _, t := range tasks {
				t.SetSafeToStop(true)
			}
			// but then we ask the env whether *all* of them are
			if env.IsSafeToStop() {
				err = env.TryTransition(environment.NewStopActivityTransition(state.taskman))
				if err != nil {
					log.WithPrefix("termination").WithError(err).Error(fmt.Sprintf("cannot transition enviroment %s from RUNNING to CONFIGURED", uid.String()))
				}
			}
		}

		// This might transition to STANDBY if needed, or do nothing if we're already there
		if env.CurrentState() == "CONFIGURED" {
			err = env.TryTransition(environment.NewResetTransition(state.taskman))
			if err != nil {
				log.WithPrefix("termination").WithError(err).Error(fmt.Sprintf("cannot transition enviroment %s from CONFIGURED to STANDBY", uid.String()))
			}
		}

		// Teardown Enviroment
		err = state.environments.TeardownEnvironment(uid, false)
		if err != nil {
			log.WithPrefix("termination").WithError(err).Error(fmt.Sprintf("cannot teardown enviroment %s", uid.String()))
		}
	}

	// Perform cleanup
	_, _, err := state.taskman.Cleanup()
	if err != nil {
		log.WithPrefix("termination").WithError(err).Error("can't perform cleanup")
	}

	// Request Mesos to kill all tasks, regardless enviroment status and if a task is locked or not.
	tasks := state.taskman.GetTasks()
	state.taskman.EmergencyKillTasks(tasks)
}
