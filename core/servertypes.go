/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2018-2020 CERN and copyright holders of ALICE O².
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

package core

import (
	pb "github.com/AliceO2Group/Control/core/protos"
)

type EnvironmentInfos []*pb.EnvironmentInfo

func (infos EnvironmentInfos) Len() int {
	return len(infos)
}
func (infos EnvironmentInfos) Less(i, j int) bool {
	iv := infos[i]
	jv := infos[j]
	if iv == nil {
		return true
	}
	if jv == nil {
		return false
	}
	if iv.CreatedWhen < jv.CreatedWhen {
		return true
	} else {
		return false
	}
}
func (infos EnvironmentInfos) Swap(i, j int) {
	var temp *pb.EnvironmentInfo
	temp = infos[i]
	infos[i] = infos[j]
	infos[j] = temp
}
