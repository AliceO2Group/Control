/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2018 CERN and copyright holders of ALICE O².
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

package environment

import (
	"context"
	"errors"

	"github.com/AliceO2Group/Control/core/workflow"

	"github.com/AliceO2Group/Control/common/event"
	"github.com/AliceO2Group/Control/common/monitoring"
	"github.com/AliceO2Group/Control/common/tracing"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"github.com/AliceO2Group/Control/core/task"
	"github.com/AliceO2Group/Control/core/task/taskop"
)

func NewConfigureTransition(taskman *task.Manager) Transition {
	return &ConfigureTransition{
		baseTransition: baseTransition{
			name:    "CONFIGURE",
			taskman: taskman,
		},
	}
}

type ConfigureTransition struct {
	baseTransition
}

func (t ConfigureTransition) do(ctx context.Context, env *Environment) (err error) {
	if env == nil {
		return errors.New("cannot transition in NIL environment")
	}

	metric := t.transitionDoMetric(env)
	defer monitoring.TimerSendSingle(&metric, monitoring.Millisecond)()

	span := tracing.NewSpan(ctx, "ConfigureTransition.do")
	defer func() {
		span.Span().SetAttributes(
			attribute.String("transition", t.name),
			attribute.String("envId", env.Id().String()),
		)
		if err != nil {
			span.Span().RecordError(err)
			span.Span().SetStatus(codes.Error, err.Error())
		} else {
			span.Span().SetStatus(codes.Ok, "")
		}
		span.End()
	}()

	wf := env.Workflow()

	activeTasks := workflow.GetActiveTasks(wf)

	if len(activeTasks) != 0 {
		// err = t.taskman.ConfigureTasks(env.Id().Array(), tasks)
		taskmanMessage := task.NewEnvironmentMessage(taskop.ConfigureTasks, env.Id(), activeTasks, nil)
		t.taskman.MessageChannel <- taskmanMessage
	}
	incomingEv := <-env.stateChangedCh
	// If some tasks failed to transition
	if tasksStateErrors := incomingEv.GetTasksStateChangedError(); tasksStateErrors != nil {
		metric.AddResult(monitoring.ERROR)
		return tasksStateErrors
	}

	env.sendEnvironmentEvent(&event.EnvironmentEvent{EnvironmentID: env.Id().String(), State: "CONFIGURED"})
	metric.AddResult(monitoring.SUCCESS)
	return
}
