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

// Package executable provides platform-specific executable management functionality
// for running and controlling tasks in the executor environment.
package executable

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/AliceO2Group/Control/common"
	"github.com/AliceO2Group/Control/common/controlmode"
	"github.com/AliceO2Group/Control/common/event"
	"github.com/AliceO2Group/Control/common/logger"
	"github.com/AliceO2Group/Control/common/logger/infologger"
	"github.com/AliceO2Group/Control/common/utils/uid"
	"github.com/AliceO2Group/Control/core/controlcommands"
	"github.com/AliceO2Group/Control/executor/executorcmd"
	"github.com/AliceO2Group/Control/executor/executorutil"
	mesos "github.com/mesos/mesos-go/api/v1/lib"
	"github.com/sirupsen/logrus"
)

const (
	startupPollingInterval = 500 * time.Millisecond
	startupTimeout         = 30 * time.Second
)

var log = logger.New(logrus.StandardLogger(), "executor")

type SendStatusFunc func(envId uid.ID, state mesos.TaskState, message string)
type SendDeviceEventFunc func(envId uid.ID, event event.DeviceEvent)
type SendMessageFunc func(message []byte)

type Task interface {
	Launch() error
	Kill() error
	Transition(transition *executorcmd.ExecutorCommand_Transition) *controlcommands.MesosCommandResponse_Transition
	UnmarshalTransition([]byte) (*executorcmd.ExecutorCommand_Transition, error)
}

type taskBase struct {
	ti  *mesos.TaskInfo
	Tci *common.TaskCommandInfo

	sendStatus      SendStatusFunc
	sendDeviceEvent SendDeviceEventFunc
	sendMessage     SendMessageFunc

	knownEnvironmentId uid.ID
	knownDetector      string
}

func NewTask(taskInfo mesos.TaskInfo, sendStatusFunc SendStatusFunc, sendDeviceEventFunc SendDeviceEventFunc, sendMessageFunc SendMessageFunc) Task {
	var commandInfo common.TaskCommandInfo
	envId := executorutil.GetEnvironmentIdFromLabelerType(&taskInfo)
	detector := executorutil.GetValueFromLabelerType(&taskInfo, "detector")

	tciData := taskInfo.GetData()

	log.WithField("json", string(tciData[:])).
		Trace("received TaskCommandInfo")
	if err := json.Unmarshal(tciData, &commandInfo); tciData != nil && err == nil {
		log.WithFields(logrus.Fields{
			"shell":       *commandInfo.Shell,
			"value":       *commandInfo.Value,
			"args":        commandInfo.Arguments,
			"task":        taskInfo.Name,
			"controlmode": commandInfo.ControlMode.String(),
			"level":       infologger.IL_Devel,
			"partition":   envId.String(),
			"detector":    detector,
		}).
			Debug("instantiating task")

		rawCommand := strings.Join(append([]string{*commandInfo.Value}, commandInfo.Arguments...), " ")
		// we deliberately print taskID, then command, so if the latter is too long for infologger to accept it,
		// we at least have the taskID and the beginning of the command.
		log.WithField("level", infologger.IL_Support).
			WithField("partition", envId.String()).
			WithField("detector", detector).
			Infof("launching task %s on executorId %s: %s",
				taskInfo.TaskID.GetValue(),
				taskInfo.GetExecutor().GetExecutorID().Value,
				rawCommand)
	} else {
		if err != nil {
			log.WithError(err).
				WithField("task", taskInfo.Name).
				WithField("partition", envId.String()).
				WithField("detector", detector).
				Error("could not launch task")
		} else {
			log.WithError(errors.New("command data is nil")).
				WithField("task", taskInfo.Name).
				WithField("partition", envId.String()).
				WithField("detector", detector).
				Error("could not launch task")
		}
		sendStatusFunc(envId, mesos.TASK_FAILED, "TaskInfo.Data is nil")
		return nil
	}

	var newTask Task
	// switch based on type of task
	switch commandInfo.ControlMode {
	case controlmode.BASIC:
		newTask = &BasicTask{
			basicTaskBase: basicTaskBase{
				taskBase: taskBase{
					ti:                 &taskInfo,
					Tci:                &commandInfo,
					sendStatus:         sendStatusFunc,
					sendDeviceEvent:    sendDeviceEventFunc,
					sendMessage:        sendMessageFunc,
					knownEnvironmentId: envId,
					knownDetector:      detector,
				},
			},
		}
	case controlmode.HOOK:
		newTask = &HookTask{
			basicTaskBase: basicTaskBase{
				taskBase: taskBase{
					ti:                 &taskInfo,
					Tci:                &commandInfo,
					sendStatus:         sendStatusFunc,
					sendDeviceEvent:    sendDeviceEventFunc,
					sendMessage:        sendMessageFunc,
					knownEnvironmentId: envId,
					knownDetector:      detector,
				},
			},
		}
	case controlmode.DIRECT:
		fallthrough
	case controlmode.FAIRMQ:
		newTask = &ControllableTask{
			taskBase: taskBase{
				ti:                 &taskInfo,
				Tci:                &commandInfo,
				sendStatus:         sendStatusFunc,
				sendDeviceEvent:    sendDeviceEventFunc,
				sendMessage:        sendMessageFunc,
				knownEnvironmentId: envId,
				knownDetector:      detector,
			},
			rpc: nil,
		}
	}

	return newTask
}

func prepareTaskCmd(commandInfo *common.TaskCommandInfo) (*exec.Cmd, error) {
	var taskCmd *exec.Cmd
	ctx := context.Background()
	if commandInfo.Timeout.Seconds() > 0 { // if a timeout is defined, we add a context
		ctx, _ = context.WithTimeout(ctx, commandInfo.Timeout)
	}

	if *commandInfo.Shell {
		rawCommand := strings.Join(append([]string{*commandInfo.Value}, commandInfo.Arguments...), " ")
		taskCmd = exec.CommandContext(ctx, "/bin/sh", []string{"-c", rawCommand}...)
	} else {
		taskCmd = exec.CommandContext(ctx, *commandInfo.Value, commandInfo.Arguments...)
	}
	taskCmd.Env = append(os.Environ(), commandInfo.Env...)

	// We must setpgid(2) in order to be able to kill the whole process group which consists of
	// the containing shell and all of its children
	taskCmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	setPdeathsig(taskCmd.SysProcAttr)

	// If the commandInfo specifies a username
	if commandInfo.User != nil && len(*commandInfo.User) > 0 {
		// we must first look up the uid/gid
		targetUser, err := user.Lookup(*commandInfo.User)
		if err != nil {
			return nil, err
		}

		userid, err := strconv.ParseUint(targetUser.Uid, 10, 32)
		if err != nil {
			return nil, err
		}
		groupid, err := strconv.ParseUint(targetUser.Gid, 10, 32)
		if err != nil {
			return nil, err
		}

		gids, gidStrings := executorutil.GetGroupIDs(targetUser)

		credential := &syscall.Credential{
			Uid:         uint32(userid),
			Gid:         uint32(groupid),
			Groups:      gids,
			NoSetGroups: false,
		}
		taskCmd.SysProcAttr.Credential = credential
		log.WithFields(logrus.Fields{
			"shell":  *commandInfo.Shell,
			"value":  *commandInfo.Value,
			"args":   commandInfo.Arguments,
			"uid":    credential.Uid,
			"gid":    credential.Gid,
			"groups": gidStrings,
		}).
			Trace("custom credentials set")
	}

	return taskCmd, nil
}
