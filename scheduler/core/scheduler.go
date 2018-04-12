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
	"encoding/json"
	"strings"
	"github.com/teo/octl/scheduler/core/environment"
	"github.com/gogo/protobuf/proto"
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
			<- state.reviveOffersTrg
			doReviveOffers(ctx, state)
			state.reviveOffersTrg <- struct{}{}
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
		scheduler.Event_OFFERS:  trackOffersReceived(state).HandleF(resourceOffers(state, fidStore)),
		scheduler.Event_UPDATE:  controller.AckStatusUpdates(state.cli).AndThen().HandleF(statusUpdate(state)),
		scheduler.Event_SUBSCRIBED: eventrules.New(
			logger,
			controller.TrackSubscription(fidStore, state.config.mesosFailoverTimeout),
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
func resourceOffers(state *internalState, fidStore store.Singleton) events.HandlerFunc {
	return func(ctx context.Context, e *scheduler.Event) error {
		var (
			offers                 = e.GetOffers().GetOffers()
			callOption             = calls.RefuseSeconds(time.Second)//calls.RefuseSecondsWithJitter(state.random, state.config.maxRefuseSeconds)
			tasksLaunchedThisCycle = 0
			offersDeclined         = 0
		)

		if state.config.veryVerbose {
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

		var roleCfgsToDeploy map[string]environment.RoleCfg
		select {
		case roleCfgsToDeploy = <- state.rolesToDeploy:
			roleNames := make([]string, 0)
			for k, _ := range roleCfgsToDeploy {
				roleNames = append(roleNames, k)
			}
			log.WithPrefix("scheduler").WithField("roles", strings.Join(roleNames, ", ")).
				Debug("received roles to deploy on this offers round")
		default:
			log.WithPrefix("scheduler").Debug("no roles need deployment")
		}

		// by default we get ready to decline all offers
		offerIDsToDecline := make([]mesos.OfferID, len(offers))
		for i := range offers {
			offerIDsToDecline[i] = offers[i].ID
		}

		rolesDeployed := make(environment.Roles)

		if len(roleCfgsToDeploy) > 0 {
			// 3 ways to make decisions
			// * FLP1, FLP2, ... , EPN1, EPN2, ... o2-roles as mesos attributes of an agent
			// * readout cards as resources
			// * o2 machine types (FLP, EPN) as mesos-roles so that other frameworks never get
			//   offers for stuff that doesn't belong to them i.e. readout cards

			// Walk through the roles list and find out if the current []offers satisfies
			// what we need.

			log.WithPrefix("scheduler").Debug("about to compute topology")

			offersUsed, offersToDecline, rolesDeployed, err :=
				matchRoles(state.roleman, roleCfgsToDeploy, offers)
			if err != nil {
				log.WithPrefix("scheduler").
					WithField("error", err.Error()).
					Warning("could not match all required roles with Mesos offers")
			}

			offerIDsToDecline = make([]mesos.OfferID, len(offersToDecline))
			for i := range offersToDecline {
				offerIDsToDecline[i] = offersToDecline[i].ID
			}

			// 1 offer per box!
			for i := range offersUsed {
				var (
					remainingResources= mesos.Resources(offers[i].Resources)
					tasks = make([]mesos.TaskInfo, 0)
				)

				log.WithPrefix("scheduler").WithFields(logrus.Fields{
					"offerId":   offers[i].ID.Value,
					"resources": remainingResources.String(),
				}).Debug("processing offer")

				remainingResourcesFlattened := resources.Flatten(remainingResources)

				// avoid the expense of computing these if we can...
				if state.config.summaryMetrics && state.config.mesosResourceTypeMetrics {
					for name, resType := range resources.TypesOf(remainingResourcesFlattened...) {
						if resType == mesos.SCALAR {
							sum, _ := name.Sum(remainingResourcesFlattened...)
							state.metricsAPI.offeredResources(sum.GetScalar().GetValue(), name.String())
						}
					}
				}

				log.Debug("state lock")
				state.Lock()
				deployingRoleName, err := func(offer mesos.Offer, roles environment.Roles) (string, error) {
					for roleName, role := range roles {
						if role.GetOfferId() == offer.ID.GetValue() {
							return roleName, nil
						}
					}
					return "", errors.New("no roles for offer")
				}(offersUsed[i], rolesDeployed)
				if err != nil {
					log.WithPrefix("scheduler").
						WithField("offerId", offersUsed[i].ID.Value).
						Error("cannot get role for offer, this should never happen")
					state.Unlock()
					log.Debug("state unlock")
					continue
				}

				// Define the O² process to run as a mesos.CommandInfo, which we'll then JSON-serialize

				cmd := rolesDeployed[deployingRoleName].GetCommand()
				processPath := *cmd.Value
				runCommand := mesos.CommandInfo{
					Value:     proto.String(processPath),
					Arguments: cmd.Arguments,
					Shell:     proto.Bool(*cmd.Shell),
				}

				// Serialize the actual command to be passed to the executor
				jsonCommand, err := json.Marshal(runCommand)
				if err != nil {
					log.WithPrefix("scheduler").
						WithFields(logrus.Fields{
							"error": err.Error(),
							"value": *runCommand.Value,
							"args":  runCommand.Arguments,
							"shell": *runCommand.Shell,
							"json":  jsonCommand,
						}).
						Error("cannot serialize mesos.CommandInfo for executor")
					state.Unlock()
					log.Debug("state unlock")
					continue
				}

				resourcesRequest := mesos.Resources(offersUsed[i].Resources).Minus(mesos.Resources(state.executor.Resources)...)

				log.WithPrefix("scheduler").
					WithField("resources", resourcesRequest).
					Debug("creating Mesos task")

				newTaskId := rolesDeployed[deployingRoleName].GetTaskId()

				task := mesos.TaskInfo{
					Name:      "Mesos Task " + newTaskId,
					TaskID:    mesos.TaskID{Value: newTaskId},
					AgentID:   offersUsed[i].AgentID,
					Executor:  state.executor,
					Resources: resourcesRequest,
					Data:      jsonCommand,  // this ends up in LAUNCH for the executor
				}

				log.WithFields(logrus.Fields{
					"taskId":     newTaskId,
					"offerId":    offersUsed[i].ID.Value,
					"executorId": state.executor.ExecutorID.Value,
					"task":       task,
				}).Debug("launching task")

				tasks = append(tasks, task)

				state.Unlock()
				log.Debug("state unlock")

				// build ACCEPT call to launch all of the tasks we've assembled
				accept := calls.Accept(
					calls.OfferOperations{calls.OpLaunch(tasks...)}.WithOffers(offersUsed[i].ID),
				).With(callOption) // handles refuseSeconds etc.

				// send ACCEPT call to mesos
				err = calls.CallNoData(ctx, state.cli, accept)
				if err != nil {
					log.WithPrefix("scheduler").WithField("error", err.Error()).
						Error("failed to launch tasks")
					// FIXME: we probably need to react to a failed ACCEPT here
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


		// build DECLINE call to reject offers we don't need any more
		decline := calls.Decline(offerIDsToDecline...).With(callOption)

		if n := len(offerIDsToDecline); n > 0 {
			err := calls.CallNoData(ctx, state.cli, decline)
			if err != nil {
				log.WithPrefix("scheduler").WithField("error", err.Error()).
					Error("failed to decline tasks")
			} else {
				log.WithPrefix("scheduler").WithField("offers", n).
					Info("offers declined")
			}
		} else {
			log.WithPrefix("scheduler").Info("no offers to decline")
		}

		// Notify listeners...
		select {
		case state.resourceOffersDone <- rolesDeployed:
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
			log.WithPrefix("scheduler").Debug("state lock")
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
			log.WithPrefix("scheduler").Debug("state unlock")

		case mesos.TASK_LOST, mesos.TASK_KILLED, mesos.TASK_FAILED, mesos.TASK_ERROR:
			log.WithPrefix("scheduler").Debug("state lock")
			state.Lock()
			log.WithPrefix("scheduler").Debug("setting global error state")
			state.err = errors.New("task " + s.GetTaskID().Value +
				" is in an unexpected state " + st.String() +
				" with reason " + s.GetReason().String() +
				" from source " + s.GetSource().String() +
				" with message '" + s.GetMessage() + "'")
			state.Unlock()
			log.WithPrefix("scheduler").Debug("state unlock")
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
	log.WithPrefix("scheduler").Debug("revive offers done")
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
