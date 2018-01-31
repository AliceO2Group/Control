/*
 * === This file is part of octl <https://github.com/teo/octl> ===
 *
 * Copyright 2017 CERN and copyright holders of ALICE O².
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

package environment

//import "github.com/looplab/fsm"



// TODO: this is the FSM of each O² process, for further reference
//fsm := fsm.NewFSM(
//	"STANDBY",
//	fsm.Events{
//		{Name: "CONFIGURE", Src: []string{"STANDBY", "CONFIGURED"},           Dst: "CONFIGURED"},
//		{Name: "START",     Src: []string{"CONFIGURED"},                      Dst: "RUNNING"},
//		{Name: "STOP",      Src: []string{"RUNNING", "PAUSED"},               Dst: "CONFIGURED"},
//		{Name: "PAUSE",     Src: []string{"RUNNING"},                         Dst: "PAUSED"},
//		{Name: "RESUME",    Src: []string{"PAUSED"},                          Dst: "RUNNING"},
//		{Name: "EXIT",      Src: []string{"CONFIGURED", "STANDBY"},           Dst: "FINAL"},
//		{Name: "GO_ERROR",  Src: []string{"CONFIGURED", "RUNNING", "PAUSED"}, Dst: "ERROR"},
//		{Name: "RESET",     Src: []string{"ERROR"},                           Dst: "STANDBY"},
//	},
//	fsm.Callbacks{},
//)


type O2Process struct {
	Name		string			`json:"name" binding:"required"`
	Command		string			`json:"command" binding:"required"`
	Args		[]string		`json:"args" binding:"required"`
	//Fsm			*fsm.FSM		`json:"-"`	// skip
	//			↑ this guy will initially only have 2 states: running and not running, or somesuch
	//			  but he doesn't belong here because all this should be immutable
}

/*func NewO2Process() *O2Process {
	return &O2Process{
		Fsm: fsm.NewFSM(
			"STOPPED",
			fsm.Events{
				{Name: "START",	Src: []string{"STOPPED"},	Dst:"STARTED"},
				{Name: "STOP",	Src: []string{"STARTED"},	Dst:"STOPPED"},
			},
			fsm.Callbacks{},
		),
		Deployed: false,
	}
}*/


type Role struct {
	Name			string			`json:"name" binding:"required"`
	Process 		O2Process		`json:"process" binding:"required"`
	RoleWantsCPU	float64			`json:"roleCPU" binding:"required"`
	RoleWantsMemory	float64			`json:"roleMemory" binding:"required"`
}

type Allocation struct {
	RoleName		string
	Role			*Role
	Hostname		string
	RoleKind		string
	AgentId			string
	OfferId			string
	TaskId			string
}