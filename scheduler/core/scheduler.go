/*
 * === This file is part of octl <https://github.com/teo/octl> ===
 *
 * Copyright 2017 CERN and copyright holders of ALICE O².
 * Author: Teo Mrnjavac <teo.mrnjavac@cern.ch>
 *
 * Portions from examples in <https://github.com/mesos/mesos-go>:
 *     Copyright 2013-2015, Mesosphere, Inc.
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

package core


import (
	"context"
	"errors"
	"io"
	"strconv"
	"time"

	"github.com/mesos/mesos-go/api/v1/lib"
	"github.com/mesos/mesos-go/api/v1/lib/backoff"
	xmetrics "github.com/mesos/mesos-go/api/v1/lib/extras/metrics"
	"github.com/mesos/mesos-go/api/v1/lib/extras/scheduler/callrules"
	"github.com/mesos/mesos-go/api/v1/lib/extras/scheduler/controller"
	"github.com/mesos/mesos-go/api/v1/lib/extras/scheduler/eventrules"
	"github.com/mesos/mesos-go/api/v1/lib/extras/store"
	"github.com/mesos/mesos-go/api/v1/lib/resources"
	"github.com/mesos/mesos-go/api/v1/lib/scheduler"
	"github.com/mesos/mesos-go/api/v1/lib/scheduler/calls"
	"github.com/mesos/mesos-go/api/v1/lib/scheduler/events"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/pborman/uuid"
	"encoding/json"
	"strings"
	"github.com/teo/octl/scheduler/core/environment"
)

var (
	RegistrationMinBackoff = 1 * time.Second
	RegistrationMaxBackoff = 15 * time.Second
)

// StateError is returned when the system encounters an unresolvable state transition error and
// should likely exit.
type StateError string

func (err StateError) Error() string { return string(err) }

var schedEventsCh = make(chan scheduler.Event_Type)

func runSchedulerController(ctx context.Context,
							state *internalState,
							fidStore store.Singleton) error {
	// Set up communication from controller to state machine.
	go func() {
		for {
			receivedEvent := <-schedEventsCh
			switch {
			case receivedEvent == scheduler.Event_SUBSCRIBED:
				if state.sm.Is("INITIAL") {
					state.sm.Event("CONNECT")
				}
			}
		}
	}()

	// Set up communication from state machine to controller
	go func() {
		for {
			<- state.reviveOffersCh
			log.WithPrefix("scheduler").Debug("received request to revive offers")
			doReviveOffers(ctx, state)
			log.WithPrefix("scheduler").Debug("revive offers done")
			state.reviveOffersCh <- struct{}{}
		}
	}()

	// The controller starts here, it takes care of connecting to Mesos and subscribing
	// as well as resubscribing if the connection is dropped.
	// It also handles incoming events on the subscription connection.
	//
	// buildFrameworkInfo returns a *mesos.FrameworkInfo which includes the framework
	// ID, as well as additional information such as Roles, WebUI URL, etc.
	return controller.Run(
		ctx,
		buildFrameworkInfo(state.config),
		state.cli, /* controller.Option...: */
		controller.WithEventHandler(buildEventHandler(state, fidStore)),
		controller.WithFrameworkID(store.GetIgnoreErrors(fidStore)),
		controller.WithRegistrationTokens(
			// Limit the rate of reregistration.
			// When the Done chan closes, the Run controller loop terminates. The
			// Done chan is closed by the context when its cancel func is called.
			backoff.Notifier(RegistrationMinBackoff, RegistrationMaxBackoff, ctx.Done()),
		),
		controller.WithSubscriptionTerminated(func(err error) {
			// Sets a handler that runs at the end of every subscription cycle.
			if err != nil {
				if err != io.EOF {
					log.WithPrefix("scheduler").WithField("error", err.Error()).
						Error("subscription terminated")
				}
				if _, ok := err.(StateError); ok {
					state.shutdown()
				}
				return
			}
			log.WithPrefix("scheduler").Info("disconnected")
		}),
	)
}


// buildEventHandler generates and returns a handler to process events received
// from the subscription. The handler is then passed as controller.Option to
// controller.Run.
func buildEventHandler(state *internalState, fidStore store.Singleton) events.Handler {
	// disable brief logs when verbose logs are enabled (there's no sense logging twice!)
	logger := controller.LogEvents(nil).Unless(state.config.verbose)

	return eventrules.New( /* eventrules.Rule... */
		logAllEvents().If(state.config.verbose),
		eventMetrics(state.metricsAPI, time.Now, state.config.summaryMetrics),
		controller.LiftErrors().DropOnError(),
		eventrules.HandleF(notifyStateMachine(state)),
	).Handle(events.Handlers{
		// scheduler.Event_Type: events.Handler
		scheduler.Event_FAILURE: logger.HandleF(failure), // wrapper + print error
		scheduler.Event_OFFERS:  trackOffersReceived(state).HandleF(resourceOffers(state)),
		scheduler.Event_UPDATE:  controller.AckStatusUpdates(state.cli).AndThen().HandleF(statusUpdate(state)),
		scheduler.Event_SUBSCRIBED: eventrules.New(
			logger,
			controller.TrackSubscription(fidStore, state.config.failoverTimeout),
		),
	}.Otherwise(logger.HandleEvent))
}


// Channel the event type of the newly received event to an asynchronous dispatcher
// in runSchedulerController
func notifyStateMachine(state *internalState) events.HandlerFunc {
	return func(ctx context.Context, e *scheduler.Event) error {
		schedEventsCh <- e.GetType()
		return nil
	}
}

// Update metrics when we receive an offer
func trackOffersReceived(state *internalState) eventrules.Rule {
	return func(ctx context.Context, e *scheduler.Event, err error, chain eventrules.Chain) (context.Context, *scheduler.Event, error) {
		if err == nil {
			state.metricsAPI.offersReceived.Int(len(e.GetOffers().GetOffers()))
		}
		return chain(ctx, e, err)
	}
}

// Handle an incoming Event_FAILURE, which may be a failure in the executor or
// in the Mesos agent.
func failure(_ context.Context, e *scheduler.Event) error {
	var (
		f              = e.GetFailure()
		eid, aid, stat = f.ExecutorID, f.AgentID, f.Status
	)
	if eid != nil {
		// executor failed..
		fields := logrus.Fields{
			"executor":	eid.Value,
		}
		if aid != nil {
			fields["agent"] = aid.Value
		}
		if stat != nil {
			fields["error"] = strconv.Itoa(int(*stat))
		}
		log.WithPrefix("scheduler").WithFields(fields).Error("executor failed")
	} else if aid != nil {
		// agent failed..
		log.WithPrefix("scheduler").WithField("agent", aid.Value).Error("agent failed")
	}
	return nil
}

// Handler for Event_OFFERS
func resourceOffers(state *internalState) events.HandlerFunc {
	return func(ctx context.Context, e *scheduler.Event) error {
		var (
			offers                 = e.GetOffers().GetOffers()
			callOption             = calls.RefuseSeconds(time.Second)//calls.RefuseSecondsWithJitter(state.random, state.config.maxRefuseSeconds)
			tasksLaunchedThisCycle = 0
			offersDeclined         = 0
		)

		if state.config.verbose {
			var(
				prettyOffers []string
				offerIds []string
			)
			for i := range offers {
				prettyOffer, _ := json.MarshalIndent(offers[i], "", "\t")
				prettyOffers = append(prettyOffers, string(prettyOffer))
				offerIds = append(offerIds, offers[i].ID.Value)
			}
			log.WithPrefix("scheduler").WithFields(logrus.Fields{
				"offerIds":	strings.Join(offerIds, ", "),
				"offers":	strings.Join(prettyOffers, "\n"),
			}).Debug("received offers")
		}

		var envIdToDeploy *uuid.Array
		select {
		case *envIdToDeploy = <- state.envToDeploy:
			log.WithPrefix("scheduler").WithField("environmentId", envIdToDeploy).
				Debug("received environment to deploy on this offer")
		default:
			log.WithPrefix("scheduler").Debug("no environment needs deployment")
		}

		environmentsChanged := make([]uuid.Array, 0)

		if envIdToDeploy != nil {
			// 3 ways to make decisions
			// * FLP1, FLP2, ... , EPN1, EPN2, ... o2-roles as mesos attributes of an agent
			// * readout cards as resources
			// * o2 machine types (FLP, EPN) as mesos-roles so that other frameworks never get
			//   offers for stuff that doesn't belong to them i.e. readout cards

			// Walk through the environment structure and find out if the current []offers satisfies
			// what we need.
			env, err := state.environments.Environment(envIdToDeploy.UUID())
			if err != nil {
				log.WithPrefix("scheduler").WithField("error", err.Error()).
					Error("cannot get environment for uuid")
			} else {
				if !env.Sm.Is("ENV_STANDBY") {
					log.WithPrefix("scheduler").
						Error("cannot deploy new environment if not in STANDBY")
				}

				log.Debug("environment lock")

				env.Mu.Lock()

				log.Debug("about to compute topology")
				// ComputeTopology proposes a topology from a list of offers based on:
				//	* attribute o2kind
				//	* attribute o2role
				//	* NOT resources, because we use up the whole machine anyway
				//	* NOT roles, because those are a higher level filter
				offersUsed, topology, err := env.ComputeTopology(offers)
				if err != nil {
					// catastrophic failure for this resourceOffers round
				}
				env.Mu.Unlock()

				log.Debug("environment unlock")

				environmentsChanged = []uuid.Array{env.Id().Array()}

				//var environmentsChanged = make([]uuid.Array, 0)
				//for _, id := range state.environments.Ids() {
				//	env, err := state.environments.Environment(id)
				//	if err != nil {
				//		log.WithPrefix("scheduler").WithField("error", err.Error()).
				//			Error("cannot fetch environment for id")
				//		continue
				//	}
				//	environmentsChanged = append(environmentsChanged, env.Id().Array())
				//}

				// 1 offer per box!
				for i := range offersUsed {
					var (
						remainingResources= mesos.Resources(offers[i].Resources)
						tasks= []mesos.TaskInfo{}
					)

					log.WithPrefix("scheduler").WithFields(logrus.Fields{
						"offerId":   offers[i].ID.Value,
						"resources": remainingResources.String(),
					}).Debug("processing offer")

					remainingResourcesFlattened := resources.Flatten(remainingResources)

					// avoid the expense of computing these if we can...
					if state.config.summaryMetrics && state.config.resourceTypeMetrics {
						for name, resType := range resources.TypesOf(remainingResourcesFlattened...) {
							if resType == mesos.SCALAR {
								sum, _ := name.Sum(remainingResourcesFlattened...)
								state.metricsAPI.offeredResources(sum.GetScalar().GetValue(), name.String())
							}
						}
					}

					log.Debug("state lock")
					state.Lock()

					allocation, err := func(offer mesos.Offer, topology map[string]environment.Allocation) (environment.Allocation, error) {
						if attrIdx := environment.IndexOfAttribute(offer.Attributes, "o2role");
							attrIdx > -1 {
							return topology[offer.Attributes[attrIdx].GetText().GetValue()], nil
						}
						return environment.Allocation{}, errors.New("no allocation for role")
					}(offersUsed[i], topology)
					if err != nil {
						log.WithField("offerId", offersUsed[i].ID.Value).
							Error("cannot get allocation for offer, this should never happen")
						state.Unlock()
						log.Debug("state unlock")
						continue
					}

					processPath := allocation.Role.Process.Command
					task := mesos.TaskInfo{
						Name:      "Mesos Task " + allocation.TaskId,
						TaskID:    mesos.TaskID{allocation.TaskId},
						AgentID:   offersUsed[i].AgentID,
						Executor:  state.executor,
						Resources: offersUsed[i].Resources,
						Command: &mesos.CommandInfo{
							Value:     &processPath,
							Arguments: allocation.Role.Process.Args,
						},
					}

					log.WithFields(logrus.Fields{
						"taskId":     allocation.TaskId,
						"offerId":    offersUsed[i].ID.Value,
						"executorId": state.executor.ExecutorID.Value,
					}).Debug("launching task")

					tasks = append(tasks, task)

					/*

					// Account for executor resources as well before building tasks
					var wantsExecutorResources mesos.Resources
					if len(offers[i].ExecutorIDs) == 0 {
						wantsExecutorResources = mesos.Resources(state.executor.Resources)
					}

					taskWantsResources := state.wantsTaskResources.Plus(wantsExecutorResources...)
					for state.tasksLaunched < state.totalTasks && resources.ContainsAll(remainingResourcesFlattened, taskWantsResources) {
						state.tasksLaunched++
						taskID := state.tasksLaunched

						if state.config.verbose {
							log.Println("Launching task " + strconv.Itoa(taskID) + " using offer " + offers[i].ID.Value)
						}

						task := mesos.TaskInfo{
							TaskID:   mesos.TaskID{Value: strconv.Itoa(taskID)},
							AgentID:  offers[i].AgentID,
							Executor: state.executor,
							Resources: resources.Find(
								resources.Flatten(state.wantsTaskResources, resources.Role(state.role).Assign()),
								remainingResources...,
							),
						}
						task.Name = "Task " + task.TaskID.Value

						// Remove the resources we just assigned to the new task from the remaningResources
						// list...
						remainingResources.Subtract(task.Resources...)

						// ...and enqueue the task to the list of tasks to be shipped to executors.
						tasks = append(tasks, task)

						remainingResourcesFlattened = resources.Flatten(remainingResources)
					}
					*/
					state.Unlock()
					log.Debug("state unlock")

					// build Accept call to launch all of the tasks we've assembled
					accept := calls.Accept(
						calls.OfferOperations{calls.OpLaunch(tasks...)}.WithOffers(offersUsed[i].ID),
					).With(callOption) // handles refuseSeconds etc.

					// send Accept call to mesos
					err = calls.CallNoData(ctx, state.cli, accept)
					if err != nil {
						log.WithPrefix("scheduler").WithField("error", err.Error()).
							Error("failed to launch tasks")
					} else {
						if n := len(tasks); n > 0 {
							tasksLaunchedThisCycle += n
							log.WithPrefix("scheduler").WithField("tasks", n).
								Info("tasks launched")
						} else {
							offersDeclined++
						}
					}
				}
			}
		}
		// Notify listeners...
		select {
		case state.resourceOffersDone <- environmentsChanged:
			log.WithPrefix("scheduler").
				Debug("notified listeners on resourceOffers done")
		default:
			log.WithPrefix("scheduler").
				Debug("no listeners notified")
		}

		// Update metrics...
		state.metricsAPI.offersDeclined.Int(offersDeclined)
		state.metricsAPI.tasksLaunched.Int(tasksLaunchedThisCycle)
		if state.config.summaryMetrics {
			state.metricsAPI.launchesPerOfferCycle(float64(tasksLaunchedThisCycle))
		}
		var msg string
		if tasksLaunchedThisCycle == 0 {
			msg = "offers cycle complete, no tasks launched"
		} else {
			msg = "offers cycle complete, tasks launched"
		}
		log.WithPrefix("scheduler").WithField("tasks", tasksLaunchedThisCycle).Debug(msg)
		return nil
	}
}

// statusUpdate handles an incoming UPDATE event.
// This func runs after acknowledgement.
func statusUpdate(state *internalState) events.HandlerFunc {
	return func(ctx context.Context, e *scheduler.Event) error {
		s := e.GetUpdate().GetStatus()
		if state.config.verbose {
			log.WithPrefix("scheduler").WithFields(logrus.Fields{
				"task":		s.TaskID.Value,
				"state":	s.GetState().String(),
				"message":	s.GetMessage(),
			}).Debug("task status update received")
		}

		// What's the new task state?
		switch st := s.GetState(); st {
		case mesos.TASK_FINISHED:
			state.Lock()
			state.tasksFinished++
			state.metricsAPI.tasksFinished()

			// TODO: this should not quit when all tasks are done, but rather do some transition
			/*
			if state.tasksFinished == state.totalTasks {
				log.Println("Mission accomplished, all tasks completed. Terminating scheduler.")
				state.shutdown()
			} else {
				tryReviveOffers(ctx, state)
			}*/
			state.Unlock()

		case mesos.TASK_LOST, mesos.TASK_KILLED, mesos.TASK_FAILED, mesos.TASK_ERROR:
			state.Lock()
			state.err = errors.New("Exiting because task " + s.GetTaskID().Value +
				" is in an unexpected state " + st.String() +
				" with reason " + s.GetReason().String() +
				" from source " + s.GetSource().String() +
				" with message '" + s.GetMessage() + "'")
			state.Unlock()
			state.shutdown()
		}
		return nil
	}
}

// tryReviveOffers sends a REVIVE call to Mesos. With this we clear all filters we might previously
// have set through ACCEPT or DECLINE calls, in the hope that Mesos then sends us new resource offers.
// This should generally run when we have received a TASK_FINISHED for some tasks, and we have more
// tasks to run.
func tryReviveOffers(ctx context.Context, state *internalState) {
	// limit the rate at which we request offer revival
	select {
	case <-state.reviveTokens:
		// not done yet, revive offers!
		doReviveOffers(ctx, state)
	default:
		// noop
	}
}

func doReviveOffers(ctx context.Context, state *internalState) {
	err := calls.CallNoData(ctx, state.cli, calls.Revive())
	if err != nil {
		log.WithPrefix("scheduler").WithField("error", err.Error()).
			Error("failed to revive offers")
		return
	}
}

// logAllEvents logs every observed event; this is somewhat expensive to do so it only happens if
// the config is verbose.
func logAllEvents() eventrules.Rule {
	return func(ctx context.Context, e *scheduler.Event, err error, ch eventrules.Chain) (context.Context, *scheduler.Event, error) {
		log.WithPrefix("scheduler").WithField("event", fmt.Sprintf("%+v", *e)).
			Debug("incoming event")
		return ch(ctx, e, err)
	}
}

// eventMetrics logs metrics for every processed API event
func eventMetrics(metricsAPI *metricsAPI, clock func() time.Time, timingMetrics bool) eventrules.Rule {
	timed := metricsAPI.eventReceivedLatency
	if !timingMetrics {
		timed = nil
	}
	harness := xmetrics.NewHarness(metricsAPI.eventReceivedCount, metricsAPI.eventErrorCount, timed, clock)
	return eventrules.Metrics(harness, nil)
}

// callMetrics logs metrics for every outgoing Mesos call
func callMetrics(metricsAPI *metricsAPI, clock func() time.Time, timingMetrics bool) callrules.Rule {
	timed := metricsAPI.callLatency
	if !timingMetrics {
		timed = nil
	}
	harness := xmetrics.NewHarness(metricsAPI.callCount, metricsAPI.callErrorCount, timed, clock)
	return callrules.Metrics(harness, nil)
}

// logCalls logs a specific message string when a particular call-type is observed
func logCalls(messages map[scheduler.Call_Type]string) callrules.Rule {
	return func(ctx context.Context, c *scheduler.Call, r mesos.Response, err error, ch callrules.Chain) (context.Context, *scheduler.Call, mesos.Response, error) {
		if message, ok := messages[c.GetType()]; ok {
			log.WithPrefix("scheduler").Info(message)
		}
		return ch(ctx, c, r, err)
	}
}
