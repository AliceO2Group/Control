/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2021 CERN and copyright holders of ALICE O².
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

package chili

import (
	"fmt"
	"net"

	"github.com/AliceO2Group/Control/chili/api"
	"github.com/AliceO2Group/Control/common/logger"
	"github.com/AliceO2Group/Control/common/logger/infologger"
	"github.com/AliceO2Group/Control/common/product"
	"github.com/AliceO2Group/Control/common/utils"
	"github.com/AliceO2Group/Control/core/the"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

//go:generate protoc -I=./ -I=../common/ --go_out=. --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative --go-grpc_out=require_unimplemented_servers=false:. protos/events.proto
//go:generate protoc -I=./ -I=../common/ --go_out=. --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative --go-grpc_out=require_unimplemented_servers=false:. protos/chili.proto

var log = logger.New(logrus.StandardLogger(), "chili")

const fileLimitWant = 65536
const fileLimitMin = 8192

func Run() (err error) {
	if viper.GetBool("verbose") {
		logrus.SetLevel(logrus.DebugLevel)
	}
	log.WithField("level", infologger.IL_Support).Infof("AliECS Control High-Level Interface Service (chili v%s build %s) starting up", product.VERSION, product.BUILD)

	// 1st service: EventBus client
	// This is actually a Go RPC server, and during `Instance` no outgoing client connections are made.
	// Actual client operation only kicks in with a `Subscribe` call or similar, and that's when a request is made to
	// coreEventsEndpoint for the core to start sending events.
	evb := the.EventBus()

	// 2nd service: gRPC server
	// Serves a curated event stream upon request, as forwarded by the EventBus
	s := api.NewServer()
	signals(s, evb) // handle UNIX signals

	// Raise soft file limit
	err = utils.SetLimits(fileLimitWant, fileLimitMin)
	if err != nil {
		return err
	}

	var lis net.Listener
	lis, err = net.Listen("tcp", fmt.Sprintf(":%d", viper.GetInt("controlPort")))
	if err != nil {
		log.WithField("error", err).
			WithField("port", viper.GetInt("controlPort")).
			Fatal("net.Listener failed to listen")
		return
	}

	log.WithField("port", viper.GetInt("controlPort")).
		WithField("coreEndpoint", viper.GetString("coreEndpoint")).
		WithField("coreEventsEndpoint", viper.GetString("coreEventsEndpoint")).
		WithField("configServiceUri", viper.GetString("configServiceUri")).
		WithField("level", infologger.IL_Support).
		Info("service started")
	if err = s.Serve(lis); err != nil {
		log.WithField("error", err).Fatal("gRPC server failed to serve")
	}
	return
}
