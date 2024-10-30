/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2020 CERN and copyright holders of ALICE O².
 * Author: Miltiadis Alexis <miltiadis.alexis@cern.ch>
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

package event

type EnvironmentEvent struct {
	eventBase
	EnvironmentID string
	Run           uint32
	State         string
	Error         error
	Message       string
}

func (e *EnvironmentEvent) GetName() string {
	return e.EnvironmentID
}

func (e *EnvironmentEvent) GetRun() uint32 {
	return e.Run
}

func (e *EnvironmentEvent) GetState() string {
	return e.State
}

func (e *EnvironmentEvent) GetError() string {
	if e.Error == nil {
		return ""
	}
	return e.Error.Error()
}

func (e *EnvironmentEvent) GetMessage() string {
	return e.Message
}
