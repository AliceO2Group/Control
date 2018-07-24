/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2018 CERN and copyright holders of ALICE O².
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

package controlcommands

import (
	"strings"
	"fmt"
	"errors"
)

type MesosCommandMultiResponse struct {
	MesosCommandResponseBase

	responses   map[MesosCommandTarget]MesosCommandResponse
}

func (m *MesosCommandMultiResponse) GetResponses() map[MesosCommandTarget]MesosCommandResponse {
	if m == nil {
		return nil
	}
	return m.responses
}

func (m *MesosCommandMultiResponse) IsMultiResponse() bool {
	return true
}

func (m *MesosCommandMultiResponse) Err() error {
	if m == nil {
		return nil
	}
	errs := make(map[MesosCommandTarget]error, 0)
	for k, v := range m.responses {
		if v.Err() != nil {
			errs[k] = v.Err()
		}
	}

	return errors.New(strings.Join(func() (out []string) {
		for k, v := range errs {
			if v != nil && len(strings.TrimSpace(v.Error())) != 0 {
				out = append(out, fmt.Sprintf("[%s/%s] %s", k.AgentId, k.ExecutorId, v.Error()))
			}
		}
		return
	}(), "\n"))
}

func consolidateResponses(command MesosCommand, responses map[MesosCommandTarget]MesosCommandResponse) MesosCommandResponse {
	if len(responses) == 0 {
		return nil
	}
	if len(responses) == 1 {
		for _, v := range responses {
			return v
		}
	}
	return &MesosCommandMultiResponse{
		MesosCommandResponseBase: *NewMesosCommandResponse(command, nil),
		responses: responses,
	}
}