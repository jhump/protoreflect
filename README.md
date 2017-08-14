# Protocol Buffer and GRPC Reflection
[![Build Status](https://travis-ci.org/jhump/protoreflect.svg?branch=master)](https://travis-ci.org/jhump/protoreflect/branches)

This repo provides reflection APIs for protocol buffers (also known as "protobufs" for short)
and GRPC. The core of reflection in protobufs is the
[descriptor](https://github.com/google/protobuf/blob/199d82fde1734ab5bc931cd0de93309e50cd7ab9/src/google/protobuf/descriptor.proto).
A descriptor is itself a protobuf message that describes a `.proto` source file or any element
therein. So a collection of descriptors can describe an entire schema of protobuf types, including
RPC services.

*Go doc: https://godoc.org/github.com/jhump/protoreflect*

----
```
import "github.com/jhump/protoreflect/desc"
```

The `desc` package herein introduces a `Descriptor` interface and implementations of it that
correspond to each of the descriptor types. These new types are effectively smart wrappers around
the [generated protobuf types](https://github.com/golang/protobuf/blob/master/protoc-gen-go/descriptor/descriptor.pb.go)
that make them *much* more useful and easier to use.

----
```
import "github.com/jhump/protoreflect/grpcreflect"
```

The `grpcreflect` package provides an easy-to-use client for the
[GRPC reflection service](https://github.com/grpc/grpc-go/blob/6bd4f6eb1ea9d81d1209494242554dcde44429a4/reflection/grpc_reflection_v1alpha/reflection.proto#L36),
making it much easier to query for and work with the schemas of remote services.

----
```
import "github.com/jhump/protoreflect/dynamic"
import "github.com/jhump/protoreflect/dynamic/grpcdynamic"
```

The `dynamic` package provides a dynamic message implementation. It implements `proto.Message` but
is backed by a message descriptor and a map of fields->values, instead of a generated struct. There
is also sub-package named `grpcdynamic`, which provides a dynamic stub implementation. The stub can
be used to issue RPC methods using method descriptors instead of generated client interfaces.
