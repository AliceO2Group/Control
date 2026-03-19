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
  OCC_CONTROL_PORT=<port> peanut
  peanut -addr host:port -mode fmq

CLI mode (non-interactive, launched when a command is given):
  peanut [flags] <command> [args]
  Run "peanut -addr x get-state" for full CLI usage.

Flags:
`)
		fs.PrintDefaults()
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
