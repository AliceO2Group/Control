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

package apricot

import (
	"fmt"
	"net"

	"github.com/AliceO2Group/Control/apricot/remote"
	"github.com/AliceO2Group/Control/common/logger"
	"github.com/AliceO2Group/Control/common/product"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

//go:generate protoc --go_out=. --go-grpc_out=require_unimplemented_servers=false:. protos/apricot.proto

var log = logger.New(logrus.StandardLogger(),"apricot")

func Run() (err error) {
	if viper.GetBool("verbose") {
		logrus.SetLevel(logrus.DebugLevel)
	}
	log.Infof("AliECS Configuration Service (apricot v%s build %s) starting up", product.VERSION, product.BUILD)

	s := remote.NewServer(Instance())
	signals(s) // handle UNIX signals
	var lis net.Listener
	lis, err = net.Listen("tcp", fmt.Sprintf(":%d", viper.GetInt("listenPort")))
	if err != nil {
		log.WithField("error", err).
			WithField("port", viper.GetInt("listenPort")).
			Fatal("net.Listener failed to listen")
		return
	}

	log.WithField("port", viper.GetInt("listenPort")).
		Info("service started")
	if err = s.Serve(lis); err != nil {
		log.WithField("error", err).Fatal("gRPC server failed to serve")
	}
	return
}
