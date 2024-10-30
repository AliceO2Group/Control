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

package channel

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/rs/xid"
)

const ipcPathFormat = "/tmp/o2ipc-%s"

type BindMap map[string]Endpoint

type Endpoint interface {
	GetAddressFormat() AddressFormat
	GetAddress() string
	GetTransport() TransportType
	ToTargetEndpoint(taskHostname string) Endpoint
	ToBoundEndpoint() Endpoint
}

func EndpointEquals(e Endpoint, f Endpoint) bool {
	if reflect.TypeOf(e) != reflect.TypeOf(f) {
		return false
	}
	switch v := e.(type) {
	case TcpEndpoint:
		w, ok := f.(TcpEndpoint)
		if !ok {
			return false
		}
		return v == w
	case IpcEndpoint:
		w, ok := f.(IpcEndpoint)
		if !ok {
			return false
		}
		return v == w
	}
	return false
}

func NewTcpEndpoint(host string, port uint64, transport TransportType) Endpoint {
	return TcpEndpoint{
		Host:      host,
		Port:      port,
		Transport: transport,
	}
}

func NewBoundTcpEndpoint(port uint64, transport TransportType) Endpoint {
	return TcpEndpoint{
		Host:      "*",
		Port:      port,
		Transport: transport,
	}
}

func NewIpcEndpoint(path string, transport TransportType) Endpoint {
	return IpcEndpoint{
		Path:      strings.TrimPrefix(path, "ipc://"),
		Transport: transport,
	}
}

func NewBoundIpcEndpoint(transport TransportType) Endpoint {
	return IpcEndpoint{
		Path:      fmt.Sprintf(ipcPathFormat, xid.New().String()),
		Transport: transport,
	}
}

type TcpEndpoint struct {
	Host      string
	Port      uint64
	Transport TransportType
}

func (TcpEndpoint) GetAddressFormat() AddressFormat {
	return TCP
}

func (t TcpEndpoint) GetAddress() string {
	if len(t.Host) == 0 || t.Host == "*" {
		return fmt.Sprintf("tcp://*:%d", t.Port)
	}
	return fmt.Sprintf("tcp://%s:%d", t.Host, t.Port)
}

func (t TcpEndpoint) GetTransport() TransportType {
	return t.Transport
}

func (t TcpEndpoint) ToTargetEndpoint(taskHostname string) Endpoint {
	return TcpEndpoint{
		Host:      taskHostname,
		Port:      t.Port,
		Transport: t.Transport,
	}
}

func (t TcpEndpoint) ToBoundEndpoint() Endpoint {
	return TcpEndpoint{
		Host:      "*",
		Port:      t.Port,
		Transport: t.Transport,
	}
}

type IpcEndpoint struct {
	Path      string
	Transport TransportType
}

func (IpcEndpoint) GetAddressFormat() AddressFormat {
	return IPC
}

func (t IpcEndpoint) GetAddress() string {
	return fmt.Sprintf("ipc://%s", t.Path)
}

func (t IpcEndpoint) GetTransport() TransportType {
	return t.Transport
}

func (t IpcEndpoint) ToTargetEndpoint(_ string) Endpoint {
	return IpcEndpoint{
		Path:      t.Path,
		Transport: t.Transport,
	}
}

func (t IpcEndpoint) ToBoundEndpoint() Endpoint {
	return IpcEndpoint{
		Path:      t.Path,
		Transport: t.Transport,
	}
}
