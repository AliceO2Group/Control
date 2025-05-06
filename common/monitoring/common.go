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

type key struct {
	nameTagsHash uint64
	timestamp    time.Time
}

func metricNameTagsToHash(hash *maphash.Hash, metric *Metric) {
	hash.WriteString(metric.name)

	for _, tag := range metric.tags {
		hash.WriteString(tag.name)
		hash.WriteString(tag.value)
	}
}

func hashValueAndReset(hash *maphash.Hash) uint64 {
	hashValue := hash.Sum64()
	hash.Reset()
	return hashValue
}
