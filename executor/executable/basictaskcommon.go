/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2020 CERN and copyright holders of ALICE O².
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
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os/exec"
	"syscall"

	"github.com/AliceO2Group/Control/common/event"
	"github.com/AliceO2Group/Control/core/controlcommands"
	"github.com/AliceO2Group/Control/executor/executorcmd"
	"github.com/AliceO2Group/Control/executor/executorcmd/transitioner"
	pb "github.com/AliceO2Group/Control/executor/protos"
	mesos "github.com/mesos/mesos-go/api/v1/lib"
	"github.com/sirupsen/logrus"
)

type basicTaskBase struct {
	taskBase
	taskCmd *exec.Cmd
	transitioner transitioner.Transitioner
	pendingFinalTaskStateCh chan mesos.TaskState
}

func (t *basicTaskBase) startBasicTask() (err error) {
	// Set up pipes for controlled process
	var errStdout, errStderr error
	var stdoutBuf, stderrBuf bytes.Buffer

	stdoutLog := log.WithPrefix("task-stdout").WithField("task", t.ti.Name).WriterLevel(logrus.DebugLevel)
	stderrLog := log.WithPrefix("task-stderr").WithField("task", t.ti.Name).WriterLevel(logrus.ErrorLevel)

	// Each of these multiwriters will push incoming lines to a buffer as well as the logger
	stdout := io.MultiWriter(stdoutLog, &stdoutBuf)
	stderr := io.MultiWriter(stderrLog, &stderrBuf)

	stdoutIn, _ := t.taskCmd.StdoutPipe()
	stderrIn, _ := t.taskCmd.StderrPipe()

	err = t.taskCmd.Start()

	if err != nil {
		log.WithFields(logrus.Fields{
			"id":      t.ti.TaskID.Value,
			"task":    t.ti.Name,
			"error":   err,
			"command": *t.tci.Value,
		}).
			Error("failed to run basic task")

		return err
	}
	log.WithField("id", t.ti.TaskID.Value).
		WithField("task", t.ti.Name).
		Debug("basic task started")

	go func() {
		_, errStdout = io.Copy(stdout, stdoutIn)
	}()
	go func() {
		_, errStderr = io.Copy(stderr, stderrIn)
	}()

	go func() {
		err = t.taskCmd.Wait()
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

		// Can be -1 if the process was killed
		exitCode := t.taskCmd.ProcessState.ExitCode()
		processTerminatedOnItsOwn := true

		select {
		case pending := <- t.pendingFinalTaskStateCh:
			pendingState = pending
			processTerminatedOnItsOwn = false
		default:
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

		deo := event.DeviceEventOrigin{
			AgentId:    t.ti.AgentID,
			ExecutorId: t.ti.GetExecutor().ExecutorID,
			TaskId:     t.ti.TaskID,
		}
		deviceEvent := event.NewDeviceEvent(deo, pb.DeviceEventType_BASIC_TASK_TERMINATED)
		if btt, ok := deviceEvent.(*event.BasicTaskTerminated); ok {
			btt.VoluntaryTermination = processTerminatedOnItsOwn
			btt.ExitCode = exitCode
			btt.FinalMesosState = pendingState
			btt.Stderr = stderrBuf.String()
			btt.Stdout = stdoutBuf.String()
			t.sendDeviceEvent(btt)
		}
	}()

	return err
}

func (t *basicTaskBase) ensureBasicTaskKilled() (err error) {
	if t.taskCmd == nil {
		return nil
	}
	if t.taskCmd.ProcessState.Exited() {
		return nil
	}

	// Preparing to kill running task
	t.pendingFinalTaskStateCh <- mesos.TASK_KILLED

	// TODO: SIGTERM before SIGKILL

	pid := t.taskCmd.Process.Pid
	err = syscall.Kill(-pid, syscall.SIGKILL)
	if err != nil {
		log.WithError(err).
			WithField("taskId", t.ti.GetTaskID()).
			Warning("could not kill task")
	}

	return
}

func (t *basicTaskBase) doLaunch(transitionFunc transitioner.DoTransitionFunc) error {
	t.pendingFinalTaskStateCh = make(chan mesos.TaskState, 1) // we use this to receive a pending status update if the task was killed
	if t.taskCmd != nil {
		return errors.New("bad internal state for basic task command")
	}

	var err error
	t.taskCmd, err = prepareTaskCmd(t.tci)
	if err != nil {
		msg := "cannot build task command"
		log.WithFields(logrus.Fields{
			"id":      t.ti.TaskID.Value,
			"task":    t.ti.Name,
			"error":   err,
		}).
			Error(msg)
		return err
	}
	if t.taskCmd == nil {
		return errors.New("could not instantiate basic task command")
	}

	t.transitioner = transitioner.NewTransitioner(t.tci.ControlMode, transitionFunc)
	log.WithField("payload", string(t.ti.GetData()[:])).
		WithField("task", t.ti.Name).
		Debug("basic task staged")

	go t.sendStatus(mesos.TASK_RUNNING, "")
	taskMessage := event.NewTaskMessage(t.ti.Name,t.ti.TaskID.GetValue(),int32(t.taskCmd.Process.Pid))
	jsonEvent, err := json.Marshal(taskMessage)
	if err != nil {
		log.WithError(err).Warning("error marshaling message from task")
	} else {
		t.sendMessage(jsonEvent)
	}

	return nil
}

func (t *basicTaskBase) UnmarshalTransition(data []byte) (cmd *executorcmd.ExecutorCommand_Transition, err error) {
	cmd = new(executorcmd.ExecutorCommand_Transition)

	cmd.Transitioner = t.transitioner
	err = json.Unmarshal(data, cmd)
	if err != nil {
		cmd = nil
	}
	return
}

func (t *basicTaskBase) Transition(cmd *executorcmd.ExecutorCommand_Transition) *controlcommands.MesosCommandResponse_Transition {
	newState, transitionError := cmd.Commit()

	response := cmd.PrepareResponse(transitionError, newState, t.ti.TaskID.Value)
	return response
}

func (t *basicTaskBase) Kill() error {
	if t.taskCmd != nil {
		t.taskCmd = nil
		log.Debug("exec.Cmd wrapper removed")
	}

	go t.sendStatus(mesos.TASK_FINISHED, "")
	return nil
}
