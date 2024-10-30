/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2020 CERN and copyright holders of ALICE O².
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
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/AliceO2Group/Control/common/controlmode"
	"github.com/AliceO2Group/Control/executor/executorcmd"
	pb "github.com/AliceO2Group/Control/executor/protos"
	"github.com/k0kubun/pp"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

func main() {
	targetPortS := os.Args[1]
	targetPort, _ := strconv.Atoi(targetPortS)
	fmt.Printf("target port: %d", targetPort)

	c := executorcmd.NewClient(
		uint64(targetPort),
		controlmode.FAIRMQ,
		executorcmd.JsonTransport,
		log.WithField("id", ""))
	if c == nil {
		fmt.Println("client is nil")
	}

	for i := 0; i < 10; i++ {
		time.Sleep(100 * time.Millisecond)

		gsr, err := c.GetState(context.TODO(), &pb.GetStateRequest{}, grpc.EmptyCallOption{})
		if err != nil {
			fmt.Println(err.Error())
			continue
		}
		if gsr == nil {
			fmt.Println("nil error and response")
			continue
		}
		fmt.Println("GetState RESPONSE:")
		_, _ = pp.Println(*gsr)
	}

	/*tr, err := c.Transition(context.TODO(), &pb.TransitionRequest{
		SrcState:             "IDLE",
		TransitionEvent:      fairmq.EvtINIT_DEVICE,
		Arguments:            nil,
	})
	if err != nil {
		fmt.Println(err.Error())
	}
	fmt.Printf("RESPONSE:\n%v\n", *tr)
	_, _ = pp.Println(*tr)

	gsr, err = c.GetState(context.TODO(), &pb.GetStateRequest{}, grpc.EmptyCallOption{})
	if err != nil {
		fmt.Println(err.Error())
	}
	fmt.Println("GetState RESPONSE:")
	_, _ = pp.Println(*gsr)*/

}
