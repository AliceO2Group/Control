// Code generated by protoc-gen-go-grpc. DO NOT EDIT.

package dcspb

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// ConfiguratorClient is the client API for Configurator service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type ConfiguratorClient interface {
	// Optional call, legal at any time, to subscribe to all future events from
	// the DCS service. The server stops serving the stream when the client closes
	// it. Multiple concurrent stream subscriptions are allowed.
	Subscribe(ctx context.Context, in *SubscriptionRequest, opts ...grpc.CallOption) (Configurator_SubscribeClient, error)
	// Single SOR request for a data taking session, with per-detector parameters.
	// Returns an event stream which returns subsequent intermediate states within
	// the SOR operation. Upon SOR completion (DetectorState.RUN_OK), the server
	// closes the stream.
	StartOfRun(ctx context.Context, in *SorRequest, opts ...grpc.CallOption) (Configurator_StartOfRunClient, error)
	// Single EOR request for a data taking session, with per-detector parameters.
	// Returns an event stream which returns subsequent intermediate states within
	// the EOR operation. Upon EOR completion (DetectorState.RUN_OK), the server
	// closes the stream.
	EndOfRun(ctx context.Context, in *EorRequest, opts ...grpc.CallOption) (Configurator_EndOfRunClient, error)
	// Optional call, legal at any time, to query the status of the DCS service
	// and either some or all of its constituent detectors. This call returns a
	// single value (not a stream), reflecting the service state at that
	// specific moment.
	GetStatus(ctx context.Context, in *StatusRequest, opts ...grpc.CallOption) (*StatusReply, error)
}

type configuratorClient struct {
	cc grpc.ClientConnInterface
}

func NewConfiguratorClient(cc grpc.ClientConnInterface) ConfiguratorClient {
	return &configuratorClient{cc}
}

func (c *configuratorClient) Subscribe(ctx context.Context, in *SubscriptionRequest, opts ...grpc.CallOption) (Configurator_SubscribeClient, error) {
	stream, err := c.cc.NewStream(ctx, &Configurator_ServiceDesc.Streams[0], "/dcs.Configurator/Subscribe", opts...)
	if err != nil {
		return nil, err
	}
	x := &configuratorSubscribeClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type Configurator_SubscribeClient interface {
	Recv() (*Event, error)
	grpc.ClientStream
}

type configuratorSubscribeClient struct {
	grpc.ClientStream
}

func (x *configuratorSubscribeClient) Recv() (*Event, error) {
	m := new(Event)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *configuratorClient) StartOfRun(ctx context.Context, in *SorRequest, opts ...grpc.CallOption) (Configurator_StartOfRunClient, error) {
	stream, err := c.cc.NewStream(ctx, &Configurator_ServiceDesc.Streams[1], "/dcs.Configurator/StartOfRun", opts...)
	if err != nil {
		return nil, err
	}
	x := &configuratorStartOfRunClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type Configurator_StartOfRunClient interface {
	Recv() (*RunEvent, error)
	grpc.ClientStream
}

type configuratorStartOfRunClient struct {
	grpc.ClientStream
}

func (x *configuratorStartOfRunClient) Recv() (*RunEvent, error) {
	m := new(RunEvent)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *configuratorClient) EndOfRun(ctx context.Context, in *EorRequest, opts ...grpc.CallOption) (Configurator_EndOfRunClient, error) {
	stream, err := c.cc.NewStream(ctx, &Configurator_ServiceDesc.Streams[2], "/dcs.Configurator/EndOfRun", opts...)
	if err != nil {
		return nil, err
	}
	x := &configuratorEndOfRunClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type Configurator_EndOfRunClient interface {
	Recv() (*RunEvent, error)
	grpc.ClientStream
}

type configuratorEndOfRunClient struct {
	grpc.ClientStream
}

func (x *configuratorEndOfRunClient) Recv() (*RunEvent, error) {
	m := new(RunEvent)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *configuratorClient) GetStatus(ctx context.Context, in *StatusRequest, opts ...grpc.CallOption) (*StatusReply, error) {
	out := new(StatusReply)
	err := c.cc.Invoke(ctx, "/dcs.Configurator/GetStatus", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ConfiguratorServer is the server API for Configurator service.
// All implementations should embed UnimplementedConfiguratorServer
// for forward compatibility
type ConfiguratorServer interface {
	// Optional call, legal at any time, to subscribe to all future events from
	// the DCS service. The server stops serving the stream when the client closes
	// it. Multiple concurrent stream subscriptions are allowed.
	Subscribe(*SubscriptionRequest, Configurator_SubscribeServer) error
	// Single SOR request for a data taking session, with per-detector parameters.
	// Returns an event stream which returns subsequent intermediate states within
	// the SOR operation. Upon SOR completion (DetectorState.RUN_OK), the server
	// closes the stream.
	StartOfRun(*SorRequest, Configurator_StartOfRunServer) error
	// Single EOR request for a data taking session, with per-detector parameters.
	// Returns an event stream which returns subsequent intermediate states within
	// the EOR operation. Upon EOR completion (DetectorState.RUN_OK), the server
	// closes the stream.
	EndOfRun(*EorRequest, Configurator_EndOfRunServer) error
	// Optional call, legal at any time, to query the status of the DCS service
	// and either some or all of its constituent detectors. This call returns a
	// single value (not a stream), reflecting the service state at that
	// specific moment.
	GetStatus(context.Context, *StatusRequest) (*StatusReply, error)
}

// UnimplementedConfiguratorServer should be embedded to have forward compatible implementations.
type UnimplementedConfiguratorServer struct {
}

func (UnimplementedConfiguratorServer) Subscribe(*SubscriptionRequest, Configurator_SubscribeServer) error {
	return status.Errorf(codes.Unimplemented, "method Subscribe not implemented")
}
func (UnimplementedConfiguratorServer) StartOfRun(*SorRequest, Configurator_StartOfRunServer) error {
	return status.Errorf(codes.Unimplemented, "method StartOfRun not implemented")
}
func (UnimplementedConfiguratorServer) EndOfRun(*EorRequest, Configurator_EndOfRunServer) error {
	return status.Errorf(codes.Unimplemented, "method EndOfRun not implemented")
}
func (UnimplementedConfiguratorServer) GetStatus(context.Context, *StatusRequest) (*StatusReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetStatus not implemented")
}

// UnsafeConfiguratorServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to ConfiguratorServer will
// result in compilation errors.
type UnsafeConfiguratorServer interface {
	mustEmbedUnimplementedConfiguratorServer()
}

func RegisterConfiguratorServer(s grpc.ServiceRegistrar, srv ConfiguratorServer) {
	s.RegisterService(&Configurator_ServiceDesc, srv)
}

func _Configurator_Subscribe_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(SubscriptionRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(ConfiguratorServer).Subscribe(m, &configuratorSubscribeServer{stream})
}

type Configurator_SubscribeServer interface {
	Send(*Event) error
	grpc.ServerStream
}

type configuratorSubscribeServer struct {
	grpc.ServerStream
}

func (x *configuratorSubscribeServer) Send(m *Event) error {
	return x.ServerStream.SendMsg(m)
}

func _Configurator_StartOfRun_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(SorRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(ConfiguratorServer).StartOfRun(m, &configuratorStartOfRunServer{stream})
}

type Configurator_StartOfRunServer interface {
	Send(*RunEvent) error
	grpc.ServerStream
}

type configuratorStartOfRunServer struct {
	grpc.ServerStream
}

func (x *configuratorStartOfRunServer) Send(m *RunEvent) error {
	return x.ServerStream.SendMsg(m)
}

func _Configurator_EndOfRun_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(EorRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(ConfiguratorServer).EndOfRun(m, &configuratorEndOfRunServer{stream})
}

type Configurator_EndOfRunServer interface {
	Send(*RunEvent) error
	grpc.ServerStream
}

type configuratorEndOfRunServer struct {
	grpc.ServerStream
}

func (x *configuratorEndOfRunServer) Send(m *RunEvent) error {
	return x.ServerStream.SendMsg(m)
}

func _Configurator_GetStatus_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(StatusRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ConfiguratorServer).GetStatus(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/dcs.Configurator/GetStatus",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ConfiguratorServer).GetStatus(ctx, req.(*StatusRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// Configurator_ServiceDesc is the grpc.ServiceDesc for Configurator service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Configurator_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "dcs.Configurator",
	HandlerType: (*ConfiguratorServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetStatus",
			Handler:    _Configurator_GetStatus_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "Subscribe",
			Handler:       _Configurator_Subscribe_Handler,
			ServerStreams: true,
		},
		{
			StreamName:    "StartOfRun",
			Handler:       _Configurator_StartOfRun_Handler,
			ServerStreams: true,
		},
		{
			StreamName:    "EndOfRun",
			Handler:       _Configurator_EndOfRun_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "protos/dcs.proto",
}
