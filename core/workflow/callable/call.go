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

	await chan error
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
	errors := make(map[*Call]error)
	for _, v := range s {
		err := v.Call()
		if err != nil {
			errors[v] = err
		}
	}
	return errors
}

func (s Calls) StartAll() {
	for _, v := range s {
		v.Start()
	}
}

func (s Calls) AwaitAll() map[*Call]error {
	// Since each Await call blocks, we call it in parallel and then collect
	errors := make(map[*Call]error)
	wg := &sync.WaitGroup{}
	wg.Add(len(s))
	for _, v := range s {
		go func(v *Call) {
			defer wg.Done()
			err := v.Await()
			if err != nil {
				errors[v] = err
			}
		}(v)
	}
	wg.Wait()
	return errors
}

func (c *Call) Call() error {
	log.WithField("trigger", c.Traits.Trigger).
		WithField("await", c.Traits.Await).
		WithField("partition", c.parentRole.GetEnvironmentId().String()).
		WithField("level", infologger.IL_Devel).
		Debugf("calling hook function %s", c.Func)

	the.EventWriterWithTopic(topic.Environment).WriteEvent(&evpb.Ev_CallEvent{
		Func:   c.Func,
		Status: "STARTED",
		Return: c.Return,
		Traits: &evpb.Traits{
			Trigger:  c.Traits.Trigger,
			Await:    c.Traits.Await,
			Timeout:  c.Traits.Timeout,
			Critical: c.Traits.Critical,
		},
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

		the.EventWriterWithTopic(topic.Environment).WriteEvent(&evpb.Ev_CallEvent{
			Func:   c.Func,
			Status: "ERROR",
			Return: c.Return,
			Traits: &evpb.Traits{
				Trigger:  c.Traits.Trigger,
				Await:    c.Traits.Await,
				Timeout:  c.Traits.Timeout,
				Critical: c.Traits.Critical,
			},
			Output: output,
			Error:  errMsg,
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
		the.EventWriterWithTopic(topic.Environment).WriteEvent(&evpb.Ev_CallEvent{
			Func:   c.Func,
			Status: "ERROR",
			Return: c.Return,
			Traits: &evpb.Traits{
				Trigger:  c.Traits.Trigger,
				Await:    c.Traits.Await,
				Timeout:  c.Traits.Timeout,
				Critical: c.Traits.Critical,
			},
			Output: output,
			Error:  errMsg,
		})

		return errors.New(errMsg)
	}

	the.EventWriterWithTopic(topic.Environment).WriteEvent(&evpb.Ev_CallEvent{
		Func:   c.Func,
		Status: "DONE",
		Return: c.Return,
		Traits: &evpb.Traits{
			Trigger:  c.Traits.Trigger,
			Await:    c.Traits.Await,
			Timeout:  c.Traits.Timeout,
			Critical: c.Traits.Critical,
		},
		Output: output,
	})

	return nil
}

func (c *Call) Start() {
	c.await = make(chan error)
	go func() {
		callId := fmt.Sprintf("hook:%s:%s", c.GetTraits().Trigger, c.GetName())
		log.Debugf("%s started", callId)
		defer utils.TimeTrack(time.Now(), callId, log.WithPrefix("callable"))
		c.await <- c.Call()
		close(c.await)
	}()
}

func (c *Call) Await() error {
	log.Trace("awaiting " + c.Func + " in trigger phase " + c.Traits.Await)
	return <-c.await
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
