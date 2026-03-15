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

// Package peanut implements a process execution and control utility for
// OCClib-based O² processes, providing debugging and development support.
package peanut

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/AliceO2Group/Control/executor/executorcmd/nopb"
	"github.com/AliceO2Group/Control/executor/executorcmd/transitioner/fairmq"
	"github.com/AliceO2Group/Control/executor/protos"
	"github.com/AliceO2Group/Control/occ/peanut/flatten"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Options configures the peanut TUI.
type Options struct {
	Addr string // host:port; if empty, falls back to OCC_CONTROL_PORT env var (occ mode only)
	Mode string // "direct" (default), "fmq", or "fmq-step"
}

var (
	app            *tview.Application
	state          string
	tuiMode        string
	tuiAddr        string
	tuiConn        *grpc.ClientConn
	streamCancel   context.CancelFunc
	transitioning  bool
	configMap      map[string]string
	controlList    *tview.List
	configTextView *tview.TextView
	configPages    *tview.Pages
	occClient      pb.OccClient
)

func monitorConnection(ctx context.Context) {
	// Try StateStream first — gives state updates and disconnect detection.
	stateStream, e := occClient.StateStream(ctx, &pb.StateStreamRequest{})
	if e == nil && stateStream != nil {
		for {
			msg, e := stateStream.Recv()
			if e != nil {
				if ctx.Err() != nil {
					return
				}
				app.QueueUpdateDraw(func() {
					state = "UNREACHABLE"
					errorMessage(configPages, "Connection lost", e.Error())
				})
				return
			}
			app.QueueUpdateDraw(func() {
				switch tuiMode {
				case "fmq":
					state = cliFMQToOCCState(msg.GetState())
				default:
					state = msg.GetState()
				}
			})
		}
	}

	// Try EventStream — disconnect detection only (no state in payload).
	eventStream, e := occClient.EventStream(ctx, &pb.EventStreamRequest{})
	if e == nil && eventStream != nil {
		for {
			if _, e := eventStream.Recv(); e != nil {
				if ctx.Err() != nil {
					return
				}
				app.QueueUpdateDraw(func() {
					state = "UNREACHABLE"
					errorMessage(configPages, "Connection lost", e.Error())
				})
				return
			}
		}
	}

	// Neither stream available — poll GetState every 2s.
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, e := occClient.GetState(ctx, &pb.GetStateRequest{}); e != nil {
				if ctx.Err() != nil {
					return
				}
				app.QueueUpdateDraw(func() {
					state = "UNREACHABLE"
					errorMessage(configPages, "Connection lost", e.Error())
				})
				return
			}
		}
	}
}

func connectRPC() {
	if streamCancel != nil {
		streamCancel() // stop any existing stream monitor
	}
	state = "CONNECTING"
	go func() {
		if tuiConn != nil {
			tuiConn.Close()
			tuiConn = nil
		}
		conn, e := grpc.Dial(tuiAddr, grpc.WithTransportCredentials(insecure.NewCredentials())) //nolint:staticcheck
		if e != nil {
			app.QueueUpdateDraw(func() {
				state = "UNREACHABLE"
				errorMessage(configPages, "Connection failed", e.Error())
			})
			return
		}
		if tuiMode == "fmq" || tuiMode == "fmq-step" {
			occClient = nopb.NewOccClient(conn)
		} else {
			occClient = pb.NewOccClient(conn)
		}
		response, e := occClient.GetState(context.TODO(), &pb.GetStateRequest{})
		if e != nil {
			app.QueueUpdateDraw(func() {
				state = "UNREACHABLE"
				errorMessage(configPages, "Connection failed", e.Error())
			})
			return
		}
		tuiConn = conn
		ctx, cancel := context.WithCancel(context.Background())
		streamCancel = cancel
		go monitorConnection(ctx)
		app.QueueUpdateDraw(func() {
			switch tuiMode {
			case "fmq":
				state = cliFMQToOCCState(response.GetState())
			default:
				state = response.GetState()
			}
		})
	}()
}

func modal(p tview.Primitive, width, height int) tview.Primitive {
	return tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(p, height, 1, false).
			AddItem(nil, 0, 1, false), width, 1, false).
		AddItem(nil, 0, 1, false)
}

func transition(evt string) error {
	args := make([]*pb.ConfigEntry, 0, len(configMap))
	for k, v := range configMap {
		args = append(args, &pb.ConfigEntry{Key: k, Value: v})
	}

	// We simulate a new run number on every START event
	if evt == "START" {
		args = append(args, &pb.ConfigEntry{
			Key:   "runNumber",
			Value: time.Now().Format("0102150405"),
		})
	}

	var (
		newState string
		err      error
	)

	if tuiMode == "fmq" {
		newState, err = tuiFMQTransition(evt, args)
	} else {
		var response *pb.TransitionReply
		response, err = occClient.Transition(context.TODO(), &pb.TransitionRequest{
			TransitionEvent: evt,
			Arguments:       args,
			SrcState:        state,
		})
		if err == nil {
			newState = response.GetState()
		}
	}

	if err != nil {
		return err
	}
	if evt == "CONFIGURE" {
		configTextView.SetTitle("runtime configuration (PUSHED)")
	}
	state = newState
	return nil
}

// tuiFMQTransition maps TUI event names to FairMQ multi-step sequences.
func tuiFMQTransition(evt string, args []*pb.ConfigEntry) (string, error) {
	kvs := make(map[string]string, len(args))
	for _, e := range args {
		kvs[e.Key] = e.Value
	}

	switch evt {
	case "CONFIGURE":
		return tuiFMQConfigure(kvs)
	case "RESET":
		return tuiFMQReset(kvs)
	case "START":
		result, err := fmqDoStep(context.TODO(), occClient, fairmq.READY, fairmq.EvtRUN, kvs)
		return cliFMQToOCCState(result), err
	case "STOP":
		result, err := fmqDoStep(context.TODO(), occClient, fairmq.RUNNING, fairmq.EvtSTOP, kvs)
		return cliFMQToOCCState(result), err
	case "RECOVER":
		result, err := fmqDoStep(context.TODO(), occClient, fairmq.ERROR, fairmq.EvtRESET_DEVICE, kvs)
		return cliFMQToOCCState(result), err
	case "EXIT":
		if state == "CONFIGURED" {
			if _, err := tuiFMQReset(nil); err != nil {
				return state, err
			}
		}
		result, err := fmqDoStep(context.TODO(), occClient, fairmq.IDLE, fairmq.EvtEND, nil)
		return cliFMQToOCCState(result), err
	default:
		return state, fmt.Errorf("unsupported transition %q in FairMQ mode", evt)
	}
}

func tuiFMQConfigure(args map[string]string) (string, error) {
	state, err := fmqDoStep(context.TODO(), occClient, fairmq.IDLE, fairmq.EvtINIT_DEVICE, args)
	if err != nil || state != fairmq.INITIALIZING_DEVICE {
		return cliFMQToOCCState(state), fmt.Errorf("INIT DEVICE: expected %q got %q: %w", fairmq.INITIALIZING_DEVICE, state, err)
	}
	state, err = fmqDoStep(context.TODO(), occClient, fairmq.INITIALIZING_DEVICE, fairmq.EvtCOMPLETE_INIT, nil)
	if err != nil || state != fairmq.INITIALIZED {
		return cliFMQToOCCState(state), fmt.Errorf("COMPLETE INIT: expected %q got %q: %w", fairmq.INITIALIZED, state, err)
	}
	state, err = fmqDoStep(context.TODO(), occClient, fairmq.INITIALIZED, fairmq.EvtBIND, nil)
	if err != nil || state != fairmq.BOUND {
		fmqDoStep(context.TODO(), occClient, fairmq.INITIALIZED, fairmq.EvtRESET_DEVICE, nil) // rollback
		return cliFMQToOCCState(state), fmt.Errorf("BIND: expected %q got %q: %w", fairmq.BOUND, state, err)
	}
	state, err = fmqDoStep(context.TODO(), occClient, fairmq.BOUND, fairmq.EvtCONNECT, nil)
	if err != nil || state != fairmq.DEVICE_READY {
		fmqDoStep(context.TODO(), occClient, fairmq.BOUND, fairmq.EvtRESET_DEVICE, nil) // rollback
		return cliFMQToOCCState(state), fmt.Errorf("CONNECT: expected %q got %q: %w", fairmq.DEVICE_READY, state, err)
	}
	state, err = fmqDoStep(context.TODO(), occClient, fairmq.DEVICE_READY, fairmq.EvtINIT_TASK, nil)
	if err != nil || state != fairmq.READY {
		fmqDoStep(context.TODO(), occClient, fairmq.DEVICE_READY, fairmq.EvtRESET_DEVICE, nil) // rollback
		return cliFMQToOCCState(state), fmt.Errorf("INIT TASK: expected %q got %q: %w", fairmq.READY, state, err)
	}
	return cliFMQToOCCState(state), nil
}

func tuiFMQReset(args map[string]string) (string, error) {
	state, err := fmqDoStep(context.TODO(), occClient, fairmq.READY, fairmq.EvtRESET_TASK, nil)
	if err != nil || state != fairmq.DEVICE_READY {
		return cliFMQToOCCState(state), fmt.Errorf("RESET TASK: expected %q got %q: %w", fairmq.DEVICE_READY, state, err)
	}
	state, err = fmqDoStep(context.TODO(), occClient, fairmq.DEVICE_READY, fairmq.EvtRESET_DEVICE, args)
	return cliFMQToOCCState(state), err
}

func drawStatus(screen tcell.Screen, x int, y int, width int, height int) (int, int, int, int) {
	tview.Print(screen, state, x, height/2, width, tview.AlignCenter, tcell.ColorLime)
	return 0, 0, 0, 0
}

// pathComplete returns filesystem completions for the given partial path.
func pathComplete(text string) []string {
	// Expand ~ to home directory
	if strings.HasPrefix(text, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			text = filepath.Join(home, text[2:])
		}
	}

	// Split into directory and filename prefix
	dir, prefix := filepath.Split(text)
	if dir == "" {
		dir = "."
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	var matches []string
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasPrefix(name, prefix) {
			continue
		}
		full := filepath.Join(dir, name)
		if entry.IsDir() {
			full += "/"
		}
		matches = append(matches, full)
	}
	return matches
}

func acquireConfigFile(configPages *tview.Pages) error {
	configInputFrame := tview.NewForm()
	configInputFrame.SetTitle("file path for runtime configuration")
	configInputFrame.SetBorder(true)
	configInputFrame.AddInputField("path:", "", 0, nil, nil)

	// Wire up filesystem tab-completion on the path input field
	pathField := configInputFrame.GetFormItemByLabel("path:").(*tview.InputField)
	pathField.SetAutocompleteFunc(func(currentText string) []string {
		return pathComplete(currentText)
	})

	configPages.AddPage("modal", modal(configInputFrame, 40, 10), true, true)
	app.SetFocus(configInputFrame)

	configCancelFunc := func() {
		configPages.RemovePage("modal")
		app.SetFocus(controlList)
	}

	configInputFrame.AddButton("Ok", func() {
		pathItem := configInputFrame.GetFormItemByLabel("path:")
		pathInput := pathItem.(*tview.InputField)
		configFilePath := pathInput.GetText()
		configCancelFunc()
		loadConfig(configFilePath, configPages)
	})

	configInputFrame.SetCancelFunc(configCancelFunc)
	configInputFrame.AddButton("Cancel", configCancelFunc)

	return nil
}

func errorMessage(configPages *tview.Pages, title string, text string) {
	modalPage := tview.NewModal().SetText(title + "\n\nError: " + text).AddButtons([]string{"Ok"}).
		SetDoneFunc(func(_ int, _ string) {
			configPages.RemovePage("modal")
			app.SetFocus(controlList)
		})

	configPages.AddPage("modal", modalPage, true, true)
	app.SetFocus(modalPage)
}

func loadConfig(configFilePath string, configPages *tview.Pages) {
	if len(configFilePath) == 0 {
		errorMessage(configPages, "Cannot open configuration file", "path empty")
		return
	}
	yamlConfig, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		errorMessage(configPages, "Cannot open configuration file", err.Error())
		return
	}
	flattened, err := flatten.FlattenString(string(yamlConfig), "", flatten.DotStyle)
	if err != nil {
		errorMessage(configPages, "Cannot prepare configuration file", err.Error())
		return
	}
	configTextView.SetText(flattened)

	configTextView.SetTitle("runtime configuration (NOT PUSHED)")

	var rawMap map[string]interface{}
	err = json.Unmarshal([]byte(configTextView.GetText(false)), &rawMap)
	if err != nil {
		errorMessage(configPages, "Cannot process configuration file", err.Error())
		return
	}
	configMap = make(map[string]string, len(rawMap))
	for k, v := range rawMap {
		configMap[k] = fmt.Sprintf("%v", v)
	}
}

func Run(opts Options) (err error) {
	state = "UNKNOWN"
	tuiMode = opts.Mode
	if tuiMode == "" {
		tuiMode = "direct"
	}

	// Validate mode
	switch tuiMode {
	case "direct", "fmq", "fmq-step":
	default:
		return fmt.Errorf("unknown mode %q — valid modes: direct, fmq, fmq-step", tuiMode)
	}

	// Resolve address
	addr := opts.Addr
	if addr == "" {
		if tuiMode == "fmq" || tuiMode == "fmq-step" {
			return fmt.Errorf("%s mode requires -addr flag", tuiMode)
		}
		// Fall back to OCC_CONTROL_PORT env var (direct mode legacy behaviour)
		occPortString := os.Getenv("OCC_CONTROL_PORT")
		if occPortString == "" {
			return fmt.Errorf("OCC_CONTROL_PORT not defined")
		}
		occPort, e := strconv.ParseUint(occPortString, 10, 64)
		if e != nil {
			return e
		}
		addr = fmt.Sprintf("127.0.0.1:%d", occPort)
	}

	// Setup UI
	app = tview.NewApplication()

	statusBox := tview.NewBox().SetBorder(true).SetTitle("state")
	configTextView = tview.NewTextView().SetChangedFunc(func() { app.QueueUpdateDraw(func() {}) })
	configTextView.SetBorder(true).SetTitle("runtime configuration (EMPTY)")
	configPages = tview.NewPages().
		AddPage("configBox", configTextView, true, true)

	doTransition := func(evt string) {
		if transitioning {
			return
		}
		transitioning = true
		go func() {
			e := transition(evt)
			app.QueueUpdateDraw(func() {
				transitioning = false
				if e != nil {
					err = e
					errorMessage(configPages, "Transition error", e.Error())
				}
			})
		}()
	}

	doFMQStep := func(event string) {
		if transitioning {
			return
		}
		transitioning = true
		go func() {
			kvs := make(map[string]string, len(configMap))
			for k, v := range configMap {
				kvs[k] = v
			}
			newState, e := fmqDoStep(context.TODO(), occClient, state, event, kvs)
			app.QueueUpdateDraw(func() {
				transitioning = false
				if e != nil {
					err = e
					errorMessage(configPages, "FMQ step error", e.Error())
				} else {
					state = newState
				}
			})
		}()
	}

	controlList = tview.NewList()
	switch tuiMode {
	case "fmq-step":
		controlList.
			AddItem("INIT DEVICE", "IDLE → INITIALIZING DEVICE", '1', func() { doFMQStep(fairmq.EvtINIT_DEVICE) }).
			AddItem("COMPLETE INIT", "INITIALIZING DEVICE → INITIALIZED", '2', func() { doFMQStep(fairmq.EvtCOMPLETE_INIT) }).
			AddItem("BIND", "INITIALIZED → BOUND", '3', func() { doFMQStep(fairmq.EvtBIND) }).
			AddItem("CONNECT", "BOUND → DEVICE READY", '4', func() { doFMQStep(fairmq.EvtCONNECT) }).
			AddItem("INIT TASK", "DEVICE READY → READY", '5', func() { doFMQStep(fairmq.EvtINIT_TASK) }).
			AddItem("RUN", "READY → RUNNING", '6', func() { doFMQStep(fairmq.EvtRUN) }).
			AddItem("STOP", "RUNNING → READY", '7', func() { doFMQStep(fairmq.EvtSTOP) }).
			AddItem("RESET TASK", "READY → DEVICE READY", '8', func() { doFMQStep(fairmq.EvtRESET_TASK) }).
			AddItem("RESET DEVICE", "→ IDLE", '9', func() { doFMQStep(fairmq.EvtRESET_DEVICE) }).
			AddItem("END", "IDLE → EXITING", '0', func() { doFMQStep(fairmq.EvtEND) })
	default: // direct, fmq
		controlList.
			AddItem("Transition CONFIGURE", "perform CONFIGURE transition", 'c', func() { doTransition("CONFIGURE") }).
			AddItem("Transition RESET", "perform RESET transition", 'r', func() { doTransition("RESET") }).
			AddItem("Transition START", "perform START transition", 's', func() { doTransition("START") }).
			AddItem("Transition STOP", "perform STOP transition", 't', func() { doTransition("STOP") }).
			AddItem("Transition RECOVER", "perform RECOVER transition", 'v', func() { doTransition("RECOVER") }).
			AddItem("Transition EXIT", "perform EXIT transition", 'x', func() { doTransition("EXIT") })
	}
	controlList.
		AddItem("Reconnect", "re-establish gRPC connection to the controlled process", 'n', func() { connectRPC() }).
		AddItem("Load configuration", "read runtime configuration from file", 'l', func() { err = acquireConfigFile(configPages) }).
		AddItem("Quit", "disconnect from the process and quit peanut", 'q', func() { app.Stop() })
	controlList.SetBorder(true).SetTitle("control")

	flex := tview.NewFlex().AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(statusBox, 3, 1, false).
		AddItem(controlList, 0, 1, true), 0, 1, false).
		AddItem(configPages, 0, 2, false)

	statusBox.SetDrawFunc(drawStatus)

	// Connect to the controlled process
	tuiAddr = addr
	connectRPC()
	return app.SetRoot(flex, true).SetFocus(controlList).Run()
}
