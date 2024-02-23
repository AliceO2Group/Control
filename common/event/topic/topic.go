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
	Event     Topic = Root + Separator + "event"

	Ev_Env            Topic = Event + Separator + "environment"
	Ev_Env_EnterState Topic = Ev_Env + Separator + "enter_state"
	Ev_Env_LeaveState Topic = Ev_Env + Separator + "leave_state"
	Ev_Env_BeforeEv   Topic = Ev_Env + Separator + "before_event"
	Ev_Env_AfterEv    Topic = Ev_Env + Separator + "after_event"

	Ev_Role Topic = Event + Separator + "role"

	Ev_Task             = Event + Separator + "task"
	Ev_Task_Lifecycle   = Ev_Task + Separator + "lifecycle"
	Ev_Task_StateChange = Ev_Task + Separator + "state_change"

	Ev_IntegratedService = Event + Separator + "integrated_service"

	Ev_Meta           = Event + Separator + "meta"
	Ev_Meta_Framework = Ev_Meta + Separator + "framework"
	Ev_Meta_Mesos     = Ev_Meta + Separator + "mesos"
)
