/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2025 CERN and copyright holders of ALICE O².
 * Author: Michal Tichak <michal.tichak@cern.ch>
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

package ecsmetrics

import (
	internalmetrics "runtime/metrics"
	"time"

	"github.com/AliceO2Group/Control/common/logger"
	"github.com/AliceO2Group/Control/common/logger/infologger"
	"github.com/AliceO2Group/Control/common/monitoring"
	"github.com/sirupsen/logrus"
)

var (
	endRequestChannel chan struct{}
	log               = logger.New(logrus.StandardLogger(), "ecsmetrics")
)

func gather() monitoring.Metric {
	samples := []internalmetrics.Sample{
		{Name: "/gc/cycles/total:gc-cycles"},
		{Name: "/memory/classes/other:bytes"},
		{Name: "/memory/classes/total:bytes"},
		{Name: "/sched/goroutines:goroutines"},
		{Name: "/sync/mutex/wait/total:seconds"},
		{Name: "/memory/classes/other:bytes"},
		{Name: "/memory/classes/total:bytes"},
		{Name: "/memory/classes/heap/free:bytes"},
		{Name: "/memory/classes/heap/objects:bytes"},
		{Name: "/memory/classes/heap/released:bytes"},
		{Name: "/memory/classes/heap/stacks:bytes"},
		{Name: "/memory/classes/heap/unused:bytes"},
	}

	internalmetrics.Read(samples)

	metric := NewMetric("golangruntimemetrics")

	for _, sample := range samples {
		switch sample.Value.Kind() {
		case internalmetrics.KindUint64:
			metric.SetFieldUInt64(sample.Name, sample.Value.Uint64())
		case internalmetrics.KindFloat64:
			metric.SetFieldFloat64(sample.Name, sample.Value.Float64())
		case internalmetrics.KindFloat64Histogram:
			log.WithField("level", infologger.IL_Devel).Warningf("Error: Histogram is not supported yet for metric [%s]", sample.Name)
			continue
		default:
			log.WithField("level", infologger.IL_Devel).Warningf("Unsupported kind %v for metric %s\n", sample.Value.Kind(), sample.Name)
			continue
		}
	}
	return metric
}

func StartGolangMetrics(period time.Duration) {
	log.WithField("level", infologger.IL_Devel).Info("Starting golang metrics reporting")
	go func() {
		log.Debug("Starting golang metrics goroutine")
		for {
			select {
			case <-endRequestChannel:
				log.Debug("ending golang metrics")
				endRequestChannel <- struct{}{}
				return
			default:
				log.Debug("sending golang metrics")
				metric := gather()
				monitoring.Send(&metric)
				time.Sleep(period)
			}
		}
	}()
}

func StopGolangMetrics() {
	endRequestChannel <- struct{}{}
	<-endRequestChannel
}
