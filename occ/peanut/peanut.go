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

package peanut

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/AliceO2Group/Control/common/controlmode"
	"github.com/AliceO2Group/Control/executor/executorcmd"
	"github.com/AliceO2Group/Control/executor/protos"
	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
	"google.golang.org/grpc"
)

var(
	app *tview.Application
	state string
	rpcClient *executorcmd.RpcClient
)

func transition(evt string) error {
	response, err := rpcClient.Transition(context.TODO(), &pb.TransitionRequest{
		TransitionEvent: evt,
		Arguments: nil,
		SrcState: state,
	}, grpc.EmptyCallOption{})
	if err != nil {
		app.Stop()
		fmt.Println(err.Error())
		return err
	}
	state = response.GetState()
	app.Draw()
	return nil
}

func drawStatus(screen tcell.Screen, x int, y int, width int, height int) (int, int, int, int) {
	tview.Print(screen, state, x, height/2, width, tview.AlignCenter, tcell.ColorLime)
	return 0, 0, 0, 0
}

func Run(cmdString string) (err error) {
	state = "UNKNOWN"

	// Setup UI
	app = tview.NewApplication()
	statusBox := tview.NewBox().SetBorder(true).SetTitle("state")

	controlList := tview.NewList().
		AddItem("Transition CONFIGURE",
			"perform CONFIGURE transition",
			'c',
			func(){
				err = transition("CONFIGURE")
				app.Draw()
			}).
		AddItem("Transition RESET",
			"perform RESET transition",
			'r',
			func(){
				err = transition("RESET")
				app.Draw()
			}).
		AddItem("Transition START",
			"perform START transition",
			's',
			func(){
				err = transition("START")
				app.Draw()
			}).
		AddItem("Transition STOP",
			"perform STOP transition",
			't',
			func(){
				err = transition("STOP")
				app.Draw()
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
			func(){
				err = transition("RECOVER")
				app.Draw()
			}).
		AddItem("Transition EXIT",
			"perform EXIT transition",
			'x',
			func(){
				err = transition("EXIT")
				app.Draw()
			}).
		AddItem("Quit",
			"disconnect from the process and quit peanut",
			'q',
			func() {
				app.Stop()
			})
	controlList.SetBorder(true).SetTitle("control")

	configBox := tview.NewBox().SetBorder(true).SetTitle("configuration to push")
	flex := tview.NewFlex().AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(statusBox, 3, 1, false).
		AddItem(controlList, 0, 1, true), 0, 1, false).
		AddItem(configBox, 0, 2, false)

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
		rpcClient = executorcmd.NewClient(occPort, controlmode.DIRECT)
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