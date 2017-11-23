package core

import (
	"context"
	"log"
	"time"

	"github.com/mesos/mesos-go/api/v1/lib/extras/scheduler/callrules"
	"github.com/mesos/mesos-go/api/v1/lib/extras/store"
	"github.com/mesos/mesos-go/api/v1/lib/scheduler"
	"github.com/gin-gonic/gin"
	"github.com/looplab/fsm"
)


// Run is the entry point for this scheduler.
// TODO: refactor Config to reflect our specific requirements
func Run(cfg Config) error {
	log.Printf("Scheduler running with configuration: %+v", cfg)

	// TODO: this is the FSM of each O² process, for further reference
	//fsm := fsm.NewFSM(
	//	"STANDBY",
	//	fsm.Events{
	//		{Name: "CONFIGURE", Src: []string{"STANDBY", "CONFIGURED"},           Dst: "CONFIGURED"},
	//		{Name: "START",     Src: []string{"CONFIGURED"},                      Dst: "RUNNING"},
	//		{Name: "STOP",      Src: []string{"RUNNING", "PAUSED"},               Dst: "CONFIGURED"},
	//		{Name: "PAUSE",     Src: []string{"RUNNING"},                         Dst: "PAUSED"},
	//		{Name: "RESUME",    Src: []string{"PAUSED"},                          Dst: "RUNNING"},
	//		{Name: "EXIT",      Src: []string{"CONFIGURED", "STANDBY"},           Dst: "FINAL"},
	//		{Name: "GO_ERROR",  Src: []string{"CONFIGURED", "RUNNING", "PAUSED"}, Dst: "ERROR"},
	//		{Name: "RESET",     Src: []string{"ERROR"},                           Dst: "STANDBY"},
	//	},
	//	fsm.Callbacks{},
	//)


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

	state.sm = fsm.NewFSM(
		"INITIAL",
		fsm.Events{
			{Name: "CONNECT",       Src: []string{"INITIAL"},                       Dst: "CONNECTED"},
			{Name: "SETUP_LAYOUT",  Src: []string{"CONNECTED", "READY"},            Dst: "READY"},
			{Name: "RUN_ACTIVITY",  Src: []string{"READY"},                         Dst: "RUNNING"},
			{Name: "STOP_ACTIVITY", Src: []string{"RUNNING"},                       Dst: "READY"},
			{Name: "EXIT",          Src: []string{"CONNECTED", "READY"},            Dst: "FINAL"},
			{Name: "GO_ERROR",      Src: []string{"CONNECTED", "READY", "RUNNING"}, Dst: "ERROR"},
			{Name: "RESET",         Src: []string{"ERROR"},                         Dst: "INITIAL"},
		},
		fsm.Callbacks{
			"after_event": func(e *fsm.Event) {
				log.Printf("State transition for event %s, source: %s, dest: %s.", e.Event, e.Src, e.Dst)
			},
			"enter_INITIAL": func(e *fsm.Event) {
				go func() {
					err = runSchedulerController(ctx, state, fidStore)
					state.RLock()
					defer state.RUnlock()
					if state.err != nil {
						err = state.err
						e.FSM.Event("ERROR")
					} else {
						e.FSM.Event("EXIT")
					}
				}()
			},
			"after_CONNECT": func(e *fsm.Event) {
			},
		},
	)


	// We now start the Control server
	if !state.config.verbose {
		gin.SetMode(gin.ReleaseMode)
	}
	controlRouter := newControlRouter(state, fidStore)
	err = controlRouter.Run(":8080")

	return err
}

// end Run
