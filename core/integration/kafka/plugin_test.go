/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2024 CERN and copyright holders of ALICE O².
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

package kafka

import (
	"testing"

	kafkapb "github.com/AliceO2Group/Control/core/integration/kafka/protos"
)

func TestSortActiveRuns(t *testing.T) {
	p := Plugin{}
	p.SortRunningEnvList(nil)

	var envs []*kafkapb.EnvInfo
	envs = append(envs,
		&kafkapb.EnvInfo{EnvironmentId: "1", Detectors: []string{"first"}},
		&kafkapb.EnvInfo{EnvironmentId: "2", Detectors: []string{"second"}},
		&kafkapb.EnvInfo{EnvironmentId: "3", Detectors: []string{"ITS", "first"}},
		&kafkapb.EnvInfo{EnvironmentId: "4", Detectors: []string{"first", "second", "third"}})

	p.SortRunningEnvList(envs)

	if len(envs) != 4 {
		t.Error("wrong number of environments")
	}

	if envs[0].EnvironmentId != "3" {
		t.Errorf("first should have been environment 3, but is %s with dets %v", envs[0].EnvironmentId, envs[0].Detectors)
	}

	if envs[1].EnvironmentId != "4" {
		t.Errorf("second should have been environment 4, but is %s with dets %v", envs[1].EnvironmentId, envs[1].Detectors)
	}

	if envs[2].EnvironmentId != "1" {
		t.Errorf("third should have been environment 1, but is %s with dets %v", envs[2].EnvironmentId, envs[2].Detectors)
	}

	if envs[3].EnvironmentId != "2" {
		t.Errorf("fourth should have been environment 2, but is %s with dets %v", envs[3].EnvironmentId, envs[3].Detectors)
	}
}
