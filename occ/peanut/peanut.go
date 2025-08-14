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
	"strconv"
	"time"

	"github.com/AliceO2Group/Control/common/controlmode"
	"github.com/AliceO2Group/Control/executor/executorcmd"
	"github.com/AliceO2Group/Control/executor/protos"
	"github.com/AliceO2Group/Control/occ/peanut/flatten"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

var (
	app            *tview.Application
	state          string
	configMap      map[string]string
	controlList    *tview.List
	configTextView *tview.TextView
	rpcClient      *executorcmd.RpcClient
)

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
	args := make([]*pb.ConfigEntry, 0)
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

	response, err := rpcClient.Transition(context.TODO(), &pb.TransitionRequest{
		TransitionEvent: evt,
		Arguments:       args,
		SrcState:        state,
	}, grpc.EmptyCallOption{})
	if err != nil {
		app.Stop()

		fmt.Println(err.Error())
		return err
	}
	if evt == "CONFIGURE" {
		configTextView.SetTitle("runtime configuration (PUSHED)")
	}
	state = response.GetState()
	return nil
}

func drawStatus(screen tcell.Screen, x int, y int, width int, height int) (int, int, int, int) {
	tview.Print(screen, state, x, height/2, width, tview.AlignCenter, tcell.ColorLime)
	return 0, 0, 0, 0
}

func acquireConfigFile(configPages *tview.Pages) error {
	configInputFrame := tview.NewForm()
	configInputFrame.SetTitle("file path for runtime configuration")
	configInputFrame.SetBorder(true)
	configInputFrame.AddInputField("path:", "", 0, nil, nil)

	configPages.AddPage("modal", modal(configInputFrame, 40, 10), true, true)

	configCancelFunc := func() {
		configPages.RemovePage("modal")
		app.SetFocus(controlList)
		app.Draw()
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

	app.SetFocus(configInputFrame)

	app.Draw()
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
	app.Draw()
}

func loadConfig(configFilePath string, configPages *tview.Pages) {
	if len(configFilePath) == 0 {
		errorMessage(configPages, "Cannot open configuration file", "path empty")
		return
	}
	/*// Make sure we trim all variants
	trimmed := strings.TrimPrefix(configFilePath, "file://")
	trimmed = strings.TrimPrefix(trimmed, "file:/")
	trimmed = strings.TrimPrefix(trimmed, "file:")
	uri := "file://" + trimmed
	cfg, err := configuration.NewConfiguration(uri)
	if err != nil {
		errorMessage(configPages, "Cannot open configuration file", err.Error())
		return
	}
	yamlConfig, err := cfg.GetRecursiveYaml("")
	if err != nil {
		errorMessage(configPages, "Cannot parse configuration file", err.Error())
		return
	}*/
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

	configMap = make(map[string]string)
	err = json.Unmarshal([]byte(configTextView.GetText(false))[:], &configMap)
	if err != nil {
		errorMessage(configPages, "Cannot process configuration file", err.Error())
		return
	}
}

func Run(cmdString string) (err error) {
	state = "UNKNOWN"

	// Setup UI
	app = tview.NewApplication()

	statusBox := tview.NewBox().SetBorder(true).SetTitle("state")
	configTextView = tview.NewTextView().SetChangedFunc(func() { app.Draw() })
	configTextView.SetBorder(true).SetTitle("runtime configuration (EMPTY)")
	configPages := tview.NewPages().
		AddPage("configBox", configTextView, true, true)

	controlList = tview.NewList().
		AddItem("Transition CONFIGURE",
			"perform CONFIGURE transition",
			'c',
			func() {
				err = transition("CONFIGURE")
			}).
		AddItem("Transition RESET",
			"perform RESET transition",
			'r',
			func() {
				err = transition("RESET")
			}).
		AddItem("Transition START",
			"perform START transition",
			's',
			func() {
				err = transition("START")
			}).
		AddItem("Transition STOP",
			"perform STOP transition",
			't',
			func() {
				err = transition("STOP")
			}).
		//AddItem("Transition GO_ERROR",
		//	"perform GO_ERROR transition",
		//	'e',
		//	func(){
		//		err = transition("GO_ERROR")
		//		app.Draw()
		//	}).
		AddItem("Transition RECOVER",
			"perform RECOVER transition",
			'v',
			func() {
				err = transition("RECOVER")
			}).
		AddItem("Transition EXIT",
			"perform EXIT transition",
			'x',
			func() {
				err = transition("EXIT")
			}).
		AddItem("Load configuration",
			"read runtime configuration from file",
			'l',
			func() {
				err = acquireConfigFile(configPages)
			}).
		AddItem("Quit",
			"disconnect from the process and quit peanut",
			'q',
			func() {
				app.Stop()
			})
	controlList.SetBorder(true).SetTitle("control")

	flex := tview.NewFlex().AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(statusBox, 3, 1, false).
		AddItem(controlList, 0, 1, true), 0, 1, false).
		AddItem(configPages, 0, 2, false)

	statusBox.SetDrawFunc(drawStatus)

	// Parse input
	var occPort uint64
	if len(cmdString) > 0 {
		// RUN process
	} else {
		// If cmdString not passed, env var OCC_CONTROL_PORT (occ/OccGlobals.h) must be defined
		occPortString := os.Getenv("OCC_CONTROL_PORT")
		if len(occPortString) == 0 {
			err = fmt.Errorf("OCC_CONTROL_PORT not defined")
			return
		}
		occPort, err = strconv.ParseUint(occPortString, 10, 64)
		if err != nil {
			return
		}
	}

	// Setup RPC
	go func() {
		// FIXME allow choice of controlmode.FAIRMQ
		rpcClient = executorcmd.NewClient(
			occPort,
			controlmode.DIRECT,
			executorcmd.ProtobufTransport,
			log.WithField("id", ""))
		var response *pb.GetStateReply
		response, err = rpcClient.GetState(context.TODO(), &pb.GetStateRequest{}, grpc.EmptyCallOption{})
		if err != nil {
			app.Stop()
			fmt.Println(err.Error())
			return
		}
		// NOTE: we acquire the transitioner-dependent STANDBY equivalent state
		state = rpcClient.FromDeviceState(response.GetState())
		app.Draw()
	}()
	return app.SetRoot(flex, true).SetFocus(controlList).Run()
}
