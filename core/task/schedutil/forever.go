/*
 * === This file is part of ALICE O² ===
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

package schedutil

import (
	"time"

	xmetrics "github.com/mesos/mesos-go/api/v1/lib/extras/metrics"
	"github.com/sirupsen/logrus"
)

func Forever(name string, jobRestartDelay time.Duration, counter xmetrics.Counter, f func() error) {
	for {
		counter(name)
		err := f()
		if err != nil {
			log.WithFields(logrus.Fields{
				"name":  name,
				"error": err.Error(),
			}).Error("job exited with error")
		} else {
			log.WithField("name", name).Info("job exited")
		}
		time.Sleep(jobRestartDelay)
	}
}
