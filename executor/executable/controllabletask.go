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
	"reflect"
	"strings"
	"syscall"
	"time"

	"github.com/AliceO2Group/Control/common/event"
	"github.com/AliceO2Group/Control/core/controlcommands"
	"github.com/AliceO2Group/Control/executor/executorcmd"
	pb "github.com/AliceO2Group/Control/executor/protos"
	mesos "github.com/mesos/mesos-go/api/v1/lib"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const(
	KILL_TIMEOUT = 3*time.Second
	TRANSITION_TIMEOUT = 10*time.Second
)

type ControllableTask struct {
	taskBase
	rpc *executorcmd.RpcClient
	pendingFinalTaskStateCh chan mesos.TaskState
	knownPid int
}

type CommitResponse struct {
	newState 	  string
	transitionError error
}

func (t *ControllableTask) Launch() error {
	t.pendingFinalTaskStateCh = make(chan mesos.TaskState, 1) // we use this to receive a pending status update if the task was killed
	taskCmd, err := prepareTaskCmd(t.tci)
	if err != nil {
		msg := "cannot build task command"
		log.WithFields(logrus.Fields{
				"id":      t.ti.TaskID.Value,
				"task":    t.ti.Name,
				"error":   err,
			}).
			Error(msg)

		t.sendStatus(mesos.TASK_FAILED, msg + ": " + err.Error())
		return err
	}

	log.WithField("payload", string(t.ti.GetData()[:])).
		WithField("task", t.ti.Name).
		Debug("starting task asynchronously")

	// We fork out into a goroutine for the actual process management.
	// Control returns to the event loop which can safely access *internalState.
	// Anything in the following goroutine must not touch *internalState, except
	// via channels.
	go func() {

		// Set up pipes for controlled process
		var errStdout, errStderr error
		stdoutIn, _ := taskCmd.StdoutPipe()
		stderrIn, _ := taskCmd.StderrPipe()

		err = taskCmd.Start()
		var tciCommandStr string
		if t.tci.Value != nil {
			tciCommandStr = *t.tci.Value
		}
		if err != nil {
			log.WithFields(logrus.Fields{
				"id":      t.ti.TaskID.Value,
				"task":    t.ti.Name,
				"error":   err.Error(),
				"command": tciCommandStr,
			}).
			Error("failed to run task")

			t.sendStatus(mesos.TASK_FAILED, err.Error())
			return
		}
		log.WithField("id", t.ti.TaskID.Value).
			WithField("task", t.ti.Name).
			Debug("task launched")

		go func() {
			_, errStdout = io.Copy(log.WithPrefix("task-stdout").WithField("task", t.ti.Name).WriterLevel(logrus.DebugLevel), stdoutIn)
		}()
		go func() {
			_, errStderr = io.Copy(log.WithPrefix("task-stderr").WithField("task", t.ti.Name).WriterLevel(logrus.ErrorLevel), stderrIn)
		}()

		log.WithFields(logrus.Fields{
			"controlPort": t.tci.ControlPort,
			"controlMode": t.tci.ControlMode.String(),
			"task":        t.ti.Name,
			"id":          t.ti.TaskID.Value,
			"path":        taskCmd.Path,
			"argv":        "[ " + strings.Join(taskCmd.Args, ", ") + " ]",
			"argc":        len(taskCmd.Args),
		}).
		Debug("starting gRPC client")

		controlTransport := executorcmd.ProtobufTransport
		for _, v := range taskCmd.Args {
			if strings.Contains(v, "-P OCClite") {
				controlTransport = executorcmd.JsonTransport
				break
			}
		}

		t.rpc = executorcmd.NewClient(t.tci.ControlPort, t.tci.ControlMode, controlTransport)
		if t.rpc == nil {
			err = errors.New("rpc client is nil")
			log.WithFields(logrus.Fields{
					"id":      t.ti.TaskID.Value,
					"task":    t.ti.Name,
					"error":   err.Error(),
					"command": tciCommandStr,
				}).
				Error("could not start gRPC client")

			t.sendStatus(mesos.TASK_FAILED, err.Error())
			return
		}
		t.rpc.TaskCmd = taskCmd

		elapsed := 0 * time.Second
		for {
			log.WithFields(logrus.Fields{
					"id":      t.ti.TaskID.Value,
					"task":    t.ti.Name,
					"command": tciCommandStr,
					"elapsed": elapsed.String(),
				}).
				Debug("polling task for IDLE state reached")

			response, err := t.rpc.GetState(context.TODO(), &pb.GetStateRequest{}, grpc.EmptyCallOption{})
			if err != nil {
				log.WithError(err).
					WithFields(logrus.Fields{
						"state": response.GetState(),
						"task":  t.ti.Name,
						"command": tciCommandStr,
					}).
					Info("cannot query task status")
			} else {
				log.WithFields(logrus.Fields{
						"state": response.GetState(),
						"task":  t.ti.Name,
						"command": tciCommandStr,
					}).
					Debug("task status queried")
				t.knownPid = int(response.GetPid())
			}
			// NOTE: we acquire the transitioner-dependent STANDBY equivalent state
			reachedState := t.rpc.FromDeviceState(response.GetState())

			if reachedState == "STANDBY" && err == nil {
				log.WithField("id", t.ti.TaskID.Value).
					WithField("task", t.ti.Name).
					WithField("command", tciCommandStr).
					Debug("task running and ready for control input")
				break
			} else if reachedState == "DONE" || reachedState == "ERROR" {
				// something went wrong, the device moved to DONE or ERROR on startup
				_ = syscall.Kill(t.knownPid, syscall.SIGKILL)

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
					WithField("command", tciCommandStr).
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
		taskMessage := event.NewAnnounceTaskPIDEvent(t.ti.TaskID.GetValue(), int32(t.knownPid))
		jsonEvent, err := json.Marshal(taskMessage)
		if err != nil {
			log.WithError(err).Warning("error marshaling message from task")
		} else {
			t.sendMessage(jsonEvent)
		}
		

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
					log.WithError(err).
						WithField("errorType", reflect.TypeOf(err)).
						Warning("error receiving event from task")
					if status.Code(err) == codes.Unavailable {
						break
					}
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
		log.WithFields(logrus.Fields{
			"id":      t.ti.TaskID.Value,
			"task":    t.ti.Name,
			"command": tciCommandStr,
		}).Debug("task done, preparing final update")

		pendingState := mesos.TASK_FINISHED
		if err != nil {
			log.WithFields(logrus.Fields{
				"id":    t.ti.TaskID.Value,
				"task":  t.ti.Name,
				"command": tciCommandStr,
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
				"command":   tciCommandStr,
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
	var(
		pid = 0
		reachedState = "UNKNOWN" // FIXME: should be LAUNCHING or similar
	)
	response, err := t.rpc.GetState(context.TODO(), &pb.GetStateRequest{}, grpc.EmptyCallOption{})
	if err == nil { // we successfully got the state from the task
		log.WithField("nativeState", response.GetState()).WithField("taskId", t.ti.GetTaskID()).Debug("task status queried for upcoming soft kill")

		// NOTE: we acquire the transitioner-dependent STANDBY equivalent state
		reachedState = t.rpc.FromDeviceState(response.GetState())

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
						AgentId:    mesos.AgentID{},    // AgentID and ExecutorID can stay empty because it's a
						ExecutorId: mesos.ExecutorID{}, // local transition
						TaskId:     t.ti.GetTaskID(),
					},
				},
				reachedState,
				evt,
				destination,
				nil,
			)
			return
		}

		for reachedState != "DONE" &&
			reachedState != "ERROR" {
			cmd := nextTransition(reachedState)
			log.WithFields(logrus.Fields{
				"evt":        cmd.Event,
				"src":        cmd.Source,
				"dst":        cmd.Destination,
				"targetList": cmd.TargetList,
			}).
				Debug("state DONE not reached, about to commit transition")

			// Call cmd.Commit() asynchronous
			commitDone := make(chan *CommitResponse)
			go func() {
				var cr CommitResponse
				cr.newState, cr.transitionError = cmd.Commit()
				commitDone <- &cr
			}()

			// Set timeout cause OCC is locking up so killing is not possible.
			var commitResponse *CommitResponse
			select {
			case commitResponse = <-commitDone:
			case <-time.After(15 * time.Second):
				log.Error("deadline exceeded")
			}
			// timeout we should break
			if commitResponse == nil {
				break
			}

			log.WithField("newState", commitResponse.newState).
				WithError(commitResponse.transitionError).
				Debug("transition committed")
			if commitResponse.transitionError != nil || len(cmd.Event) == 0 {
				log.WithError(commitResponse.transitionError).Error("cannot gracefully end task")
				break
			}
			reachedState = commitResponse.newState
		}

		log.Debug("end transition loop done")
		pid = int(response.GetPid())
		if pid == 0 {
			// t.knownPid must be valid because GetState was sure to have been successful in the past
			pid = t.knownPid
		}
	} else {
		log.WithError(err).WithField("taskId", t.ti.GetTaskID()).Warn("cannot query task status for graceful process termination")
		pid = t.knownPid
		if pid == 0 {
			// The pid was never known through a successful `GetState` in the lifetime
			// of this process, so we must rely on the PGID of the containing shell
			pid = -t.rpc.TaskCmd.Process.Pid
			// When killing the containing shell we must use syscall.Kill with a negative PID, in order to kill all
			// children which were assigned the same PGID at launch.

			// Otherwise, when we kill the child process directly, it should also
			// terminate the shell that is wrapping the command, so we avoid using
			// negative PID is all other cases in order to allow FairMQ cleanup to
			// run.
			log.WithError(err).WithField("taskId", t.ti.GetTaskID()).Warn("task PID not known from task, using containing shell PGID")
		}
	}

	_ = t.rpc.Close()
	t.rpc = nil

	if reachedState == "DONE" {
		log.Debug("task exited correctly")
		t.pendingFinalTaskStateCh <- mesos.TASK_FINISHED
	} else { // something went wrong
		log.Debug("task killed")
		t.pendingFinalTaskStateCh <- mesos.TASK_KILLED
	}

	killErrCh := make(chan error)
	go func() {
		err := syscall.Kill(pid, syscall.SIGTERM)
		if err != nil {
			log.WithError(err).
				WithField("taskId", t.ti.GetTaskID()).
				Warning("could not gracefully kill task")
		}
		killErrCh <- err
	}()

	// Set a small timeout to SIGTERM if SIGTERM fails or timeout passes,
	// we perform a SIGKILL.
	select {
	case killErr := <- killErrCh:
		if killErr == nil {
			time.Sleep(KILL_TIMEOUT)
			if pidExists(pid) {
				// SIGINT for the "Waiting for graceful device shutdown. 
				// Hit Ctrl-C again to abort immediately" message.
				killErr = syscall.Kill(pid, syscall.SIGINT)
				if killErr != nil {
					log.WithError(killErr).
						WithField("taskId", t.ti.GetTaskID()).
						Warning("could not gracefully kill task")
				}
			}
			time.Sleep(KILL_TIMEOUT)
			if !pidExists(pid) {
				return killErr
			}
		}
	case <-time.After(TRANSITION_TIMEOUT):
	}

	killErr := syscall.Kill(pid, syscall.SIGKILL)
	if killErr != nil {
		log.WithError(killErr).
			WithField("taskId", t.ti.GetTaskID()).
			Warning("could not kill task")
	}

	return killErr
}