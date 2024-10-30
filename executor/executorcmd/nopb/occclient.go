package nopb

import (
	"context"

	"google.golang.org/grpc"

	"github.com/AliceO2Group/Control/executor/protos"
)

// Equivalent to the pb.OccClient interface so we can use protobuf-generated code
type OccClient interface {
	EventStream(ctx context.Context, in *pb.EventStreamRequest, opts ...grpc.CallOption) (pb.Occ_EventStreamClient, error)
	StateStream(ctx context.Context, in *pb.StateStreamRequest, opts ...grpc.CallOption) (pb.Occ_StateStreamClient, error)
	GetState(ctx context.Context, in *pb.GetStateRequest, opts ...grpc.CallOption) (*pb.GetStateReply, error)
	Transition(ctx context.Context, in *pb.TransitionRequest, opts ...grpc.CallOption) (*pb.TransitionReply, error)
}
type occClient struct {
	cc *grpc.ClientConn
}

func NewOccClient(cc *grpc.ClientConn) OccClient {
	return &occClient{cc}
}

type occEventStreamClient struct {
	grpc.ClientStream
}

func (x *occEventStreamClient) Recv() (*pb.EventStreamReply, error) {
	m := new(pb.EventStreamReply)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *occClient) EventStream(ctx context.Context, in *pb.EventStreamRequest, opts ...grpc.CallOption) (pb.Occ_EventStreamClient, error) {
	opts = append(opts,
		[]grpc.CallOption{
			grpc.CallContentSubtype("json"),
		}...,
	)
	streamDesc := grpc.StreamDesc{
		StreamName:    "EventStream",
		Handler:       nil,
		ServerStreams: true,
		ClientStreams: false,
	}
	stream, err := c.cc.NewStream(ctx, &streamDesc, "EventStream", opts...)
	if err != nil {
		return nil, err
	}
	x := &occEventStreamClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

func (c *occClient) StateStream(ctx context.Context, in *pb.StateStreamRequest, opts ...grpc.CallOption) (pb.Occ_StateStreamClient, error) {
	return nil, nil
}

func (c *occClient) GetState(ctx context.Context, in *pb.GetStateRequest, opts ...grpc.CallOption) (*pb.GetStateReply, error) {
	out := pb.GetStateReply{}
	opts = append(opts,
		[]grpc.CallOption{
			grpc.CallContentSubtype("json"),
		}...,
	)
	err := c.cc.Invoke(ctx, "GetState", in, &out, opts...)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *occClient) Transition(ctx context.Context, in *pb.TransitionRequest, opts ...grpc.CallOption) (*pb.TransitionReply, error) {
	out := new(pb.TransitionReply)
	opts = append(opts,
		[]grpc.CallOption{
			grpc.CallContentSubtype("json"),
		}...,
	)
	err := c.cc.Invoke(ctx, "Transition", in, &out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}
