# Protocol Buffer Reflection

This repo provides reflection APIs for protocol buffers (also known as "protobufs" for short).
The core of reflection in protobufs is the [descriptor](https://github.com/google/protobuf/blob/199d82fde1734ab5bc931cd0de93309e50cd7ab9/src/google/protobuf/descriptor.proto).
A descriptor is itself a protobuf message that describes a `.proto` source file or any element
therein. So a collection of descriptors can describe an entire schema of protobuf types, including
RPC services.

The `desc` package herein introduces a `Descriptor` interface and implementations of it that
correspond to each of the built-in descriptor typess. These new types are effectively smart
wrappers around the built-in protobuf types that make descriptors *much* more useful and easier
to use.

The `grpcreflect` package provides an easy-to-use client for the
[GRPC reflection service](https://github.com/grpc/grpc-go/blob/6bd4f6eb1ea9d81d1209494242554dcde44429a4/reflection/grpc_reflection_v1alpha/reflection.proto#L36),
making it much easier to query for and work with the schemas of remote services.
