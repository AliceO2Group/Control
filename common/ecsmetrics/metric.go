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
	"time"

	"github.com/AliceO2Group/Control/common/monitoring"
)

func NewMetric(name string) monitoring.Metric {
	metric := monitoring.NewMetric(name, time.Now())
	metric.AddTag("subsystem", "ECS")
	return metric
}

// Timer* functions are meant to be used with defer statement to measure runtime of given function:
// defer TimerNS(&metric)()
func TimerMS(metric *monitoring.Metric) func() {
	start := time.Now()
	return func() {
		metric.SetFieldInt64("execution_time_ms", time.Since(start).Milliseconds())
	}
}

func TimerNS(metric *monitoring.Metric) func() {
	start := time.Now()
	return func() {
		metric.SetFieldInt64("execution_time_ns", time.Since(start).Nanoseconds())
	}
}
