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
	"math/rand"
	"sort"
)

// Documentation how does the calculation of percentiles work
// https://en.wikipedia.org/wiki/Reservoir_sampling

type reservoirSampling struct {
	samples         []float64
	samplesLimit    uint64
	name            string
	countSinceReset uint64
}

func newReservoirSampling(name string, limit uint64) reservoirSampling {
	return reservoirSampling{
		samples:         make([]float64, 0, limit),
		samplesLimit:    limit,
		name:            name,
		countSinceReset: 0,
	}
}

func (this *reservoirSampling) AddPoint(val float64) {
	this.countSinceReset += 1
	if len(this.samples) < int(this.samplesLimit) {
		this.samples = append(this.samples, val)
	} else {
		if j := rand.Int63n(int64(this.countSinceReset)); j < int64(len(this.samples)) {
			this.samples[j] = val
		}
	}
}

func (this *reservoirSampling) indexForPercentile(percentile int) int {
	return int(float64(len(this.samples)) * 0.01 * float64(percentile))
}

func (this *reservoirSampling) Reset() {
	this.samples = this.samples[:0]
	this.countSinceReset = 0
}

func (this *reservoirSampling) GetStats() (mean float64, median float64, min float64, percentile10 float64, percentile30 float64, percentile70 float64, percentile90 float64, max float64, count uint64, poolSize uint64) {
	sort.Slice(this.samples, func(i, j int) bool { return this.samples[i] < this.samples[j] })

	samplesCount := len(this.samples)
	if samplesCount == 0 {
		return 0, 0, 0, 0, 0, 0, 0, 0, 0, 0
	}

	var sum float64
	for _, val := range this.samples {
		sum += float64(val)
	}

	return sum / float64(samplesCount),
		this.samples[this.indexForPercentile(50)],
		this.samples[0],
		this.samples[this.indexForPercentile(10)],
		this.samples[this.indexForPercentile(30)],
		this.samples[this.indexForPercentile(70)],
		this.samples[this.indexForPercentile(90)],
		this.samples[len(this.samples)-1],
		this.countSinceReset,
		uint64(len(this.samples))
}
