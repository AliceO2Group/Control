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
	"strings"

	"github.com/spf13/viper"
	evbus "github.com/teo/EventBus"
)

type EventClient struct {
	Cli *evbus.Client
}

func NewEventClient() (*EventClient, error) {
	eventsEndpoint := viper.GetString("coreEventsEndpoint")
	eventsEndpoint = strings.TrimPrefix(eventsEndpoint, "//")
	s := &EventClient{
		Cli: evbus.NewClient(eventsEndpoint, EVENTBUS_SRVPATH, evbus.New()),
	}
	err := s.Cli.Start()
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (s *EventClient) Publish(object any) {
	s.Cli.EventBus().Publish("general", object)
}

func (s *EventClient) Subscribe(fn any) error {
	return s.Cli.EventBus().Subscribe("general", fn)
}

func (s *EventClient) SubscribeAsync(fn any, transactional bool) error {
	return s.Cli.EventBus().SubscribeAsync("general", fn, transactional)
}

func (s *EventClient) Unsubscribe(fn any) error {
	return s.Cli.EventBus().Unsubscribe("general", fn)
}
