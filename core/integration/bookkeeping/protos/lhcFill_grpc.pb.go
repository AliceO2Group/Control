// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.3.0
// - protoc             v5.28.2
// source: protos/lhcFill.proto

package bkpb

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

const (
	LhcFillService_GetLast_FullMethodName = "/o2.bookkeeping.LhcFillService/GetLast"
)

// LhcFillServiceClient is the client API for LhcFillService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type LhcFillServiceClient interface {
	GetLast(ctx context.Context, in *LastLhcFillFetchRequest, opts ...grpc.CallOption) (*LhcFillWithRelations, error)
}

type lhcFillServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewLhcFillServiceClient(cc grpc.ClientConnInterface) LhcFillServiceClient {
	return &lhcFillServiceClient{cc}
}

func (c *lhcFillServiceClient) GetLast(ctx context.Context, in *LastLhcFillFetchRequest, opts ...grpc.CallOption) (*LhcFillWithRelations, error) {
	out := new(LhcFillWithRelations)
	err := c.cc.Invoke(ctx, LhcFillService_GetLast_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// LhcFillServiceServer is the server API for LhcFillService service.
// All implementations should embed UnimplementedLhcFillServiceServer
// for forward compatibility
type LhcFillServiceServer interface {
	GetLast(context.Context, *LastLhcFillFetchRequest) (*LhcFillWithRelations, error)
}

// UnimplementedLhcFillServiceServer should be embedded to have forward compatible implementations.
type UnimplementedLhcFillServiceServer struct {
}

func (UnimplementedLhcFillServiceServer) GetLast(context.Context, *LastLhcFillFetchRequest) (*LhcFillWithRelations, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetLast not implemented")
}

// UnsafeLhcFillServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to LhcFillServiceServer will
// result in compilation errors.
type UnsafeLhcFillServiceServer interface {
	mustEmbedUnimplementedLhcFillServiceServer()
}

func RegisterLhcFillServiceServer(s grpc.ServiceRegistrar, srv LhcFillServiceServer) {
	s.RegisterService(&LhcFillService_ServiceDesc, srv)
}

func _LhcFillService_GetLast_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(LastLhcFillFetchRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(LhcFillServiceServer).GetLast(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: LhcFillService_GetLast_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(LhcFillServiceServer).GetLast(ctx, req.(*LastLhcFillFetchRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// LhcFillService_ServiceDesc is the grpc.ServiceDesc for LhcFillService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var LhcFillService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "o2.bookkeeping.LhcFillService",
	HandlerType: (*LhcFillServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetLast",
			Handler:    _LhcFillService_GetLast_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "protos/lhcFill.proto",
}
