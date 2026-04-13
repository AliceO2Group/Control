/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2024 CERN and copyright holders of ALICE O².
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

package peanut

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/AliceO2Group/Control/executor/executorcmd/nopb"
	"github.com/AliceO2Group/Control/executor/executorcmd/transitioner/fairmq"
	pb "github.com/AliceO2Group/Control/executor/protos"
	"github.com/AliceO2Group/Control/occ/peanut/flatten"
)

// RunCLI runs peanut in non-interactive CLI mode.
// args should be os.Args[1:].
func RunCLI(args []string) error {
	fs := flag.NewFlagSet("peanut", flag.ExitOnError)
	addr := fs.String("addr", "localhost:47100", "OCC gRPC address (host:port)")
	mode := fs.String("mode", "fmq", "control mode: fmq (json codec, default) or direct (protobuf)")
	timeout := fs.Duration("timeout", 30*time.Second, "request timeout for unary calls")
	configFile := fs.String("config", "", "path to YAML/JSON file whose flattened key=value pairs are sent as arguments (inline key=val args take precedence)")
	fs.Usage = cliUsage
	_ = fs.Parse(args)

	cmds := fs.Args()
	if len(cmds) == 0 {
		cliUsage()
		return fmt.Errorf("no command specified")
	}

	conn, err := grpc.Dial(*addr, grpc.WithTransportCredentials(insecure.NewCredentials())) //nolint:staticcheck
	if err != nil {
		return fmt.Errorf("dial %s: %w", *addr, err)
	}
	defer conn.Close()

	var client pb.OccClient
	if *mode == "fmq" {
		client = nopb.NewOccClient(conn)
	} else {
		client = pb.NewOccClient(conn)
	}

	loadedKVs, err := cliLoadConfig(*configFile)
	if err != nil {
		return fmt.Errorf("config file: %w", err)
	}

	switch cmds[0] {
	case "get-state":
		ctx, cancel := context.WithTimeout(context.Background(), *timeout)
		defer cancel()
		reply, err := client.GetState(ctx, &pb.GetStateRequest{})
		if err != nil {
			return fmt.Errorf("GetState: %w", err)
		}
		fmt.Println(reply.GetState())

	case "transition":
		if len(cmds) < 3 {
			return fmt.Errorf("usage: transition <fromState> <toState> [key=val ...]")
		}
		from := strings.ToUpper(cmds[1])
		to := strings.ToUpper(cmds[2])
		kvs := cliMergeKVs(loadedKVs, cliParseKVs(cmds[3:]))

		ctx, cancel := context.WithTimeout(context.Background(), *timeout)
		defer cancel()

		if *mode == "fmq" {
			result, err := cliFMQTransition(ctx, client, from, to, kvs)
			if err != nil {
				return fmt.Errorf("transition %s→%s: %w", from, to, err)
			}
			fmt.Printf("ok  state=%s\n", result)
		} else {
			event := cliOCCEventForTransition(from, to)
			reply, err := client.Transition(ctx, &pb.TransitionRequest{
				SrcState:        from,
				TransitionEvent: event,
				Arguments:       cliKVsToEntries(kvs),
			})
			if err != nil {
				return fmt.Errorf("Transition: %w", err)
			}
			fmt.Printf("ok  state=%s trigger=%s\n", reply.GetState(), reply.GetTrigger())
		}

	case "direct-step":
		// Low-level single OCC gRPC call. Mirrors what the TUI does.
		// Usage: direct-step <srcState> <event> [key=val ...]
		if len(cmds) < 3 {
			return fmt.Errorf("usage: direct-step <srcState> <event> [key=val ...]\n  e.g. direct-step STANDBY CONFIGURE key=val")
		}
		kvs := cliMergeKVs(loadedKVs, cliParseKVs(cmds[3:]))
		ctx, cancel := context.WithTimeout(context.Background(), *timeout)
		defer cancel()
		reply, err := client.Transition(ctx, &pb.TransitionRequest{
			SrcState:        cmds[1],
			TransitionEvent: cmds[2],
			Arguments:       cliKVsToEntries(kvs),
		})
		if err != nil {
			return fmt.Errorf("occ-step: %w", err)
		}
		fmt.Printf("ok  state=%s trigger=%s\n", reply.GetState(), reply.GetTrigger())

	case "fmq-step":
		if len(cmds) < 3 {
			return fmt.Errorf("usage: fmq-step <srcFMQState> <fmqEvent> [key=val ...]\n  e.g. fmq-step IDLE \"INIT DEVICE\" key=val")
		}
		kvs := cliMergeKVs(loadedKVs, cliParseKVs(cmds[3:]))
		ctx, cancel := context.WithTimeout(context.Background(), *timeout)
		defer cancel()
		state, err := cliFMQDoStep(ctx, client, cmds[1], cmds[2], kvs)
		if err != nil {
			return fmt.Errorf("fmq-step: %w", err)
		}
		fmt.Printf("ok  fmq-state=%s  occ-state=%s\n", state, cliFMQToOCCState(state))

	case "state-stream":
		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer cancel()
		stream, err := client.StateStream(ctx, &pb.StateStreamRequest{})
		if err != nil {
			return fmt.Errorf("StateStream: %w", err)
		}
		if stream == nil {
			return fmt.Errorf("StateStream not supported by this server (try polling with get-state)")
		}
		fmt.Fprintf(os.Stderr, "streaming state updates from %s  (ctrl-c to stop)\n", *addr)
		for {
			msg, err := stream.Recv()
			if err != nil {
				if ctx.Err() != nil {
					return nil
				}
				return fmt.Errorf("StateStream recv: %w", err)
			}
			fmt.Printf("type=%-12s  state=%s\n", msg.GetType(), msg.GetState())
		}

	case "event-stream":
		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer cancel()
		stream, err := client.EventStream(ctx, &pb.EventStreamRequest{})
		if err != nil {
			return fmt.Errorf("EventStream: %w", err)
		}
		if stream == nil {
			return fmt.Errorf("EventStream not supported by this server")
		}
		fmt.Fprintf(os.Stderr, "streaming events from %s  (ctrl-c to stop)\n", *addr)
		for {
			msg, err := stream.Recv()
			if err != nil {
				if ctx.Err() != nil {
					return nil
				}
				return fmt.Errorf("EventStream recv: %w", err)
			}
			fmt.Printf("event=%s\n", msg.GetEvent())
		}

	default:
		return fmt.Errorf("unknown command %q — valid: get-state, transition, direct-step, fmq-step, state-stream, event-stream", cmds[0])
	}
	return nil
}

// cliFMQStateToOCC maps FairMQ device states to OCC states.
var cliFMQStateToOCC = map[string]string{
	fairmq.IDLE:                "STANDBY",
	fairmq.INITIALIZING_DEVICE: "STANDBY",
	fairmq.INITIALIZED:         "STANDBY",
	fairmq.BOUND:               "STANDBY",
	fairmq.DEVICE_READY:        "STANDBY",
	fairmq.READY:               "CONFIGURED",
	fairmq.RUNNING:             "RUNNING",
	fairmq.ERROR:               "ERROR",
}

func cliFMQToOCCState(fmqState string) string {
	if s, ok := cliFMQStateToOCC[strings.ToUpper(fmqState)]; ok {
		return s
	}
	return "STANDBY"
}

// fmqDoStep performs a single raw FairMQ gRPC Transition call with no output.
func fmqDoStep(ctx context.Context, client pb.OccClient, srcFMQ, event string, kvs map[string]string) (string, error) {
	reply, err := client.Transition(ctx, &pb.TransitionRequest{
		SrcState:        srcFMQ,
		TransitionEvent: event,
		Arguments:       cliKVsToEntries(kvs),
	})
	if err != nil {
		return "", fmt.Errorf("step %q failed: %w", event, err)
	}
	state := reply.GetState()
	if !reply.GetOk() {
		return state, fmt.Errorf("step %q not ok, state=%s", event, state)
	}
	return state, nil
}

// cliFMQDoStep wraps fmqDoStep with stderr progress output for CLI use.
func cliFMQDoStep(ctx context.Context, client pb.OccClient, srcFMQ, event string, kvs map[string]string) (string, error) {
	fmt.Fprintf(os.Stderr, "  fmq-step  src=%-20q  event=%q\n", srcFMQ, event)
	state, err := fmqDoStep(ctx, client, srcFMQ, event, kvs)
	ok := err == nil
	fmt.Fprintf(os.Stderr, "           result=%-20q  ok=%v\n", state, ok)
	return state, err
}

func cliFMQTransition(ctx context.Context, client pb.OccClient, from, to string, kvs map[string]string) (string, error) {
	switch {
	case from == "STANDBY" && to == "CONFIGURED":
		return cliFMQConfigure(ctx, client, kvs)
	case from == "CONFIGURED" && to == "STANDBY":
		return cliFMQReset(ctx, client, kvs)
	case from == "CONFIGURED" && to == "RUNNING":
		state, err := cliFMQDoStep(ctx, client, fairmq.READY, fairmq.EvtRUN, kvs)
		return cliFMQToOCCState(state), err
	case from == "RUNNING" && to == "CONFIGURED":
		state, err := cliFMQDoStep(ctx, client, fairmq.RUNNING, fairmq.EvtSTOP, kvs)
		return cliFMQToOCCState(state), err
	default:
		return from, fmt.Errorf("unsupported FairMQ transition %s → %s", from, to)
	}
}

// fmqStepErr formats a FairMQ step failure, omitting the cause when err is nil
// (state arrived but was wrong) to avoid a trailing ": <nil>" in the message.
func fmqStepErr(step, want, got string, err error) error {
	if err != nil {
		return fmt.Errorf("%s: expected %q got %q: %w", step, want, got, err)
	}
	return fmt.Errorf("%s: expected %q got %q", step, want, got)
}

func cliFMQConfigure(ctx context.Context, client pb.OccClient, args map[string]string) (string, error) {
	state, err := cliFMQDoStep(ctx, client, fairmq.IDLE, fairmq.EvtINIT_DEVICE, args)
	if err != nil || state != fairmq.INITIALIZING_DEVICE {
		return cliFMQToOCCState(state), fmqStepErr("INIT DEVICE", fairmq.INITIALIZING_DEVICE, state, err)
	}
	state, err = cliFMQDoStep(ctx, client, fairmq.INITIALIZING_DEVICE, fairmq.EvtCOMPLETE_INIT, nil)
	if err != nil || state != fairmq.INITIALIZED {
		return cliFMQToOCCState(state), fmqStepErr("COMPLETE INIT", fairmq.INITIALIZED, state, err)
	}
	state, err = cliFMQDoStep(ctx, client, fairmq.INITIALIZED, fairmq.EvtBIND, nil)
	if err != nil || state != fairmq.BOUND {
		cliFMQDoStep(ctx, client, fairmq.INITIALIZED, fairmq.EvtRESET_DEVICE, nil) // rollback
		return cliFMQToOCCState(state), fmqStepErr("BIND", fairmq.BOUND, state, err)
	}
	state, err = cliFMQDoStep(ctx, client, fairmq.BOUND, fairmq.EvtCONNECT, nil)
	if err != nil || state != fairmq.DEVICE_READY {
		cliFMQDoStep(ctx, client, fairmq.BOUND, fairmq.EvtRESET_DEVICE, nil) // rollback
		return cliFMQToOCCState(state), fmqStepErr("CONNECT", fairmq.DEVICE_READY, state, err)
	}
	state, err = cliFMQDoStep(ctx, client, fairmq.DEVICE_READY, fairmq.EvtINIT_TASK, nil)
	if err != nil || state != fairmq.READY {
		cliFMQDoStep(ctx, client, fairmq.DEVICE_READY, fairmq.EvtRESET_DEVICE, nil) // rollback
		return cliFMQToOCCState(state), fmqStepErr("INIT TASK", fairmq.READY, state, err)
	}
	return cliFMQToOCCState(state), nil
}

func cliFMQReset(ctx context.Context, client pb.OccClient, args map[string]string) (string, error) {
	state, err := cliFMQDoStep(ctx, client, fairmq.READY, fairmq.EvtRESET_TASK, nil)
	if err != nil || state != fairmq.DEVICE_READY {
		return cliFMQToOCCState(state), fmqStepErr("RESET TASK", fairmq.DEVICE_READY, state, err)
	}
	state, err = cliFMQDoStep(ctx, client, fairmq.DEVICE_READY, fairmq.EvtRESET_DEVICE, args)
	return cliFMQToOCCState(state), err
}

func cliOCCEventForTransition(from, to string) string {
	type edge struct{ from, to string }
	table := map[edge]string{
		{"STANDBY", "CONFIGURED"}: "CONFIGURE",
		{"CONFIGURED", "RUNNING"}: "START",
		{"RUNNING", "CONFIGURED"}: "STOP",
		{"CONFIGURED", "STANDBY"}: "RESET",
		{"ERROR", "STANDBY"}:      "RECOVER",
	}
	if ev, ok := table[edge{from, to}]; ok {
		return ev
	}
	return to
}

func cliKVsToEntries(kvs map[string]string) []*pb.ConfigEntry {
	entries := make([]*pb.ConfigEntry, 0, len(kvs))
	for k, v := range kvs {
		entries = append(entries, &pb.ConfigEntry{Key: k, Value: v})
	}
	return entries
}

func cliParseKVs(args []string) map[string]string {
	m := make(map[string]string, len(args))
	for _, kv := range args {
		parts := strings.SplitN(kv, "=", 2)
		if len(parts) == 2 {
			m[parts[0]] = parts[1]
		}
	}
	return m
}

// cliLoadConfig reads and flattens a YAML/JSON config file into a key=value map.
// Returns an empty map (not an error) if path is empty.
func cliLoadConfig(path string) (map[string]string, error) {
	if path == "" {
		return map[string]string{}, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read %q: %w", path, err)
	}
	flattened, err := flatten.FlattenString(string(data), "", flatten.DotStyle)
	if err != nil {
		return nil, fmt.Errorf("cannot flatten %q: %w", path, err)
	}
	var rawMap map[string]interface{}
	if err := json.Unmarshal([]byte(flattened), &rawMap); err != nil {
		return nil, fmt.Errorf("cannot parse flattened config: %w", err)
	}
	kvs := make(map[string]string, len(rawMap))
	for k, v := range rawMap {
		kvs[k] = fmt.Sprintf("%v", v)
	}
	return kvs, nil
}

// cliMergeKVs merges base and override maps; keys in override take precedence.
func cliMergeKVs(base, override map[string]string) map[string]string {
	merged := make(map[string]string, len(base)+len(override))
	for k, v := range base {
		merged[k] = v
	}
	for k, v := range override {
		merged[k] = v
	}
	return merged
}

func cliUsage() {
	fmt.Fprint(os.Stderr, `peanut — process execution and control utility for OCC / FairMQ processes

TUI mode (interactive, no command given):
  peanut                                   direct mode via OCC_CONTROL_PORT env var
  peanut -addr host:port                   direct mode (OCC protobuf)
  peanut -addr host:port -mode fmq         fmq batched mode (full FairMQ sequence per transition)
  peanut -addr host:port -mode fmq-step    fmq single-step mode (one button per raw FairMQ event)

CLI mode (non-interactive, command given):
  peanut [flags] <command> [args]

CLI Flags:
  -addr     string    gRPC address (default "localhost:47100")
  -mode     string    fmq (FairMQ json codec, default) or direct (OCC protobuf)
  -timeout  duration  unary call timeout (default 30s)
  -config   string    path to YAML/JSON file with arguments to push (inline key=val args take precedence)

CLI Commands:
  get-state
        Print the current FSM state.

  transition <fromState> <toState> [key=val ...]
        High-level OCC transition. In fmq mode drives the full multi-step
        FairMQ sequence automatically:
          STANDBY→CONFIGURED  runs INIT DEVICE, COMPLETE INIT, BIND, CONNECT, INIT TASK
          CONFIGURED→RUNNING  runs RUN
          RUNNING→CONFIGURED  runs STOP
          CONFIGURED→STANDBY  runs RESET TASK, RESET DEVICE
        In direct mode sends a single OCC protobuf Transition RPC.
        key=val pairs are forwarded as ConfigEntry arguments.

  direct-step <srcState> <event> [key=val ...]
        Low-level: send a single raw OCC protobuf Transition RPC regardless of -mode.
        Events: CONFIGURE, RESET, START, STOP, RECOVER, EXIT

  fmq-step <srcFMQState> <fmqEvent> [key=val ...]
        Low-level: send a single raw FairMQ gRPC Transition call regardless of -mode.
        FairMQ state/event names that contain spaces must be quoted.

  state-stream
        Subscribe to StateStream; print updates until interrupted.

  event-stream
        Subscribe to EventStream; print events until interrupted.

Examples:
  peanut -addr localhost:47100 get-state
  peanut -addr localhost:47100 transition STANDBY CONFIGURED chans.x.0.address=ipc://@foo
  peanut -addr localhost:47100 fmq-step IDLE "INIT DEVICE" chans.x.0.address=ipc://@foo
  peanut -addr localhost:47100 state-stream
  peanut -addr localhost:47100 -mode direct transition STANDBY CONFIGURED
`)
}
