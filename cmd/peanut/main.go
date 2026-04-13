/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2019 CERN and copyright holders of ALICE O².
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

package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/AliceO2Group/Control/occ/peanut"
)

func main() {
	fs := flag.NewFlagSet("peanut", flag.ExitOnError)
	addr := fs.String("addr", "", "OCC gRPC address (host:port); if empty, OCC_CONTROL_PORT env var is used in direct mode")
	mode := fs.String("mode", "direct", "control mode: direct (default), fmq, or fmq-step")
	fs.Usage = func() {
		fmt.Fprint(os.Stderr, `peanut — process execution and control utility for OCC / FairMQ processes

TUI mode (interactive, launched when no command is given):
  peanut                                   direct mode via OCC_CONTROL_PORT env var
  peanut -addr host:port                   direct mode (OCC protobuf, one button per transition)
  peanut -addr host:port -mode fmq         fmq batched mode (drives full FairMQ sequence per transition)
  peanut -addr host:port -mode fmq-step    fmq single-step mode (one button per raw FairMQ event)

CLI mode (non-interactive, launched when a command is given):
  peanut [flags] <command> [args]

TUI Flags:
  -addr  string   gRPC address (host:port); if empty, uses OCC_CONTROL_PORT env var in direct mode
  -mode  string   direct (default), fmq, or fmq-step

CLI Flags:
  -addr     string    gRPC address (default "localhost:47100")
  -mode     string    fmq (default) or direct
  -timeout  duration  unary call timeout (default 30s)
  -config   string    path to YAML/JSON file with arguments to push (inline key=val args take precedence)

CLI Commands:
  get-state
        Print the current FSM state.

  transition <fromState> <toState> [key=val ...]
        High-level OCC transition. In fmq mode drives the full multi-step FairMQ sequence:
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
        Subscribe to StateStream; print updates until interrupted (ctrl-c to stop).

  event-stream
        Subscribe to EventStream; print events until interrupted (ctrl-c to stop).

Examples:
  peanut -addr localhost:47100 get-state
  peanut -addr localhost:47100 transition STANDBY CONFIGURED chans.x.0.address=ipc://@foo
  peanut -addr localhost:47100 -config args.yaml transition STANDBY CONFIGURED
  peanut -addr localhost:47100 fmq-step IDLE "INIT DEVICE" chans.x.0.address=ipc://@foo
  peanut -addr localhost:47100 direct-step STANDBY CONFIGURE key=val
  peanut -addr localhost:47100 state-stream
  peanut -addr localhost:47100 -mode direct transition STANDBY CONFIGURED
`)
	}
	_ = fs.Parse(os.Args[1:])

	if fs.NArg() > 0 {
		// CLI mode — pass all original args so RunCLI can re-parse its own flags
		if err := peanut.RunCLI(os.Args[1:]); err != nil {
			fmt.Fprintf(os.Stderr, "peanut: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// TUI mode
	if err := peanut.Run(peanut.Options{Addr: *addr, Mode: *mode}); err != nil {
		fmt.Fprintf(os.Stderr, "peanut: %v\n", err)
		os.Exit(1)
	}
}
