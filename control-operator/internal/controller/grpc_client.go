/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2026 CERN and copyright holders of ALICE O².
 * Author: Michal Tichak <michal.tichak@cern.ch>
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

package controller

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"k8s.io/apimachinery/pkg/types"

	aliecsv1alpha1 "github.com/AliceO2Group/ControlOperator/api/v1alpha1"
	pb "github.com/AliceO2Group/ControlOperator/internal/controller/protos/generated"
)

// jsonOccClient mirrors executor/executorcmd/nopb/occclient.go: uses short method
// names and JSON content subtype for OCC lite / FairMQ processes.
type jsonOccClient struct{ conn *grpc.ClientConn }

type jsonEventStreamClient struct{ grpc.ClientStream }

func (x *jsonEventStreamClient) Recv() (*pb.EventStreamReply, error) {
	m := new(pb.EventStreamReply)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

type jsonStateStreamClient struct{ grpc.ClientStream }

func (x *jsonStateStreamClient) Recv() (*pb.StateStreamReply, error) {
	m := new(pb.StateStreamReply)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *jsonOccClient) EventStream(ctx context.Context, in *pb.EventStreamRequest, opts ...grpc.CallOption) (pb.Occ_EventStreamClient, error) {
	opts = append(opts, grpc.CallContentSubtype("json"))
	stream, err := c.conn.NewStream(ctx, &grpc.StreamDesc{StreamName: "EventStream", ServerStreams: true}, "EventStream", opts...)
	if err != nil {
		return nil, err
	}
	x := &jsonEventStreamClient{stream}
	if err := x.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

func (c *jsonOccClient) StateStream(ctx context.Context, in *pb.StateStreamRequest, opts ...grpc.CallOption) (pb.Occ_StateStreamClient, error) {
	opts = append(opts, grpc.CallContentSubtype("json"))
	stream, err := c.conn.NewStream(ctx, &grpc.StreamDesc{StreamName: "StateStream", ServerStreams: true}, "StateStream", opts...)
	if err != nil {
		return nil, err
	}
	x := &jsonStateStreamClient{stream}
	if err := x.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

func (c *jsonOccClient) GetState(ctx context.Context, in *pb.GetStateRequest, opts ...grpc.CallOption) (*pb.GetStateReply, error) {
	out := new(pb.GetStateReply)
	opts = append(opts, grpc.CallContentSubtype("json"))
	if err := c.conn.Invoke(ctx, "GetState", in, out, opts...); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *jsonOccClient) Transition(ctx context.Context, in *pb.TransitionRequest, opts ...grpc.CallOption) (*pb.TransitionReply, error) {
	out := new(pb.TransitionReply)
	opts = append(opts, grpc.CallContentSubtype("json"))
	if err := c.conn.Invoke(ctx, "Transition", in, out, opts...); err != nil {
		return nil, err
	}
	return out, nil
}

type OccClient struct {
	client      pb.OccClient
	conn        *grpc.ClientConn
	crdName     types.NamespacedName
	reconciler  *TaskReconciler
	cancel      *context.CancelFunc
	log         logr.Logger
	controlMode string
}

// fromDeviceState translates a raw device state to an OCC state name,
// mirroring executor/executorcmd/transitioner FairMQ and Direct FromDeviceState logic.
func fromDeviceState(controlMode string, state string) string {
	if controlMode == "fairmq" {
		return occStateForFmqState(state)
	}
	return strings.ToLower(state)
}

func NewOccClient(ctx context.Context, address string, controlMode string, reconciler *TaskReconciler, crdName types.NamespacedName, log logr.Logger) (*OccClient, error) {
	// grpc.WithBlock() ensures that the dialer waits for the connection to be established.
	// If the server isn't listening, this will return an error after the context timeout.
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	var occClient pb.OccClient
	if controlMode == "fairmq" {
		occClient = &jsonOccClient{conn}
	} else {
		occClient = pb.NewOccClient(conn)
	}

	client := &OccClient{client: occClient, conn: conn, crdName: crdName, reconciler: reconciler, log: log, controlMode: controlMode}

	client.conn.Connect()

	// ctxWithTimeout, cancel := context.WithTimeout(ctx, 1*time.Second)
	// defer cancel()
	// client.GetState(ctxWithTimeout)

	return client, nil
}

func (c *OccClient) ConsumeIfReady(ctx context.Context) bool {
	if c.cancel != nil {
		return true
	}

	if connState := c.conn.GetState(); connState != connectivity.Ready {
		c.log.V(1).Info("connection is in different state than ready", "conn state", connState.String())
		return false
	}
	clientCtx, clientCancel := context.WithCancel(context.Background())
	c.cancel = &clientCancel

	go c.ConsumeEventStream(clientCtx)
	go c.ConsumeStateStream(clientCtx)
	return true
}

func (c *OccClient) WaitUntilConnected(ctx context.Context) error {
	for {
		state := c.conn.GetState()
		if state == connectivity.Ready {
			return nil
		}

		if !c.conn.WaitForStateChange(ctx, state) {
			return fmt.Errorf("connection failed: %w", ctx.Err())
		}
	}
}

func (c *OccClient) Close() error {
	if c.cancel != nil {
		(*c.cancel)()
	}
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (c *OccClient) GetState(ctx context.Context) (*pb.GetStateReply, error) {
	if c == nil || c.client == nil {
		return nil, errors.New("nil client for TransitionRequest")
	}
	c.log.V(1).Info("GetState")
	result, err := c.client.GetState(ctx, &pb.GetStateRequest{})
	if err == nil {
		result.State = fromDeviceState(c.controlMode, result.State)
	}
	return result, err
}

func (c *OccClient) TransitionRequest(ctx context.Context, fromState string, toState string, args map[string]string) (*pb.TransitionReply, error) {
	if c == nil || c.client == nil {
		return nil, errors.New("nil client for TransitionRequest")
	}

	from, err := StateFromString(fromState)
	if err != nil {
		return nil, err
	}
	to, err := StateFromString(toState)
	if err != nil {
		return nil, err
	}

	transition, err := FromStatesToTransition(from, to)
	if err != nil {
		return nil, err
	}

	var configEntries []*pb.ConfigEntry
	for k, v := range args {
		configEntries = append(configEntries, &pb.ConfigEntry{Key: k, Value: v})
	}

	request := &pb.TransitionRequest{
		SrcState:        strings.ToUpper(fromState),
		TransitionEvent: strings.ToUpper(transition.String()),
		Arguments:       configEntries,
	}
	c.log.V(1).Info("TransitionRequest", "req", request)

	return c.client.Transition(ctx, request)
}

func (c *OccClient) ConsumeEventStream(ctx context.Context) {
	c.log.Info("starting to consume EventStream")
	stream, err := c.client.EventStream(ctx, &pb.EventStreamRequest{})
	if err != nil {
		c.log.Error(err, "failed to start event stream")
		return
	}
	for {
		resp, err := stream.Recv()
		if err != nil {
			if st, ok := status.FromError(err); ok && st.Code() == codes.Canceled {
				c.log.Info("EventStream stopped: context cancelled")
				return
			}
			c.log.Error(err, "EventStream stopped with error")
			return
		}
		c.log.Info("received event", "event", resp.GetEvent())
	}
}

func (c *OccClient) ConsumeStateStream(ctx context.Context) {
	c.log.Info("starting to consume StateStream")
	stream, err := c.client.StateStream(ctx, &pb.StateStreamRequest{})
	if err != nil {
		c.log.Error(err, "failed to start state stream")
		return
	}

	for {
		resp, err := stream.Recv()
		if err != nil {
			if st, ok := status.FromError(err); ok && st.Code() == codes.Canceled {
				c.log.Info("StateStream stopped: context cancelled")
				return
			}
			c.log.Error(err, "StateStream stopped with error")
			return
		}
		c.log.V(1).Info("received state update", "type", resp.GetType(), "state", resp.GetState())

		task := &aliecsv1alpha1.Task{}
		if err := c.reconciler.Get(ctx, c.crdName, task); err != nil {
			c.log.V(1).Error(err, "state event could not find task")
			continue
		}
		// TODO: add some checks??
		task.Status.State = fromDeviceState(c.controlMode, resp.GetState())
		updateCtx, cancel := context.WithTimeout(context.Background(), time.Second)
		if err := c.reconciler.Status().Update(updateCtx, task); err != nil {
			c.log.Error(err, "state event did not apply state change to task", "state", task.Status.State)
		}
		cancel()
	}
}
