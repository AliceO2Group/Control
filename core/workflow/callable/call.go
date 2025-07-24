/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2021-2022 CERN and copyright holders of ALICE O².
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

package callable

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	texttemplate "text/template"
	"time"

	"github.com/AliceO2Group/Control/apricot"
	"github.com/AliceO2Group/Control/common/event/topic"
	"github.com/AliceO2Group/Control/common/logger"
	"github.com/AliceO2Group/Control/common/logger/infologger"
	"github.com/AliceO2Group/Control/common/monitoring"
	evpb "github.com/AliceO2Group/Control/common/protos"
	"github.com/AliceO2Group/Control/common/utils"
	"github.com/AliceO2Group/Control/configuration/template"
	"github.com/AliceO2Group/Control/core/integration"
	"github.com/AliceO2Group/Control/core/task"
	"github.com/AliceO2Group/Control/core/the"
	"github.com/sirupsen/logrus"
)

var log = logger.New(logrus.StandardLogger(), "callable")

type Call struct {
	Func       string
	Return     string
	VarStack   map[string]string
	Traits     task.Traits
	parentRole ParentRole

	await       chan error
	awaitCancel context.CancelFunc
}

type Calls []*Call

func NewCall(funcCall string, returnVar string, parent ParentRole) (call *Call) {
	return &Call{
		Func:       funcCall,
		Return:     returnVar,
		VarStack:   map[string]string{},
		Traits:     parent.GetTaskTraits(),
		parentRole: parent,
	}
}

func (s Calls) CallAll() map[*Call]error {
	errs := make(map[*Call]error)
	for _, v := range s {
		err := v.Call()
		if err != nil {
			errs[v] = err
		}
	}
	return errs
}

func (s Calls) StartAll() {
	for _, v := range s {
		v.Start()
	}
}

func (s Calls) AwaitAll() map[*Call]error {
	// Since each Await call blocks, we call it in parallel and then collect
	errs := make(map[*Call]error)
	wg := &sync.WaitGroup{}
	wg.Add(len(s))
	for _, v := range s {
		go func(v *Call) {
			defer wg.Done()
			err := v.Await()
			if err != nil {
				errs[v] = err
			}
		}(v)
	}
	wg.Wait()
	return errs
}

func (c *Call) Call() error {
	log.WithField("trigger", c.Traits.Trigger).
		WithField("await", c.Traits.Await).
		WithField("partition", c.parentRole.GetEnvironmentId().String()).
		WithField("level", infologger.IL_Devel).
		Debugf("calling hook function %s", c.Func)

	metric := c.newMetric("callablecall")
	defer monitoring.TimerSendSingle(&metric, monitoring.Millisecond)()

	the.EventWriterWithTopic(topic.Call).WriteEvent(&evpb.Ev_CallEvent{
		Path:       c.GetParentRolePath(),
		Func:       c.Func,
		CallStatus: evpb.OpStatus_STARTED,
		Return:     c.Return,
		Traits: &evpb.Traits{
			Trigger:  c.Traits.Trigger,
			Await:    c.Traits.Await,
			Timeout:  c.Traits.Timeout,
			Critical: c.Traits.Critical,
		},
		EnvironmentId: c.parentRole.GetEnvironmentId().String(),
	})

	output := "{{" + c.Func + "}}"
	returnVar := c.Return
	fields := template.Fields{
		template.WrapPointer(&output),
		template.WrapPointer(&returnVar),
	}
	var err error
	c.VarStack, err = c.parentRole.ConsolidatedVarStack()
	if err != nil {
		log.WithField("trigger", c.Traits.Trigger).
			WithField("partition", c.parentRole.GetEnvironmentId().String()).
			Debug("could not instantiate varStack")
	}
	c.VarStack["environment_id"] = c.parentRole.GetEnvironmentId().String()
	c.VarStack["__call_func"] = c.Func
	c.VarStack["__call_timeout"] = c.Traits.Timeout
	c.VarStack["__call_trigger"] = c.Traits.Trigger
	c.VarStack["__call_await"] = c.Traits.Await
	c.VarStack["__call_critical"] = strconv.FormatBool(c.Traits.Critical)
	c.VarStack["__call_rolepath"] = c.GetParentRolePath()

	objStack := integration.PluginsInstance().CallStack(c)

	var errMsg string
	err = fields.Execute(apricot.Instance(), c.GetName(), c.VarStack, objStack, nil, make(map[string]texttemplate.Template), nil)
	if err != nil {
		errMsg = err.Error()

		the.EventWriterWithTopic(topic.Call).WriteEvent(&evpb.Ev_CallEvent{
			Path:       c.GetParentRolePath(),
			Func:       c.Func,
			CallStatus: evpb.OpStatus_DONE_ERROR,
			Return:     c.Return,
			Traits: &evpb.Traits{
				Trigger:  c.Traits.Trigger,
				Await:    c.Traits.Await,
				Timeout:  c.Traits.Timeout,
				Critical: c.Traits.Critical,
			},
			Output:        output,
			Error:         errMsg,
			EnvironmentId: c.parentRole.GetEnvironmentId().String(),
		})

		return err
	}
	if len(returnVar) > 0 {
		c.parentRole.SetRuntimeVar(returnVar, output)
	}

	// if __call_error was written into the VarStack we treat it as an error exit from the call
	var ok bool
	if errMsg, ok = c.VarStack["__call_error"]; ok && len(errMsg) > 0 {
		if errReason, ok := c.VarStack["__call_error_reason"]; ok && len(errReason) > 0 {
			errMsg += ". REASON: " + errReason
		}
		the.EventWriterWithTopic(topic.Call).WriteEvent(&evpb.Ev_CallEvent{
			Path:       c.GetParentRolePath(),
			Func:       c.Func,
			CallStatus: evpb.OpStatus_DONE_ERROR,
			Return:     c.Return,
			Traits: &evpb.Traits{
				Trigger:  c.Traits.Trigger,
				Await:    c.Traits.Await,
				Timeout:  c.Traits.Timeout,
				Critical: c.Traits.Critical,
			},
			Output:        output,
			Error:         errMsg,
			EnvironmentId: c.parentRole.GetEnvironmentId().String(),
		})

		return errors.New(errMsg)
	}

	the.EventWriterWithTopic(topic.Call).WriteEvent(&evpb.Ev_CallEvent{
		Path:       c.GetParentRolePath(),
		Func:       c.Func,
		CallStatus: evpb.OpStatus_DONE_OK,
		Return:     c.Return,
		Traits: &evpb.Traits{
			Trigger:  c.Traits.Trigger,
			Await:    c.Traits.Await,
			Timeout:  c.Traits.Timeout,
			Critical: c.Traits.Critical,
		},
		Output:        output,
		EnvironmentId: c.parentRole.GetEnvironmentId().String(),
	})

	return nil
}

func (c *Call) newMetric(name string) monitoring.Metric {
	metric := monitoring.NewMetric(name)
	metric.AddTag("name", c.GetName())
	metric.AddTag("trigger", c.GetTraits().Trigger)
	metric.AddTag("envId", c.parentRole.GetEnvironmentId().String())
	return metric
}

func (c *Call) Start() {
	c.await = make(chan error)
	ctx, cancel := context.WithCancel(context.Background())
	c.awaitCancel = cancel
	go func() {
		metric := c.newMetric("callablewrapped")
		defer monitoring.TimerSendSingle(&metric, monitoring.Millisecond)()

		callId := fmt.Sprintf("hook:%s:%s", c.GetTraits().Trigger, c.GetName())
		log.Debugf("%s started", callId)
		defer utils.TimeTrack(time.Now(), callId, log.WithPrefix("callable"))
		select {
		case c.await <- c.Call():
		case <-ctx.Done():
			log.Debugf("%s cancelled", callId)
		}
		close(c.await)
	}()
}

func (c *Call) Await() error {
	log.Trace("awaiting " + c.Func + " in trigger phase " + c.Traits.Await)
	return <-c.await
}

func (c *Call) Cancel() bool {
	if c.awaitCancel != nil {
		c.awaitCancel()
		c.awaitCancel = nil
		return true
	}
	return false
}

func (c *Call) GetParentRole() interface{} {
	return c.parentRole
}

func (c *Call) GetParentRolePath() string {
	return c.parentRole.GetPath()
}

func (c *Call) GetName() string {
	return c.parentRole.GetPath()
}

func (c *Call) GetTraits() task.Traits {
	return c.Traits
}
