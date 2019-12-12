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
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/AliceO2Group/Control/common"
	"github.com/AliceO2Group/Control/common/controlmode"
	"github.com/AliceO2Group/Control/common/event"
	"github.com/AliceO2Group/Control/common/logger"
	"github.com/AliceO2Group/Control/core/controlcommands"
	"github.com/AliceO2Group/Control/executor/executorcmd"
	mesos "github.com/mesos/mesos-go/api/v1/lib"
	"github.com/sirupsen/logrus"
)

const(
	startupPollingInterval = 500 * time.Millisecond
	startupTimeout         = 30 * time.Second
)

var log = logger.New(logrus.StandardLogger(), "executor")

type SendStatusFunc func(state mesos.TaskState, message string)
type SendDeviceEventFunc func(event event.DeviceEvent)

type Task interface {
	Launch() error
	Kill() error
	Transition(transition *executorcmd.ExecutorCommand_Transition) *controlcommands.MesosCommandResponse_Transition
	UnmarshalTransition([]byte) (*executorcmd.ExecutorCommand_Transition, error)
}

type taskBase struct {
	ti *mesos.TaskInfo
	tci *common.TaskCommandInfo

	sendStatus SendStatusFunc
	sendDeviceEvent SendDeviceEventFunc
}

func NewTask(taskInfo mesos.TaskInfo, sendStatusFunc SendStatusFunc, sendDeviceEventFunc SendDeviceEventFunc) Task {
	var commandInfo common.TaskCommandInfo

	tciData := taskInfo.GetData()

	log.WithField("json", string(tciData[:])).Trace("received TaskCommandInfo")
	if err := json.Unmarshal(tciData, &commandInfo); tciData != nil && err == nil {
		log.WithFields(logrus.Fields{
			"shell": *commandInfo.Shell,
			"value": *commandInfo.Value,
			"args":  commandInfo.Arguments,
			"task":  taskInfo.Name,
		}).
		Info("instantiating task")
	} else {
		if err != nil {
			log.WithError(err).WithField("task", taskInfo.Name).Error("could not launch task")
		} else {
			log.WithError(errors.New("command data is nil")).WithField("task", taskInfo.Name).Error("could not launch task")
		}
		sendStatusFunc(mesos.TASK_FAILED, "TaskInfo.Data is nil")
		return nil
	}

	var newTask Task
	// switch based on type of task
	switch commandInfo.ControlMode {
	case controlmode.BASIC:
		newTask = &BasicTask{
			taskBase: taskBase{
				ti: &taskInfo,
				tci: &commandInfo,
				sendStatus: sendStatusFunc,
				sendDeviceEvent: sendDeviceEventFunc,
			},
		}
	case controlmode.DIRECT:
		fallthrough
	case controlmode.FAIRMQ:
		newTask = &ControllableTask{
			taskBase: taskBase{
				ti: &taskInfo,
				tci: &commandInfo,
				sendStatus: sendStatusFunc,
				sendDeviceEvent: sendDeviceEventFunc,
			},
			rpc: nil,
		}
	}

	return newTask
}

func prepareTaskCmd(commandInfo *common.TaskCommandInfo) *exec.Cmd {
	var taskCmd *exec.Cmd
	if *commandInfo.Shell {
		rawCommand := strings.Join(append([]string{*commandInfo.Value}, commandInfo.Arguments...), " ")
		taskCmd = exec.Command("/bin/sh", []string{"-c", rawCommand}...)
	} else {
		taskCmd = exec.Command(*commandInfo.Value, commandInfo.Arguments...)
	}
	taskCmd.Env = append(os.Environ(), commandInfo.Env...)

	// We must setpgid(2) in order to be able to kill the whole process group which consists of
	// the containing shell and all of its children
	taskCmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	return taskCmd
}
