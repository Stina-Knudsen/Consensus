// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.5.1
// - protoc             v5.28.2
// source: proto.proto

package grpc

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.64.0 or later.
const _ = grpc.SupportPackageIsVersion9

const (
	MutexService_RequestMessage_FullMethodName = "/proto.MutexService/RequestMessage"
)

// MutexServiceClient is the client API for MutexService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type MutexServiceClient interface {
	RequestMessage(ctx context.Context, in *Request, opts ...grpc.CallOption) (*Reply, error)
}

type mutexServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewMutexServiceClient(cc grpc.ClientConnInterface) MutexServiceClient {
	return &mutexServiceClient{cc}
}

func (c *mutexServiceClient) RequestMessage(ctx context.Context, in *Request, opts ...grpc.CallOption) (*Reply, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Reply)
	err := c.cc.Invoke(ctx, MutexService_RequestMessage_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// MutexServiceServer is the server API for MutexService service.
// All implementations must embed UnimplementedMutexServiceServer
// for forward compatibility.
type MutexServiceServer interface {
	RequestMessage(context.Context, *Request) (*Reply, error)
	mustEmbedUnimplementedMutexServiceServer()
}

// UnimplementedMutexServiceServer must be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedMutexServiceServer struct{}

func (UnimplementedMutexServiceServer) RequestMessage(context.Context, *Request) (*Reply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method RequestMessage not implemented")
}
func (UnimplementedMutexServiceServer) mustEmbedUnimplementedMutexServiceServer() {}
func (UnimplementedMutexServiceServer) testEmbeddedByValue()                      {}

// UnsafeMutexServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to MutexServiceServer will
// result in compilation errors.
type UnsafeMutexServiceServer interface {
	mustEmbedUnimplementedMutexServiceServer()
}

func RegisterMutexServiceServer(s grpc.ServiceRegistrar, srv MutexServiceServer) {
	// If the following call pancis, it indicates UnimplementedMutexServiceServer was
	// embedded by pointer and is nil.  This will cause panics if an
	// unimplemented method is ever invoked, so we test this at initialization
	// time to prevent it from happening at runtime later due to I/O.
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&MutexService_ServiceDesc, srv)
}

func _MutexService_RequestMessage_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Request)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MutexServiceServer).RequestMessage(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: MutexService_RequestMessage_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MutexServiceServer).RequestMessage(ctx, req.(*Request))
	}
	return interceptor(ctx, in, info, handler)
}

// MutexService_ServiceDesc is the grpc.ServiceDesc for MutexService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var MutexService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "proto.MutexService",
	HandlerType: (*MutexServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "RequestMessage",
			Handler:    _MutexService_RequestMessage_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "proto.proto",
}
