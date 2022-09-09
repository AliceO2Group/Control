/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2022 CERN and copyright holders of ALICE O².
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

package eventbus

import (
	"fmt"

	"github.com/spf13/viper"
	evbus "github.com/teo/EventBus"
)

type EventServer struct {
	Srv *evbus.Server
}

func NewEventServer() (*EventServer, error) {
	s := &EventServer{
		Srv: evbus.NewServer(fmt.Sprintf(":%d", viper.GetInt("eventsPort")), EVENTBUS_SRVPATH, evbus.New()),
	}
	err := s.Srv.Start()
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (s *EventServer) Publish(object any) {
	s.Srv.EventBus().Publish("general", object)
}

func (s *EventServer) Subscribe(fn any) error {
	return s.Srv.EventBus().Subscribe("general", fn)
}

func (s *EventServer) SubscribeAsync(fn any, transactional bool) error {
	return s.Srv.EventBus().SubscribeAsync("general", fn, transactional)
}

func (s *EventServer) Unsubscribe(fn any) error {
	return s.Srv.EventBus().Unsubscribe("general", fn)
}
