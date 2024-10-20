/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2017-2018 CERN and copyright holders of ALICE O².
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

package task

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/AliceO2Group/Control/apricot"
	"github.com/AliceO2Group/Control/common"
	"github.com/AliceO2Group/Control/common/controlmode"
	"github.com/AliceO2Group/Control/common/logger/infologger"
	"github.com/AliceO2Group/Control/common/utils"
	"github.com/AliceO2Group/Control/common/utils/uid"
	"github.com/AliceO2Group/Control/core/task/channel"
	"github.com/AliceO2Group/Control/core/task/schedutil"
	"github.com/AliceO2Group/Control/core/the"
	"github.com/spf13/viper"

	"github.com/AliceO2Group/Control/common/event"
	"github.com/AliceO2Group/Control/core/controlcommands"
	"github.com/AliceO2Group/Control/core/task/constraint"
	pb "github.com/AliceO2Group/Control/executor/protos"
	mesos "github.com/mesos/mesos-go/api/v1/lib"
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
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
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
	state *schedulerState,
	fidStore store.Singleton) error {
	// Set up communication from controller to state machine.
	go func() {
		for {
			receivedEvent := <-schedEventsCh
			switch {
			case receivedEvent == scheduler.Event_SUBSCRIBED:
				if state.sm.Is("INITIAL") {
					state.sm.Event(context.Background(), "CONNECT")
				}
			}
		}
	}()

	// Set up communication from state machine to controller
	go func() {
		for {
			<-state.reviveOffersTrg
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
		schedutil.BuildFrameworkInfo(),
		state.cli, /* controller.Option...: */
		controller.WithEventHandler(state.buildEventHandler(fidStore)),
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
			log.WithPrefix("scheduler").
				Info("disconnected")
		}),
	)
}

// buildEventHandler generates and returns a handler to process events received
// from the subscription. The handler is then passed as controller.Option to
// controller.Run.
func (state *schedulerState) buildEventHandler(fidStore store.Singleton) events.Handler {
	// disable brief logs when verbose logs are enabled (there's no sense logging twice!)
	logger := controller.LogEvents(nil).Unless(viper.GetBool("verbose"))

	return eventrules.New( /* eventrules.Rule... */
		logAllEvents().If(viper.GetBool("verbose")),
		eventMetrics(state.metricsAPI, time.Now, viper.GetBool("summaryMetrics")),
		controller.LiftErrors().DropOnError(),
		eventrules.HandleF(state.notifyStateMachine()),
	).Handle(events.Handlers{
		// scheduler.Event_Type: events.Handler
		scheduler.Event_FAILURE: logger.HandleF(state.failure), // wrapper + print error
		scheduler.Event_OFFERS:  state.trackOffersReceived().HandleF(state.resourceOffers(fidStore)),
		scheduler.Event_UPDATE:  controller.AckStatusUpdates(state.cli).AndThen().HandleF(state.statusUpdate()),
		scheduler.Event_SUBSCRIBED: eventrules.New(
			logger,
			controller.TrackSubscription(fidStore, viper.GetDuration("mesosFailoverTimeout")),
			eventrules.New().HandleF(state.reconciliationCall()),
		),
		scheduler.Event_MESSAGE: eventrules.HandleF(state.incomingMessageHandler()),
	}.Otherwise(logger.HandleEvent))
}

// Channel the event type of the newly received event to an asynchronous dispatcher
// in runSchedulerController
func (state *schedulerState) notifyStateMachine() events.HandlerFunc {
	return func(ctx context.Context, e *scheduler.Event) error {
		schedEventsCh <- e.GetType()
		return nil
	}
}

// Implicit Reconciliation Call that sends an empty list of tasks and the master responds
// with the latest state for all currently known non-terminal tasks.
func (state *schedulerState) reconciliationCall() events.HandlerFunc {
	return func(ctx context.Context, e *scheduler.Event) error {
		reconcileCall := calls.Reconcile(calls.ReconcileTasks(nil))
		_ = calls.CallNoData(ctx, state.cli, reconcileCall)
		return nil
	}
}

// Update metrics when we receive an offer
func (state *schedulerState) trackOffersReceived() eventrules.Rule {
	return func(ctx context.Context, e *scheduler.Event, err error, chain eventrules.Chain) (context.Context, *scheduler.Event, error) {
		if err == nil {
			state.metricsAPI.offersReceived.Int(len(e.GetOffers().GetOffers()))
		}
		return chain(ctx, e, err)
	}
}

// Handle an incoming Event_FAILURE, which may be a failure in the executor or
// in the Mesos agent.
func (state *schedulerState) failure(_ context.Context, e *scheduler.Event) error {
	var (
		f              = e.GetFailure()
		eid, aid, stat = f.ExecutorID, f.AgentID, f.Status
	)
	if eid != nil {
		// executor failed..
		fields := logrus.Fields{
			"executor": eid.Value,
		}
		if aid != nil {
			fields["agent"] = aid.Value
			host := state.getAgentCacheHostname(*aid)
			fields["srcHost"] = host
			detector, err := apricot.Instance().GetDetectorForHost(host)
			if err == nil {
				fields["detector"] = detector
			}
		}
		if stat != nil {
			fields["error"] = strconv.Itoa(int(*stat))
		}
		log.WithPrefix("scheduler").
			WithFields(fields).
			WithField("level", infologger.IL_Support).
			Error("executor failed")
		state.taskman.internalEventCh <- event.NewExecutorFailedEvent(eid)
	} else if aid != nil {
		// agent failed..
		fields := logrus.Fields{}

		fields["agent"] = aid.Value
		host := state.getAgentCacheHostname(*aid)
		fields["srcHost"] = host
		detector, err := apricot.Instance().GetDetectorForHost(host)
		if err == nil {
			fields["detector"] = detector
		}

		if stat != nil {
			fields["error"] = strconv.Itoa(int(*stat))
		}
		log.WithPrefix("scheduler").
			WithFields(fields).
			WithField("level", infologger.IL_Support).
			Error("agent failed")
		state.taskman.internalEventCh <- event.NewAgentFailedEvent(aid)
	}
	return nil
}

func (state *schedulerState) getAgentCacheHostname(id mesos.AgentID) string {
	if state == nil ||
		state.taskman == nil {
		return ""
	}
	if entry := state.taskman.AgentCache.Get(id); entry != nil {
		return entry.Hostname
	}
	return ""
}

// Handler for Event_MESSAGE
func (state *schedulerState) incomingMessageHandler() events.HandlerFunc {
	// instantiate map of MCtargets, command IDs and timeouts here
	// what should happen
	// sendCommand sends a command, pushes the targets list, command id and timeout (maybe
	// through a channel) to a structure accessible here.
	// then when we receive a response, if its id, target and timeout is satisfied by one and
	// only one entry in the list, we signal back to commandqueue
	// otherwise, we log and ignore.
	return func(ctx context.Context, e *scheduler.Event) (err error) {

		mesosMessage := e.GetMessage()
		if mesosMessage == nil {
			err = errors.New("message handler got bad MESSAGE")
			log.WithPrefix("scheduler").
				WithError(err).
				Warning("message handler cannot continue")
			return
		}
		agentId := mesosMessage.GetAgentID()
		executorId := mesosMessage.GetExecutorID()
		if len(agentId.GetValue()) == 0 || len(executorId.GetValue()) == 0 {
			err = errors.New("message handler got MESSAGE with no valid sender")
			log.WithPrefix("scheduler").
				WithFields(logrus.Fields{
					"agentId":    agentId.GetValue(),
					"executorId": executorId.GetValue(),
					"error":      err.Error(),
				}).
				Warning("message handler cannot continue")
			return
		}

		data := mesosMessage.GetData()

		var incomingType struct {
			MessageType string `json:"_messageType"`
		}
		err = json.Unmarshal(data, &incomingType)
		if err != nil {
			return
		}

		switch incomingType.MessageType {
		case "DeviceEvent":
			var incomingEvent struct {
				Type   pb.DeviceEventType      `json:"type"`
				Origin event.DeviceEventOrigin `json:"origin"`
				Labels map[string]string       `json:"labels"`
			}
			err = json.Unmarshal(data, &incomingEvent)
			if err != nil {
				return
			}
			envId := uid.NilID()
			if len(incomingEvent.Labels) > 0 {
				envIdS, ok := incomingEvent.Labels["environmentId"]
				if ok {
					envId, err = uid.FromString(envIdS)
					if err != nil {
						envId = uid.NilID()
					}
				}
			}

			ev := event.NewDeviceEvent(incomingEvent.Origin, incomingEvent.Type)
			if ev != nil {
				ev.SetLabels(incomingEvent.Labels)

				err = json.Unmarshal(data, &ev)
				if err != nil {
					return
				}
				state.taskman.internalEventCh <- ev
				//state.handleDeviceEvent(ev)
			} else {
				log.WithFields(logrus.Fields{
					"type":       incomingEvent.Type.String(),
					"originTask": incomingEvent.Origin.TaskId.Value,
					"partition":  envId.String(),
				}).
					Error("cannot handle incoming device event")
			}

		case "MesosCommandResponse":
			var incomingCommand struct {
				CommandName string `json:"name"`
			}
			err = json.Unmarshal(data, &incomingCommand)
			if err != nil {
				return
			}

			log.WithPrefix("scheduler").
				WithField("commandName", incomingCommand.CommandName).
				Trace("processing incoming MESSAGE")
			switch incomingCommand.CommandName {
			case "MesosCommand_TriggerHook":
				var res controlcommands.MesosCommandResponse_TriggerHook
				err = json.Unmarshal(data, &res)
				if err != nil {
					log.WithPrefix("scheduler").WithFields(logrus.Fields{
						"commandName": incomingCommand.CommandName,
						"agentId":     agentId.GetValue(),
						"executorId":  executorId.GetValue(),
						"message":     string(data[:]),
						"error":       err.Error(),
					}).
						Error("cannot unmarshal incoming MESSAGE")
					return
				}
				sender := controlcommands.MesosCommandTarget{
					AgentId:    agentId,
					ExecutorId: executorId,
					TaskId:     mesos.TaskID{Value: res.TaskId},
				}

				go func() {
					state.servent.ProcessResponse(&res, sender)
				}()
				return
			case "MesosCommand_Transition":
				var res controlcommands.MesosCommandResponse_Transition
				err = json.Unmarshal(data, &res)
				if err != nil {
					log.WithPrefix("scheduler").WithFields(logrus.Fields{
						"commandName": incomingCommand.CommandName,
						"agentId":     agentId.GetValue(),
						"executorId":  executorId.GetValue(),
						"message":     string(data[:]),
						"error":       err.Error(),
					}).
						Error("cannot unmarshal incoming MESSAGE")
					return
				}
				sender := controlcommands.MesosCommandTarget{
					AgentId:    agentId,
					ExecutorId: executorId,
					TaskId:     mesos.TaskID{Value: res.TaskId},
				}

				go func() {
					taskmanMessage := NewTaskStateMessage(res.TaskId, res.CurrentState)
					state.taskman.MessageChannel <- taskmanMessage

					// servent should be inside taskman and eventually
					// all this handling.
					state.servent.ProcessResponse(&res, sender)
				}()
				return
			default:
				return errors.New(fmt.Sprintf("unrecognized response for controlcommand %s", incomingCommand.CommandName))
			}
		case "AnnounceTaskPIDEvent":
			var taskMessage event.AnnounceTaskPIDEvent
			err = json.Unmarshal(data, &taskMessage)
			if err != nil {
				return
			}

			t := state.taskman.GetTask(taskMessage.GetTaskId())
			if t != nil {
				t.setTaskPID(taskMessage.GetTaskPID())
			}
		}
		return
	}
}

// Handler for Event_OFFERS
func (state *schedulerState) resourceOffers(fidStore store.Singleton) events.HandlerFunc {
	return func(ctx context.Context, e *scheduler.Event) error {
		timeResourceOffersCall := time.Now()
		var (
			offers                 = e.GetOffers().GetOffers()
			callOption             = calls.RefuseSeconds(time.Second) //calls.RefuseSecondsWithJitter(state.random, state.config.maxRefuseSeconds)
			tasksLaunchedThisCycle = 0
			offersDeclined         = 0
		)

		if viper.GetBool("veryVerbose") {
			var (
				prettyOffers []string
				offerIds     []string
			)
			for i := range offers {
				prettyOffer, _ := json.MarshalIndent(offers[i], "", "\t")
				prettyOffers = append(prettyOffers, string(prettyOffer))
				offerIds = append(offerIds, offers[i].ID.Value)
			}
			log.WithPrefix("scheduler").WithFields(logrus.Fields{
				"offerIds": strings.Join(offerIds, ", "),
				//"offers":	strings.Join(prettyOffers, "\n"),
				"offers": len(offerIds),
			}).
				Trace("received offers")
		}

		var descriptorsStillToDeploy Descriptors
		envId := uid.NilID()

		var deploymentRequestPayload *ResourceOffersDeploymentRequest

		// receive deployment request from channel, if any
		select {
		case deploymentRequestPayload = <-state.tasksToDeploy:
			if deploymentRequestPayload == nil {
				break
			}
			descriptorsStillToDeploy = deploymentRequestPayload.tasksToDeploy
			envId = deploymentRequestPayload.envId

			if viper.GetBool("veryVerbose") {
				rolePaths := make([]string, len(descriptorsStillToDeploy))
				taskClasses := make([]string, len(descriptorsStillToDeploy))
				for i, d := range descriptorsStillToDeploy {
					rolePaths[i] = d.TaskRole.GetPath()
					taskClasses[i] = d.TaskClassName
				}
				log.WithPrefix("scheduler").
					WithField("partition", envId.String()).
					WithFields(logrus.Fields{
						"roles":       strings.Join(rolePaths, ", "),
						"classes":     strings.Join(taskClasses, ", "),
						"descriptors": len(descriptorsStillToDeploy),
					}).
					Debugf("received %d descriptors for tasks to deploy on this offers round", len(deploymentRequestPayload.tasksToDeploy))
				utils.TimeTrack(timeResourceOffersCall, "resourceOffers: start to descriptors channel receive", log.WithField("descriptors", len(descriptorsStillToDeploy)))
			}
		default:
			if viper.GetBool("veryVerbose") {
				log.WithPrefix("scheduler").
					Trace("no roles need deployment")
			}
		}

		timeGotDescriptors := time.Now()

		machinesUsed := make(map[string]struct{})

		// by default we get ready to decline all offers
		offerIDsToDecline := make(map[mesos.OfferID]struct{}, len(offers))
		for i := range offers {
			offerIDsToDecline[offers[i].ID] = struct{}{}
		}

		tasksDeployed := make(DeploymentMap)
		tasksDeployedMutex := sync.Mutex{}

		// list of descriptors that we find impossible to deploy due to wants/constraints
		descriptorsUndeployable := make(Descriptors, 0)

		if len(descriptorsStillToDeploy) > 0 {
			// 3 ways to make decisions
			// * FLP1, FLP2, ... , EPN1, EPN2, ... o2-roles as mesos attributes of an agent
			// * readout cards as resources
			// * o2 machine types (FLP, EPN) as mesos-roles so that other frameworks never get
			//   offers for stuff that doesn't belong to them i.e. readout cards

			// Walk through the roles list and find out if the current []offers satisfies
			// what we need.

			log.WithPrefix("scheduler").
				WithField("partition", envId.String()).
				Debug("about to deploy workflow tasks")

			var err error

			// We make a map[Descriptor]constraint.Constraints and for each descriptor to deploy we
			// fill it with the pre-computed total constraints for that Descriptor.

			descriptorConstraints := state.taskman.BuildDescriptorConstraints(descriptorsStillToDeploy)

			utils.TimeTrack(timeGotDescriptors, "resourceOffers: descriptors channel receive to constraints built", log.
				WithField("descriptors", len(descriptorsStillToDeploy)).
				WithField("partition", envId.String()))
			timePreProcessing := time.Now()

			// Pre-processing: for each descriptorConstraint, if it includes a machine_id, means the descriptor can only
			// match a single offer. If such a descriptorConstraint is unsatisfiable, we should bail out early without
			// even trying to match it to an offer.

			// Here's where we accumulate descriptors that we know can only be matched to a single offer, provided the
			// offer can satisfy constraints, which we don't know yet.
			offerDescriptorsPrematchToDeploy := make(map[mesos.OfferID]Descriptors)

			// Each offer *must* have a machine_id, otherwise the O² system instance is broken at install time
			offersByMachineId := make(map[string]mesos.Offer)
			for _, offer := range offers {
				machineIdAttr := ""
				for _, attr := range offer.Attributes {
					if attr.GetName() == "machine_id" {
						machineIdAttr = attr.GetText().GetValue()
						break
					}
				}
				if machineIdAttr != "" {
					offersByMachineId[machineIdAttr] = offer
				}
			}

			for i := len(descriptorsStillToDeploy) - 1; i >= 0; i-- {
				descriptor := descriptorsStillToDeploy[i]
				requiredMachineId := ""
				for _, descriptorConstraint := range descriptorConstraints[descriptor] {
					if descriptorConstraint.Attribute == "machine_id" {
						requiredMachineId = descriptorConstraint.Value
						break
					}
				}
				if requiredMachineId != "" {
					// We have a constraint on the machine_id, so we need to find an offer that matches it.
					// If we don't find any, we can bail out early.
					offer, found := offersByMachineId[requiredMachineId]
					if found {
						// We found an offer that matches the machine_id constraint, so we can add the descriptor to the
						// pre-match list. It doesn't mean the offer can be accepted straight away, but it means that if
						// the descriptor has any hope of being deployed, it will only be on this offer, provided other
						// constraints and resources are satisfied.
						if offerDescriptorsPrematchToDeploy[offer.ID] == nil {
							offerDescriptorsPrematchToDeploy[offer.ID] = make(Descriptors, 0)
						}
						offerDescriptorsPrematchToDeploy[offer.ID] = append(offerDescriptorsPrematchToDeploy[offer.ID], descriptor)
						descriptorsStillToDeploy = append(descriptorsStillToDeploy[:i], descriptorsStillToDeploy[i+1:]...)
					} else {
						// We have a constraint on the machine_id, but we didn't find any offer that matches it.
						// We can bail out early.
						descriptorsUndeployable = append(descriptorsUndeployable, descriptor)
						descriptorsStillToDeploy = append(descriptorsStillToDeploy[:i], descriptorsStillToDeploy[i+1:]...)
						log.WithField("partition", envId.String()).
							WithField("descriptor", descriptor.TaskClassName).
							Errorf("no resource offer for required host %s, deployment will be aborted", requiredMachineId)
					}
				}
			}

			utils.TimeTrack(timePreProcessing, "resourceOffers: constraints built to offers pre-processing done", log.
				WithField("descriptors", len(descriptorsStillToDeploy)).
				WithField("partition", envId.String()))
			timeForOffers := time.Now()

			// We protect the descriptors structures with a mutex, within the scope of the current offers round, because
			// concurrent stuff starts to happen here.
			descriptorsMu := sync.Mutex{}

			if len(descriptorsUndeployable) == 0 { // we still have hope to deploy something

				// Parallelized offer processing, exhaustive search for each offer
				var offerWaitGroup sync.WaitGroup
				offerWaitGroup.Add(len(offers))

				for offerIndex, _ := range offers {
					go func(offerIndex int) {
						defer offerWaitGroup.Done()

						offer := offers[offerIndex]

						timeSingleOffer := time.Now()
						var (
							remainingResourcesInOffer        = mesos.Resources(offer.Resources)
							taskInfosToLaunchForCurrentOffer = make([]mesos.TaskInfo, 0)
							tasksDeployedForCurrentOffer     = make(DeploymentMap)
							targetExecutorId                 = mesos.ExecutorID{}
						)

						// If there are no executors provided by the offer,
						// we start a new one by generating a new ID
						if len(offer.ExecutorIDs) == 0 {
							targetExecutorId.Value = uid.New().String()
							log.WithField("executorId", targetExecutorId.Value).
								WithField("offerHost", offer.GetHostname()).
								WithField("level", infologger.IL_Support).
								Info("received offer without executor ID, will start new executor if accepted")
						} else {
							targetExecutorId.Value = offer.ExecutorIDs[0].Value
							if len(offer.ExecutorIDs) == 1 {
								log.WithField("executorId", targetExecutorId.Value).
									WithField("offerHost", offer.GetHostname()).
									WithField("level", infologger.IL_Support).
									Info("received offer with one executor ID, will use existing executor")
							} else if len(offer.ExecutorIDs) > 1 {
								log.WithField("executorId", targetExecutorId.Value).
									WithField("executorIds", offer.ExecutorIDs).
									WithField("offerHost", offer.GetHostname()).
									WithField("level", infologger.IL_Support).
									Warn("received offer with more than one executor ID, will use first one")
							}
						}

						host := offer.GetHostname()
						var detector string
						detector, err = apricot.Instance().GetDetectorForHost(host)
						if err != nil {
							detector = ""
						}

						log.WithPrefix("scheduler").
							WithFields(logrus.Fields{
								"offerId":   offer.ID.Value,
								"offerHost": host,
								"resources": remainingResourcesInOffer.String(),
								"partition": envId.String(),
								"detector":  detector,
							}).
							Debug("processing offer")

						remainingResourcesFlattened := resources.Flatten(remainingResourcesInOffer)

						// avoid the expense of computing these if we can...
						if viper.GetBool("summaryMetrics") && viper.GetBool("mesosResourceTypeMetrics") {
							for name, resType := range resources.TypesOf(remainingResourcesFlattened...) {
								if resType == mesos.SCALAR {
									sum, _ := name.Sum(remainingResourcesFlattened...)
									state.metricsAPI.offeredResources(sum.GetScalar().GetValue(), name.String())
								}
							}
						}

						log.WithPrefix("scheduler").
							WithField("partition", envId.String()).
							WithField("detector", detector).
							Trace("state lock to process descriptors to deploy")

						timeDescriptorsSection := time.Now()
						descriptorsMu.Lock()

						descriptorsPrematchToDeploy, prematchedDescriptorsExistForThisOffer := offerDescriptorsPrematchToDeploy[offer.ID]
						if prematchedDescriptorsExistForThisOffer {
						FOR_PREMATCH_DESCRIPTORS:
							for i := len(descriptorsPrematchToDeploy) - 1; i >= 0; i-- {
								descriptor := descriptorsPrematchToDeploy[i]

								descriptorDetector, ok := descriptor.TaskRole.GetVars().Get("detector")
								if !ok {
									descriptorDetector = ""
								}

								offerAttributes := constraint.Attributes(offer.Attributes)
								if !offerAttributes.Satisfy(descriptorConstraints[descriptor]) {
									if viper.GetBool("veryVerbose") {
										log.WithPrefix("scheduler").
											WithField("partition", envId.String()).
											WithField("detector", descriptorDetector).
											WithFields(logrus.Fields{
												"taskClass":   descriptor.TaskClassName,
												"constraints": descriptorConstraints[descriptor],
												"offerId":     offer.ID.Value,
												"resources":   remainingResourcesInOffer.String(),
												"attributes":  offerAttributes.String(),
											}).
											Trace("descriptor constraints not satisfied by pre-matched offer attributes, descriptor undeployable")
									}

									// we know this descriptor will never be satisfiable, no point in continuing
									descriptorsPrematchToDeploy = append(descriptorsPrematchToDeploy[:i], descriptorsPrematchToDeploy[i+1:]...)
									descriptorsUndeployable = append(descriptorsUndeployable, descriptor)

									break FOR_PREMATCH_DESCRIPTORS
								}
								log.WithPrefix("scheduler").
									WithField("partition", envId.String()).
									WithField("detector", descriptorDetector).
									Debug("pre-matched offer attributes satisfy constraints")

								var wants *Wants
								wants, err = state.taskman.GetWantsForDescriptor(descriptor, envId)
								if err != nil {
									log.WithPrefix("scheduler").
										WithError(err).
										WithField("partition", envId.String()).
										WithField("detector", descriptorDetector).
										WithFields(logrus.Fields{
											"class":       descriptor.TaskClassName,
											"constraints": descriptor.RoleConstraints.String(),
											"level":       infologger.IL_Devel,
											"offerHost":   offer.Hostname,
										}).
										Error("invalid task class: no task class or no resource demands for pre-matched descriptor, WILL NOT BE DEPLOYED")

									// we know this descriptor will never be satisfiable, no point in continuing
									descriptorsPrematchToDeploy = append(descriptorsPrematchToDeploy[:i], descriptorsPrematchToDeploy[i+1:]...)
									descriptorsUndeployable = append(descriptorsUndeployable, descriptor)

									break FOR_PREMATCH_DESCRIPTORS
								}
								if !Resources(remainingResourcesInOffer).Satisfy(wants) {
									if viper.GetBool("veryVerbose") {
										log.WithPrefix("scheduler").
											WithField("partition", envId.String()).
											WithField("detector", descriptorDetector).
											WithFields(logrus.Fields{
												"taskClass": descriptor.TaskClassName,
												"wants":     *wants,
												"offerId":   offer.ID.Value,
												"resources": remainingResourcesInOffer.String(),
												"level":     infologger.IL_Devel,
												"offerHost": offer.Hostname,
											}).
											Warn("descriptor wants not satisfied by pre-matched offer resources")
									}

									// we know this descriptor will never be satisfiable, no point in continuing
									descriptorsPrematchToDeploy = append(descriptorsPrematchToDeploy[:i], descriptorsPrematchToDeploy[i+1:]...)
									descriptorsUndeployable = append(descriptorsUndeployable, descriptor)

									break FOR_PREMATCH_DESCRIPTORS
								}

								var limits *Limits
								limits = state.taskman.GetLimitsForDescriptor(descriptor, envId)

								// Point of no return, we start subtracting resources
								taskPtr, mesosTaskInfo := makeTaskForMesosResources(
									state,
									&offer,
									descriptor,
									wants,
									limits,
									remainingResourcesInOffer,
									machinesUsed,
									targetExecutorId,
									envId,
									descriptorDetector,
									offerIDsToDecline,
								)
								if taskPtr == nil || mesosTaskInfo == nil {
									break FOR_PREMATCH_DESCRIPTORS
								}

								log.WithPrefix("scheduler").
									WithField("partition", envId.String()).
									WithField("detector", descriptorDetector).
									WithFields(logrus.Fields{
										"name":       mesosTaskInfo.Name,
										"taskId":     mesosTaskInfo.TaskID.Value,
										"offerId":    offer.ID.Value,
										"executorId": state.executor.ExecutorID.Value,
										"limits":     mesosTaskInfo.Limits,
									}).Debug("launching task")

								taskPtr.SendEvent(&event.TaskEvent{
									Name:      taskPtr.GetName(),
									TaskID:    mesosTaskInfo.TaskID.Value,
									State:     "LAUNCHED",
									Hostname:  taskPtr.hostname,
									ClassName: taskPtr.GetClassName(),
								})

								taskInfosToLaunchForCurrentOffer = append(taskInfosToLaunchForCurrentOffer, *mesosTaskInfo)
								descriptorsPrematchToDeploy = append(descriptorsPrematchToDeploy[:i], descriptorsPrematchToDeploy[i+1:]...)
								tasksDeployedForCurrentOffer[taskPtr] = descriptor

							}
						} // end FOR_PREMATCH_DESCRIPTORS

						if len(descriptorsUndeployable) == 0 { // still hope of deploying something

							// We iterate down over the descriptors, and we remove them as we match
						FOR_DESCRIPTORS:
							for i := len(descriptorsStillToDeploy) - 1; i >= 0; i-- {
								descriptor := descriptorsStillToDeploy[i]

								descriptorDetector, ok := descriptor.TaskRole.GetVars().Get("detector")
								if !ok {
									descriptorDetector = ""
								}

								offerAttributes := constraint.Attributes(offer.Attributes)
								if !offerAttributes.Satisfy(descriptorConstraints[descriptor]) {
									if viper.GetBool("veryVerbose") {
										log.WithPrefix("scheduler").
											WithField("partition", envId.String()).
											WithField("detector", descriptorDetector).
											WithFields(logrus.Fields{
												"taskClass":   descriptor.TaskClassName,
												"constraints": descriptorConstraints[descriptor],
												"offerId":     offer.ID.Value,
												"resources":   remainingResourcesInOffer.String(),
												"attributes":  offerAttributes.String(),
											}).
											Trace("descriptor constraints not satisfied by offer attributes")
									}
									continue FOR_DESCRIPTORS // next descriptor
								}
								log.WithPrefix("scheduler").
									WithField("partition", envId.String()).
									WithField("detector", descriptorDetector).
									Debug("offer attributes satisfy constraints")

								var wants *Wants
								wants, err = state.taskman.GetWantsForDescriptor(descriptor, envId)
								if err != nil {
									log.WithPrefix("scheduler").
										WithError(err).
										WithField("partition", envId.String()).
										WithField("detector", descriptorDetector).
										WithFields(logrus.Fields{
											"class":       descriptor.TaskClassName,
											"constraints": descriptor.RoleConstraints.String(),
											"level":       infologger.IL_Devel,
											"offerHost":   offer.Hostname,
										}).
										Error("invalid task class: no task class or no resource demands for descriptor, WILL NOT BE DEPLOYED")
									continue FOR_DESCRIPTORS // next descriptor
								}
								if !Resources(remainingResourcesInOffer).Satisfy(wants) {
									if viper.GetBool("veryVerbose") {
										log.WithPrefix("scheduler").
											WithField("partition", envId.String()).
											WithField("detector", descriptorDetector).
											WithFields(logrus.Fields{
												"taskClass": descriptor.TaskClassName,
												"wants":     *wants,
												"offerId":   offer.ID.Value,
												"resources": remainingResourcesInOffer.String(),
												"level":     infologger.IL_Devel,
												"offerHost": offer.Hostname,
											}).
											Warn("descriptor wants not satisfied by offer resources")
									}
									continue FOR_DESCRIPTORS // next descriptor
								}

								var limits *Limits
								limits = state.taskman.GetLimitsForDescriptor(descriptor, envId)

								// Point of no return, we start subtracting resources
								taskPtr, mesosTaskInfo := makeTaskForMesosResources(
									state,
									&offer,
									descriptor,
									wants,
									limits,
									remainingResourcesInOffer,
									machinesUsed,
									targetExecutorId,
									envId,
									descriptorDetector,
									offerIDsToDecline,
								)
								if taskPtr == nil || mesosTaskInfo == nil {
									continue FOR_DESCRIPTORS // next descriptor
								}

								log.WithPrefix("scheduler").
									WithField("partition", envId.String()).
									WithField("detector", descriptorDetector).
									WithFields(logrus.Fields{
										"name":       mesosTaskInfo.Name,
										"taskId":     mesosTaskInfo.TaskID.Value,
										"offerId":    offer.ID.Value,
										"executorId": state.executor.ExecutorID.Value,
										"limits":     mesosTaskInfo.Limits,
									}).Debug("launching task")

								taskPtr.SendEvent(&event.TaskEvent{
									Name:      taskPtr.GetName(),
									TaskID:    mesosTaskInfo.TaskID.Value,
									State:     "LAUNCHED",
									Hostname:  taskPtr.hostname,
									ClassName: taskPtr.GetClassName(),
								})

								taskInfosToLaunchForCurrentOffer = append(taskInfosToLaunchForCurrentOffer, *mesosTaskInfo)
								descriptorsStillToDeploy = append(descriptorsStillToDeploy[:i], descriptorsStillToDeploy[i+1:]...)
								tasksDeployedForCurrentOffer[taskPtr] = descriptor

							} // end FOR_DESCRIPTORS
						}
						descriptorsMu.Unlock()

						utils.TimeTrack(timeDescriptorsSection, "resourceOffers: single offer descriptors section", log.
							WithField("partition", envId.String()).
							WithField("offerHost", host).
							WithField("tasksDeployed", len(tasksDeployedForCurrentOffer)).
							WithField("descriptorsStillToDeploy", len(descriptorsStillToDeploy)).
							WithField("offers", len(offers)))
						timeOfferAcceptance := time.Now()

						log.WithPrefix("scheduler").
							WithField("offerHost", host).
							WithField("detector", detector).
							WithField("partition", envId.String()).
							Trace("state unlock")

						// build ACCEPT call to launch all of the tasks we've assembled
						accept := calls.Accept(
							calls.OfferOperations{calls.OpLaunch(taskInfosToLaunchForCurrentOffer...)}.WithOffers(offer.ID),
						).With(callOption) // handles refuseSeconds etc.

						// send ACCEPT call to mesos
						err = calls.CallNoData(ctx, state.cli, accept)
						if err != nil {
							log.WithPrefix("scheduler").
								WithError(err).
								WithField("detector", detector).
								WithField("partition", envId.String()).
								WithField("offerHost", host).
								Error("failed to launch tasks")
							// FIXME: we probably need to react to a failed ACCEPT here
						} else {
							if n := len(taskInfosToLaunchForCurrentOffer); n > 0 {
								tasksLaunchedThisCycle += n
								log.WithPrefix("scheduler").
									WithField("tasks", n).
									WithField("partition", envId.String()).
									WithField("detector", detector).
									WithField("level", infologger.IL_Support).
									WithField("offerHost", offer.Hostname).
									WithField("executorId", targetExecutorId.Value).
									Infof("launch request sent to %s: %d tasks", offer.Hostname, n)
								for _, taskInfo := range taskInfosToLaunchForCurrentOffer {
									log.WithPrefix("scheduler").
										WithFields(logrus.Fields{
											"executorId":   taskInfo.GetExecutor().ExecutorID.Value,
											"executorName": taskInfo.GetExecutor().GetName(),
											"agentId":      taskInfo.GetAgentID().Value,
											"taskId":       taskInfo.GetTaskID().Value,
											"level":        infologger.IL_Devel,
										}).
										WithField("offerHost", offer.Hostname).
										WithField("partition", envId.String()).
										WithField("detector", detector).
										Debug("task launch requested")
								}

								tasksDeployedMutex.Lock()
								// update deployment map
								for k, v := range tasksDeployedForCurrentOffer {
									tasksDeployed[k] = v
								}
								tasksDeployedMutex.Unlock()
							} else {
								offersDeclined++
							}
						}
						utils.TimeTrack(timeOfferAcceptance, "resourceOffers: single offer acceptance section", log.
							WithField("partition", envId.String()).
							WithField("detector", detector).
							WithField("tasksDeployed", len(tasksDeployedForCurrentOffer)).
							WithField("descriptorsStillToDeploy", len(descriptorsStillToDeploy)).
							WithField("offers", len(offers)).
							WithField("offerHost", offer.Hostname))

						utils.TimeTrack(timeSingleOffer, "resourceOffers: process and accept host offer", log.
							WithField("partition", envId.String()).
							WithField("detector", detector).
							WithField("tasksDeployed", len(tasksDeployedForCurrentOffer)).
							WithField("descriptorsStillToDeploy", len(descriptorsStillToDeploy)).
							WithField("offers", len(offers)).
							WithField("offerHost", offer.Hostname))

					}(offerIndex) // end for offer closure
				} // end for _, offer := range offers
				offerWaitGroup.Wait()

			}

			utils.TimeTrack(timeForOffers, "resourceOffers: pre-processing done to for_offers done (concurrent)", log.
				WithField("partition", envId.String()).
				WithField("descriptorsStillToDeploy", len(descriptorsStillToDeploy)).
				WithField("offers", len(offers)).
				WithField("offersDeclined", len(offerIDsToDecline)))
		} // end if len(descriptorsStillToDeploy) > 0

		timeBeforeDecline := time.Now()
		// build DECLINE call to reject offers we don't need any more
		declineSlice := make([]mesos.OfferID, len(offerIDsToDecline))
		j := 0
		for k := range offerIDsToDecline {
			declineSlice[j] = k
			j++
		}
		decline := calls.Decline(declineSlice...).With(callOption)

		if n := len(offerIDsToDecline); n > 0 {
			err := calls.CallNoData(ctx, state.cli, decline)
			if err != nil {
				log.WithPrefix("scheduler").
					WithField("error", err.Error()).
					WithField("partition", envId.String()).
					Error("failed to decline tasks")
			} else {
				log.WithPrefix("scheduler").
					WithField("offers", n).
					WithField("partition", envId.String()).
					Trace("offers declined")
			}
		} else {
			log.WithPrefix("scheduler").
				WithField("partition", envId.String()).
				Trace("no offers to decline")
		}

		// Notify listeners...
		if deploymentRequestPayload != nil {
			select {
			case deploymentRequestPayload.outcomeCh <- ResourceOffersOutcome{
				deployed:     tasksDeployed,
				undeployed:   descriptorsStillToDeploy,
				undeployable: descriptorsUndeployable,
			}:
				log.WithPrefix("scheduler").
					WithField("tasksDeployed", len(tasksDeployed)).
					WithField("partition", envId.String()).
					Trace("notified listeners on resourceOffers done")
			default:
				if viper.GetBool("veryVerbose") {
					log.WithPrefix("scheduler").
						WithField("partition", envId.String()).
						Trace("no listeners notified")
				}
			}
		}

		// Update metrics...
		state.metricsAPI.offersDeclined.Int(offersDeclined)
		state.metricsAPI.tasksLaunched.Int(tasksLaunchedThisCycle)
		if viper.GetBool("summaryMetrics") {
			state.metricsAPI.launchesPerOfferCycle(float64(tasksLaunchedThisCycle))
		}
		if tasksLaunchedThisCycle == 0 {
			if viper.GetBool("veryVerbose") {
				log.WithPrefix("scheduler").
					WithField("partition", envId.String()).
					WithField("level", infologger.IL_Devel).
					Trace("offers cycle complete, no tasks launched")
			}
		} else {
			machinesUsedSlice := func(machines map[string]struct{}) []string { // StringSet to StringSlice
				out := make([]string, len(machines))
				i := 0
				for k, _ := range machines {
					out[i] = k
					i++
				}
				return out
			}(machinesUsed)

			detectorsForHosts, _ := the.ConfSvc().GetDetectorsForHosts(machinesUsedSlice)

			log.WithPrefix("scheduler").
				WithField("partition", envId.String()).
				WithField("hosts", strings.Join(machinesUsedSlice, " ")).
				WithField("hostCount", len(machinesUsedSlice)).
				WithField("detectors", strings.Join(detectorsForHosts, " ")).
				Debugf("offers cycle complete, %d tasks launched for %d detectors", tasksLaunchedThisCycle, len(detectorsForHosts))
		}
		if viper.GetBool("veryVerbose") {
			utils.TimeTrack(timeBeforeDecline, "resourceOffers: decline-notify-return", log.
				WithField("partition", envId.String()).
				WithField("tasksLaunched", tasksLaunchedThisCycle).
				WithField("level", infologger.IL_Devel).
				WithField("offersDeclined", offersDeclined))
		}

		return nil
	}
}

// statusUpdate handles an incoming UPDATE event.
// This func runs after acknowledgement.
func (state *schedulerState) statusUpdate() events.HandlerFunc {
	return func(ctx context.Context, e *scheduler.Event) error {
		s := e.GetUpdate().GetStatus()

		fields := logrus.Fields{
			"task":    s.TaskID.Value,
			"state":   s.GetState().String(),
			"message": s.GetMessage(),
		}

		aid := s.GetAgentID()
		if aid != nil {
			host := state.getAgentCacheHostname(*aid)
			fields["srcHost"] = host
			detector, err := apricot.Instance().GetDetectorForHost(host)
			if err == nil {
				fields["detector"] = detector
			}
		}

		if viper.GetBool("verbose") {
			log.WithPrefix("scheduler").
				WithFields(fields).
				Trace("task status update received")
		}

		// What's the new task state?
		updatedState := s.GetState()
		switch updatedState {

		case mesos.TASK_FINISHED:
			// log.WithPrefix("scheduler").Debug("state lock")
			state.metricsAPI.tasksFinished()

		// FIXME: this should not quit when all tasks are done, but rather do some transition
		/*
			if state.tasksFinished == state.totalTasks {
				log.Println("Mission accomplished, all tasks completed. Terminating scheduler.")
				state.shutdown()
			} else {
				state.tryReviveOffers(ctx)
			}*/
		// log.WithPrefix("scheduler").Debug("state unlock")
		case mesos.TASK_RUNNING:
			log.WithPrefix("scheduler").
				WithFields(fields).
				Trace("task status update received")
		}

		taskmanMessage := NewTaskStatusMessage(s)
		state.taskman.MessageChannel <- taskmanMessage

		return nil
	}
}

// tryReviveOffers sends a REVIVE call to Mesos. With this we clear all filters we might previously
// have set through ACCEPT or DECLINE calls, in the hope that Mesos then sends us new resource offers.
// This should generally run when we have received a TASK_FINISHED for some tasks, and we have more
// tasks to run.
func (state *schedulerState) tryReviveOffers(ctx context.Context) {
	// limit the rate at which we request offer revival
	select {
	case <-state.reviveTokens:
		// not done yet, revive offers!
		doReviveOffers(ctx, state)
	default:
		// noop
	}
}

func doReviveOffers(ctx context.Context, state *schedulerState) {
	err := calls.CallNoData(ctx, state.cli, calls.Revive())
	if err != nil {
		log.WithPrefix("scheduler").
			WithField("error", err.Error()).
			Error("failed to revive offers")
		return
	}
	log.WithPrefix("scheduler").
		Debug("revive offers done")
}

func (state *schedulerState) killTask(ctx context.Context, receiver controlcommands.MesosCommandTarget) (err error) {
	killCall := calls.Kill(receiver.TaskId.GetValue(), receiver.AgentId.GetValue())

	err = calls.CallNoData(ctx, state.cli, killCall)
	return
}

func (state *schedulerState) sendCommand(ctx context.Context, command controlcommands.MesosCommand, receiver controlcommands.MesosCommandTarget) (err error) {
	var bytes []byte
	bytes, err = json.Marshal(command)
	if err != nil {
		return
	}

	message := calls.Message(receiver.AgentId.Value, receiver.ExecutorId.Value, bytes)

	err = calls.CallNoData(ctx, state.cli, message)

	detector := common.GetValueFromLabelerType(command, "detector")

	log.WithPrefix("scheduler").
		WithField("partition", command.GetEnvironmentId().String()).
		WithField("detector", detector).
		WithFields(logrus.Fields{
			"agentId":    receiver.AgentId.Value,
			"executorId": receiver.ExecutorId.Value,
			"payload":    string(bytes),
			"error": func() string {
				if err == nil {
					return "nil"
				} else {
					return err.Error()
				}
			}(),
		}).
		Trace("outgoing MESSAGE call")
	return err
}

// logAllEvents logs every observed event; this is somewhat expensive to do so it only happens if
// the config is verbose.
func logAllEvents() eventrules.Rule {
	return func(ctx context.Context, e *scheduler.Event, err error, ch eventrules.Chain) (context.Context, *scheduler.Event, error) {
		fields := logrus.Fields{
			"type": e.GetType().String(),
		}
		switch e.GetType() {
		case scheduler.Event_MESSAGE:
			fields["agentId"] = e.GetMessage().GetAgentID().Value
			fields["executorId"] = e.GetMessage().GetExecutorID().Value
		case scheduler.Event_UPDATE:
			status := e.GetUpdate().GetStatus()
			fields["agentId"] = status.GetAgentID().GetValue()
			fields["executorId"] = status.GetExecutorID().GetValue()
			fields["taskId"] = status.GetTaskID().Value
			fields["taskStatus"] = status.GetState().String()
		case scheduler.Event_OFFERS:
			off := e.GetOffers().Offers
			if len(off) > 0 {
				fields["agentId"] = off[0].GetAgentID().Value
				fields["hostname"] = off[0].GetHostname()
				exids := off[0].GetExecutorIDs()
				if len(exids) > 0 {
					fields["executorId"] = exids[0].GetValue()
				}
			}
			offerIds := make([]string, len(off))
			for i, _ := range off {
				offerIds[i] = off[i].GetID().Value
			}
			fields["offerIds"] = strings.Join(offerIds, ",")
		case scheduler.Event_SUBSCRIBED:
			fields["frameworkId"] = e.GetSubscribed().GetFrameworkID().Value
			fields["heartbeatInterval"] = fmt.Sprintf("%f", e.GetSubscribed().GetHeartbeatIntervalSeconds())
		case scheduler.Event_HEARTBEAT:
		default:
			fields["raw"] = fmt.Sprintf("%+v", *e)
		}
		log.WithPrefix("scheduler").
			WithFields(fields).
			Trace("incoming event")
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
			log.WithPrefix("scheduler").Debug(message)
		}
		return ch(ctx, c, r, err)
	}
}

func makeTaskForMesosResources(
	state *schedulerState,
	offer *mesos.Offer,
	descriptor *Descriptor,
	wants *Wants,
	limits *Limits,
	remainingResourcesInOffer mesos.Resources,
	machinesUsed map[string]struct{},
	targetExecutorId mesos.ExecutorID,
	envId uid.ID,
	descriptorDetector string,
	offerIDsToDecline map[mesos.OfferID]struct{},
) (*Task, *mesos.TaskInfo) {

	bindMap := make(channel.BindMap)
	for _, ch := range wants.InboundChannels {
		if ch.Addressing == channel.IPC {
			bindMap[ch.Name] = channel.NewBoundIpcEndpoint(ch.Transport)
		} else {
			var availPorts mesos.Ranges
			var ok bool
			availPorts, ok = resources.Ports(remainingResourcesInOffer...)
			if !ok {
				return nil, nil
			}
			// TODO: this can be optimized by excluding the base range outside the loop
			availPorts = availPorts.Remove(mesos.Value_Range{Begin: 0, End: 8999})
			port := availPorts.Min()
			builder := resources.Build().
				Name(resources.Name("ports")).
				Ranges(resources.BuildRanges().Span(port, port).Ranges)
			remainingResourcesInOffer.Subtract(builder.Resource)
			bindMap[ch.Name] = channel.NewBoundTcpEndpoint(port, ch.Transport)
		}

		// global channel alias processing
		if len(ch.Global) != 0 {
			bindMap["::"+ch.Global] = bindMap[ch.Name]
		}
	}

	agentForCache := AgentCacheInfo{
		AgentId:    offer.AgentID,
		Attributes: offer.Attributes,
		Hostname:   offer.Hostname,
	}
	state.taskman.AgentCache.Update(agentForCache) //thread safe
	machinesUsed[offer.Hostname] = struct{}{}

	taskPtr := state.taskman.newTaskForMesosOffer(offer, descriptor, bindMap, targetExecutorId)
	if taskPtr == nil {
		log.WithPrefix("scheduler").
			WithField("partition", envId.String()).
			WithField("detector", descriptorDetector).
			WithField("offerId", offer.ID.Value).
			Error("cannot get task for offer+descriptor, this should never happen")
		log.WithPrefix("scheduler").
			WithField("partition", envId.String()).
			WithField("detector", descriptorDetector).
			Trace("state unlock")
		return nil, nil
	}

	// Do not decline this offer
	_, contains := offerIDsToDecline[offer.ID]
	if contains {
		delete(offerIDsToDecline, offer.ID)
	}

	// Build the O² process to run as a mesos.CommandInfo, which we'll then JSON-serialize
	err := taskPtr.BuildTaskCommand(descriptor.TaskRole)
	if err != nil {
		log.WithPrefix("scheduler").
			WithField("offerId", offer.ID.Value).
			WithError(err).
			WithField("partition", envId.String()).
			WithField("detector", descriptorDetector).
			Error("cannot build task command")
		return nil, nil
	}
	cmd := taskPtr.GetTaskCommandInfo()

	// Claim the control port
	availPorts, ok := resources.Ports(remainingResourcesInOffer...)
	if !ok {
		return nil, nil
	}
	// The control port range starts at 47101
	// FIXME: make the control ports cutoff configurable
	availPorts = availPorts.Remove(mesos.Value_Range{Begin: 0, End: 29999})
	controlPort := availPorts.Min()
	builder := resources.Build().
		Name(resources.Name("ports")).
		Ranges(resources.BuildRanges().Span(controlPort, controlPort).Ranges)
	remainingResourcesInOffer.Subtract(builder.Resource)

	// Append control port to arguments
	// For the control port parameter and/or environment variable, see occ/OccGlobals.h
	if cmd.ControlMode != controlmode.BASIC &&
		cmd.ControlMode != controlmode.HOOK {
		cmd.ControlPort = controlPort
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%d", "OCC_CONTROL_PORT", controlPort))
	}

	if cmd.ControlMode == controlmode.FAIRMQ {
		cmd.Arguments = append(cmd.Arguments, "--control-port", strconv.FormatUint(controlPort, 10))
	}

	// Convenience function that scans through cmd.Env and appends the
	// given key-value pair if the given variable isn't already
	// provided by the user.
	fillEnvDefault := func(varName string, defaultValue string) {
		varIsUserProvided := false
		for _, envVar := range cmd.Env {
			if strings.HasPrefix(envVar, varName) {
				varIsUserProvided = true
				break
			}
		}
		if !varIsUserProvided {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", varName, defaultValue))
		}
	}

	// Iterated call of the above function for the given kv-map of
	// env var defaults
	for varName, defaultValue := range map[string]string{
		"O2_ROLE":   offer.Hostname,
		"O2_SYSTEM": "FLP",
	} {
		fillEnvDefault(varName, defaultValue)
	}

	runCommand := *cmd

	// Serialize the actual command to be passed to the executor
	var jsonCommand []byte
	jsonCommand, err = json.Marshal(&runCommand)
	if err != nil {
		log.WithPrefix("scheduler").
			WithField("partition", envId.String()).
			WithField("detector", descriptorDetector).
			WithFields(logrus.Fields{
				"error": err.Error(),
				"value": *runCommand.Value,
				"args":  runCommand.Arguments,
				"shell": *runCommand.Shell,
				"json":  jsonCommand,
			}).
			Error("cannot serialize mesos.CommandInfo for executor")

		return nil, nil
	}

	// Build resources request
	resourcesRequest := make(mesos.Resources, 0)
	resourcesRequest.Add1(resources.NewCPUs(wants.Cpu).Resource)
	resourcesRequest.Add1(resources.NewMemory(wants.Memory).Resource)
	portsBuilder := resources.BuildRanges()
	for _, rng := range wants.StaticPorts {
		portsBuilder = portsBuilder.Span(rng.Begin, rng.End)
	}
	for _, endpoint := range bindMap {
		// We only add the endpoint to the portsBuilder if it has a port
		if tcpEndpoint, endpointOk := endpoint.(channel.TcpEndpoint); endpointOk {
			portsBuilder = portsBuilder.Span(tcpEndpoint.Port, tcpEndpoint.Port)
		}
	}
	portsBuilder = portsBuilder.Span(controlPort, controlPort)

	portRanges := portsBuilder.Ranges.Sort().Squash()
	portsResources := resources.Build().Name(resources.Name("ports")).Ranges(portRanges)
	resourcesRequest.Add1(portsResources.Resource)

	// Append executor resources to request
	executorResources := mesos.Resources(state.executor.Resources)
	log.WithPrefix("scheduler").
		WithField("taskRole", taskPtr.GetParent().GetPath()).
		WithField("targetHost", offer.Hostname).
		WithField("partition", envId.String()).
		WithField("detector", descriptorDetector).
		WithField("taskResources", resourcesRequest).
		WithField("executorResources", executorResources).
		WithField("limits", limits).
		WithField("controlPort", controlPort).
		WithField("inboundChannels", func() string {
			accu := make([]string, len(wants.InboundChannels))
			for i := 0; i < len(wants.InboundChannels); i++ {
				channel := wants.InboundChannels[i]
				accu[i] = channel.Name
				if len(channel.Global) > 0 {
					accu[i] += fmt.Sprintf(" (global: %s)", channel.Global)
				}
				if endpoint, ok := bindMap[channel.Name]; ok {
					accu[i] += fmt.Sprintf(" -> %s", endpoint.GetAddress())
				}
			}
			return strings.Join(accu, ", ")
		}()).
		Debug("creating Mesos task")
	resourcesRequest.Add(executorResources...)

	newTaskId := taskPtr.GetTaskId()

	executor := state.CopyExecutorInfo()
	executor.ExecutorID.Value = taskPtr.GetExecutorId()
	envIdS := envId.String()

	mesosTaskInfo := mesos.TaskInfo{
		Name:      taskPtr.GetName(),
		TaskID:    mesos.TaskID{Value: newTaskId},
		AgentID:   offer.AgentID,
		Executor:  executor,
		Resources: resourcesRequest,
		Data:      jsonCommand, // this ends up in LAUNCH for the executor
		Labels: &mesos.Labels{Labels: []mesos.Label{
			{
				Key:   "environmentId",
				Value: &envIdS,
			},
			{
				Key:   "detector",
				Value: &descriptorDetector,
			},
		}},
		Limits: map[string]mesos.Value_Scalar{},
	}

	if limits != nil && limits.Cpu > 0 {
		mesosTaskInfo.Limits["cpus"] = *resources.NewCPUs(limits.Cpu).Resource.Scalar
	} else {
		mesosTaskInfo.Limits["cpus"] = mesos.Value_Scalar{Value: math.Inf(1)} // magic value for infinity
	}
	if limits != nil && limits.Memory > 0 {
		mesosTaskInfo.Limits["mem"] = *resources.NewMemory(limits.Memory).Resource.Scalar
	} else {
		mesosTaskInfo.Limits["mem"] = mesos.Value_Scalar{Value: math.Inf(1)} // magic value for infinity
	}

	// We must run the executor with a special LD_LIBRARY_PATH because
	// its InfoLogger binding is built with GCC-Toolchain
	ldLibPath, ok := agentForCache.Attributes.Get("executor_env_LD_LIBRARY_PATH")
	mesosTaskInfo.Executor.Command.Environment = &mesos.Environment{}
	if ok {
		mesosTaskInfo.Executor.Command.Environment.Variables =
			append(mesosTaskInfo.Executor.Command.Environment.Variables,
				mesos.Environment_Variable{
					Name:  "LD_LIBRARY_PATH",
					Value: proto.String(ldLibPath),
				})
	}

	return taskPtr, &mesosTaskInfo
}
