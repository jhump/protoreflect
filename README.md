# Protocol Buffer and gRPC Reflection
[![Build Status](https://circleci.com/gh/jhump/protoreflect/tree/main.svg?style=svg)](https://circleci.com/gh/jhump/protoreflect/tree/main)
[![Go Report Card](https://goreportcard.com/badge/github.com/jhump/protoreflect)](https://goreportcard.com/report/github.com/jhump/protoreflect)

This repo provides reflection APIs for [protocol buffers](https://developers.google.com/protocol-buffers/) (also known as "protobufs" for short)
and [gRPC](https://grpc.io/). The core of reflection in protobufs is the
[descriptor](https://github.com/google/protobuf/blob/199d82fde1734ab5bc931cd0de93309e50cd7ab9/src/google/protobuf/descriptor.proto).
A descriptor is itself a protobuf message that describes a `.proto` source file or any element
therein. So a collection of descriptors can describe an entire schema of protobuf types, including
RPC services.

[![GoDoc](https://godoc.org/github.com/jhump/protoreflect?status.svg)](https://godoc.org/github.com/jhump/protoreflect)

----
## Descriptors and Reflection Utilities

```go
import "github.com/jhump/protoreflect/v2/protobuilder"
```

The `protobuilder` package allows for programmatic construction of rich descriptors. Descriptors can
be constructed programmatically by creating trees of descriptor protos and using the `desc` package
to link those into rich descriptors. But constructing a valid tree of descriptor protos is far from
trivial.

So this package provides generous API to greatly simplify that task. It also allows for converting
rich descriptors into builders, which means you can programmatically modify/tweak existing
descriptors.

*[Read more ≫](https://pkg.go.dev/github.com/jhump/protoreflect/v2/protobuilder)*

```go
import "github.com/jhump/protoreflect/v2/protomessage"
```

...

*[Read more ≫](https://pkg.go.dev/github.com/jhump/protoreflect/v2/protomessage)*

```go
import "github.com/jhump/protoreflect/v2/protoprint"
```

The `protoprint` package allows for printing of descriptors to `.proto` source files. This is
effectively the inverse of the `protoparse` package. Combined with the `builder` package, this
is a useful tool for programmatically generating protocol buffer sources.

*[Read more ≫](https://pkg.go.dev/github.com/jhump/protoreflect/v2/protoprint)*

```go
import "github.com/jhump/protoreflect/v2/protoresolve"
```

...

*[Read more ≫](https://pkg.go.dev/github.com/jhump/protoreflect/v2/protoresolve)*

```go
import "github.com/jhump/protoreflect/v2/protowrap"
```

...

*[Read more ≫](https://pkg.go.dev/github.com/jhump/protoreflect/v2/protowrap)*

```go
import "github.com/jhump/protoreflect/v2/sourceloc"
```

...

*[Read more ≫](https://pkg.go.dev/github.com/jhump/protoreflect/v2/sourceloc)*

----
## gRPC Reflection and Dynamic RPC

```go
import "github.com/jhump/protoreflect/v2/grpcdynamic"
```

There is also sub-package named `grpcdynamic`, which provides a dynamic stub implementation. The stub can
be used to issue RPC methods using method descriptors instead of generated client interfaces.

*[Read more ≫](https://pkg.go.dev/github.com/jhump/protoreflect/v2/grpcdynamic)*

```go
import "github.com/jhump/protoreflect/v2/grpcreflect"
```

The `grpcreflect` package provides an easy-to-use client for the
[gRPC reflection service](https://github.com/grpc/grpc-go/blob/6bd4f6eb1ea9d81d1209494242554dcde44429a4/reflection/grpc_reflection_v1alpha/reflection.proto#L36),
making it much easier to query for and work with the schemas of remote services.

It also provides some helper methods for querying for rich service descriptors for the
services registered in a gRPC server.

*[Read more ≫](https://pkg.go.dev/github.com/jhump/protoreflect/v2/grpcreflect)*

----
## Source Code Info for Embedded Descriptors

```go
import "github.com/jhump/protoreflect/v2/sourceinfo"
```

...

*[Read more ≫](https://pkg.go.dev/github.com/jhump/protoreflect/v2/sourceinfo)*
