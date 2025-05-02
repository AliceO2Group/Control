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
	"testing"
	"time"

	"github.com/AliceO2Group/Control/common/monitoring"
)

func measureFunc(metric *monitoring.Metric) {
	defer TimerMS(metric)()
	defer TimerNS(metric)()
	time.Sleep(100 * time.Millisecond)
}

func TestSimpleStartStop(t *testing.T) {
	metric := NewMetric("test")
	measureFunc(&metric)
	fields := metric.GetFields()
	if fields["execution_time_ms"].(int64) < 100 {
		t.Error("wrong milliseconds")
	}
	if fields["execution_time_ns"].(int64) < 100000000 {
		t.Error("wrong nanoseconds")
	}
}
