/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2024 CERN and copyright holders of ALICE O².
 * Author: Michal Tichak <michal.tichak@cern.ch>
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

package controller

import (
	"context"
	"fmt"
	"strings"

	pb "github.com/AliceO2Group/Control/operator/internal/controller/protos/generated"
)

// FairMQ internal state names as used by the OCC plugin
const (
	fmqIdle               = "IDLE"
	fmqInitializingDevice = "INITIALIZING DEVICE"
	fmqInitialized        = "INITIALIZED"
	fmqBound              = "BOUND"
	fmqDeviceReady        = "DEVICE READY"
	fmqReady              = "READY"
	fmqRunning            = "RUNNING"
	fmqError              = "ERROR"
)

// FairMQ transition event names as used by the OCC plugin
const (
	fmqEvtInitDevice   = "INIT DEVICE"
	fmqEvtCompleteInit = "COMPLETE INIT"
	fmqEvtBind         = "BIND"
	fmqEvtConnect      = "CONNECT"
	fmqEvtInitTask     = "INIT TASK"
	fmqEvtRun          = "RUN"
	fmqEvtStop         = "STOP"
	fmqEvtResetTask    = "RESET TASK"
	fmqEvtResetDevice  = "RESET DEVICE"
)

// fmqToOCCState maps FairMQ internal states to lowercase OCC state names
var fmqToOCCState = map[string]string{
	fmqIdle:               "standby",
	fmqInitializingDevice: "standby",
	fmqInitialized:        "standby",
	fmqBound:              "standby",
	fmqDeviceReady:        "standby",
	fmqReady:              "configured",
	fmqRunning:            "running",
	fmqError:              "error",
}

func occStateForFmqState(fmqState string) string {
	if occ, ok := fmqToOCCState[strings.ToUpper(fmqState)]; ok {
		return occ
	}
	return "standby"
}

// doFairMQStep sends a single FairMQ-level gRPC Transition and returns the
// resulting FairMQ state as reported by the OCC plugin.
func (c *OccClient) doFairMQStep(ctx context.Context, srcFmqState, fmqEvent string, args map[string]string) (string, error) {
	var configEntries []*pb.ConfigEntry
	for k, v := range args {
		configEntries = append(configEntries, &pb.ConfigEntry{Key: k, Value: v})
	}

	request := &pb.TransitionRequest{
		SrcState:        srcFmqState,
		TransitionEvent: fmqEvent,
		Arguments:       configEntries,
	}
	c.log.V(1).Info("FairMQ step", "event", fmqEvent, "src", srcFmqState)

	reply, err := c.client.Transition(ctx, request)
	if err != nil {
		return "", fmt.Errorf("FairMQ gRPC transition %q failed: %w", fmqEvent, err)
	}

	resultState := reply.GetState()
	if !reply.GetOk() {
		return resultState, fmt.Errorf("FairMQ transition %q not ok, state: %s", fmqEvent, resultState)
	}
	c.log.V(1).Info("FairMQ step done", "event", fmqEvent, "result", resultState)
	return resultState, nil
}

// fairMQConfigure drives the FairMQ device through the 5-step CONFIGURE sequence:
//
//	IDLE → (INIT DEVICE + args) → INITIALIZING DEVICE
//	     → (COMPLETE INIT)       → INITIALIZED
//	     → (BIND)                → BOUND
//	     → (CONNECT)             → DEVICE READY
//	     → (INIT TASK)           → READY (= CONFIGURED in OCC terms)
//
// On failure in any intermediate step, a RESET DEVICE rollback is attempted.
func (c *OccClient) fairMQConfigure(ctx context.Context, args map[string]string) (string, error) {
	// Step 1: INIT DEVICE — channel config args go here
	state, err := c.doFairMQStep(ctx, fmqIdle, fmqEvtInitDevice, args)
	if state != fmqInitializingDevice {
		return occStateForFmqState(state), fmt.Errorf("INIT DEVICE: expected %s, got %s: %w", fmqInitializingDevice, state, err)
	}

	// Step 2: COMPLETE INIT
	state, err = c.doFairMQStep(ctx, fmqInitializingDevice, fmqEvtCompleteInit, nil)
	if state != fmqInitialized {
		return occStateForFmqState(state), fmt.Errorf("COMPLETE INIT: expected %s, got %s: %w", fmqInitialized, state, err)
	}

	// Step 3: BIND
	state, err = c.doFairMQStep(ctx, fmqInitialized, fmqEvtBind, nil)
	if state == fmqInitialized {
		// Stuck — roll back to IDLE
		rollback, _ := c.doFairMQStep(ctx, fmqInitialized, fmqEvtResetDevice, nil)
		return occStateForFmqState(rollback), fmt.Errorf("BIND: stuck in %s, rolled back to %s", fmqInitialized, rollback)
	} else if state != fmqBound {
		return occStateForFmqState(state), fmt.Errorf("BIND: expected %s, got %s: %w", fmqBound, state, err)
	}

	// Step 4: CONNECT
	state, err = c.doFairMQStep(ctx, fmqBound, fmqEvtConnect, nil)
	if state == fmqBound {
		// Stuck — roll back to IDLE
		rollback, _ := c.doFairMQStep(ctx, fmqBound, fmqEvtResetDevice, nil)
		return occStateForFmqState(rollback), fmt.Errorf("CONNECT: stuck in %s, rolled back to %s", fmqBound, rollback)
	} else if state != fmqDeviceReady {
		return occStateForFmqState(state), fmt.Errorf("CONNECT: expected %s, got %s: %w", fmqDeviceReady, state, err)
	}

	// Step 5: INIT TASK
	state, err = c.doFairMQStep(ctx, fmqDeviceReady, fmqEvtInitTask, nil)
	if state == fmqDeviceReady {
		// Stuck — roll back to IDLE
		rollback, _ := c.doFairMQStep(ctx, fmqDeviceReady, fmqEvtResetDevice, nil)
		return occStateForFmqState(rollback), fmt.Errorf("INIT TASK: stuck in %s, rolled back to %s", fmqDeviceReady, rollback)
	}

	return occStateForFmqState(state), err
}

// fairMQReset drives the FairMQ device through the 2-step RESET sequence:
//
//	READY → (RESET TASK)         → DEVICE READY
//	      → (RESET DEVICE + args) → IDLE (= STANDBY in OCC terms)
func (c *OccClient) fairMQReset(ctx context.Context, args map[string]string) (string, error) {
	// Step 1: RESET TASK
	state, err := c.doFairMQStep(ctx, fmqReady, fmqEvtResetTask, nil)
	if state != fmqDeviceReady {
		return occStateForFmqState(state), fmt.Errorf("RESET TASK: expected %s, got %s: %w", fmqDeviceReady, state, err)
	}

	// Step 2: RESET DEVICE — args go here, matching executor doReset behaviour
	state, err = c.doFairMQStep(ctx, fmqDeviceReady, fmqEvtResetDevice, args)
	if state == fmqDeviceReady {
		// Stuck — roll back to READY
		rollback, _ := c.doFairMQStep(ctx, fmqDeviceReady, fmqEvtInitTask, nil)
		return occStateForFmqState(rollback), fmt.Errorf("RESET DEVICE: stuck in %s, rolled back to %s", fmqDeviceReady, rollback)
	}

	return occStateForFmqState(state), err
}

// FairMQTransitionRequest drives a FairMQ OCC device through the multi-step
// sequence that corresponds to a single OCC-level transition (CONFIGURE, RESET,
// START, STOP). Returns the resulting OCC state in lowercase ("configured",
// "running", "standby", "error").
func (c *OccClient) FairMQTransitionRequest(ctx context.Context, fromState, toState string, args map[string]string) (string, error) {
	if c == nil || c.client == nil {
		return fromState, fmt.Errorf("nil client for FairMQTransitionRequest")
	}

	from, err := StateFromString(fromState)
	if err != nil {
		return fromState, err
	}
	to, err := StateFromString(toState)
	if err != nil {
		return fromState, err
	}

	transition, err := FromStatesToTransition(from, to)
	if err != nil {
		return fromState, err
	}

	c.log.Info("FairMQTransitionRequest", "from", fromState, "to", toState, "transition", transition.String(), "args", args)

	switch transition {
	case CONFIGURE:
		return c.fairMQConfigure(ctx, args)
	case RESET:
		return c.fairMQReset(ctx, args)
	case START:
		state, err := c.doFairMQStep(ctx, fmqReady, fmqEvtRun, args)
		return occStateForFmqState(state), err
	case STOP:
		state, err := c.doFairMQStep(ctx, fmqRunning, fmqEvtStop, args)
		return occStateForFmqState(state), err
	default:
		return fromState, fmt.Errorf("FairMQ transition %s not implemented", transition.String())
	}
}
