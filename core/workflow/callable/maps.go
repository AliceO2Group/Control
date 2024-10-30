/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2021-2022 CERN and copyright holders of ALICE O².
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

package callable

import "sort"

type HooksMap map[HookWeight]Hooks
type CallsMap map[HookWeight]Calls
type HookWeight int

func (m HooksMap) GetWeights() []HookWeight {
	weights := make([]int, len(m))
	i := 0
	for k := range m {
		weights[i] = int(k)
		i++
	}
	sort.Ints(weights)
	out := make([]HookWeight, len(weights))
	for i, v := range weights {
		out[i] = HookWeight(v)
	}
	return out
}

func (m CallsMap) GetWeights() []HookWeight {
	weights := make([]int, len(m))
	i := 0
	for k := range m {
		weights[i] = int(k)
		i++
	}
	sort.Ints(weights)
	out := make([]HookWeight, len(weights))
	for i, v := range weights {
		out[i] = HookWeight(v)
	}
	return out
}
