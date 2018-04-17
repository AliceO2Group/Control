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

package metrics

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	Subsystem = "octl_scheduler"
)

// TODO(jdef) time in between offers

var (
	CallErrorCount = prometheus.NewCounterVec(prometheus.CounterOpts{
		Subsystem: Subsystem,
		Name:      "call_error_count",
		Help:      "The number of errors for outgoing calls.",
	}, []string{"type"})
	CallCount = prometheus.NewCounterVec(prometheus.CounterOpts{
		Subsystem: Subsystem,
		Name:      "call_count",
		Help:      "The number of outgoing calls.",
	}, []string{"type"})
	CallLatency = prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Subsystem: Subsystem,
		Name:      "call_latency",
		Help:      "Time to execute various calls, by type.",
	}, []string{"type"})
	EventErrorCount = prometheus.NewCounterVec(prometheus.CounterOpts{
		Subsystem: Subsystem,
		Name:      "event_error_count",
		Help:      "The number of event processing errors.",
	}, []string{"type"})
	EventReceivedCount = prometheus.NewCounterVec(prometheus.CounterOpts{
		Subsystem: Subsystem,
		Name:      "event_received_count",
		Help:      "The number of events received.",
	}, []string{"type"})
	EventReceivedLatency = prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Subsystem: Subsystem,
		Name:      "event_received_latency",
		Help:      "Time to process various events, by type.",
	}, []string{"type"})
	OffersReceived = prometheus.NewCounter(prometheus.CounterOpts{
		Subsystem: Subsystem,
		Name:      "offers_received",
		Help:      "The number of individual offers received.",
	})
	OffersDeclined = prometheus.NewCounter(prometheus.CounterOpts{
		Subsystem: Subsystem,
		Name:      "offers_declined",
		Help:      "The number of offers declined.",
	})
	TasksFinished = prometheus.NewCounter(prometheus.CounterOpts{
		Subsystem: Subsystem,
		Name:      "tasks_finished",
		Help:      "The number of tasks finished.",
	})
	TasksLaunched = prometheus.NewCounter(prometheus.CounterOpts{
		Subsystem: Subsystem,
		Name:      "tasks_launched",
		Help:      "The number of tasks launched.",
	})
	JobStartCount = prometheus.NewCounterVec(prometheus.CounterOpts{
		Subsystem: Subsystem,
		Name:      "job_start_count",
		Help:      "The number of internal background jobs started.",
	}, []string{"job"})
	OfferedResources = prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Subsystem: Subsystem,
		Name:      "offered_resources",
		Help:      "Scalar resources offered by type.",
	}, []string{"type"})
	TasksLaunchedPerOfferCycle = prometheus.NewSummary(prometheus.SummaryOpts{
		Subsystem: Subsystem,
		Name:      "tasks_launched_per_cycle",
		Help:      "Number of tasks launched per-offers cycle (event).",
	})
	ArtifactDownloads = prometheus.NewCounter(prometheus.CounterOpts{
		Subsystem: Subsystem,
		Name:      "artifact_downloads",
		Help:      "The number of artifacts served by the built-in http server.",
	})
)

var registerMetrics sync.Once

func Register() {
	registerMetrics.Do(func() {
		prometheus.MustRegister(CallErrorCount)
		prometheus.MustRegister(CallCount)
		prometheus.MustRegister(CallLatency)
		prometheus.MustRegister(EventErrorCount)
		prometheus.MustRegister(EventReceivedCount)
		prometheus.MustRegister(EventReceivedLatency)
		prometheus.MustRegister(OffersReceived)
		prometheus.MustRegister(OffersDeclined)
		prometheus.MustRegister(JobStartCount)
		prometheus.MustRegister(TasksFinished)
		prometheus.MustRegister(TasksLaunched)
		prometheus.MustRegister(OfferedResources)
		prometheus.MustRegister(TasksLaunchedPerOfferCycle)
		prometheus.MustRegister(ArtifactDownloads)
	})
}
