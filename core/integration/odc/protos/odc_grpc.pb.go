// Code generated by protoc-gen-go-grpc. DO NOT EDIT.

package odc

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

// ODCClient is the client API for ODC service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type ODCClient interface {
	// Creates a new DDS session or attaches to an existing DDS session.
	Initialize(ctx context.Context, in *InitializeRequest, opts ...grpc.CallOption) (*GeneralReply, error)
	// Submits DDS agents (deploys a dynamic cluster) according to a specified computing resources.
	// Can be called multiple times in order to submit more DDS agents (allocate more resources).
	Submit(ctx context.Context, in *SubmitRequest, opts ...grpc.CallOption) (*GeneralReply, error)
	// Activates a given topology.
	Activate(ctx context.Context, in *ActivateRequest, opts ...grpc.CallOption) (*GeneralReply, error)
	// Run request combines Initialize, Submit and Activate into a single request.
	// Run request always creates a new DDS session.
	Run(ctx context.Context, in *RunRequest, opts ...grpc.CallOption) (*GeneralReply, error)
	// Updates a topology (up or down scale number of tasks or any other topology change).
	// It consists of 3 commands: Reset, Activate and Configure.
	// Can be called multiple times.
	Update(ctx context.Context, in *UpdateRequest, opts ...grpc.CallOption) (*GeneralReply, error)
	// Transitions devices into Ready state.
	Configure(ctx context.Context, in *ConfigureRequest, opts ...grpc.CallOption) (*StateReply, error)
	// Changes devices configuration.
	SetProperties(ctx context.Context, in *SetPropertiesRequest, opts ...grpc.CallOption) (*GeneralReply, error)
	// Get current aggregated state of devices.
	GetState(ctx context.Context, in *StateRequest, opts ...grpc.CallOption) (*StateReply, error)
	// Transition devices into Running state.
	Start(ctx context.Context, in *StartRequest, opts ...grpc.CallOption) (*StateReply, error)
	// Transitions devices into Ready state.
	Stop(ctx context.Context, in *StopRequest, opts ...grpc.CallOption) (*StateReply, error)
	// Transitions devices into Idle state.
	Reset(ctx context.Context, in *ResetRequest, opts ...grpc.CallOption) (*StateReply, error)
	// Shuts devices down via End transition.
	Terminate(ctx context.Context, in *TerminateRequest, opts ...grpc.CallOption) (*StateReply, error)
	// Shutdown DDS session.
	Shutdown(ctx context.Context, in *ShutdownRequest, opts ...grpc.CallOption) (*GeneralReply, error)
}

type oDCClient struct {
	cc grpc.ClientConnInterface
}

func NewODCClient(cc grpc.ClientConnInterface) ODCClient {
	return &oDCClient{cc}
}

func (c *oDCClient) Initialize(ctx context.Context, in *InitializeRequest, opts ...grpc.CallOption) (*GeneralReply, error) {
	out := new(GeneralReply)
	err := c.cc.Invoke(ctx, "/odc.ODC/Initialize", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *oDCClient) Submit(ctx context.Context, in *SubmitRequest, opts ...grpc.CallOption) (*GeneralReply, error) {
	out := new(GeneralReply)
	err := c.cc.Invoke(ctx, "/odc.ODC/Submit", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *oDCClient) Activate(ctx context.Context, in *ActivateRequest, opts ...grpc.CallOption) (*GeneralReply, error) {
	out := new(GeneralReply)
	err := c.cc.Invoke(ctx, "/odc.ODC/Activate", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *oDCClient) Run(ctx context.Context, in *RunRequest, opts ...grpc.CallOption) (*GeneralReply, error) {
	out := new(GeneralReply)
	err := c.cc.Invoke(ctx, "/odc.ODC/Run", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *oDCClient) Update(ctx context.Context, in *UpdateRequest, opts ...grpc.CallOption) (*GeneralReply, error) {
	out := new(GeneralReply)
	err := c.cc.Invoke(ctx, "/odc.ODC/Update", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *oDCClient) Configure(ctx context.Context, in *ConfigureRequest, opts ...grpc.CallOption) (*StateReply, error) {
	out := new(StateReply)
	err := c.cc.Invoke(ctx, "/odc.ODC/Configure", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *oDCClient) SetProperties(ctx context.Context, in *SetPropertiesRequest, opts ...grpc.CallOption) (*GeneralReply, error) {
	out := new(GeneralReply)
	err := c.cc.Invoke(ctx, "/odc.ODC/SetProperties", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *oDCClient) GetState(ctx context.Context, in *StateRequest, opts ...grpc.CallOption) (*StateReply, error) {
	out := new(StateReply)
	err := c.cc.Invoke(ctx, "/odc.ODC/GetState", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *oDCClient) Start(ctx context.Context, in *StartRequest, opts ...grpc.CallOption) (*StateReply, error) {
	out := new(StateReply)
	err := c.cc.Invoke(ctx, "/odc.ODC/Start", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *oDCClient) Stop(ctx context.Context, in *StopRequest, opts ...grpc.CallOption) (*StateReply, error) {
	out := new(StateReply)
	err := c.cc.Invoke(ctx, "/odc.ODC/Stop", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *oDCClient) Reset(ctx context.Context, in *ResetRequest, opts ...grpc.CallOption) (*StateReply, error) {
	out := new(StateReply)
	err := c.cc.Invoke(ctx, "/odc.ODC/Reset", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *oDCClient) Terminate(ctx context.Context, in *TerminateRequest, opts ...grpc.CallOption) (*StateReply, error) {
	out := new(StateReply)
	err := c.cc.Invoke(ctx, "/odc.ODC/Terminate", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *oDCClient) Shutdown(ctx context.Context, in *ShutdownRequest, opts ...grpc.CallOption) (*GeneralReply, error) {
	out := new(GeneralReply)
	err := c.cc.Invoke(ctx, "/odc.ODC/Shutdown", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ODCServer is the server API for ODC service.
// All implementations should embed UnimplementedODCServer
// for forward compatibility
type ODCServer interface {
	// Creates a new DDS session or attaches to an existing DDS session.
	Initialize(context.Context, *InitializeRequest) (*GeneralReply, error)
	// Submits DDS agents (deploys a dynamic cluster) according to a specified computing resources.
	// Can be called multiple times in order to submit more DDS agents (allocate more resources).
	Submit(context.Context, *SubmitRequest) (*GeneralReply, error)
	// Activates a given topology.
	Activate(context.Context, *ActivateRequest) (*GeneralReply, error)
	// Run request combines Initialize, Submit and Activate into a single request.
	// Run request always creates a new DDS session.
	Run(context.Context, *RunRequest) (*GeneralReply, error)
	// Updates a topology (up or down scale number of tasks or any other topology change).
	// It consists of 3 commands: Reset, Activate and Configure.
	// Can be called multiple times.
	Update(context.Context, *UpdateRequest) (*GeneralReply, error)
	// Transitions devices into Ready state.
	Configure(context.Context, *ConfigureRequest) (*StateReply, error)
	// Changes devices configuration.
	SetProperties(context.Context, *SetPropertiesRequest) (*GeneralReply, error)
	// Get current aggregated state of devices.
	GetState(context.Context, *StateRequest) (*StateReply, error)
	// Transition devices into Running state.
	Start(context.Context, *StartRequest) (*StateReply, error)
	// Transitions devices into Ready state.
	Stop(context.Context, *StopRequest) (*StateReply, error)
	// Transitions devices into Idle state.
	Reset(context.Context, *ResetRequest) (*StateReply, error)
	// Shuts devices down via End transition.
	Terminate(context.Context, *TerminateRequest) (*StateReply, error)
	// Shutdown DDS session.
	Shutdown(context.Context, *ShutdownRequest) (*GeneralReply, error)
}

// UnimplementedODCServer should be embedded to have forward compatible implementations.
type UnimplementedODCServer struct {
}

func (UnimplementedODCServer) Initialize(context.Context, *InitializeRequest) (*GeneralReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Initialize not implemented")
}
func (UnimplementedODCServer) Submit(context.Context, *SubmitRequest) (*GeneralReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Submit not implemented")
}
func (UnimplementedODCServer) Activate(context.Context, *ActivateRequest) (*GeneralReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Activate not implemented")
}
func (UnimplementedODCServer) Run(context.Context, *RunRequest) (*GeneralReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Run not implemented")
}
func (UnimplementedODCServer) Update(context.Context, *UpdateRequest) (*GeneralReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Update not implemented")
}
func (UnimplementedODCServer) Configure(context.Context, *ConfigureRequest) (*StateReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Configure not implemented")
}
func (UnimplementedODCServer) SetProperties(context.Context, *SetPropertiesRequest) (*GeneralReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SetProperties not implemented")
}
func (UnimplementedODCServer) GetState(context.Context, *StateRequest) (*StateReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetState not implemented")
}
func (UnimplementedODCServer) Start(context.Context, *StartRequest) (*StateReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Start not implemented")
}
func (UnimplementedODCServer) Stop(context.Context, *StopRequest) (*StateReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Stop not implemented")
}
func (UnimplementedODCServer) Reset(context.Context, *ResetRequest) (*StateReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Reset not implemented")
}
func (UnimplementedODCServer) Terminate(context.Context, *TerminateRequest) (*StateReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Terminate not implemented")
}
func (UnimplementedODCServer) Shutdown(context.Context, *ShutdownRequest) (*GeneralReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Shutdown not implemented")
}

// UnsafeODCServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to ODCServer will
// result in compilation errors.
type UnsafeODCServer interface {
	mustEmbedUnimplementedODCServer()
}

func RegisterODCServer(s grpc.ServiceRegistrar, srv ODCServer) {
	s.RegisterService(&ODC_ServiceDesc, srv)
}

func _ODC_Initialize_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(InitializeRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ODCServer).Initialize(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/odc.ODC/Initialize",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ODCServer).Initialize(ctx, req.(*InitializeRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ODC_Submit_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SubmitRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ODCServer).Submit(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/odc.ODC/Submit",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ODCServer).Submit(ctx, req.(*SubmitRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ODC_Activate_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ActivateRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ODCServer).Activate(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/odc.ODC/Activate",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ODCServer).Activate(ctx, req.(*ActivateRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ODC_Run_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RunRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ODCServer).Run(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/odc.ODC/Run",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ODCServer).Run(ctx, req.(*RunRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ODC_Update_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UpdateRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ODCServer).Update(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/odc.ODC/Update",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ODCServer).Update(ctx, req.(*UpdateRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ODC_Configure_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ConfigureRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ODCServer).Configure(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/odc.ODC/Configure",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ODCServer).Configure(ctx, req.(*ConfigureRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ODC_SetProperties_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SetPropertiesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ODCServer).SetProperties(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/odc.ODC/SetProperties",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ODCServer).SetProperties(ctx, req.(*SetPropertiesRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ODC_GetState_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(StateRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ODCServer).GetState(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/odc.ODC/GetState",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ODCServer).GetState(ctx, req.(*StateRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ODC_Start_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(StartRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ODCServer).Start(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/odc.ODC/Start",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ODCServer).Start(ctx, req.(*StartRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ODC_Stop_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(StopRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ODCServer).Stop(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/odc.ODC/Stop",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ODCServer).Stop(ctx, req.(*StopRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ODC_Reset_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ResetRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ODCServer).Reset(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/odc.ODC/Reset",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ODCServer).Reset(ctx, req.(*ResetRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ODC_Terminate_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(TerminateRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ODCServer).Terminate(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/odc.ODC/Terminate",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ODCServer).Terminate(ctx, req.(*TerminateRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ODC_Shutdown_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ShutdownRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ODCServer).Shutdown(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/odc.ODC/Shutdown",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ODCServer).Shutdown(ctx, req.(*ShutdownRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// ODC_ServiceDesc is the grpc.ServiceDesc for ODC service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var ODC_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "odc.ODC",
	HandlerType: (*ODCServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Initialize",
			Handler:    _ODC_Initialize_Handler,
		},
		{
			MethodName: "Submit",
			Handler:    _ODC_Submit_Handler,
		},
		{
			MethodName: "Activate",
			Handler:    _ODC_Activate_Handler,
		},
		{
			MethodName: "Run",
			Handler:    _ODC_Run_Handler,
		},
		{
			MethodName: "Update",
			Handler:    _ODC_Update_Handler,
		},
		{
			MethodName: "Configure",
			Handler:    _ODC_Configure_Handler,
		},
		{
			MethodName: "SetProperties",
			Handler:    _ODC_SetProperties_Handler,
		},
		{
			MethodName: "GetState",
			Handler:    _ODC_GetState_Handler,
		},
		{
			MethodName: "Start",
			Handler:    _ODC_Start_Handler,
		},
		{
			MethodName: "Stop",
			Handler:    _ODC_Stop_Handler,
		},
		{
			MethodName: "Reset",
			Handler:    _ODC_Reset_Handler,
		},
		{
			MethodName: "Terminate",
			Handler:    _ODC_Terminate_Handler,
		},
		{
			MethodName: "Shutdown",
			Handler:    _ODC_Shutdown_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "protos/odc.proto",
}
