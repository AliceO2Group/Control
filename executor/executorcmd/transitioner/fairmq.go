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

package transitioner

import "github.com/AliceO2Group/Control/executor/executorcmd/transitioner/fairmq"

type FairMQ struct {
	DoTransition DoTransitionFunc

	stateMap    map[string]string
	invStateMap map[string]string
}

func NewFairMQTransitioner(transitionFunc DoTransitionFunc) *FairMQ {
	stateMap := map[string]string{
		"STANDBY":    fairmq.IDLE,
		"CONFIGURED": fairmq.READY,
		"RUNNING":    fairmq.RUNNING,
		"ERROR":      fairmq.ERROR,
		"DONE":       fairmq.EXITING,
	}
	return &FairMQ{
		DoTransition: transitionFunc,
		stateMap:     stateMap,
		invStateMap: func() (inv map[string]string) { // invert stateMap
			inv = make(map[string]string, len(stateMap))
			for k, v := range stateMap {
				inv[v] = k
			}
			return
		}(),
	}
}

func (cm *FairMQ) Commit(evt string, src string, dst string, args map[string]string) (finalState string, err error) {
	//FIXME: Use an enum for O²C states and strings for FMQ states
	switch evt {
	case "START":
		finalState, err = cm.DoTransition(EventInfo{fairmq.EvtRUN, cm.fmqStateForState(src), cm.fmqStateForState(dst), args})
		finalState = cm.stateForFmqState(finalState)
	case "STOP":
		finalState, err = cm.DoTransition(EventInfo{fairmq.EvtSTOP, cm.fmqStateForState(src), cm.fmqStateForState(dst), args})
		finalState = cm.stateForFmqState(finalState)
	case "RECOVER":
		fallthrough
	case "GO_ERROR":
		log.WithField("event", evt).Error("transition not implemented yet")
		finalState = src
	case "CONFIGURE":
		finalState, err = cm.doConfigure(evt, src, dst, args)
	case "RESET":
		finalState, err = cm.doReset(evt, src, dst, args)
	case "EXIT":
		var state string
		if src == "CONFIGURED" { // We need to RESET first
			state, err = cm.doReset(evt, src, dst, args)
			if state != "STANDBY" {
				finalState = state
				break
			}
		}
		finalState, err = cm.DoTransition(EventInfo{fairmq.EvtEND, cm.fmqStateForState(src), cm.fmqStateForState(dst), args})
		finalState = cm.stateForFmqState(finalState)
	default:
		log.WithField("event", evt).Error("transition impossible")
	}

	return
}

func (cm *FairMQ) fmqStateForState(stateName string) string {
	if cm == nil {
		return ""
	}

	fmqState, ok := cm.stateMap[stateName]
	if !ok {
		return ""
	}
	return fmqState
}

func (cm *FairMQ) stateForFmqState(fmqStateName string) string {
	if cm == nil {
		return ""
	}

	state, ok := cm.invStateMap[fmqStateName]
	if !ok {
		return ""
	}
	return state
}

func (cm *FairMQ) doConfigure(evt string, src string, dst string, args map[string]string) (finalState string, err error) {
	var state string
	state, err = cm.DoTransition(EventInfo{fairmq.EvtINIT_DEVICE, cm.fmqStateForState(src), fairmq.INITIALIZING_DEVICE, args})
	if state != fairmq.INITIALIZING_DEVICE {
		finalState = cm.stateForFmqState(state)
		return
	}

	state, err = cm.DoTransition(EventInfo{fairmq.EvtCOMPLETE_INIT, fairmq.INITIALIZING_DEVICE, fairmq.INITIALIZED, nil})
	if state != fairmq.INITIALIZED {
		finalState = cm.stateForFmqState(state)
		return
	}

	state, err = cm.DoTransition(EventInfo{fairmq.EvtBIND, fairmq.INITIALIZED, fairmq.BOUND, nil})
	if state == fairmq.INITIALIZED { // If we're stuck in the intermediate INITIALIZED state, we roll back to IDLE
		state, _ = cm.DoTransition(EventInfo{fairmq.EvtRESET_DEVICE, fairmq.INITIALIZED, cm.fmqStateForState(src), nil})
	} else if state != fairmq.BOUND {
		finalState = cm.stateForFmqState(state)
		return
	}

	state, err = cm.DoTransition(EventInfo{fairmq.EvtCONNECT, fairmq.BOUND, fairmq.DEVICE_READY, nil})
	if state == fairmq.BOUND { // If we're stuck in the intermediate BOUND state, we roll back to IDLE
		state, _ = cm.DoTransition(EventInfo{fairmq.EvtRESET_DEVICE, fairmq.BOUND, cm.fmqStateForState(src), nil})
	} else if state != fairmq.DEVICE_READY {
		finalState = cm.stateForFmqState(state)
		return
	}

	state, err = cm.DoTransition(EventInfo{fairmq.EvtINIT_TASK, fairmq.DEVICE_READY, cm.fmqStateForState(dst), nil})
	if state == fairmq.DEVICE_READY { // If we're stuck in the intermediate DEVICE READY state, we roll back to IDLE
		state, _ = cm.DoTransition(EventInfo{fairmq.EvtRESET_DEVICE, fairmq.DEVICE_READY, cm.fmqStateForState(src), nil})
	}
	finalState = cm.stateForFmqState(state)
	return
}

func (cm *FairMQ) doReset(evt string, src string, dst string, args map[string]string) (finalState string, err error) {
	var state string
	state, err = cm.DoTransition(EventInfo{fairmq.EvtRESET_TASK, cm.fmqStateForState(src), fairmq.DEVICE_READY, nil})
	if state != fairmq.DEVICE_READY {
		finalState = cm.stateForFmqState(state)
		return
	}
	state, err = cm.DoTransition(EventInfo{fairmq.EvtRESET_DEVICE, fairmq.DEVICE_READY, cm.fmqStateForState(dst), args})
	if state == fairmq.DEVICE_READY { // If we're stuck in the intermediate DEVICE READY state, we roll back to READY
		state, _ = cm.DoTransition(EventInfo{fairmq.EvtINIT_TASK, fairmq.DEVICE_READY, cm.fmqStateForState(src), nil})
	}
	finalState = cm.stateForFmqState(state)
	return
}

func (cm *FairMQ) FromDeviceState(state string) string {
	return cm.stateForFmqState(state)
}
