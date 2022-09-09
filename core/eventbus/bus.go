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
	"sync"

	"github.com/AliceO2Group/Control/common/logger"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var log = logger.New(logrus.StandardLogger(), "eventbus")

var (
	once     sync.Once
	instance Bus
)

const EVENTBUS_SRVPATH = "/_server_bus_"

type Publisher interface {
	Publish(any)
}

type Subscriber interface {
	Subscribe(fn any) error
	SubscribeAsync(fn any, transactional bool) error
	Unsubscribe(fn any) error
}

type Bus interface {
	Publisher
	Subscriber
}

func Instance() Bus {
	once.Do(func() {
		isServer := viper.GetInt("eventsPort") > 0
		isClient := len(viper.GetString("coreEventsEndpoint")) > 0
		var err error
		if isServer && !isClient {
			instance, err = NewEventServer()
			if err != nil {
				log.WithField("port", viper.GetInt("eventsPort")).
					WithError(err).
					Fatal("cannot start event bus server")
			}
		} else {
			instance, err = NewEventClient()
			if err != nil {
				log.WithField("eventsEndpoint", viper.GetString("coreEventsEndpoint")).
					WithError(err).
					Fatal("cannot start event bus client")
			}
		}
	})
	return instance
}
