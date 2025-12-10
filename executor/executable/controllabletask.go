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
	log.WithFields(logrus.Fields{
		"taskId":    t.ti.TaskID.GetValue(),
		"taskName":  t.ti.Name,
		"level":     infologger.IL_Devel,
		"partition": t.knownEnvironmentId.String(),
		"detector":  t.knownDetector,
	}).Debug("executor.ControllableTask.Launch begin")

	launchStartTime := time.Now()

	defer utils.TimeTrack(launchStartTime,
		"executor.ControllableTask.Launch",
		log.WithFields(logrus.Fields{
			"taskId":    t.ti.TaskID.GetValue(),
			"taskName":  t.ti.Name,
			"level":     infologger.IL_Devel,
			"partition": t.knownEnvironmentId.String(),
			"detector":  t.knownDetector,
		}))

	t.pendingFinalTaskStateCh = make(chan mesos.TaskState, 1) // we use this to receive a pending status update if the task was killed
	taskCmd, err := prepareTaskCmd(t.Tci)
	if err != nil {
		msg := "cannot build task command"
		log.WithFields(logrus.Fields{
			"id":        t.ti.TaskID.Value,
			"task":      t.ti.Name,
			"error":     err,
			"partition": t.knownEnvironmentId.String(),
			"detector":  t.knownDetector,
		}).
			Error(msg)

		t.sendStatus(t.knownEnvironmentId, mesos.TASK_FAILED, msg+": "+err.Error())
		return err
	}

	log.WithField("payload", string(t.ti.GetData()[:])).
		WithField("task", t.ti.Name).
		WithField("level", infologger.IL_Devel).
		WithField("partition", t.knownEnvironmentId.String()).
		WithField("detector", t.knownDetector).
		Debug("starting task asynchronously")

	// We fork out into a goroutine for the actual process management.
	// Control returns to the event loop which can safely access *internalState.
	// Anything in the following goroutine must not touch *internalState, except
	// via channels.
	go func() {
		truncatedCmd := executorutil.TruncateCommandBeforeTheLastPipe(t.Tci.GetValue(), 500)
		log.WithFields(logrus.Fields{
			"cmd":       truncatedCmd,
			"taskId":    t.ti.TaskID.GetValue(),
			"taskName":  t.ti.Name,
			"level":     infologger.IL_Devel,
			"partition": t.knownEnvironmentId.String(),
			"detector":  t.knownDetector,
		}).Debug("executor.ControllableTask.Launch.async begin")

		// Set up pipes for controlled process
		var errStdout, errStderr error
		stdoutIn, _ := taskCmd.StdoutPipe()
		stderrIn, _ := taskCmd.StderrPipe()

		err = taskCmd.Start()
		if err != nil {
			log.WithFields(logrus.Fields{
				"id":        t.ti.TaskID.Value,
				"task":      t.ti.Name,
				"error":     err.Error(),
				"command":   truncatedCmd,
				"partition": t.knownEnvironmentId.String(),
				"detector":  t.knownDetector,
			}).
				Error("failed to run task")

			t.sendStatus(t.knownEnvironmentId, mesos.TASK_FAILED, err.Error())
			_ = t.doTermIntKill(-taskCmd.Process.Pid)
			return
		}
		log.WithField("id", t.ti.TaskID.Value).
			WithField("task", t.ti.Name).
			WithField("partition", t.knownEnvironmentId.String()).
			WithField("detector", t.knownDetector).
			Debug("task launched")

		utils.TimeTrack(launchStartTime,
			"executor.ControllableTask.Launch.async: Launch begin to taskCmd.Start() complete",
			log.WithField("partition", t.knownEnvironmentId.String()).
				WithField("detector", t.knownDetector).
				WithFields(logrus.Fields{
					"cmd":      truncatedCmd,
					"taskId":   t.ti.TaskID.GetValue(),
					"taskName": t.ti.Name,
					"level":    infologger.IL_Devel,
				}))

		if t.Tci.Stdout == nil {
			none := "none"
			t.Tci.Stdout = &none
		}
		if t.Tci.Stderr == nil {
			none := "none"
			t.Tci.Stderr = &none
		}

		switch *t.Tci.Stdout {
		case "stdout":
			go func() {
				entry := log.WithPrefix("task-stdout").
					WithField("level", infologger.IL_Support).
					WithField("partition", t.knownEnvironmentId.String()).
					WithField("detector", t.knownDetector).
					WithField("task", t.ti.Name).
					WithField("nohooks", true)
				writer := &logger.SafeLogrusWriter{
					Entry:     entry,
					PrintFunc: entry.Debug,
				}
				_, errStdout = io.Copy(writer, stdoutIn)
				writer.Flush()
			}()
		case "all":
			go func() {
				entry := log.WithPrefix("task-stdout").
					WithField("level", infologger.IL_Support).
					WithField("partition", t.knownEnvironmentId.String()).
					WithField("detector", t.knownDetector).
					WithField("task", t.ti.Name)
				writer := &logger.SafeLogrusWriter{
					Entry:     entry,
					PrintFunc: entry.Debug,
				}
				_, errStdout = io.Copy(writer, stdoutIn)
				writer.Flush()
			}()
		default:
			go func() {
				_, errStdout = io.Copy(io.Discard, stdoutIn)
			}()
		}

		switch *t.Tci.Stderr {
		case "stdout":
			go func() {
				entry := log.WithPrefix("task-stderr").
					WithField("level", infologger.IL_Support).
					WithField("partition", t.knownEnvironmentId.String()).
					WithField("detector", t.knownDetector).
					WithField("task", t.ti.Name).
					WithField("nohooks", true)
				writer := &logger.SafeLogrusWriter{
					Entry:     entry,
					PrintFunc: entry.Warn,
				}
				_, errStderr = io.Copy(writer, stderrIn)
				writer.Flush()
			}()
		case "all":
			go func() {
				entry := log.WithPrefix("task-stderr").
					WithField("level", infologger.IL_Support).
					WithField("partition", t.knownEnvironmentId.String()).
					WithField("detector", t.knownDetector).
					WithField("task", t.ti.Name)
				writer := &logger.SafeLogrusWriter{
					Entry:     entry,
					PrintFunc: entry.Warn,
				}
				_, errStderr = io.Copy(writer, stderrIn)
				writer.Flush()
			}()
		default:
			go func() {
				_, errStderr = io.Copy(io.Discard, stderrIn)
			}()
		}

		log.WithField("partition", t.knownEnvironmentId.String()).
			WithField("detector", t.knownDetector).
			WithFields(logrus.Fields{
				"controlPort": t.Tci.ControlPort,
				"controlMode": t.Tci.ControlMode.String(),
				"task":        t.ti.Name,
				"id":          t.ti.TaskID.Value,
				"path":        taskCmd.Path,
				"argv":        "[ " + strings.Join(taskCmd.Args, ", ") + " ]",
				"argc":        len(taskCmd.Args),
				"level":       infologger.IL_Devel,
			}).
			Debug("starting gRPC client")

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
				WithField("partition", t.knownEnvironmentId.String()).
				WithField("detector", t.knownDetector).
				WithFields(logrus.Fields{
					"id":      t.ti.TaskID.Value,
					"task":    t.ti.Name,
					"command": truncatedCmd,
				},
				),
		)
		if t.rpc == nil {
			err = errors.New("rpc client is nil")
			log.WithField("partition", t.knownEnvironmentId.String()).
				WithField("detector", t.knownDetector).
				WithFields(logrus.Fields{
					"id":      t.ti.TaskID.Value,
					"task":    t.ti.Name,
					"error":   err.Error(),
					"command": truncatedCmd,
				}).
				WithField("level", infologger.IL_Devel).
				Error("could not start gRPC client")

			t.sendStatus(t.knownEnvironmentId, mesos.TASK_FAILED, err.Error())
			_ = t.doTermIntKill(-taskCmd.Process.Pid)
			return
		}
		t.rpc.TaskCmd = taskCmd

		utils.TimeTrack(launchStartTime,
			"executor.ControllableTask.Launch.async: Launch begin to gRPC client dial success",
			log.WithField("partition", t.knownEnvironmentId.String()).
				WithField("detector", t.knownDetector).
				WithFields(logrus.Fields{
					"command":  truncatedCmd,
					"taskId":   t.ti.TaskID.GetValue(),
					"taskName": t.ti.Name,
					"level":    infologger.IL_Devel,
				}))

		utils.TimeTrack(rpcDialStartTime,
			"executor.ControllableTask.Launch.async: gRPC client dial begin to gRPC client dial success",
			log.WithField("partition", t.knownEnvironmentId.String()).
				WithField("detector", t.knownDetector).
				WithFields(logrus.Fields{
					"command":  truncatedCmd,
					"taskId":   t.ti.TaskID.GetValue(),
					"taskName": t.ti.Name,
					"level":    infologger.IL_Devel,
				}))

		statePollingStartTime := time.Now()
		elapsed := 0 * time.Second
		for {
			log.WithField("partition", t.knownEnvironmentId.String()).
				WithField("detector", t.knownDetector).
				WithFields(logrus.Fields{
					"id":      t.ti.TaskID.Value,
					"task":    t.ti.Name,
					"command": truncatedCmd,
					"elapsed": elapsed.String(),
					"level":   infologger.IL_Devel,
				}).
				Debug("polling task for IDLE state reached")

			response, err := t.rpc.GetState(context.TODO(), &pb.GetStateRequest{}, grpc.EmptyCallOption{})
			if err != nil {
				log.WithError(err).
					WithField("partition", t.knownEnvironmentId.String()).
					WithField("detector", t.knownDetector).
					WithFields(logrus.Fields{
						"state":   response.GetState(),
						"task":    t.ti.Name,
						"command": truncatedCmd,
					}).
					Info("cannot query task status")
			} else {
				log.WithField("partition", t.knownEnvironmentId.String()).
					WithField("detector", t.knownDetector).
					WithFields(logrus.Fields{
						"state":   response.GetState(),
						"task":    t.ti.Name,
						"command": truncatedCmd,
						"level":   infologger.IL_Devel,
					}).
					Debug("task status queried")
				t.knownPid = int(response.GetPid())
			}
			// NOTE: we acquire the transitioner-dependent STANDBY equivalent state
			reachedState := t.rpc.FromDeviceState(response.GetState())

			if reachedState == "STANDBY" && err == nil {
				log.WithField("partition", t.knownEnvironmentId.String()).
					WithField("detector", t.knownDetector).
					WithField("id", t.ti.TaskID.Value).
					WithField("task", t.ti.Name).
					WithField("command", truncatedCmd).
					WithField("level", infologger.IL_Devel).
					Debug("task running and ready for control input")
				break
			} else if reachedState == "DONE" || reachedState == "ERROR" {
				// something went wrong, the device moved to DONE or ERROR on startup
				pid := t.knownPid
				if pid == 0 {
					// The pid was never known through a successful `GetState` in the lifetime
					// of this process, so we must rely on the PGID of the containing shell
					pid = -t.rpc.TaskCmd.Process.Pid
				}
				log.WithField("partition", t.knownEnvironmentId.String()).
					WithField("detector", t.knownDetector).
					WithField("taskId", t.ti.Name).
					Debug("sending SIGKILL (9) to task")
				_ = syscall.Kill(pid, syscall.SIGKILL)
				_ = stdoutIn.Close()
				_ = stderrIn.Close()

				log.WithField("partition", t.knownEnvironmentId.String()).
					WithField("detector", t.knownDetector).
					WithField("task", t.ti.Name).Debug("task killed")
				t.sendStatus(t.knownEnvironmentId, mesos.TASK_FAILED, "task reached wrong state on startup")
				return
			} else if elapsed >= startupTimeout {
				err = errors.New("timeout while waiting for task startup")
				log.WithField("partition", t.knownEnvironmentId.String()).
					WithField("detector", t.knownDetector).
					WithField("task", t.ti.Name).Error(err.Error())
				t.sendStatus(t.knownEnvironmentId, mesos.TASK_FAILED, err.Error())
				_ = t.rpc.Close()
				t.rpc = nil

				_ = stdoutIn.Close()
				_ = stderrIn.Close()

				return
			} else {
				log.WithField("partition", t.knownEnvironmentId.String()).
					WithField("detector", t.knownDetector).
					WithField("task", t.ti.Name).
					WithField("command", truncatedCmd).
					Debugf("task not ready yet, waiting %s", startupPollingInterval.String())
				time.Sleep(startupPollingInterval)
				elapsed += startupPollingInterval
			}
		}

		utils.TimeTrack(launchStartTime,
			"executor.ControllableTask.Launch.async: Launch begin to gRPC state polling done",
			log.WithField("partition", t.knownEnvironmentId.String()).
				WithField("detector", t.knownDetector).
				WithFields(logrus.Fields{
					"command":  truncatedCmd,
					"taskId":   t.ti.TaskID.GetValue(),
					"taskName": t.ti.Name,
					"level":    infologger.IL_Devel,
				}))

		utils.TimeTrack(statePollingStartTime,
			"executor.ControllableTask.Launch.async: gRPC state polling begin to gRPC state polling done",
			log.WithField("partition", t.knownEnvironmentId.String()).
				WithField("detector", t.knownDetector).
				WithFields(logrus.Fields{
					"command":  truncatedCmd,
					"taskId":   t.ti.TaskID.GetValue(),
					"taskName": t.ti.Name,
					"level":    infologger.IL_Devel,
				}))

		// Set up event stream from task
		esc, err := t.rpc.EventStream(context.TODO(), &pb.EventStreamRequest{}, grpc.EmptyCallOption{})
		if err != nil {
			log.WithField("task", t.ti.Name).
				WithError(err).
				WithField("partition", t.knownEnvironmentId.String()).
				WithField("detector", t.knownDetector).
				Error("cannot set up event stream from task")
			t.sendStatus(t.knownEnvironmentId, mesos.TASK_FAILED, err.Error())
			_ = t.rpc.Close()
			t.rpc = nil
			return
		}

		// send RUNNING
		t.sendStatus(t.knownEnvironmentId, mesos.TASK_RUNNING, "")
		taskMessage := event.NewAnnounceTaskPIDEvent(t.ti.TaskID.GetValue(), int32(t.knownPid))
		taskMessage.SetLabels(map[string]string{"detector": t.knownDetector, "environmentId": t.knownEnvironmentId.String()})

		jsonEvent, err := json.Marshal(taskMessage)
		if err != nil {
			log.WithField("partition", t.knownEnvironmentId.String()).
				WithField("detector", t.knownDetector).
				WithField("taskId", t.ti.TaskID.GetValue()).
				WithField("taskName", t.ti.Name).
				WithError(err).
				Warning("error marshaling message")
		} else {
			t.sendMessage(jsonEvent)
			log.WithField("partition", t.knownEnvironmentId.String()).
				WithField("detector", t.knownDetector).
				WithFields(logrus.Fields{
					"command":  truncatedCmd,
					"taskId":   t.ti.TaskID.GetValue(),
					"taskName": t.ti.Name,
					"level":    infologger.IL_Devel,
				}).Debug("executor.ControllableTask.Launch.async: TASK_RUNNING sent back to core")
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
					log.WithField("partition", t.knownEnvironmentId.String()).
						WithField("detector", t.knownDetector).
						WithField("taskId", deo.TaskId.GetValue()).
						WithField("taskName", t.ti.Name).
						WithError(err).
						Debug("event stream done")
					break
				}
				esr, err := esc.Recv()
				if err == io.EOF {
					log.WithField("partition", t.knownEnvironmentId.String()).
						WithField("detector", t.knownDetector).
						WithField("taskId", deo.TaskId.GetValue()).
						WithField("taskName", t.ti.Name).
						WithError(err).
						Debug("event stream EOF")
					break
				}
				if err != nil {
					log.WithError(err).
						WithField("partition", t.knownEnvironmentId.String()).
						WithField("detector", t.knownDetector).
						WithField("errorType", reflect.TypeOf(err)).
						WithField("level", infologger.IL_Devel).
						WithField("taskId", deo.TaskId.GetValue()).
						WithField("taskName", t.ti.Name).
						Warning("error receiving event")
					if status.Code(err) == codes.Unavailable {
						break
					}
					continue
				}
				ev := esr.GetEvent()

				deviceEvent := event.NewDeviceEvent(deo, ev.GetType())
				if deviceEvent == nil {
					log.WithField("partition", t.knownEnvironmentId.String()).
						WithField("detector", t.knownDetector).
						WithField("taskId", deo.TaskId.GetValue()).
						WithField("taskName", t.ti.Name).
						Debug("nil DeviceEvent received (NULL_DEVICE_EVENT) - closing stream")
					break
				} else {
					taskId := deo.TaskId.Value

					if deviceEvent.GetType() == pb.DeviceEventType_END_OF_STREAM {
						log.WithField("partition", t.knownEnvironmentId.String()).
							WithField("detector", t.knownDetector).
							WithField("taskId", taskId).
							WithField("taskName", t.ti.Name).
							WithField("taskPid", t.knownPid).
							Debug("END_OF_STREAM DeviceEvent received - notifying environment")
					} else if ev.GetType() == pb.DeviceEventType_TASK_INTERNAL_ERROR {
						log.WithField("partition", t.knownEnvironmentId.String()).
							WithField("detector", t.knownDetector).
							WithField("taskId", taskId).
							WithField("taskName", t.ti.Name).
							WithField("taskPid", t.knownPid).
							WithField("level", infologger.IL_Support).
							Warningf("task transitioned to ERROR on its own - notifying environment")
					}
				}
				deviceEvent.SetLabels(map[string]string{"detector": t.knownDetector, "environmentId": t.knownEnvironmentId.String()})

				t.sendDeviceEvent(t.knownEnvironmentId, deviceEvent)
			}
		}()

		err = taskCmd.Wait()
		// ^ when this unblocks, the task is done
		log.WithField("partition", t.knownEnvironmentId.String()).
			WithField("detector", t.knownDetector).
			WithFields(logrus.Fields{
				"id":      t.ti.TaskID.Value,
				"task":    t.ti.Name,
				"command": truncatedCmd,
				"level":   infologger.IL_Devel,
			}).Debug("task done (taskCmd.Wait unblocks), preparing final update")

		pendingState := mesos.TASK_FINISHED
		if err != nil {
			taskClassName, _ := utils.ExtractTaskClassName(t.ti.Name)
			log.WithField("partition", t.knownEnvironmentId.String()).
				WithField("detector", t.knownDetector).
				WithField("level", infologger.IL_Ops).
				Errorf("task '%s' terminated with error: %s", utils.TrimJitPrefix(taskClassName), err.Error())
			log.WithField("partition", t.knownEnvironmentId.String()).
				WithField("detector", t.knownDetector).
				WithFields(logrus.Fields{
					"id":      t.ti.TaskID.Value,
					"task":    t.ti.Name,
					"command": truncatedCmd,
					"error":   err.Error(),
					"level":   infologger.IL_Devel,
				}).
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
			log.WithField("partition", t.knownEnvironmentId.String()).
				WithField("detector", t.knownDetector).
				WithField("taskId", t.ti.TaskID.GetValue()).
				WithField("taskName", t.ti.Name).
				Debug("rpc client closed")
			t.rpc = nil
			log.WithField("partition", t.knownEnvironmentId.String()).
				WithField("detector", t.knownDetector).
				WithField("taskId", t.ti.TaskID.GetValue()).
				WithField("taskName", t.ti.Name).
				Debug("rpc client removed")
		}

		if errStdout != nil || errStderr != nil {
			log.WithField("partition", t.knownEnvironmentId.String()).
				WithField("detector", t.knownDetector).
				WithFields(logrus.Fields{
					"errStderr": errStderr,
					"errStdout": errStdout,
					"id":        t.ti.TaskID.Value,
					"task":      t.ti.Name,
					"command":   truncatedCmd,
					"level":     infologger.IL_Devel,
				}).
				Warning("failed to capture stdout or stderr of task")
		}

		t.sendStatus(t.knownEnvironmentId, pendingState, "")
	}()

	log.WithField("partition", t.knownEnvironmentId.String()).
		WithField("detector", t.knownDetector).
		WithFields(logrus.Fields{
			"task":  t.ti.Name,
			"level": infologger.IL_Devel,
		}).
		Debug("gRPC client starting, handler forked: executor.ControllableTask.Launch end")
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
	var (
		pid          = 0
		reachedState = "UNKNOWN" // FIXME: should be LAUNCHING or similar
	)
	cxt, cancel := context.WithTimeout(context.Background(), KILL_TRANSITION_TIMEOUT)
	defer cancel()
	response, err := t.rpc.GetState(cxt, &pb.GetStateRequest{}, grpc.EmptyCallOption{})
	if err == nil { // we successfully got the state from the task
		log.WithField("nativeState", response.GetState()).
			WithField("taskId", t.ti.GetTaskID()).
			WithField("level", infologger.IL_Devel).
			WithField("partition", t.knownEnvironmentId.String()).
			WithField("detector", t.knownDetector).
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
			log.WithField("partition", t.knownEnvironmentId.String()).
				WithField("detector", t.knownDetector).
				WithFields(logrus.Fields{
					"evt":        cmd.Event,
					"src":        cmd.Source,
					"dst":        cmd.Destination,
					"targetList": cmd.TargetList,
					"level":      infologger.IL_Devel,
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
				log.WithField("partition", t.knownEnvironmentId.String()).
					WithField("detector", t.knownDetector).
					WithField("task", t.ti.TaskID.Value).
					WithField("level", infologger.IL_Devel).
					Warn("teardown transition sequence timed out")
			}
			// timeout we should break
			if commitResponse == nil {
				break
			}

			log.WithField("partition", t.knownEnvironmentId.String()).
				WithField("detector", t.knownDetector).
				WithField("newState", commitResponse.newState).
				WithError(commitResponse.transitionError).
				WithField("task", t.ti.TaskID.Value).
				WithField("level", infologger.IL_Devel).
				Debug("transition committed")
			if commitResponse.transitionError != nil || len(cmd.Event) == 0 {
				log.WithField("partition", t.knownEnvironmentId.String()).
					WithField("detector", t.knownDetector).
					WithError(commitResponse.transitionError).
					WithField("task", t.ti.TaskID.Value).
					WithField("level", infologger.IL_Devel).
					Warn("teardown transition sequence error")
				break
			}
			reachedState = commitResponse.newState
		}

		log.WithField("partition", t.knownEnvironmentId.String()).
			WithField("detector", t.knownDetector).
			WithField("task", t.ti.TaskID.Value).
			WithField("level", infologger.IL_Devel).
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
		log.WithField("partition", t.knownEnvironmentId.String()).
			WithField("detector", t.knownDetector).
			WithError(err).
			WithField("taskId", t.ti.GetTaskID()).
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
			log.WithField("partition", t.knownEnvironmentId.String()).
				WithField("detector", t.knownDetector).
				WithError(err).WithField("taskId", t.ti.GetTaskID()).
				Warn("task PID not known from task, using containing shell PGID")
		}
	}

	_ = t.rpc.Close()
	t.rpc = nil

	if reachedState == "DONE" {
		log.WithField("partition", t.knownEnvironmentId.String()).
			WithField("detector", t.knownDetector).
			WithField("taskId", t.ti.TaskID.Value).
			Debugf("task reached DONE, will wait %.1fs before terminating it", DONE_TIMEOUT.Seconds())
		t.pendingFinalTaskStateCh <- mesos.TASK_FINISHED
		time.Sleep(DONE_TIMEOUT)
	} else { // something went wrong
		log.WithField("partition", t.knownEnvironmentId.String()).
			WithField("detector", t.knownDetector).
			WithField("taskId", t.ti.TaskID.Value).
			Debug("task died already or will be killed soon")
		t.pendingFinalTaskStateCh <- mesos.TASK_KILLED
	}

	if pidExists(pid) {
		return t.doTermIntKill(pid)
	} else {
		log.WithField("taskId", t.ti.GetTaskID()).
			WithField("partition", t.knownEnvironmentId.String()).
			WithField("detector", t.knownDetector).
			Debugf("task terminated on its own")
		return nil
	}
}

func (t *ControllableTask) doKill9(pid int) error {
	log.WithField("partition", t.knownEnvironmentId.String()).
		WithField("detector", t.knownDetector).
		WithField("taskId", t.ti.GetTaskID()).
		Debug("sending SIGKILL (9) to task")
	killErr := syscall.Kill(pid, syscall.SIGKILL)
	if killErr != nil {
		log.WithField("partition", t.knownEnvironmentId.String()).
			WithField("detector", t.knownDetector).
			WithError(killErr).
			WithField("taskId", t.ti.GetTaskID()).
			Warning("task SIGKILL failed")
	}

	return killErr
}

func (t *ControllableTask) doTermIntKill(pid int) error {
	killErrCh := make(chan error)
	go func() {
		log.WithField("partition", t.knownEnvironmentId.String()).
			WithField("detector", t.knownDetector).
			WithField("taskId", t.ti.GetTaskID()).
			Debug("sending SIGTERM (15) to task")
		err := syscall.Kill(pid, syscall.SIGTERM)
		if err != nil {
			log.WithField("partition", t.knownEnvironmentId.String()).
				WithField("detector", t.knownDetector).
				WithError(err).
				WithField("taskId", t.ti.GetTaskID()).
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
				log.WithField("partition", t.knownEnvironmentId.String()).
					WithField("detector", t.knownDetector).
					WithField("taskId", t.ti.GetTaskID()).
					Debug("sending SIGINT (2) to task")
				killErr = syscall.Kill(pid, syscall.SIGINT)
				if killErr != nil {
					log.WithField("partition", t.knownEnvironmentId.String()).
						WithField("detector", t.knownDetector).
						WithError(killErr).
						WithField("taskId", t.ti.GetTaskID()).
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
