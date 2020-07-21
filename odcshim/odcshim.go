/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2020 CERN and copyright holders of ALICE O².
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

package odcshim

import (
	"fmt"
	"net"
	"os"

	"github.com/AliceO2Group/Control/common/logger"
	"github.com/AliceO2Group/Control/odcshim/occserver"
	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

//go:generate protoc -I ../occ     --gofast_out=plugins=grpc:. protos/occ.proto
//go:generate protoc -I ../odcshim --gofast_out=plugins=grpc:. odcprotos/odc.proto

var log = logger.New(logrus.StandardLogger(), "o2-aliecs-odc-shim")


func setDefaults() error {
	viper.SetDefault("odcPort", 50051)
	viper.SetDefault("odcHost", "127.0.0.1")
	viper.SetDefault("verbose", false)
	return nil
}

func setFlags() error {
	pflag.String("control-port", viper.GetString("control-port"), "OCC control port")
	pflag.Int("odcPort", viper.GetInt("odcPort"), "Remote ODC server port")
	pflag.String("odcHost", viper.GetString("odcHost"), "Remote ODC server hostname")
	pflag.BoolP("verbose", "v", viper.GetBool("verbose"), "Verbose logging")

	pflag.Parse()
	return viper.BindPFlags(pflag.CommandLine)
}

func Run() (err error) {
	if err = setDefaults(); err != nil {
		return
	}
	if err = setFlags(); err != nil {
		return
	}

	host := viper.GetString("odcHost")
	port := viper.GetInt("odcPort")
	occPort := viper.GetString("control-port")
	if len(occPort) == 0 {
		occPort = os.Getenv("OCC_CONTROL_PORT")
	}
	topology := os.Getenv("ODC_TOPOLOGY")

	server := occserver.NewServer(host, port, topology)
	if server == nil {
		return fmt.Errorf("cannot start OCC-ODC bridge to %s:%d", host, port)
	}

	var lis net.Listener
	lis, err = net.Listen("tcp", fmt.Sprintf(":%s", occPort))
	if err != nil {
		log.WithField("error", err).
			WithField("port", occPort).
			Fatal("net.Listener failed to listen")
		return
	}
	if err = server.Serve(lis); err != nil {
		log.WithField("error", err).Fatal("GRPC server failed to serve")
	}

	return
}
