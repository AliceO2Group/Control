/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2017-2018 CERN and copyright holders of ALICE O².
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
	"context"
	"github.com/AliceO2Group/Control/common/product"
	"github.com/AliceO2Group/Control/core/the"
	"github.com/spf13/viper"
	"time"

	"fmt"
	"github.com/AliceO2Group/Control/common/logger"
	"github.com/looplab/fsm"
	"github.com/mesos/mesos-go/api/v1/lib/extras/scheduler/callrules"
	"github.com/mesos/mesos-go/api/v1/lib/extras/store"
	"github.com/mesos/mesos-go/api/v1/lib/scheduler"
	"github.com/sirupsen/logrus"
	"net"
)

var log = logger.New(logrus.StandardLogger(),"core")

// Run is the entry point for this scheduler.
// TODO: refactor Config to reflect our specific requirements
func Run() error {
	if viper.GetBool("verbose") || viper.GetBool("veryVerbose") {
		logrus.SetLevel(logrus.DebugLevel)
	}

	if viper.GetBool("veryVerbose") {
		log.WithField("configuration", viper.AllSettings()).Debug("core starting up")
	}
	log.Infof("%s (%s v%s build %s) starting up", product.PRETTY_FULLNAME, product.PRETTY_SHORTNAME, product.VERSION, product.BUILD)

	// We create a context and use its cancel func as a shutdown func to release
	// all resources. The shutdown func is stored in the app.internalState.
	ctx, cancel := context.WithCancel(context.Background())

	// This only runs once to create a container for all data which comprises the
	// scheduler's state.
	// It also keeps count of the tasks launched/finished
	state, err := newInternalState(cancel)
	if err != nil {
		return err
	}

	// TODO(jdef) how to track/handle timeout errors that occur for SUBSCRIBE calls? we should
	// probably tolerate X number of subsequent subscribe failures before bailing. we'll need
	// to track the lastCallAttempted along with subsequentSubscribeTimeouts.

	// store.Singleton is a thread-safe abstraction to load and store and string,
	// provided by mesos-go.
	// We also make sure that a log message is printed with the FrameworkID.
	fidStore := store.DecorateSingleton(
		store.NewInMemorySingleton(),
		store.DoSet().AndThen(func(_ store.Setter, v string, _ error) error {
			// Store Mesos Framework ID to configuration.
			err = the.ConfSvc().NewMesosFID(v)
			if err != nil {
				log.WithField("error", err).Error("cannot write to configuration")
			}
			log.WithField("frameworkId", v).Debug("frameworkId")
			return nil
		}))
	
	// Set Framework ID from the configuration 
	if fidValue, err := the.ConfSvc().GetMesosFID(); err == nil {
		store.SetOrPanic(fidStore)(fidValue)
	}

	// callrules.New returns a Rules and accept a bunch of Rule values as arguments.
	// WithFrameworkID returns a Rule which injects a frameworkID to outgoing calls.
	// logCalls returns a rule which prints to the log all calls of type SUBSCRIBE.
	// callMetrics logs metrics for every outgoing call.
	state.cli = callrules.New(
		callrules.WithFrameworkID(store.GetIgnoreErrors(fidStore)),
		logCalls(map[scheduler.Call_Type]string{scheduler.Call_SUBSCRIBE: "subscribe connecting"}),
		callMetrics(state.metricsAPI, time.Now, viper.GetBool("summaryMetrics")),
	).Caller(state.cli)

	state.sm = fsm.NewFSM(
		"INITIAL",
		fsm.Events{
			{Name: "CONNECT",			Src: []string{"INITIAL"},   Dst: "CONNECTED"},
			{Name: "NEW_ENVIRONMENT",	Src: []string{"CONNECTED"},	Dst: "CONNECTED"},
			{Name: "GO_ERROR", 			Src: []string{"CONNECTED"}, Dst: "ERROR"},
			{Name: "RESET",    			Src: []string{"ERROR"},     Dst: "INITIAL"},
			{Name: "EXIT",     			Src: []string{"CONNECTED"}, Dst: "FINAL"},
		},
		fsm.Callbacks{
			"before_event": func(e *fsm.Event) {
				log.WithFields(logrus.Fields{
					"event": e.Event,
					"src": e.Src,
					"dst": e.Dst,
				}).Debug("state.sm starting transition")
			},
			"enter_state": func(e *fsm.Event) {
				log.WithFields(logrus.Fields{
					"event": e.Event,
					"src": e.Src,
					"dst": e.Dst,
				}).Debug("state.sm entering state")
			},
			"leave_CONNECTED": func(e *fsm.Event) {
				log.Debug("leave_CONNECTED")

			},
			"before_NEW_ENVIRONMENT": func(e *fsm.Event) {
				log.Debug("before_NEW_ENVIRONMENT")
				e.Async() //transition frozen until the corresponding fsm.Transition call
			},
			"enter_CONNECTED": func(e *fsm.Event) {
				log.Debug("enter_CONNECTED")
				log.Info("scheduler connected")
			},
			"after_NEW_ENVIRONMENT": func(e *fsm.Event) {
				log.Debug("after_NEW_ENVIRONMENT")
			},
		},
	)

	// We now build the Control server
	s := NewServer(state, fidStore)

	// Async start of the scheduler controller. This runs in parallel with the grpc server.
	go func() {
		err = runSchedulerController(ctx, state, fidStore)
		state.RLock()
		defer state.RUnlock()
		if state.err != nil {
			err = state.err
			log.WithField("error", err.Error()).Debug("scheduler quit with error, main state machine GO_ERROR")
			state.sm.Event("GO_ERROR", err)	 //TODO: use error information in GO_ERROR
		} else {
			log.Debug("scheduler quit, no errors")
			state.sm.Event("EXIT")
		}
	}()

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", viper.GetInt("controlPort")))
	if err != nil {
		log.WithField("error", err).
			WithField("port", viper.GetInt("controlPort")).
			Fatal("net.Listener failed to listen")
	}
	if err := s.Serve(lis); err != nil {
		log.WithField("error", err).Fatal("GRPC server failed to serve")
	}

	return err
}

// end Run
