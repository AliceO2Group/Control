// Code generated by protoc-gen-go-grpc. DO NOT EDIT.

package pb

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion7

// ControlClient is the client API for Control service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type ControlClient interface {
	TrackStatus(ctx context.Context, in *StatusRequest, opts ...grpc.CallOption) (Control_TrackStatusClient, error)
	GetFrameworkInfo(ctx context.Context, in *GetFrameworkInfoRequest, opts ...grpc.CallOption) (*GetFrameworkInfoReply, error)
	Teardown(ctx context.Context, in *TeardownRequest, opts ...grpc.CallOption) (*TeardownReply, error)
	GetEnvironments(ctx context.Context, in *GetEnvironmentsRequest, opts ...grpc.CallOption) (*GetEnvironmentsReply, error)
	NewEnvironment(ctx context.Context, in *NewEnvironmentRequest, opts ...grpc.CallOption) (*NewEnvironmentReply, error)
	GetEnvironment(ctx context.Context, in *GetEnvironmentRequest, opts ...grpc.CallOption) (*GetEnvironmentReply, error)
	ControlEnvironment(ctx context.Context, in *ControlEnvironmentRequest, opts ...grpc.CallOption) (*ControlEnvironmentReply, error)
	ModifyEnvironment(ctx context.Context, in *ModifyEnvironmentRequest, opts ...grpc.CallOption) (*ModifyEnvironmentReply, error)
	DestroyEnvironment(ctx context.Context, in *DestroyEnvironmentRequest, opts ...grpc.CallOption) (*DestroyEnvironmentReply, error)
	GetTasks(ctx context.Context, in *GetTasksRequest, opts ...grpc.CallOption) (*GetTasksReply, error)
	GetTask(ctx context.Context, in *GetTaskRequest, opts ...grpc.CallOption) (*GetTaskReply, error)
	CleanupTasks(ctx context.Context, in *CleanupTasksRequest, opts ...grpc.CallOption) (*CleanupTasksReply, error)
	GetRoles(ctx context.Context, in *GetRolesRequest, opts ...grpc.CallOption) (*GetRolesReply, error)
	GetWorkflowTemplates(ctx context.Context, in *GetWorkflowTemplatesRequest, opts ...grpc.CallOption) (*GetWorkflowTemplatesReply, error)
	ListRepos(ctx context.Context, in *ListReposRequest, opts ...grpc.CallOption) (*ListReposReply, error)
	AddRepo(ctx context.Context, in *AddRepoRequest, opts ...grpc.CallOption) (*AddRepoReply, error)
	RemoveRepo(ctx context.Context, in *RemoveRepoRequest, opts ...grpc.CallOption) (*RemoveRepoReply, error)
	RefreshRepos(ctx context.Context, in *RefreshReposRequest, opts ...grpc.CallOption) (*Empty, error)
	SetDefaultRepo(ctx context.Context, in *SetDefaultRepoRequest, opts ...grpc.CallOption) (*Empty, error)
	SetGlobalDefaultRevision(ctx context.Context, in *SetGlobalDefaultRevisionRequest, opts ...grpc.CallOption) (*Empty, error)
	SetRepoDefaultRevision(ctx context.Context, in *SetRepoDefaultRevisionRequest, opts ...grpc.CallOption) (*SetRepoDefaultRevisionReply, error)
	Subscribe(ctx context.Context, in *SubscribeRequest, opts ...grpc.CallOption) (Control_SubscribeClient, error)
}

type controlClient struct {
	cc grpc.ClientConnInterface
}

func NewControlClient(cc grpc.ClientConnInterface) ControlClient {
	return &controlClient{cc}
}

func (c *controlClient) TrackStatus(ctx context.Context, in *StatusRequest, opts ...grpc.CallOption) (Control_TrackStatusClient, error) {
	stream, err := c.cc.NewStream(ctx, &_Control_serviceDesc.Streams[0], "/o2control.Control/TrackStatus", opts...)
	if err != nil {
		return nil, err
	}
	x := &controlTrackStatusClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type Control_TrackStatusClient interface {
	Recv() (*StatusReply, error)
	grpc.ClientStream
}

type controlTrackStatusClient struct {
	grpc.ClientStream
}

func (x *controlTrackStatusClient) Recv() (*StatusReply, error) {
	m := new(StatusReply)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *controlClient) GetFrameworkInfo(ctx context.Context, in *GetFrameworkInfoRequest, opts ...grpc.CallOption) (*GetFrameworkInfoReply, error) {
	out := new(GetFrameworkInfoReply)
	err := c.cc.Invoke(ctx, "/o2control.Control/GetFrameworkInfo", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *controlClient) Teardown(ctx context.Context, in *TeardownRequest, opts ...grpc.CallOption) (*TeardownReply, error) {
	out := new(TeardownReply)
	err := c.cc.Invoke(ctx, "/o2control.Control/Teardown", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *controlClient) GetEnvironments(ctx context.Context, in *GetEnvironmentsRequest, opts ...grpc.CallOption) (*GetEnvironmentsReply, error) {
	out := new(GetEnvironmentsReply)
	err := c.cc.Invoke(ctx, "/o2control.Control/GetEnvironments", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *controlClient) NewEnvironment(ctx context.Context, in *NewEnvironmentRequest, opts ...grpc.CallOption) (*NewEnvironmentReply, error) {
	out := new(NewEnvironmentReply)
	err := c.cc.Invoke(ctx, "/o2control.Control/NewEnvironment", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *controlClient) GetEnvironment(ctx context.Context, in *GetEnvironmentRequest, opts ...grpc.CallOption) (*GetEnvironmentReply, error) {
	out := new(GetEnvironmentReply)
	err := c.cc.Invoke(ctx, "/o2control.Control/GetEnvironment", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *controlClient) ControlEnvironment(ctx context.Context, in *ControlEnvironmentRequest, opts ...grpc.CallOption) (*ControlEnvironmentReply, error) {
	out := new(ControlEnvironmentReply)
	err := c.cc.Invoke(ctx, "/o2control.Control/ControlEnvironment", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *controlClient) ModifyEnvironment(ctx context.Context, in *ModifyEnvironmentRequest, opts ...grpc.CallOption) (*ModifyEnvironmentReply, error) {
	out := new(ModifyEnvironmentReply)
	err := c.cc.Invoke(ctx, "/o2control.Control/ModifyEnvironment", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *controlClient) DestroyEnvironment(ctx context.Context, in *DestroyEnvironmentRequest, opts ...grpc.CallOption) (*DestroyEnvironmentReply, error) {
	out := new(DestroyEnvironmentReply)
	err := c.cc.Invoke(ctx, "/o2control.Control/DestroyEnvironment", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *controlClient) GetTasks(ctx context.Context, in *GetTasksRequest, opts ...grpc.CallOption) (*GetTasksReply, error) {
	out := new(GetTasksReply)
	err := c.cc.Invoke(ctx, "/o2control.Control/GetTasks", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *controlClient) GetTask(ctx context.Context, in *GetTaskRequest, opts ...grpc.CallOption) (*GetTaskReply, error) {
	out := new(GetTaskReply)
	err := c.cc.Invoke(ctx, "/o2control.Control/GetTask", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *controlClient) CleanupTasks(ctx context.Context, in *CleanupTasksRequest, opts ...grpc.CallOption) (*CleanupTasksReply, error) {
	out := new(CleanupTasksReply)
	err := c.cc.Invoke(ctx, "/o2control.Control/CleanupTasks", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *controlClient) GetRoles(ctx context.Context, in *GetRolesRequest, opts ...grpc.CallOption) (*GetRolesReply, error) {
	out := new(GetRolesReply)
	err := c.cc.Invoke(ctx, "/o2control.Control/GetRoles", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *controlClient) GetWorkflowTemplates(ctx context.Context, in *GetWorkflowTemplatesRequest, opts ...grpc.CallOption) (*GetWorkflowTemplatesReply, error) {
	out := new(GetWorkflowTemplatesReply)
	err := c.cc.Invoke(ctx, "/o2control.Control/GetWorkflowTemplates", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *controlClient) ListRepos(ctx context.Context, in *ListReposRequest, opts ...grpc.CallOption) (*ListReposReply, error) {
	out := new(ListReposReply)
	err := c.cc.Invoke(ctx, "/o2control.Control/ListRepos", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *controlClient) AddRepo(ctx context.Context, in *AddRepoRequest, opts ...grpc.CallOption) (*AddRepoReply, error) {
	out := new(AddRepoReply)
	err := c.cc.Invoke(ctx, "/o2control.Control/AddRepo", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *controlClient) RemoveRepo(ctx context.Context, in *RemoveRepoRequest, opts ...grpc.CallOption) (*RemoveRepoReply, error) {
	out := new(RemoveRepoReply)
	err := c.cc.Invoke(ctx, "/o2control.Control/RemoveRepo", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *controlClient) RefreshRepos(ctx context.Context, in *RefreshReposRequest, opts ...grpc.CallOption) (*Empty, error) {
	out := new(Empty)
	err := c.cc.Invoke(ctx, "/o2control.Control/RefreshRepos", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *controlClient) SetDefaultRepo(ctx context.Context, in *SetDefaultRepoRequest, opts ...grpc.CallOption) (*Empty, error) {
	out := new(Empty)
	err := c.cc.Invoke(ctx, "/o2control.Control/SetDefaultRepo", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *controlClient) SetGlobalDefaultRevision(ctx context.Context, in *SetGlobalDefaultRevisionRequest, opts ...grpc.CallOption) (*Empty, error) {
	out := new(Empty)
	err := c.cc.Invoke(ctx, "/o2control.Control/SetGlobalDefaultRevision", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *controlClient) SetRepoDefaultRevision(ctx context.Context, in *SetRepoDefaultRevisionRequest, opts ...grpc.CallOption) (*SetRepoDefaultRevisionReply, error) {
	out := new(SetRepoDefaultRevisionReply)
	err := c.cc.Invoke(ctx, "/o2control.Control/SetRepoDefaultRevision", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *controlClient) Subscribe(ctx context.Context, in *SubscribeRequest, opts ...grpc.CallOption) (Control_SubscribeClient, error) {
	stream, err := c.cc.NewStream(ctx, &_Control_serviceDesc.Streams[1], "/o2control.Control/Subscribe", opts...)
	if err != nil {
		return nil, err
	}
	x := &controlSubscribeClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type Control_SubscribeClient interface {
	Recv() (*Event, error)
	grpc.ClientStream
}

type controlSubscribeClient struct {
	grpc.ClientStream
}

func (x *controlSubscribeClient) Recv() (*Event, error) {
	m := new(Event)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// ControlServer is the server API for Control service.
// All implementations should embed UnimplementedControlServer
// for forward compatibility
type ControlServer interface {
	TrackStatus(*StatusRequest, Control_TrackStatusServer) error
	GetFrameworkInfo(context.Context, *GetFrameworkInfoRequest) (*GetFrameworkInfoReply, error)
	Teardown(context.Context, *TeardownRequest) (*TeardownReply, error)
	GetEnvironments(context.Context, *GetEnvironmentsRequest) (*GetEnvironmentsReply, error)
	NewEnvironment(context.Context, *NewEnvironmentRequest) (*NewEnvironmentReply, error)
	GetEnvironment(context.Context, *GetEnvironmentRequest) (*GetEnvironmentReply, error)
	ControlEnvironment(context.Context, *ControlEnvironmentRequest) (*ControlEnvironmentReply, error)
	ModifyEnvironment(context.Context, *ModifyEnvironmentRequest) (*ModifyEnvironmentReply, error)
	DestroyEnvironment(context.Context, *DestroyEnvironmentRequest) (*DestroyEnvironmentReply, error)
	GetTasks(context.Context, *GetTasksRequest) (*GetTasksReply, error)
	GetTask(context.Context, *GetTaskRequest) (*GetTaskReply, error)
	CleanupTasks(context.Context, *CleanupTasksRequest) (*CleanupTasksReply, error)
	GetRoles(context.Context, *GetRolesRequest) (*GetRolesReply, error)
	GetWorkflowTemplates(context.Context, *GetWorkflowTemplatesRequest) (*GetWorkflowTemplatesReply, error)
	ListRepos(context.Context, *ListReposRequest) (*ListReposReply, error)
	AddRepo(context.Context, *AddRepoRequest) (*AddRepoReply, error)
	RemoveRepo(context.Context, *RemoveRepoRequest) (*RemoveRepoReply, error)
	RefreshRepos(context.Context, *RefreshReposRequest) (*Empty, error)
	SetDefaultRepo(context.Context, *SetDefaultRepoRequest) (*Empty, error)
	SetGlobalDefaultRevision(context.Context, *SetGlobalDefaultRevisionRequest) (*Empty, error)
	SetRepoDefaultRevision(context.Context, *SetRepoDefaultRevisionRequest) (*SetRepoDefaultRevisionReply, error)
	Subscribe(*SubscribeRequest, Control_SubscribeServer) error
}

// UnimplementedControlServer should be embedded to have forward compatible implementations.
type UnimplementedControlServer struct {
}

func (UnimplementedControlServer) TrackStatus(*StatusRequest, Control_TrackStatusServer) error {
	return status.Errorf(codes.Unimplemented, "method TrackStatus not implemented")
}
func (UnimplementedControlServer) GetFrameworkInfo(context.Context, *GetFrameworkInfoRequest) (*GetFrameworkInfoReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetFrameworkInfo not implemented")
}
func (UnimplementedControlServer) Teardown(context.Context, *TeardownRequest) (*TeardownReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Teardown not implemented")
}
func (UnimplementedControlServer) GetEnvironments(context.Context, *GetEnvironmentsRequest) (*GetEnvironmentsReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetEnvironments not implemented")
}
func (UnimplementedControlServer) NewEnvironment(context.Context, *NewEnvironmentRequest) (*NewEnvironmentReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method NewEnvironment not implemented")
}
func (UnimplementedControlServer) GetEnvironment(context.Context, *GetEnvironmentRequest) (*GetEnvironmentReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetEnvironment not implemented")
}
func (UnimplementedControlServer) ControlEnvironment(context.Context, *ControlEnvironmentRequest) (*ControlEnvironmentReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ControlEnvironment not implemented")
}
func (UnimplementedControlServer) ModifyEnvironment(context.Context, *ModifyEnvironmentRequest) (*ModifyEnvironmentReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ModifyEnvironment not implemented")
}
func (UnimplementedControlServer) DestroyEnvironment(context.Context, *DestroyEnvironmentRequest) (*DestroyEnvironmentReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DestroyEnvironment not implemented")
}
func (UnimplementedControlServer) GetTasks(context.Context, *GetTasksRequest) (*GetTasksReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetTasks not implemented")
}
func (UnimplementedControlServer) GetTask(context.Context, *GetTaskRequest) (*GetTaskReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetTask not implemented")
}
func (UnimplementedControlServer) CleanupTasks(context.Context, *CleanupTasksRequest) (*CleanupTasksReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CleanupTasks not implemented")
}
func (UnimplementedControlServer) GetRoles(context.Context, *GetRolesRequest) (*GetRolesReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetRoles not implemented")
}
func (UnimplementedControlServer) GetWorkflowTemplates(context.Context, *GetWorkflowTemplatesRequest) (*GetWorkflowTemplatesReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetWorkflowTemplates not implemented")
}
func (UnimplementedControlServer) ListRepos(context.Context, *ListReposRequest) (*ListReposReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListRepos not implemented")
}
func (UnimplementedControlServer) AddRepo(context.Context, *AddRepoRequest) (*AddRepoReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method AddRepo not implemented")
}
func (UnimplementedControlServer) RemoveRepo(context.Context, *RemoveRepoRequest) (*RemoveRepoReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method RemoveRepo not implemented")
}
func (UnimplementedControlServer) RefreshRepos(context.Context, *RefreshReposRequest) (*Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method RefreshRepos not implemented")
}
func (UnimplementedControlServer) SetDefaultRepo(context.Context, *SetDefaultRepoRequest) (*Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SetDefaultRepo not implemented")
}
func (UnimplementedControlServer) SetGlobalDefaultRevision(context.Context, *SetGlobalDefaultRevisionRequest) (*Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SetGlobalDefaultRevision not implemented")
}
func (UnimplementedControlServer) SetRepoDefaultRevision(context.Context, *SetRepoDefaultRevisionRequest) (*SetRepoDefaultRevisionReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SetRepoDefaultRevision not implemented")
}
func (UnimplementedControlServer) Subscribe(*SubscribeRequest, Control_SubscribeServer) error {
	return status.Errorf(codes.Unimplemented, "method Subscribe not implemented")
}

// UnsafeControlServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to ControlServer will
// result in compilation errors.
type UnsafeControlServer interface {
	mustEmbedUnimplementedControlServer()
}

func RegisterControlServer(s grpc.ServiceRegistrar, srv ControlServer) {
	s.RegisterService(&_Control_serviceDesc, srv)
}

func _Control_TrackStatus_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(StatusRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(ControlServer).TrackStatus(m, &controlTrackStatusServer{stream})
}

type Control_TrackStatusServer interface {
	Send(*StatusReply) error
	grpc.ServerStream
}

type controlTrackStatusServer struct {
	grpc.ServerStream
}

func (x *controlTrackStatusServer) Send(m *StatusReply) error {
	return x.ServerStream.SendMsg(m)
}

func _Control_GetFrameworkInfo_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetFrameworkInfoRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ControlServer).GetFrameworkInfo(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/o2control.Control/GetFrameworkInfo",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ControlServer).GetFrameworkInfo(ctx, req.(*GetFrameworkInfoRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Control_Teardown_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(TeardownRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ControlServer).Teardown(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/o2control.Control/Teardown",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ControlServer).Teardown(ctx, req.(*TeardownRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Control_GetEnvironments_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetEnvironmentsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ControlServer).GetEnvironments(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/o2control.Control/GetEnvironments",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ControlServer).GetEnvironments(ctx, req.(*GetEnvironmentsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Control_NewEnvironment_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(NewEnvironmentRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ControlServer).NewEnvironment(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/o2control.Control/NewEnvironment",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ControlServer).NewEnvironment(ctx, req.(*NewEnvironmentRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Control_GetEnvironment_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetEnvironmentRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ControlServer).GetEnvironment(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/o2control.Control/GetEnvironment",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ControlServer).GetEnvironment(ctx, req.(*GetEnvironmentRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Control_ControlEnvironment_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ControlEnvironmentRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ControlServer).ControlEnvironment(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/o2control.Control/ControlEnvironment",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ControlServer).ControlEnvironment(ctx, req.(*ControlEnvironmentRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Control_ModifyEnvironment_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ModifyEnvironmentRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ControlServer).ModifyEnvironment(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/o2control.Control/ModifyEnvironment",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ControlServer).ModifyEnvironment(ctx, req.(*ModifyEnvironmentRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Control_DestroyEnvironment_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DestroyEnvironmentRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ControlServer).DestroyEnvironment(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/o2control.Control/DestroyEnvironment",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ControlServer).DestroyEnvironment(ctx, req.(*DestroyEnvironmentRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Control_GetTasks_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetTasksRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ControlServer).GetTasks(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/o2control.Control/GetTasks",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ControlServer).GetTasks(ctx, req.(*GetTasksRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Control_GetTask_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetTaskRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ControlServer).GetTask(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/o2control.Control/GetTask",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ControlServer).GetTask(ctx, req.(*GetTaskRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Control_CleanupTasks_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CleanupTasksRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ControlServer).CleanupTasks(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/o2control.Control/CleanupTasks",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ControlServer).CleanupTasks(ctx, req.(*CleanupTasksRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Control_GetRoles_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetRolesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ControlServer).GetRoles(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/o2control.Control/GetRoles",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ControlServer).GetRoles(ctx, req.(*GetRolesRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Control_GetWorkflowTemplates_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetWorkflowTemplatesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ControlServer).GetWorkflowTemplates(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/o2control.Control/GetWorkflowTemplates",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ControlServer).GetWorkflowTemplates(ctx, req.(*GetWorkflowTemplatesRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Control_ListRepos_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListReposRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ControlServer).ListRepos(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/o2control.Control/ListRepos",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ControlServer).ListRepos(ctx, req.(*ListReposRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Control_AddRepo_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(AddRepoRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ControlServer).AddRepo(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/o2control.Control/AddRepo",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ControlServer).AddRepo(ctx, req.(*AddRepoRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Control_RemoveRepo_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RemoveRepoRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ControlServer).RemoveRepo(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/o2control.Control/RemoveRepo",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ControlServer).RemoveRepo(ctx, req.(*RemoveRepoRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Control_RefreshRepos_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RefreshReposRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ControlServer).RefreshRepos(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/o2control.Control/RefreshRepos",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ControlServer).RefreshRepos(ctx, req.(*RefreshReposRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Control_SetDefaultRepo_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SetDefaultRepoRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ControlServer).SetDefaultRepo(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/o2control.Control/SetDefaultRepo",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ControlServer).SetDefaultRepo(ctx, req.(*SetDefaultRepoRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Control_SetGlobalDefaultRevision_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SetGlobalDefaultRevisionRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ControlServer).SetGlobalDefaultRevision(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/o2control.Control/SetGlobalDefaultRevision",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ControlServer).SetGlobalDefaultRevision(ctx, req.(*SetGlobalDefaultRevisionRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Control_SetRepoDefaultRevision_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SetRepoDefaultRevisionRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ControlServer).SetRepoDefaultRevision(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/o2control.Control/SetRepoDefaultRevision",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ControlServer).SetRepoDefaultRevision(ctx, req.(*SetRepoDefaultRevisionRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Control_Subscribe_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(SubscribeRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(ControlServer).Subscribe(m, &controlSubscribeServer{stream})
}

type Control_SubscribeServer interface {
	Send(*Event) error
	grpc.ServerStream
}

type controlSubscribeServer struct {
	grpc.ServerStream
}

func (x *controlSubscribeServer) Send(m *Event) error {
	return x.ServerStream.SendMsg(m)
}

var _Control_serviceDesc = grpc.ServiceDesc{
	ServiceName: "o2control.Control",
	HandlerType: (*ControlServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetFrameworkInfo",
			Handler:    _Control_GetFrameworkInfo_Handler,
		},
		{
			MethodName: "Teardown",
			Handler:    _Control_Teardown_Handler,
		},
		{
			MethodName: "GetEnvironments",
			Handler:    _Control_GetEnvironments_Handler,
		},
		{
			MethodName: "NewEnvironment",
			Handler:    _Control_NewEnvironment_Handler,
		},
		{
			MethodName: "GetEnvironment",
			Handler:    _Control_GetEnvironment_Handler,
		},
		{
			MethodName: "ControlEnvironment",
			Handler:    _Control_ControlEnvironment_Handler,
		},
		{
			MethodName: "ModifyEnvironment",
			Handler:    _Control_ModifyEnvironment_Handler,
		},
		{
			MethodName: "DestroyEnvironment",
			Handler:    _Control_DestroyEnvironment_Handler,
		},
		{
			MethodName: "GetTasks",
			Handler:    _Control_GetTasks_Handler,
		},
		{
			MethodName: "GetTask",
			Handler:    _Control_GetTask_Handler,
		},
		{
			MethodName: "CleanupTasks",
			Handler:    _Control_CleanupTasks_Handler,
		},
		{
			MethodName: "GetRoles",
			Handler:    _Control_GetRoles_Handler,
		},
		{
			MethodName: "GetWorkflowTemplates",
			Handler:    _Control_GetWorkflowTemplates_Handler,
		},
		{
			MethodName: "ListRepos",
			Handler:    _Control_ListRepos_Handler,
		},
		{
			MethodName: "AddRepo",
			Handler:    _Control_AddRepo_Handler,
		},
		{
			MethodName: "RemoveRepo",
			Handler:    _Control_RemoveRepo_Handler,
		},
		{
			MethodName: "RefreshRepos",
			Handler:    _Control_RefreshRepos_Handler,
		},
		{
			MethodName: "SetDefaultRepo",
			Handler:    _Control_SetDefaultRepo_Handler,
		},
		{
			MethodName: "SetGlobalDefaultRevision",
			Handler:    _Control_SetGlobalDefaultRevision_Handler,
		},
		{
			MethodName: "SetRepoDefaultRevision",
			Handler:    _Control_SetRepoDefaultRevision_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "TrackStatus",
			Handler:       _Control_TrackStatus_Handler,
			ServerStreams: true,
		},
		{
			StreamName:    "Subscribe",
			Handler:       _Control_Subscribe_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "protos/o2control.proto",
}
