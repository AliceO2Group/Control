/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2021 CERN and copyright holders of ALICE O².
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
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	texttemplate "text/template"
	"time"

	"github.com/AliceO2Group/Control/apricot"
	"github.com/AliceO2Group/Control/common/event"
	"github.com/AliceO2Group/Control/common/logger"
	"github.com/AliceO2Group/Control/common/logger/infologger"
	"github.com/AliceO2Group/Control/common/utils"
	"github.com/AliceO2Group/Control/common/utils/uid"
	"github.com/AliceO2Group/Control/configuration/template"
	"github.com/AliceO2Group/Control/core/integration"
	"github.com/AliceO2Group/Control/core/task"
	"github.com/sirupsen/logrus"
)

var log = logger.New(logrus.StandardLogger(), "callable")

type HookWeight int
type Calls []*Call
type Hooks []Hook
type HooksMap map[HookWeight]Hooks
type CallsMap map[HookWeight]Calls

type Hook interface {
	GetParentRole() interface{}
	GetParentRolePath() string
	GetName() string
	GetTraits() task.Traits
}

type Call struct {
	Func string
	Return string
	VarStack map[string]string
	Traits task.Traits
	parentRole ParentRole

	await chan error
}

func ParseTriggerExpression(triggerExpr string) (triggerName string, triggerWeight HookWeight) {
	var (
		triggerWeightS string
		triggerWeightI int
		err error
	)

	// Split the trigger expression of this task by + or -
	if splitIndex := strings.LastIndexFunc(triggerExpr, func(r rune) bool {
		return r == '+' || r == '-'
	}); splitIndex >= 0 {
		triggerName, triggerWeightS = triggerExpr[:splitIndex], triggerExpr[splitIndex:]
	} else {
		triggerName, triggerWeightS = triggerExpr, "+0"
	}

	triggerWeightI, err = strconv.Atoi(triggerWeightS)
	if err != nil {
		log.Warnf("invalid trigger weight definition %s, defaulting to %s", triggerExpr, triggerName + "+0")
		triggerWeightI = 0
	}
	triggerWeight = HookWeight(triggerWeightI)

	return
}

func (m HooksMap) GetWeights() []HookWeight {
	weights := make([]int, len(m))
	i := 0
	for k, _ := range m {
		weights[i] = int(k)
		i++
	}
	sort.Ints(weights)
	out := make([]HookWeight, len(weights))
	for i, v := range weights{
		out[i] = HookWeight(v)
	}
	return out
}

func (m CallsMap) GetWeights() []HookWeight {
	weights := make([]int, len(m))
	i := 0
	for k, _ := range m {
		weights[i] = int(k)
		i++
	}
	sort.Ints(weights)
	out := make([]HookWeight, len(weights))
	for i, v := range weights{
		out[i] = HookWeight(v)
	}
	return out
}

func (s Hooks) FilterCalls() (calls Calls) {
	calls = make(Calls, 0)
	for _, v := range s {
		if c, ok := v.(*Call); ok {
			calls = append(calls, c)
		}
	}
	return
}

func (s Hooks) FilterTasks() (tasks task.Tasks) {
	tasks = make(task.Tasks, 0)
	for _, v := range s {
		if t, ok := v.(*task.Task); ok {
			tasks = append(tasks, t)
		}
	}
	return
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


func NewCall(funcCall string, returnVar string, varStack map[string]string, parent ParentRole) (call *Call) {
	return &Call{
		Func:       funcCall,
		Return:     returnVar,
		VarStack:   varStack,
		Traits:     parent.GetTaskTraits(),
		parentRole: parent,
	}
}

func (c *Call) Call() error {
	log.WithField("trigger", c.Traits.Trigger).
		WithField("await", c.Traits.Await).
		WithField("partition", c.parentRole.GetEnvironmentId().String()).
		WithField("level", infologger.IL_Devel).
		Debugf("calling hook function %s", c.Func)
	output := "{{" + c.Func + "}}"
	returnVar := c.Return
	fields := template.Fields{
			template.WrapPointer(&output),
			template.WrapPointer(&returnVar),
		}
	c.VarStack["environment_id"] = c.parentRole.GetEnvironmentId().String()
	c.VarStack["__call_func"] = c.Func
	c.VarStack["__call_timeout"] = c.Traits.Timeout
	c.VarStack["__call_trigger"] = c.Traits.Trigger
	c.VarStack["__call_await"] = c.Traits.Await
	c.VarStack["__call_critical"] = strconv.FormatBool(c.Traits.Critical)
	c.VarStack["__call_rolepath"] = c.GetParentRolePath()

	objStack := integration.PluginsInstance().CallStack(c)

	//TODO: Can the repo be fetched from somewhere?
	err := fields.Execute(apricot.Instance(), c.GetName(), c.VarStack, objStack, make(map[string]texttemplate.Template), nil)
	if err != nil {
		return err
	}
	if len(returnVar) > 0 {
		c.parentRole.SetRuntimeVar(returnVar, output)
	}
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

type ParentRole interface {
	GetPath() string
	GetTaskTraits() task.Traits
	GetEnvironmentId() uid.ID
	ConsolidatedVarStack() (varStack map[string]string, err error)
	SendEvent(event.Event)
	SetRuntimeVar(key string, value string)
	GetCurrentRunNumber() uint32
}

func AcquireTimeout(defaultTimeout time.Duration, varStack map[string]string, callName string, envId string) time.Duration {
	timeout := defaultTimeout
	timeoutStr, ok := varStack["__call_timeout"] // the Call interface ensures we'll find this key
	// see Call.Call in callable/call.go for details
	if ok {
		var err error
		timeout, err = time.ParseDuration(timeoutStr)
		if err != nil {
			timeout = defaultTimeout
			log.WithField("partition", envId).
				WithField("call", callName).
				WithField("default", timeout.String()).
				Warn("could not parse timeout declaration for hook call")
		}
	} else {
		log.WithField("partition", envId).
			WithField("call", callName).
			WithField("default", timeout.String()).
			Warn("could not get timeout declaration for hook call")
	}
	return timeout
}