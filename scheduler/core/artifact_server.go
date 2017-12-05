/*
 * === This file is part of octl <https://github.com/teo/octl> ===
 *
 * Copyright 2017 CERN and copyright holders of ALICE O².
 * Author: Teo Mrnjavac <teo.mrnjavac@cern.ch>
 *
 * Portions from examples in <https://github.com/mesos/mesos-go>:
 *     Copyright 2013-2015, Mesosphere, Inc.
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
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
)

func serveFile(filename string) (handler http.Handler, err error) {
	_, err = os.Stat(filename)
	if err != nil {
		err = fmt.Errorf("failed to locate artifact: %+v", err)
	} else {
		handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, filename)
		})
	}
	return
}

// returns (downloadURI, basename(path))
func serveExecutorArtifact(server server, path string, mux *http.ServeMux) (string, string, error) {
	// Create base path (http://foobar:5000/<base>)
	pathSplit := strings.Split(path, "/")
	var base string
	if len(pathSplit) > 0 {
		base = pathSplit[len(pathSplit)-1]
	} else {
		base = path
	}
	pattern := "/" + base
	h, err := serveFile(path)
	if err != nil {
		return "", "", err
	}

	mux.Handle(pattern, h)

	hostURI := fmt.Sprintf("http://%s:%d/%s", server.address, server.port, base)
	log.Println("Hosting artifact '" + path + "' at '" + hostURI + "'")

	return hostURI, base, nil
}

func newListener(server server) (*net.TCPListener, int, error) {
	addr, err := net.ResolveTCPAddr("tcp", net.JoinHostPort(server.address, strconv.Itoa(server.port)))
	if err != nil {
		return nil, 0, err
	}
	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return nil, 0, err
	}
	bindAddress := listener.Addr().String()
	_, port, err := net.SplitHostPort(bindAddress)
	if err != nil {
		return nil, 0, err
	}
	iport, err := strconv.Atoi(port)
	if err != nil {
		return nil, 0, err
	}
	return listener, iport, nil
}
