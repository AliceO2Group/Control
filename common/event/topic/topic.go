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

package topic

type Topic string

const (
	Separator       = "." // used to separate topic segments
	Root      Topic = "aliecs"

	Run         Topic = Root + Separator + "run" // currently unused, intended for detailed run information e.g. SOR, SOEOR, EOEOR
	Environment Topic = Root + Separator + "environment"
	Role        Topic = Root + Separator + "role" // currently unused, intended for role state change events
	Task        Topic = Root + Separator + "task"
	Call        Topic = Root + Separator + "call"

	Core Topic = Root + Separator + "core"

	IntegratedService Topic = Root + Separator + "integrated_service"
)
