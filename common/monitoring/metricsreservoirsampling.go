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

package monitoring

import (
	"hash/maphash"
	"time"
)

type bucketsReservoirSampleType map[key]*metricReservoirSample

var reservoirSize uint64 = 1000

type metricReservoirSample struct {
	metric    Metric
	reservoir reservoirSampling
}

type MetricsReservoirSampling struct {
	hash           maphash.Hash
	metricsBuckets bucketsReservoirSampleType
}

func NewMetricsReservoirSampling() *MetricsReservoirSampling {
	metrics := &MetricsReservoirSampling{}
	metrics.metricsBuckets = make(bucketsReservoirSampleType)
	metrics.hash.SetSeed(maphash.MakeSeed())
	return metrics
}

func metricFieldToFloat64(field any) float64 {
	var asserted float64
	switch v := field.(type) {
	case int64:
		asserted = float64(v)
	case uint64:
		asserted = float64(v)
	case float64:
		asserted = v
	}
	return asserted
}

func (this *MetricsReservoirSampling) AddMetric(metric *Metric) {
	for fieldName, field := range metric.fields {
		metricNameTagsToHash(&this.hash, metric)
		_, _ = this.hash.WriteString(fieldName)
		k := key{nameTagsHash: hashValueAndReset(&this.hash), timestamp: time.Unix(metric.timestamp.Unix(), 0)}
		if storedMetric, ok := this.metricsBuckets[k]; !ok {
			newReservoir := newReservoirSampling(fieldName, reservoirSize)
			newReservoir.AddPoint(metricFieldToFloat64(field))
			this.metricsBuckets[k] = &metricReservoirSample{metric: *metric, reservoir: newReservoir}
		} else {
			storedMetric.reservoir.AddPoint(metricFieldToFloat64(field))
		}
	}
}

func (this *MetricsReservoirSampling) Clear() {
	this.hash.Reset()
	clear(this.metricsBuckets)
}

func (this *MetricsReservoirSampling) GetMetrics() []Metric {
	var result []Metric
	for key, reservoirMetric := range this.metricsBuckets {
		m := Metric{name: reservoirMetric.metric.name, tags: reservoirMetric.metric.tags, timestamp: key.timestamp}

		mean, median, minimum, p10, p30, p70, p90, maximum, count, poolSize := reservoirMetric.reservoir.GetStats()

		m.SetFieldFloat64(reservoirMetric.reservoir.name+"_mean", mean)
		m.SetFieldFloat64(reservoirMetric.reservoir.name+"_median", median)
		m.SetFieldFloat64(reservoirMetric.reservoir.name+"_min", minimum)
		m.SetFieldFloat64(reservoirMetric.reservoir.name+"_p10", p10)
		m.SetFieldFloat64(reservoirMetric.reservoir.name+"_p30", p30)
		m.SetFieldFloat64(reservoirMetric.reservoir.name+"_p70", p70)
		m.SetFieldFloat64(reservoirMetric.reservoir.name+"_p90", p90)
		m.SetFieldFloat64(reservoirMetric.reservoir.name+"_max", maximum)
		m.SetFieldUInt64(reservoirMetric.reservoir.name+"_count", count)
		m.SetFieldUInt64(reservoirMetric.reservoir.name+"_poolsize", poolSize)

		result = append(result, m)
	}
	return result
}
