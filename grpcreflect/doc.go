// Package grpcreflect provides gRPC-specific extensions to protobuf reflection.
// This includes a way to access rich service descriptors for all services that
// a gRPC server exports.
//
// Also included is an easy-to-use client for the [gRPC reflection service]. This
// client makes it easy to ask a server (that supports the reflection service)
// for metadata on its exported services, which could be used to construct a
// dynamic client. (See the grpcdynamic package in this same repo for more on
// that.)
//
// [gRPC reflection service]: https://github.com/grpc/grpc/blob/master/src/proto/grpc/reflection/v1/reflection.proto
package grpcreflect
