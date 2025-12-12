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
	"os/exec"
	"reflect"
	"strings"
	"syscall"
	"time"

	"github.com/AliceO2Group/Control/common/logger"
	"github.com/AliceO2Group/Control/executor/executorutil"

	"github.com/AliceO2Group/Control/common/event"
	"github.com/AliceO2Group/Control/common/logger/infologger"
	"github.com/AliceO2Group/Control/common/utils"
	"github.com/AliceO2Group/Control/core/controlcommands"
	"github.com/AliceO2Group/Control/executor/executorcmd"
	pb "github.com/AliceO2Group/Control/executor/protos"
	mesos "github.com/mesos/mesos-go/api/v1/lib"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	DONE_TIMEOUT            = 1 * time.Second
	SIGTERM_TIMEOUT         = 2 * time.Second
	SIGINT_TIMEOUT          = 3 * time.Second
	KILL_TRANSITION_TIMEOUT = 5 * time.Second // to be on the safe side, readout might need up to 5s to go from RUNNING to DONE
	TRANSITION_TIMEOUT      = 10 * time.Second
)

type ControllableTask struct {
	taskBase
	rpc                     *executorcmd.RpcClient
	pendingFinalTaskStateCh chan mesos.TaskState
	knownPid                int
}

type CommitResponse struct {
	newState        string
	transitionError error
}

func (t *ControllableTask) Launch() error {
	defaultLogFields := logrus.Fields{
		"taskId":    t.ti.TaskID.GetValue(),
		"taskName":  t.ti.Name,
		"partition": t.knownEnvironmentId.String(),
		"detector":  t.knownDetector,
	}

	log.WithFields(defaultLogFields).
		WithField(infologger.Level, infologger.IL_Devel).
		Debug("executor.ControllableTask.Launch begin")

	launchStartTime := time.Now()

	defer utils.TimeTrack(launchStartTime,
		"executor.ControllableTask.Launch",
		log.WithFields(defaultLogFields).
			WithField(infologger.Level, infologger.IL_Devel))

	t.pendingFinalTaskStateCh = make(chan mesos.TaskState, 1) // we use this to receive a pending status update if the task was killed
	taskCmd, err := prepareTaskCmd(t.Tci)
	if err != nil {
		msg := "cannot build task command"
		log.WithFields(defaultLogFields).
			WithError(err).
			Error(msg)

		t.sendStatus(t.knownEnvironmentId, mesos.TASK_FAILED, msg+": "+err.Error())
		return err
	}

	log.WithFields(defaultLogFields).
		WithField("payload", string(t.ti.GetData()[:])).
		WithField(infologger.Level, infologger.IL_Devel).
		Debug("starting task asynchronously")

	// We fork out into a goroutine for the actual process management.
	// Control returns to the event loop which can safely access *internalState.
	// Anything in the following goroutine must not touch *internalState, except
	// via channels.
	go func() {
		t.doLaunchTask(taskCmd, launchStartTime)
	}()

	log.WithFields(defaultLogFields).
		WithField(infologger.Level, infologger.IL_Devel).
		Debug("gRPC client starting, handler forked: executor.ControllableTask.Launch end")
	return nil
}

func (t *ControllableTask) doLaunchTask(taskCmd *exec.Cmd, launchStartTime time.Time) {
	defaultLogFields := logrus.Fields{
		"taskId":    t.ti.TaskID.GetValue(),
		"taskName":  t.ti.Name,
		"partition": t.knownEnvironmentId.String(),
		"detector":  t.knownDetector,
	}
	truncatedCmd := executorutil.TruncateCommandBeforeTheLastPipe(t.Tci.GetValue(), 500)

	log.WithFields(defaultLogFields).
		WithField("command", truncatedCmd).
		WithField(infologger.Level, infologger.IL_Devel).
		Debug("executor.ControllableTask.Launch.async begin")

	// Set up pipes for controlled process. They have to be retrieved before starting the task.
	stdoutIn, _ := taskCmd.StdoutPipe()
	stderrIn, _ := taskCmd.StderrPipe()

	err := taskCmd.Start()
	if err != nil {
		log.WithFields(defaultLogFields).
			WithField("command", truncatedCmd).
			WithError(err).
			Error("failed to run task")

		t.sendStatus(t.knownEnvironmentId, mesos.TASK_FAILED, err.Error())
		// no need to close IO pipes, Cmd.Start does it on failure
		return
	}
	log.WithFields(defaultLogFields).Debug("task launched")

	utils.TimeTrack(launchStartTime,
		"executor.ControllableTask.Launch.async: Launch begin to taskCmd.Start() complete",
		log.WithFields(defaultLogFields).
			WithField(infologger.Level, infologger.IL_Devel))

	t.initTaskStdLogging(stdoutIn, stderrIn)

	log.WithFields(defaultLogFields).
		WithFields(logrus.Fields{
			"controlPort":    t.Tci.ControlPort,
			"controlMode":    t.Tci.ControlMode.String(),
			"path":           taskCmd.Path,
			"argv":           "[ " + strings.Join(taskCmd.Args, ", ") + " ]",
			"argc":           len(taskCmd.Args),
			infologger.Level: infologger.IL_Devel,
		}).Debug("starting gRPC client")

	controlTransport := executorcmd.ProtobufTransport
	for _, v := range taskCmd.Args {
		if strings.Contains(v, "-P OCClite") {
			controlTransport = executorcmd.JsonTransport
			break
		}
	}

	rpcDialStartTime := time.Now()
	t.rpc = executorcmd.NewClient(
		t.Tci.ControlPort,
		t.Tci.ControlMode,
		controlTransport,
		log.WithPrefix("executorcmd").
			WithFields(defaultLogFields),
	)
	if t.rpc == nil {
		err = errors.New("rpc client is nil")
		log.WithFields(defaultLogFields).
			WithField("command", truncatedCmd).
			WithError(err).
			WithField(infologger.Level, infologger.IL_Devel).
			Error("could not start gRPC client")

		t.sendStatus(t.knownEnvironmentId, mesos.TASK_FAILED, err.Error())
		_ = t.doTermIntKill(-taskCmd.Process.Pid)
		return
	}
	t.rpc.TaskCmd = taskCmd

	utils.TimeTrack(launchStartTime,
		"executor.ControllableTask.Launch.async: Launch begin to gRPC client dial success",
		log.WithFields(defaultLogFields).
			WithField(infologger.Level, infologger.IL_Devel))

	utils.TimeTrack(rpcDialStartTime,
		"executor.ControllableTask.Launch.async: gRPC client dial begin to gRPC client dial success",
		log.WithFields(defaultLogFields).
			WithField(infologger.Level, infologger.IL_Devel))

	err = t.pollTaskForStandbyState()
	if err != nil {
		log.WithFields(defaultLogFields).
			WithField(infologger.Level, infologger.IL_Support).
			WithError(err).
			Error("failed to poll task for standby state")

		t.sendStatus(t.knownEnvironmentId, mesos.TASK_FAILED, err.Error())

		t.cleanupFailedTask(taskCmd)
		return
	}

	utils.TimeTrack(launchStartTime,
		"executor.ControllableTask.Launch.async: Launch begin to gRPC state polling done",
		log.WithFields(defaultLogFields).
			WithField(infologger.Level, infologger.IL_Devel))

	// Set up event stream from task
	esc, err := t.rpc.EventStream(context.TODO(), &pb.EventStreamRequest{}, grpc.EmptyCallOption{})
	if err != nil {
		log.WithFields(defaultLogFields).
			WithError(err).
			Error("cannot set up event stream from task")
		t.sendStatus(t.knownEnvironmentId, mesos.TASK_FAILED, err.Error())
		_ = t.rpc.Close()
		t.rpc = nil
		// fixme: why don't we kill the task in this error case, but we do in others?
		return
	}

	// send RUNNING
	t.sendStatus(t.knownEnvironmentId, mesos.TASK_RUNNING, "")
	taskMessage := event.NewAnnounceTaskPIDEvent(t.ti.TaskID.GetValue(), int32(t.knownPid))
	taskMessage.SetLabels(map[string]string{"detector": t.knownDetector, "environmentId": t.knownEnvironmentId.String()})

	jsonEvent, err := json.Marshal(taskMessage)
	if err != nil {
		log.WithFields(defaultLogFields).
			WithError(err).
			Warning("error marshaling message")
	} else {
		t.sendMessage(jsonEvent)
		log.WithFields(defaultLogFields).
			WithField("command", truncatedCmd).
			WithField(infologger.Level, infologger.IL_Devel).
			Debug("executor.ControllableTask.Launch.async: TASK_RUNNING sent back to core")
	}

	// Process events from task in yet another goroutine
	go func() {
		t.processEventsFromTask(esc)
	}()

	err = taskCmd.Wait()
	// ^ when this unblocks, the task is done
	log.WithFields(defaultLogFields).
		WithField("command", truncatedCmd).
		WithField(infologger.Level, infologger.IL_Devel).
		Debug("task done (taskCmd.Wait unblocks), preparing final update")

	pendingState := mesos.TASK_FINISHED
	if err != nil {
		taskClassName, _ := utils.ExtractTaskClassName(t.ti.Name)
		log.WithFields(defaultLogFields).
			WithField(infologger.Level, infologger.IL_Ops).
			Errorf("task '%s' terminated with error: %s", utils.TrimJitPrefix(taskClassName), err.Error())
		log.WithFields(defaultLogFields).
			WithField("command", truncatedCmd).
			WithField(infologger.Level, infologger.IL_Devel).
			WithError(err).
			Error("task terminated with error (details):")
		pendingState = mesos.TASK_FAILED
	}

	select {
	case pending := <-t.pendingFinalTaskStateCh:
		pendingState = pending
	default:
	}

	if t.rpc != nil {
		_ = t.rpc.Close() // NOTE: might return non-nil error, but we don't care much
		log.WithFields(defaultLogFields).
			Debug("rpc client closed")
		t.rpc = nil
		log.WithFields(defaultLogFields).
			Debug("rpc client removed")
	}

	t.sendStatus(t.knownEnvironmentId, pendingState, "")
	return
}

func (t *ControllableTask) cleanupFailedTask(taskCmd *exec.Cmd) {

	defaultLogFields := logrus.Fields{
		"taskId":    t.ti.TaskID.GetValue(),
		"taskName":  t.ti.Name,
		"partition": t.knownEnvironmentId.String(),
		"detector":  t.knownDetector,
	}

	if taskCmd.Process == nil {
		// task never started or was already terminated
		return
	}

	if t.rpc != nil {
		_ = t.rpc.Close()
		t.rpc = nil
	}

	pid := t.knownPid
	if pid == 0 {
		// The pid was never known through a successful `GetState` in the lifetime
		// of this process, so we must rely on the PGID of the containing shell
		pid = -taskCmd.Process.Pid
	}

	_ = t.doTermIntKill(-taskCmd.Process.Pid)

	err := taskCmd.Wait()
	if err != nil {
		log.WithFields(defaultLogFields).
			WithField(infologger.Level, infologger.IL_Support).
			WithError(err).
			Warning("task terminated and exited with error")
	} else {
		log.WithFields(defaultLogFields).
			WithField(infologger.Level, infologger.IL_Support).
			Debug("task terminated")
	}
}

func (t *ControllableTask) initTaskStdLogging(stdoutIn io.ReadCloser, stderrIn io.ReadCloser) {
	defaultLogFields := logrus.Fields{
		"taskId":    t.ti.TaskID.GetValue(),
		"taskName":  t.ti.Name,
		"partition": t.knownEnvironmentId.String(),
		"detector":  t.knownDetector,
	}

	if t.Tci.Stdout == nil {
		none := "none"
		t.Tci.Stdout = &none
	}
	if t.Tci.Stderr == nil {
		none := "none"
		t.Tci.Stderr = &none
	}

	go func() {
		var errStdout error
		switch *t.Tci.Stdout {
		case "stdout":
			entry := log.WithPrefix("task-stdout").
				WithFields(defaultLogFields).
				WithField(infologger.Level, infologger.IL_Support).
				WithField("nohooks", true)
			writer := &logger.SafeLogrusWriter{
				Entry:     entry,
				PrintFunc: entry.Debug,
			}
			_, errStdout = io.Copy(writer, stdoutIn)
			writer.Flush()
		case "all":
			entry := log.WithPrefix("task-stdout").
				WithFields(defaultLogFields).
				WithField(infologger.Level, infologger.IL_Support)
			writer := &logger.SafeLogrusWriter{
				Entry:     entry,
				PrintFunc: entry.Debug,
			}
			_, errStdout = io.Copy(writer, stdoutIn)
			writer.Flush()
		default:
			_, errStdout = io.Copy(io.Discard, stdoutIn)
		}
		if errStdout != nil {
			log.WithFields(defaultLogFields).
				WithError(errStdout).
				WithField(infologger.Level, infologger.IL_Devel).
				Warning("failed to capture stdout of task")
		}
	}()

	go func() {
		var errStderr error
		switch *t.Tci.Stderr {
		case "stdout":
			entry := log.WithPrefix("task-stderr").
				WithFields(defaultLogFields).
				WithField(infologger.Level, infologger.IL_Support).
				WithField("nohooks", true)
			writer := &logger.SafeLogrusWriter{
				Entry:     entry,
				PrintFunc: entry.Warn,
			}
			_, errStderr = io.Copy(writer, stderrIn)
			writer.Flush()
		case "all":
			entry := log.WithPrefix("task-stderr").
				WithFields(defaultLogFields).
				WithField(infologger.Level, infologger.IL_Support)
			writer := &logger.SafeLogrusWriter{
				Entry:     entry,
				PrintFunc: entry.Warn,
			}
			_, errStderr = io.Copy(writer, stderrIn)
			writer.Flush()
		default:
			_, errStderr = io.Copy(io.Discard, stderrIn)
		}
		if errStderr != nil {
			log.WithFields(defaultLogFields).
				WithError(errStderr).
				WithField(infologger.Level, infologger.IL_Devel).
				Warning("failed to capture stderr of task")
		}
	}()
}

func (t *ControllableTask) pollTaskForStandbyState() error {
	defaultLogFields := logrus.Fields{
		"taskId":    t.ti.TaskID.GetValue(),
		"taskName":  t.ti.Name,
		"partition": t.knownEnvironmentId.String(),
		"detector":  t.knownDetector,
	}
	statePollingStartTime := time.Now()
	elapsed := 0 * time.Second
	for {
		log.WithFields(defaultLogFields).
			WithField("elapsed", elapsed.String()).
			WithField(infologger.Level, infologger.IL_Devel).
			Debug("polling task for STANDBY state reached")

		response, err := t.rpc.GetState(context.TODO(), &pb.GetStateRequest{}, grpc.EmptyCallOption{})
		if err != nil {
			log.WithError(err).
				WithFields(defaultLogFields).
				WithField("state", response.GetState()).
				Info("cannot query task status")
		} else {
			log.WithFields(defaultLogFields).
				WithField("state", response.GetState()).
				WithField(infologger.Level, infologger.IL_Devel).
				Debug("task status queried")
			t.knownPid = int(response.GetPid())
		}
		// NOTE: we acquire the transitioner-dependent STANDBY equivalent state
		// fixme: that's a possible nil access there, because we do not "continue" on error
		reachedState := t.rpc.FromDeviceState(response.GetState())

		if reachedState == "STANDBY" && err == nil {
			log.WithFields(defaultLogFields).
				WithField(infologger.Level, infologger.IL_Devel).
				Debug("task running and ready for control input")
			break
		} else if reachedState == "DONE" || reachedState == "ERROR" {
			// something went wrong, the device moved to DONE or ERROR on startup
			return errors.New("task reached wrong state on startup")
		} else if elapsed >= startupTimeout {
			return errors.New("timeout while trying to poll task")
		} else {
			log.WithFields(defaultLogFields).
				Debugf("task not ready yet, waiting %s", startupPollingInterval.String())
			time.Sleep(startupPollingInterval)
			elapsed += startupPollingInterval
		}
	}

	utils.TimeTrack(statePollingStartTime,
		"executor.ControllableTask.Launch.async: gRPC state polling begin to gRPC state polling done",
		log.WithFields(defaultLogFields).
			WithField(infologger.Level, infologger.IL_Devel))
	return nil
}

func (t *ControllableTask) processEventsFromTask(esc pb.Occ_EventStreamClient) {
	defaultLogFields := logrus.Fields{
		"taskId":    t.ti.TaskID.GetValue(),
		"taskName":  t.ti.Name,
		"partition": t.knownEnvironmentId.String(),
		"detector":  t.knownDetector,
	}
	deo := event.DeviceEventOrigin{
		AgentId:    t.ti.AgentID,
		ExecutorId: t.ti.GetExecutor().ExecutorID,
		TaskId:     t.ti.TaskID,
	}

	for {
		if t.rpc == nil {
			log.WithFields(defaultLogFields).
				Debug("event stream done")
			break
		}
		esr, err := esc.Recv()
		if err == io.EOF {
			log.WithFields(defaultLogFields).
				WithError(err).
				Debug("event stream EOF")
			break
		}
		if err != nil {
			log.WithFields(defaultLogFields).
				WithField("errorType", reflect.TypeOf(err)).
				WithField(infologger.Level, infologger.IL_Devel).
				WithError(err).
				Warning("error receiving event")
			if status.Code(err) == codes.Unavailable {
				break
			}
			// fixme: we also get codes.Canceled sometimes, it's probably OK and we should not complain
			continue
		}
		ev := esr.GetEvent()

		deviceEvent := event.NewDeviceEvent(deo, ev.GetType())
		if deviceEvent == nil {
			log.WithFields(defaultLogFields).
				Debug("nil DeviceEvent received (NULL_DEVICE_EVENT) - closing stream")
			break
		} else {
			if deviceEvent.GetType() == pb.DeviceEventType_END_OF_STREAM {
				log.WithFields(defaultLogFields).
					WithField("taskPid", t.knownPid).
					Debug("END_OF_STREAM DeviceEvent received - notifying environment")
			} else if ev.GetType() == pb.DeviceEventType_TASK_INTERNAL_ERROR {
				log.WithFields(defaultLogFields).
					WithField("taskPid", t.knownPid).
					WithField(infologger.Level, infologger.IL_Support).
					Warningf("task transitioned to ERROR on its own - notifying environment")
			}
		}
		deviceEvent.SetLabels(map[string]string{"detector": t.knownDetector, "environmentId": t.knownEnvironmentId.String()})

		t.sendDeviceEvent(t.knownEnvironmentId, deviceEvent)
	}
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
	defaultLogFields := logrus.Fields{
		"taskId":    t.ti.TaskID.GetValue(),
		"taskName":  t.ti.Name,
		"partition": t.knownEnvironmentId.String(),
		"detector":  t.knownDetector,
	}

	var (
		pid          = 0
		reachedState = "UNKNOWN" // FIXME: should be LAUNCHING or similar
	)
	cxt, cancel := context.WithTimeout(context.Background(), KILL_TRANSITION_TIMEOUT)
	defer cancel()
	response, err := t.rpc.GetState(cxt, &pb.GetStateRequest{}, grpc.EmptyCallOption{})
	if err == nil { // we successfully got the state from the task
		log.WithFields(defaultLogFields).
			WithField("nativeState", response.GetState()).
			WithField(infologger.Level, infologger.IL_Devel).
			Debug("task status queried for upcoming soft kill")

		// NOTE: we acquire the transitioner-dependent STANDBY equivalent state
		reachedState = t.rpc.FromDeviceState(response.GetState())

		nextTransition := func(currentState string) (exc *executorcmd.ExecutorCommand_Transition) {
			var evt, destination string
			switch currentState {
			case "RUNNING":
				evt = "STOP"
				destination = "CONFIGURED"
			case "CONFIGURED":
				evt = "RESET"
				destination = "STANDBY"
			case "ERROR":
				evt = "EXIT"
				destination = "DONE"
			case "STANDBY":
				evt = "EXIT"
				destination = "DONE"
			}

			exc = executorcmd.NewLocalExecutorCommand_Transition(
				t.rpc.Transitioner,
				t.knownEnvironmentId,
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

		for reachedState != "DONE" {
			cmd := nextTransition(reachedState)
			log.WithFields(defaultLogFields).
				WithFields(logrus.Fields{
					"evt":            cmd.Event,
					"src":            cmd.Source,
					"dst":            cmd.Destination,
					"targetList":     cmd.TargetList,
					infologger.Level: infologger.IL_Devel,
				}).
				Debug("state DONE not reached, about to commit transition")

			// Call cmd.Commit() asynchronous with buffered channel so it doesn't get stuck in a case of timeout
			commitDone := make(chan *CommitResponse, 1)
			go func() {
				var cr CommitResponse
				cr.newState, cr.transitionError = cmd.Commit()
				commitDone <- &cr
			}()

			// Set timeout cause OCC is locking up so killing is not possible.
			var commitResponse *CommitResponse
			select {
			case commitResponse = <-commitDone:
			case <-time.After(KILL_TRANSITION_TIMEOUT):
				log.WithFields(defaultLogFields).
					WithField(infologger.Level, infologger.IL_Devel).
					Warn("teardown transition sequence timed out")
			}
			// timeout we should break
			if commitResponse == nil {
				break
			}

			log.WithFields(defaultLogFields).
				WithField("newState", commitResponse.newState).
				WithError(commitResponse.transitionError).
				WithField(infologger.Level, infologger.IL_Devel).
				Debug("transition committed")
			if commitResponse.transitionError != nil || len(cmd.Event) == 0 {
				log.WithFields(defaultLogFields).
					WithError(commitResponse.transitionError).
					WithField(infologger.Level, infologger.IL_Devel).
					Warn("teardown transition sequence error")
				break
			}
			reachedState = commitResponse.newState
		}

		log.WithFields(defaultLogFields).
			WithField(infologger.Level, infologger.IL_Devel).
			Debug("teardown transition sequence done")
		pid = int(response.GetPid())
		if pid == 0 {
			// t.knownPid must be valid because GetState was sure to have been successful in the past
			pid = t.knownPid
		}
	} else {
		// If GetState didn't succeed during this Kill code path, but might still have
		// at some earlier point during the lifetime of this task.
		// Either way, we might or might not have the true PID.
		log.WithFields(defaultLogFields).
			WithError(err).
			Warn("cannot query task status for graceful process termination")
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
			log.WithFields(defaultLogFields).
				WithError(err).
				Warn("task PID not known from task, using containing shell PGID")
		}
	}

	_ = t.rpc.Close()
	t.rpc = nil

	if reachedState == "DONE" {
		log.WithFields(defaultLogFields).
			Debugf("task reached DONE, will wait %.1fs before terminating it", DONE_TIMEOUT.Seconds())
		t.pendingFinalTaskStateCh <- mesos.TASK_FINISHED
		time.Sleep(DONE_TIMEOUT)
	} else { // something went wrong
		log.WithFields(defaultLogFields).
			Debug("task died already or will be killed soon")
		t.pendingFinalTaskStateCh <- mesos.TASK_KILLED
	}

	if pidExists(pid) {
		return t.doTermIntKill(pid)
	} else {
		log.WithFields(defaultLogFields).
			Debugf("task terminated on its own")
		return nil
	}
}

func (t *ControllableTask) doKill9(pid int) error {
	defaultLogFields := logrus.Fields{
		"taskId":    t.ti.TaskID.GetValue(),
		"taskName":  t.ti.Name,
		"partition": t.knownEnvironmentId.String(),
		"detector":  t.knownDetector,
	}

	log.WithFields(defaultLogFields).
		Debug("sending SIGKILL (9) to task")
	killErr := syscall.Kill(pid, syscall.SIGKILL)
	if killErr != nil {
		log.WithFields(defaultLogFields).
			WithError(killErr).
			Warning("task SIGKILL failed")
	}

	return killErr
}

func (t *ControllableTask) doTermIntKill(pid int) error {
	defaultLogFields := logrus.Fields{
		"taskId":    t.ti.TaskID.GetValue(),
		"taskName":  t.ti.Name,
		"partition": t.knownEnvironmentId.String(),
		"detector":  t.knownDetector,
	}

	killErrCh := make(chan error)
	go func() {
		log.WithFields(defaultLogFields).
			Debug("sending SIGTERM (15) to task")
		err := syscall.Kill(pid, syscall.SIGTERM)
		if err != nil {
			log.WithFields(defaultLogFields).
				WithError(err).
				Warning("task SIGTERM failed")
		}
		killErrCh <- err
	}()

	// Set a small timeout to SIGTERM if SIGTERM fails or timeout passes,
	// we perform a SIGKILL.
	select {
	case killErr := <-killErrCh:
		if killErr == nil {
			time.Sleep(SIGTERM_TIMEOUT) // Waiting for the SIGTERM to kick in
			if pidExists(pid) {
				// SIGINT for the "Waiting for graceful device shutdown.
				// Hit Ctrl-C again to abort immediately" message.
				log.WithFields(defaultLogFields).
					Debug("sending SIGINT (2) to task")
				killErr = syscall.Kill(pid, syscall.SIGINT)
				if killErr != nil {
					log.WithFields(defaultLogFields).
						WithError(killErr).
						Warning("task SIGINT failed")
				}
				time.Sleep(SIGINT_TIMEOUT)
			}
		}
		if !pidExists(pid) {
			return killErr
		}
	case <-time.After(SIGTERM_TIMEOUT + SIGINT_TIMEOUT):
	}

	return t.doKill9(pid)
}
