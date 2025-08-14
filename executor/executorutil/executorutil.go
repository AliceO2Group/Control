/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2022 CERN and copyright holders of ALICE O².
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

// Package executorutil provides utility functions for the executor,
// including resource management and task execution helpers.
package executorutil

import (
	"os/exec"
	"os/user"
	"strconv"
	"strings"

	"github.com/AliceO2Group/Control/common/utils/uid"
	mesos "github.com/mesos/mesos-go/api/v1/lib"
)

type Labeler interface {
	GetLabels() *mesos.Labels
}

func GetEnvironmentIdFromLabelerType(labeler Labeler) uid.ID {
	envId := uid.NilID()
	var err error
	labels := labeler.GetLabels()

	if labels != nil && len(labels.GetLabels()) > 0 {
		for _, label := range labels.GetLabels() {
			if label.GetKey() == "environmentId" && label.GetValue() != "" {
				envId, err = uid.FromString(label.GetValue())
				if err != nil {
					envId = uid.NilID()
				}
				break
			}
		}
	}
	return envId
}

func GetValueFromLabelerType(labeler Labeler, key string) string {
	labels := labeler.GetLabels()

	if labels != nil && len(labels.GetLabels()) > 0 {
		for _, label := range labels.GetLabels() {
			if label.GetKey() == key && label.GetValue() != "" {
				return label.GetValue()
			}
		}
	}
	return ""
}

func GetGroupIDs(targetUser *user.User) ([]uint32, []string) {
	gidStrings, err := targetUser.GroupIds()
	if err != nil {
		// Non-nil error means we're building with !osusergo, so our binary is fully
		// static and therefore certain CGO calls needed by os.user aren't available.
		// We work around by calling `id -G username` like in the shell.
		idCmd := exec.Command("id", "-G", targetUser.Username) // list of GIDs for a given user
		output, err := idCmd.Output()
		if err != nil {
			return []uint32{}, []string{}
		}
		gidStrings = strings.Split(string(output[:]), " ")
	}

	gids := make([]uint32, len(gidStrings))
	for i, v := range gidStrings {
		parsed, err := strconv.ParseUint(strings.TrimSpace(v), 10, 32)
		if err != nil {
			return []uint32{}, []string{}
		}
		gids[i] = uint32(parsed)
	}
	return gids, gidStrings
}

// TruncateCommandBeforeTheLastPipe
// DPL commands are often aggregated in pipes, one after another.
// In certain contexts only the piece after the last pipe actually matters,
// while the previous ones only provide awareness of the whole topology.
// The current IL does not allow us to fit messages longer than 1024 characters,
// so we end up with truncated logs anyway.
// This method attempts to keep the piece after the last '|' intact, but will trim anything before
// if the total command length is going to exceed maxLength.
func TruncateCommandBeforeTheLastPipe(cmd string, maxLength int) string {
	lastPipeIdx := strings.LastIndex(cmd, "|")
	const separator = " <CUT> |"
	if lastPipeIdx == -1 || lastPipeIdx+1 == len(cmd) || len(cmd) < maxLength {
		return cmd
	} else {
		budgetForBeginning := maxLength - (len(cmd) - lastPipeIdx) - len(separator)
		return cmd[:budgetForBeginning] + separator + cmd[lastPipeIdx+1:]
	}
}
