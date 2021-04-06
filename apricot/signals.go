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
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"
    "github.com/AliceO2Group/Control/apricot/local"
)

func signals(srv *grpc.Server, httpsvr *http.Server) {

	// Create channel to receive unix signals
	signal_chan := make(chan os.Signal, 1)

	//Register channel to receive SIGINT and SIGTERM signals
	signal.Notify(signal_chan,
		syscall.SIGINT,
		syscall.SIGTERM)

	// Goroutine executes a blocking receive for signals
	go func() {
		s := <-signal_chan
		log.WithField("signal", s.String()).
			Debug("received signal")

		srv.Stop()
		if err := httpsvr.Shutdown(context.Background()); err != nil {
			log.Printf("Error while shutting down http server.")
		}

		// Mesos calls are async.Sleep for 2s to mark tasks as completed.
		time.Sleep(2 * time.Second)
		log.WithField("signal", s.String()).
			Info("service stopped")

		switch s {
		case syscall.SIGINT:
			os.Exit(130) // 128+2
		case syscall.SIGTERM:
			os.Exit(143) // 128+15
		}
	}()
}
