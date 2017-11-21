package app

import (
	"context"
	"errors"
	"io"
	"log"
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
	"github.com/gin-gonic/gin"
)

var (
	RegistrationMinBackoff = 1 * time.Second
	RegistrationMaxBackoff = 15 * time.Second
)

// StateError is returned when the system encounters an unresolvable state transition error and
// should likely exit.
type StateError string

func (err StateError) Error() string { return string(err) }

// Run is the entry point for this scheduler.
// TODO: refactor Config to reflect our specific requirements
func Run(cfg Config) error {
	log.Printf("Scheduler running with configuration: %+v", cfg)

	// We create a context and use its cancel func as a shutdown func to release
	// all resources. The shutdown func is stored in the app.internalState.
	ctx, cancel := context.WithCancel(context.Background())

	// This only runs once to create a container for all data which comprises the
	// scheduler's state.
	// Included values.
	//   * cfg          - a config struct, TODO: overhaul.
	//   * totalTasks   - as specified in the cfg, TODO: this needs to be fancier
	//   * reviveTokens - FIXME: from the example, not sure of its purpose yet.
	//                    It's supposed to be a chan which yields a struct{}{}
	//                    at intervals between minWait and maxWait. The more you
	//                    read from the chan, the more you have to wait for the next
	//                    ping. I believe this might be used for reconnect retries?
	//                    TODO: understand Notifier vs BurstNotifier.
	//   * wantsTaskResources - builds a mesos.Resources value to describe resource
	//                    requirements for all tasks, TODO: this needs to be fancier
	//                    to allow configurable resource requirements for different
	//                    kinds of tasks.
	//   * executor     - a struct for all executor related information, including
	//                    binary path, artifact server, container image, executor-
	//                    specific resources, configuration, etc.
	//   * metricsApi   - a pointer to the metrics collector.
	//   * cli          - keeps an object of interface calls.Caller, which is the
	//                    main interface a Mesos scheduler should consume.
	//                    The interface itself is generated. The implementation is
	//                    provided by mesos-go as a HTTP API client.
	//   * random       - a random generator instance.
	//   * shutdown     - a shutdown func
	// It also keeps count of the tasks launched/finished
	state, err := newInternalState(cfg, cancel)
	if err != nil {
		return err
	}

	// TODO(jdef) how to track/handle timeout errors that occur for SUBSCRIBE calls? we should
	// probably tolerate X number of subsequent subscribe failures before bailing. we'll need
	// to track the lastCallAttempted along with subsequentSubscribeTimeouts.

	// store.Singleton is a thread-safe abstraction to load and store and string,
	// provided by mesos-go.
	// We also make sure that a log message is printed when the FrameworkID changes.
	fidStore := store.DecorateSingleton(
		store.NewInMemorySingleton(),
		store.DoSet().AndThen(func(_ store.Setter, v string, _ error) error {
			log.Println("New FrameworkID stored into fidStore singleton:", v)
			return nil
		}))

	// callrules.New returns a Rules and accept a bunch of Rule values as arguments.
	// WithFrameworkID returns a Rule which injects a frameworkID to outgoing calls.
	// logCalls returns a rule which prints to the log all calls of type SUBSCRIBE.
	// callMetrics logs metrics for every outgoing call.
	state.cli = callrules.New(
		callrules.WithFrameworkID(store.GetIgnoreErrors(fidStore)),
		logCalls(map[scheduler.Call_Type]string{scheduler.Call_SUBSCRIBE: "[SUBSCRIBE] Connecting..."}),
		callMetrics(state.metricsAPI, time.Now, state.config.summaryMetrics),
	).Caller(state.cli)

	// We now start the Control server
	if !state.config.verbose {
		gin.SetMode(gin.ReleaseMode)
	}
	controlRouter := gin.Default()
	controlRouter.GET("/status", func(c *gin.Context){
		state.RLock()
		defer state.RUnlock()
		msg := gin.H{
			"tasksLaunched": state.tasksLaunched,
			"tasksFinished": state.tasksFinished,
			"totalTasks": state.totalTasks,
			"frameworkId": store.GetIgnoreErrors(fidStore)(),
			"config" : state.config,
		}
		if state.config.verbose {
			c.IndentedJSON(200, msg)
		} else {
			c.JSON(200, msg)
		}
	})
	go controlRouter.Run(":8080")

	// The controller starts here, it takes care of connecting to Mesos and subscribing
	// as well as resubscribing if the connection is dropped.
	// It also handles incoming events on the subscription connection.
	//
	// buildFrameworkInfo returns a *mesos.FrameworkInfo which includes the framework
	// ID, as well as additional information such as Roles, WebUI URL, etc.
	err = controller.Run(
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
					log.Println(err)
				}
				if _, ok := err.(StateError); ok {
					state.shutdown()
				}
				return
			}
			log.Println("disconnected")
		}),
	)
	state.RLock()
	defer state.RUnlock()
	if state.err != nil {
		err = state.err
	}
	return err
}

// end Run

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
		msg := "executor '" + eid.Value + "' terminated"
		if aid != nil {
			msg += " on agent '" + aid.Value + "'"
		}
		if stat != nil {
			msg += " with status=" + strconv.Itoa(int(*stat))
		}
		log.Println(msg)
	} else if aid != nil {
		// agent failed..
		log.Println("agent '" + aid.Value + "' terminated")
	}
	return nil
}

// Handler for Event_OFFERS
func resourceOffers(state *internalState) events.HandlerFunc {
	return func(ctx context.Context, e *scheduler.Event) error {
		var (
			offers                 = e.GetOffers().GetOffers()
			callOption             = calls.RefuseSecondsWithJitter(state.random, state.config.maxRefuseSeconds)
			tasksLaunchedThisCycle = 0
			offersDeclined         = 0
		)

		// 1 offer per box!
		// TODO: implement role or attribute checking and/or reservations
		for i := range offers {
			var (
				remainingResources = mesos.Resources(offers[i].Resources)
				tasks              = []mesos.TaskInfo{}
			)

			if state.config.verbose {
				log.Println("Received offer id '" + offers[i].ID.Value +
					"' with resources " + remainingResources.String())
			}

			var wantsExecutorResources mesos.Resources
			if len(offers[i].ExecutorIDs) == 0 {
				wantsExecutorResources = mesos.Resources(state.executor.Resources)
			}

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

			// Account for executor resources as well before building tasks
			state.Lock()
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
			state.Unlock()

			// build Accept call to launch all of the tasks we've assembled
			accept := calls.Accept(
				calls.OfferOperations{calls.OpLaunch(tasks...)}.WithOffers(offers[i].ID),
			).With(callOption) // handles refuseSeconds etc.

			// send Accept call to mesos
			err := calls.CallNoData(ctx, state.cli, accept)
			if err != nil {
				log.Printf("Failed to launch tasks: %+v", err)
			} else {
				if n := len(tasks); n > 0 {
					tasksLaunchedThisCycle += n
					log.Printf("Successfully launched %+v tasks.", n)
				} else {
					offersDeclined++
				}
			}
		}

		// Update metrics...
		state.metricsAPI.offersDeclined.Int(offersDeclined)
		state.metricsAPI.tasksLaunched.Int(tasksLaunchedThisCycle)
		if state.config.summaryMetrics {
			state.metricsAPI.launchesPerOfferCycle(float64(tasksLaunchedThisCycle))
		}
		if tasksLaunchedThisCycle == 0 && state.config.verbose {
			log.Println("zero tasks launched this OFFERS cycle")
		} else {
			log.Printf("%+v tasks launched this OFFERS cycle.\n", tasksLaunchedThisCycle)
		}
		return nil
	}
}

// statusUpdate handles an incoming UPDATE event.
// This func runs after acknowledgement.
func statusUpdate(state *internalState) events.HandlerFunc {
	return func(ctx context.Context, e *scheduler.Event) error {
		s := e.GetUpdate().GetStatus()
		if state.config.verbose {
			msg := "Task " + s.TaskID.Value + " is in state " + s.GetState().String()
			if m := s.GetMessage(); m != "" {
				msg += " with message '" + m + "'"
			}
			log.Println(msg)
		}

		// What's the new task state?
		switch st := s.GetState(); st {
		case mesos.TASK_FINISHED:
			state.Lock()
			state.tasksFinished++
			state.metricsAPI.tasksFinished()

			if state.tasksFinished == state.totalTasks {
				log.Println("Mission accomplished, all tasks completed. Terminating scheduler.")
				state.shutdown()
			} else {
				tryReviveOffers(ctx, state)
			}
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
		err := calls.CallNoData(ctx, state.cli, calls.Revive())
		if err != nil {
			log.Printf("Failed to revive offers: %+v", err)
			return
		}
	default:
		// noop
	}
}

// logAllEvents logs every observed event; this is somewhat expensive to do so it only happens if
// the config is verbose.
func logAllEvents() eventrules.Rule {
	return func(ctx context.Context, e *scheduler.Event, err error, ch eventrules.Chain) (context.Context, *scheduler.Event, error) {
		log.Printf("Incoming EVENT: %+v\n", *e)
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
			log.Println(message)
		}
		return ch(ctx, c, r, err)
	}
}
