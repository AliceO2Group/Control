/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2018-2019 CERN and copyright holders of ALICE O².
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

package executable

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"syscall"
	"time"

	"github.com/AliceO2Group/Control/common/event"
	"github.com/AliceO2Group/Control/core/controlcommands"
	"github.com/AliceO2Group/Control/executor/executorcmd"
	pb "github.com/AliceO2Group/Control/executor/protos"
	mesos "github.com/mesos/mesos-go/api/v1/lib"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

type ControllableTask struct {
	taskBase
	rpc *executorcmd.RpcClient
	pendingFinalTaskStateCh chan mesos.TaskState
}

func (t *ControllableTask) Launch() error {
	t.pendingFinalTaskStateCh = make(chan mesos.TaskState, 1) // we use this to receive a pending status update if the task was killed
	taskCmd := prepareTaskCmd(t.tci)
	log.WithField("payload", string(t.ti.GetData()[:])).
		WithField("task", t.ti.Name).
		Debug("starting task")

	// Set up pipes for controlled process
	var errStdout, errStderr error
	stdoutIn, _ := taskCmd.StdoutPipe()
	stderrIn, _ := taskCmd.StderrPipe()

	err := taskCmd.Start()
	if err != nil {
		log.WithFields(logrus.Fields{
			"id":      t.ti.TaskID.Value,
			"task":    t.ti.Name,
			"error":   err,
			"command": *t.tci.Value,
		}).
		Error("failed to run task")

		t.sendStatus(mesos.TASK_FAILED, err.Error())
		return err
	}
	log.WithField("id", t.ti.TaskID.Value).
		WithField("task", t.ti.Name).
		Debug("task started")

	go func() {
		_, errStdout = io.Copy(log.WithPrefix("task-stdout").WithField("task", t.ti.Name).Writer(), stdoutIn)
	}()
	go func() {
		_, errStderr = io.Copy(log.WithPrefix("task-stderr").WithField("task", t.ti.Name).Writer(), stderrIn)
	}()

	log.WithFields(logrus.Fields{
		"controlPort": t.tci.ControlPort,
		"controlMode": t.tci.ControlMode.String(),
		"task":        t.ti.Name,
		"id":          t.ti.TaskID.Value,
	}).
	Debug("starting gRPC client")
	t.rpc = executorcmd.NewClient(t.tci.ControlPort, t.tci.ControlMode)
	if t.rpc == nil {
		return errors.New("could not start gRPC client")
	}
	t.rpc.TaskCmd = taskCmd

	// We fork out into a goroutine for the actual process management.
	// Control returns to the event loop which can safely access *internalState.
	// Anything in the following goroutine must not touch *internalState, except via channels.
	go func() {
		elapsed := 0 * time.Second
		for {
			log.WithFields(logrus.Fields{
				"id":      t.ti.TaskID.Value,
				"task":    t.ti.Name,
				"elapsed": elapsed.String(),
			}).
				Debug("polling task for IDLE state reached")

			response, err := t.rpc.GetState(context.TODO(), &pb.GetStateRequest{}, grpc.EmptyCallOption{})
			if err != nil {
				log.WithError(err).
					WithField("task", t.ti.Name).
					Info("cannot query task status")
			} else {
				log.WithField("state", response.GetState()).
					WithField("task", t.ti.Name).
					Debug("task status queried")
			}
			// NOTE: we acquire the transitioner-dependent STANDBY equivalent state
			reachedState := t.rpc.FromDeviceState(response.GetState())

			if reachedState == "STANDBY" && err == nil {
				log.WithField("id", t.ti.TaskID.Value).
					WithField("task", t.ti.Name).
					Debug("task running and ready for control input")
				break
			} else if reachedState == "DONE" || reachedState == "ERROR" {
				// something went wrong, the device moved to DONE or ERROR on startup
				_ = syscall.Kill(-taskCmd.Process.Pid, syscall.SIGKILL)

				log.WithField("task", t.ti.Name).Debug("task killed")
				t.sendStatus(mesos.TASK_FAILED, "task reached wrong state on startup")
				return
			} else if elapsed >= startupTimeout {
				err = errors.New("timeout while waiting for task startup")
				log.WithField("task", t.ti.Name).Error(err.Error())
				t.sendStatus(mesos.TASK_FAILED, err.Error())
				_ = t.rpc.Close()
				t.rpc = nil
				return
			} else {
				log.WithField("task", t.ti.Name).
					Debugf("task not ready yet, waiting %s", startupPollingInterval.String())
				time.Sleep(startupPollingInterval)
				elapsed += startupPollingInterval
			}
		}

		// Set up event stream from task
		esc, err := t.rpc.EventStream(context.TODO(), &pb.EventStreamRequest{}, grpc.EmptyCallOption{})
		if err != nil {
			log.WithField("task", t.ti.Name).
				WithError(err).
				Error("cannot set up event stream from task")
			t.sendStatus(mesos.TASK_FAILED, err.Error())
			_ = t.rpc.Close()
			t.rpc = nil
			return
		}

		log.WithField("task", t.ti.Name).Debug("notifying of task running state")

		// send RUNNING
		t.sendStatus(mesos.TASK_RUNNING, "")

		// Process events from task in yet another goroutine
		go func() {
			deo := event.DeviceEventOrigin{
				AgentId:    t.ti.AgentID,
				ExecutorId: t.ti.GetExecutor().ExecutorID,
				TaskId:     t.ti.TaskID,
			}
			for {
				if t.rpc == nil {
					log.WithError(err).Warning("event stream done")
					break
				}
				esr, err := esc.Recv()
				if err == io.EOF {
					log.WithError(err).Warning("event stream EOF")
					break
				}
				if err != nil {
					log.WithError(err).Warning("error receiving event from task")
					continue
				}
				ev := esr.GetEvent()

				deviceEvent := event.NewDeviceEvent(deo, ev.GetType())
				if deviceEvent == nil {
					log.Debug("nil DeviceEvent received (NULL_DEVICE_EVENT) - closing stream")
					break
				}

				t.sendDeviceEvent(deviceEvent)
			}
		}()

		err = taskCmd.Wait()
		// ^ when this unblocks, the task is done

		pendingState := mesos.TASK_FINISHED
		if err != nil {
			log.WithFields(logrus.Fields{
				"id":    t.ti.TaskID.Value,
				"task":  t.ti.Name,
				"error": err.Error(),
			}).
			Error("process terminated with error")
			pendingState = mesos.TASK_FAILED
		}

		select {
		case pending := <- t.pendingFinalTaskStateCh:
			pendingState = pending
		default:
		}

		if t.rpc != nil {
			_ = t.rpc.Close() // NOTE: might return non-nil error, but we don't care much
			log.Debug("rpc client closed")
			t.rpc = nil
			log.Debug("rpc client removed")
		}

		if errStdout != nil || errStderr != nil {
			log.WithFields(logrus.Fields{
				"errStderr": errStderr,
				"errStdout": errStdout,
				"id":        t.ti.TaskID.Value,
				"task":      t.ti.Name,
			}).
			Warning("failed to capture stdout or stderr of task")
		}

		log.WithField("task", t.ti.Name).
			WithField("status", pendingState.String()).
			Debug("sending final status update")
		t.sendStatus(pendingState, "")
	}()

	log.WithField("task", t.ti.Name).Debug("gRPC client running, handler forked")
	return nil
}

func (t *ControllableTask) UnmarshalTransition(data []byte) (cmd *executorcmd.ExecutorCommand_Transition, err error) {
	cmd = new(executorcmd.ExecutorCommand_Transition)
	if t.rpc == nil {
		err = errors.New("cannot unmarshal transition: RPC is down")
		cmd = nil
		return
	}
	cmd.Transitioner = t.rpc.Transitioner
	err = json.Unmarshal(data, cmd)
	if err != nil {
		cmd = nil
	}
	return
}

func (t *ControllableTask) Transition(cmd *executorcmd.ExecutorCommand_Transition) *controlcommands.MesosCommandResponse_Transition {
	newState, transitionError := cmd.Commit()

	response := cmd.PrepareResponse(transitionError, newState, t.ti.TaskID.Value)
	return response
}

func (t *ControllableTask) Kill() error {
	response, err := t.rpc.GetState(context.TODO(), &pb.GetStateRequest{}, grpc.EmptyCallOption{})
	if err != nil {
		log.WithError(err).WithField("taskId", t.ti.GetTaskID()).Error("cannot query task status")
	} else {
		log.WithField("state", response.GetState()).WithField("taskId", t.ti.GetTaskID()).Debug("task status queried")
	}

	// NOTE: we acquire the transitioner-dependent STANDBY equivalent state
	reachedState := t.rpc.FromDeviceState(response.GetState())


	nextTransition := func(currentState string) (exc *executorcmd.ExecutorCommand_Transition) {
		log.WithField("currentState", currentState).
			Debug("nextTransition(currentState) BEGIN")
		var evt, destination string
		switch currentState {
		case "RUNNING":
			evt = "STOP"
			destination = "CONFIGURED"
		case "CONFIGURED":
			evt = "RESET"
			destination = "STANDBY"
		case "ERROR":
			evt = "RECOVER"
			destination = "STANDBY"
		case "STANDBY":
			evt = "EXIT"
			destination = "DONE"
		}
		log.WithField("evt", evt).
			WithField("dst", destination).
			Debug("nextTransition(currentState) BEGIN")

		exc = executorcmd.NewLocalExecutorCommand_Transition(
			t.rpc.Transitioner,
			[]controlcommands.MesosCommandTarget{
				{
					AgentId: mesos.AgentID{},       // AgentID and ExecutorID can stay empty because it's a
					ExecutorId: mesos.ExecutorID{}, // local transition
					TaskId: t.ti.GetTaskID(),
				},
			},
			reachedState,
			evt,
			destination,
			nil,
		)
		return
	}

	for reachedState != "DONE" {
		cmd := nextTransition(reachedState)
		log.WithFields(logrus.Fields{
			"evt": cmd.Event,
			"src": cmd.Source,
			"dst": cmd.Destination,
			"targetList": cmd.TargetList,
		}).
		Debug("state DONE not reached, about to commit transition")

		newState, transitionError := cmd.Commit()

		log.WithField("newState", newState).
			WithError(transitionError).
			Debug("transition committed")
		if transitionError != nil || len(cmd.Event) == 0 {
			log.WithError(transitionError).Error("cannot gracefully end task")
			break
		}
		reachedState = newState
	}

	log.Debug("end transition loop done")

	pid := t.rpc.TaskCmd.Process.Pid
	_ = t.rpc.Close()
	t.rpc = nil

	if reachedState == "DONE" {
		log.Debug("task exited correctly")
		t.pendingFinalTaskStateCh <- mesos.TASK_FINISHED
	} else { // something went wrong
		log.Debug("task killed")
		t.pendingFinalTaskStateCh <- mesos.TASK_KILLED
	}


	// TODO: do a SIGTERM before the SIGKILL

	// When killing we must always use syscall.Kill with a negative PID, in order to kill all
	// children which were assigned the same PGID at launch
	killErr := syscall.Kill(-pid, syscall.SIGKILL)
	if killErr != nil {
		log.WithError(killErr).
			WithField("taskId", t.ti.GetTaskID()).
			Warning("could not kill task")
	}

	return killErr
}