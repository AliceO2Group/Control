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

package task

import (
	"net"
	"net/http"
	"strconv"

	schedmetrics "github.com/AliceO2Group/Control/core/metrics"
	"github.com/AliceO2Group/Control/core/task/schedutil"
	xmetrics "github.com/mesos/mesos-go/api/v1/lib/extras/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/spf13/viper"
)

func initMetrics() *metricsAPI {
	schedmetrics.Register()
	metricsAddress := net.JoinHostPort(viper.GetString("metrics.address"), strconv.Itoa(viper.GetInt("metrics.port")))
	http.Handle(viper.GetString("metrics.path"), promhttp.Handler())
	api := newMetricsAPI()
	go schedutil.Forever("api-server", viper.GetDuration("mesosJobRestartDelay"), api.jobStartCount, func() error { return http.ListenAndServe(metricsAddress, nil) })
	return api
}

func newMetricAdder(m prometheus.Counter) xmetrics.Adder {
	return func(x float64, _ ...string) { m.Add(x) }
}

func newMetricCounter(m prometheus.Counter) xmetrics.Counter {
	return func(_ ...string) { m.Inc() }
}

func newMetricCounters(m *prometheus.CounterVec) xmetrics.Counter {
	return func(s ...string) { m.WithLabelValues(s...).Inc() }
}

func newMetricWatcher(m prometheus.Summary) xmetrics.Watcher {
	return func(x float64, _ ...string) { m.Observe(x) }
}

func newMetricWatchers(m *prometheus.SummaryVec) xmetrics.Watcher {
	return func(x float64, s ...string) { m.WithLabelValues(s...).Observe(x) }
}

type metricsAPI struct {
	eventErrorCount       xmetrics.Counter
	eventReceivedCount    xmetrics.Counter
	eventReceivedLatency  xmetrics.Watcher
	callCount             xmetrics.Counter
	callErrorCount        xmetrics.Counter
	callLatency           xmetrics.Watcher
	offersReceived        xmetrics.Adder
	offersDeclined        xmetrics.Adder
	tasksLaunched         xmetrics.Adder
	tasksFinished         xmetrics.Counter
	launchesPerOfferCycle xmetrics.Watcher
	offeredResources      xmetrics.Watcher
	jobStartCount         xmetrics.Counter
	artifactDownloads     xmetrics.Counter
}

func newMetricsAPI() *metricsAPI {
	return &metricsAPI{
		callCount:             newMetricCounters(schedmetrics.CallCount),
		callErrorCount:        newMetricCounters(schedmetrics.CallErrorCount),
		callLatency:           newMetricWatchers(schedmetrics.CallLatency),
		eventErrorCount:       newMetricCounters(schedmetrics.EventErrorCount),
		eventReceivedCount:    newMetricCounters(schedmetrics.EventReceivedCount),
		eventReceivedLatency:  newMetricWatchers(schedmetrics.EventReceivedLatency),
		offersReceived:        newMetricAdder(schedmetrics.OffersReceived),
		offersDeclined:        newMetricAdder(schedmetrics.OffersDeclined),
		tasksLaunched:         newMetricAdder(schedmetrics.TasksLaunched),
		tasksFinished:         newMetricCounter(schedmetrics.TasksFinished),
		launchesPerOfferCycle: newMetricWatcher(schedmetrics.TasksLaunchedPerOfferCycle),
		offeredResources:      newMetricWatchers(schedmetrics.OfferedResources),
		jobStartCount:         newMetricCounters(schedmetrics.JobStartCount),
		artifactDownloads:     newMetricCounter(schedmetrics.ArtifactDownloads),
	}
}
