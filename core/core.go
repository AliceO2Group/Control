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
	"fmt"
	"net"

	"github.com/AliceO2Group/Control/common/logger"
	"github.com/AliceO2Group/Control/common/product"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var log = logger.New(logrus.StandardLogger(),"core")

// Run is the entry point for this scheduler.
// TODO: refactor Config to reflect our specific requirements
func Run() error {
	if viper.GetBool("verbose") {
		logrus.SetLevel(logrus.DebugLevel)
	}
	if viper.GetBool("veryVerbose") {
		logrus.SetLevel(logrus.TraceLevel)
	}

	if viper.GetBool("veryVerbose") {
		log.WithField("configuration", viper.AllSettings()).Debug("core starting up")
	}
	log.Infof("%s (%s v%s build %s) starting up", product.PRETTY_FULLNAME, product.PRETTY_SHORTNAME, product.VERSION, product.BUILD)

	// We create a context and use its cancel func as a shutdown func to release
	// all resources. The shutdown func is stored in the scheduler.internalState.
	ctx, cancel := context.WithCancel(context.Background())

	// This only runs once to create a container for all data which comprises the
	// scheduler's state.
	// It also keeps count of the tasks launched/finished
	state, err := newGlobalState(cancel)
	if err != nil {
		return err
	}

	// Set up channel to receive Unix Signals.
	signals(state)


	// We now build the Control server
	s := NewServer(state)

	state.taskman.Start(ctx)

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
