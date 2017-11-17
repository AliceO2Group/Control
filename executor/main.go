package main

import (
	"context"
	"errors"
	"io"
	"log"
	"net/url"
	"os"
	"time"

	"github.com/mesos/mesos-go/api/v1/lib"
	"github.com/mesos/mesos-go/api/v1/lib/backoff"
	"github.com/mesos/mesos-go/api/v1/lib/encoding"
	"github.com/mesos/mesos-go/api/v1/lib/encoding/codecs"
	"github.com/mesos/mesos-go/api/v1/lib/executor"
	"github.com/mesos/mesos-go/api/v1/lib/executor/calls"
	"github.com/mesos/mesos-go/api/v1/lib/executor/config"
	"github.com/mesos/mesos-go/api/v1/lib/executor/events"
	"github.com/mesos/mesos-go/api/v1/lib/httpcli"
	"github.com/mesos/mesos-go/api/v1/lib/httpcli/httpexec"
	"github.com/pborman/uuid"
)

const (
	apiPath     = "/api/v1/executor"
	httpTimeout = 10 * time.Second
)

var errMustAbort = errors.New("executor received abort signal from Mesos, will attempt to re-subscribe")

// Entry point, reads configuration from environment variables.
func main() {
	cfg, err := config.FromEnv()
	if err != nil {
		log.Fatal("Failed to load configuration: " + err.Error())
	}
	log.Printf("Configuration loaded: %+v", cfg)
	run(cfg)
	os.Exit(0)
}

// mnaybeReconnect returns a backoff.Notifier chan if framework checkpointing is enabled.
func maybeReconnect(cfg config.Config) <-chan struct{} {
	if cfg.Checkpoint {
		return backoff.Notifier(1*time.Second, cfg.SubscriptionBackoffMax*3/4, nil)
	}
	return nil
}

// run is the actual entry point of the executor.
func run(cfg config.Config) {
	var (
		apiURL = url.URL{
			Scheme: "http", // TODO(jdef) make this configurable
			Host:   cfg.AgentEndpoint,
			Path:   apiPath,
		}
		http = httpcli.New(
			httpcli.Endpoint(apiURL.String()),
			httpcli.Codec(codecs.ByMediaType[codecs.MediaTypeProtobuf]),
			httpcli.Do(httpcli.With(httpcli.Timeout(httpTimeout))),
		)

		// Fill in the Framework and Executor IDs as call parameters
		callOptions = executor.CallOptions{
			calls.Framework(cfg.FrameworkID),
			calls.Executor(cfg.ExecutorID),
		}
		state = &internalState{
			// With this we inject the callOptions into all outgoing calls
			cli: calls.SenderWith(
				httpexec.NewSender(http.Send),
				callOptions...,
			),
			// The executor keeps lists of unacknowledged tasks and unacknowledged updates. In case of unexpected
			// disconnection, the executor should SUBSCRIBE again and send these lists to Mesos in the SUBSCRIBE
			// call.
			unackedTasks:   make(map[mesos.TaskID]mesos.TaskInfo),
			unackedUpdates: make(map[string]executor.Call_Update),
			failedTasks:    make(map[mesos.TaskID]mesos.TaskStatus),
		}
		subscriber = calls.SenderWith(
			// Here too, callOptions for all outgoing subscriber calls
			httpexec.NewSender(http.Send, httpcli.Close(true)),
			callOptions...,
		)

		// Chan which receives a struct every once in a while
		shouldReconnect = maybeReconnect(cfg)
		disconnected    = time.Now()
		handler         = buildEventHandler(state)
	)

	// Main loop for (re)subscription. Once we're subscribed, we jump into the event loop for handling the agent.
	for {
		func() {
			// We create the subscription call. If we haven't had an unclean disconnect, the lists of unacknowledged
			// tasks and updates are empty.
			subscribe := calls.Subscribe(unacknowledgedTasks(state), unacknowledgedUpdates(state))

			log.Println("Subscribing to agent for events..")
			//                           ↓ empty context       ↓ we get a plain RequestFunc from the executor.Call value
			resp, err := subscriber.Send(context.TODO(), calls.NonStreaming(subscribe))
			if resp != nil {
				defer resp.Close()
			}
			if err == nil {
				// We're officially connected, start decoding events
				err = eventLoop(state, resp, handler)
				// If we're out of the eventLoop, means a disconnect happened, willingly or not.
				disconnected = time.Now()
			}
			if err != nil && err != io.EOF {
				log.Println(err)
			} else {
				log.Println("Executor disconnected.")
			}
		}()
		if state.shouldQuit {
			log.Println("Gracefully shutting down because we were told to.")
			return
		}
		// The purpose of checkpointing is to handle recovery when mesos-agent exits.
		if !cfg.Checkpoint {
			log.Println("Gracefully exiting because framework checkpointing is NOT enabled.")
			return
		}
		if time.Now().Sub(disconnected) > cfg.RecoveryTimeout {
			log.Printf("Failed to re-establish subscription with agent within %v, aborting.", cfg.RecoveryTimeout)
			return
		}
		log.Println("Waiting for reconnect timeout")
		<-shouldReconnect // wait for some amount of time before retrying subscription
	}
}

// unacknowledgedTasks generates the value of the UnacknowledgedTasks field of a Subscribe call.
func unacknowledgedTasks(state *internalState) (result []mesos.TaskInfo) {
	if n := len(state.unackedTasks); n > 0 {
		result = make([]mesos.TaskInfo, 0, n)
		for k := range state.unackedTasks {
			result = append(result, state.unackedTasks[k])
		}
	}
	return
}

// unacknowledgedUpdates generates the value of the UnacknowledgedUpdates field of a Subscribe call.
func unacknowledgedUpdates(state *internalState) (result []executor.Call_Update) {
	if n := len(state.unackedUpdates); n > 0 {
		result = make([]executor.Call_Update, 0, n)
		for k := range state.unackedUpdates {
			result = append(result, state.unackedUpdates[k])
		}
	}
	return
}

// eventLoop dispatches incoming events from mesos-agent to the events.Handler (built in buildEventhandler).
func eventLoop(state *internalState, decoder encoding.Decoder, h events.Handler) (err error) {
	log.Println("listening for events from agent...")
	ctx := context.TODO() // dummy context
	for err == nil && !state.shouldQuit {
		// housekeeping
		sendFailedTasks(state)

		var e executor.Event
		if err = decoder.Decode(&e); err == nil {
			err = h.HandleEvent(ctx, &e)
		}
	}
	return err
}

// buildEventHandler builds an events.Handler, whose HandleEvent is triggered from the eventLoop.
func buildEventHandler(state *internalState) events.Handler {
	return events.HandlerFuncs{
		executor.Event_SUBSCRIBED: func(_ context.Context, e *executor.Event) error {
			log.Println("Mesos agent reports executor SUBSCRIBED.")
			// With this event we get FrameworkInfo, ExecutorInfo, AgentInfo:
			state.framework = e.Subscribed.FrameworkInfo
			state.executor = e.Subscribed.ExecutorInfo
			state.agent = e.Subscribed.AgentInfo
			return nil
		},
		executor.Event_LAUNCH: func(_ context.Context, e *executor.Event) error {
			// Launch one task. We're not handling LAUNCH_GROUP.
			log.Println("LAUNCH received")
			launch(state, e.Launch.Task)
			return nil
		},
		executor.Event_KILL: func(_ context.Context, e *executor.Event) error {
			// TODO: ask the process to kindly transition to the end state and quit, and if it doesn't do so gracefully,
			// kill it and report back with a MESSAGE.
			log.Println("Warning: KILL not implemented")
			return nil
		},
		executor.Event_ACKNOWLEDGED: func(_ context.Context, e *executor.Event) error {
			delete(state.unackedTasks, e.Acknowledged.TaskID)
			delete(state.unackedUpdates, string(e.Acknowledged.UUID))
			return nil
		},
		executor.Event_MESSAGE: func(_ context.Context, e *executor.Event) error {
			log.Printf("MESSAGE: received %d bytes of message data", len(e.Message.Data))
			log.Println(e.Message.Data)
			return nil
		},
		executor.Event_SHUTDOWN: func(_ context.Context, e *executor.Event) error {
			log.Println("SHUTDOWN received")
			state.shouldQuit = true
			return nil
		},
		executor.Event_ERROR: func(_ context.Context, e *executor.Event) error {
			log.Println("ERROR received")
			return errMustAbort
		},
	}.Otherwise(func(_ context.Context, e *executor.Event) error {
		log.Fatal("unexpected event", e)
		return nil
	})
}

// sendFailedTasks runs on every iteration of eventLoop to send an UPDATE on any failed tasks to the agent.
func sendFailedTasks(state *internalState) {
	for taskID, status := range state.failedTasks {
		updateErr := update(state, status)
		if updateErr != nil {
			log.Printf("failed to send status update for task %s: %+v", taskID.Value, updateErr)
		} else {
			// If we have successfully notified Mesos, we clear our list of failed tasks.
			delete(state.failedTasks, taskID)
		}
	}
}

// launch tries to launch a task described by a mesos.TaskInfo.
func launch(state *internalState, task mesos.TaskInfo) {
	state.unackedTasks[task.TaskID] = task
	if task.GetCommand() != nil {
		log.Printf("Launching task: {Shell: %s, Value: %s, Arguments: %s}",
			task.GetCommand().Shell, task.GetCommand().Value, task.GetCommand().Arguments)
	} else {
		log.Println("Could not launch task: CommandInfo is nil.")
	}
	// TODO: launch task here?

	// send RUNNING
	status := newStatus(state, task.TaskID)
	status.State = mesos.TASK_RUNNING.Enum()
	err := update(state, status)
	if err != nil {
		log.Printf("Failed to send TASK_RUNNING for task %s: %+v", task.TaskID.Value, err)
		status.State = mesos.TASK_FAILED.Enum()
		status.Message = protoString(err.Error())
		state.failedTasks[task.TaskID] = status
		return
	}

	// send FINISHED
	status = newStatus(state, task.TaskID)
	status.State = mesos.TASK_FINISHED.Enum()
	err = update(state, status)
	if err != nil {
		log.Printf("failed to send TASK_FINISHED for task %s: %+v", task.TaskID.Value, err)
		status.State = mesos.TASK_FAILED.Enum()
		status.Message = protoString(err.Error())
		state.failedTasks[task.TaskID] = status
	}
}

// helper func to package strings up nicely for protobuf
// NOTE(jdef): if we need any more proto funcs like this, just import the
// proto package and use those.
func protoString(s string) *string { return &s }

// update sends UPDATE to agent.
func update(state *internalState, status mesos.TaskStatus) error {
	upd := calls.Update(status)
	resp, err := state.cli.Send(context.TODO(), calls.NonStreaming(upd))
	if resp != nil {
		resp.Close()
	}
	if err != nil {
		log.Printf("failed to send update: %+v", err)
		debugJSON(upd)
	} else {
		state.unackedUpdates[string(status.UUID)] = *upd.Update
	}
	return err
}

// newStatus constructs a new mesos.TaskStatus to describe a task.
func newStatus(state *internalState, id mesos.TaskID) mesos.TaskStatus {
	return mesos.TaskStatus{
		TaskID:     id,
		Source:     mesos.SOURCE_EXECUTOR.Enum(),
		ExecutorID: &state.executor.ExecutorID,
		UUID:       []byte(uuid.NewRandom()),
	}
}

// internalState of the executor.
type internalState struct {
	cli            calls.Sender
	cfg            config.Config
	framework      mesos.FrameworkInfo
	executor       mesos.ExecutorInfo
	agent          mesos.AgentInfo
	unackedTasks   map[mesos.TaskID]mesos.TaskInfo
	unackedUpdates map[string]executor.Call_Update
	failedTasks    map[mesos.TaskID]mesos.TaskStatus // send updates for these as we can
	shouldQuit     bool
}
