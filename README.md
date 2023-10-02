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

### ⚠️ Note

This repo was originally built to work with the "V1" API of the Protobuf runtime for Go: `github.com/golang/protobuf`.

Since the creation of this repo, a new runtime for Go has been release, a "V2" of the API in `google.golang.org/protobuf`. This new API now includes support for functionality that this repo implements:
  * _Descriptors_: This repo provides `github.com/jhump/protoreflect/desc`. The new API now provides alternative types in [`google.golang.org/protobuf/reflect/protoreflect`](https://pkg.go.dev/google.golang.org/protobuf/reflect/protoreflect). It also provides ways to access descriptors for statically linked types in [`google.golang.org/protobuf/reflect/protoregistry`](https://pkg.go.dev/google.golang.org/protobuf/reflect/protoregistry) and the ability to convert between these descriptor types and their "poorer cousins", [descriptor protos](https://pkg.go.dev/google.golang.org/protobuf/types/descriptorpb), in [`google.golang.org/protobuf/reflect/protodesc`](https://pkg.go.dev/google.golang.org/protobuf/reflect/protodesc)
  * _Dynamic Messages_: This repo provides `github.com/jhump/protoreflect/dynamic`. The new API now provides an alternative in [`google.golang.org/protobuf/types/dynamicpb`](https://pkg.go.dev/google.golang.org/protobuf/types/dynamicpb).
  * _Binary Wire Format_: This repo provides `github.com/jhump/protoreflect/codec`. The new API now provides an alternative in [`google.golang.org/protobuf/encoding/protowire`](https://pkg.go.dev/google.golang.org/protobuf/encoding/protowire).

Most protobuf users have likely upgraded to that newer runtime and thus encounter some friction using this repo. It is now recommended to use the above packages in the V2 Protobuf API _instead of_ using the corresponding packages in this repo. But that still leaves a lot of functionality in this repo, such as the `desc/builder`, `desc/protoparse`, `desc/protoprint`, `dynamic/grpcdynamic`, `dynamic/msgregistry`, and `grpcreflect` packages herein. And all of these packages build on the core `desc.Descriptor` types in this repo. As of v1.15.0, you can convert between this repo's `desc.Descriptor` types and the V2 API's `protoreflect.Descriptor` types using `Wrap` functions in the `desc` package and `Unwrap` methods on the `desc.Descriptor` types. That allows easier interop between these remaining useful packages and new V2 API descriptor implementations.

If you have code that uses the `dynamic` package in this repo and are trying to interop with V2 APIs, in some cases you can use the [`proto.MessageV2`](https://pkg.go.dev/github.com/golang/protobuf/proto#MessageV2) converter function (defined in the V1 `proto` package in `github.com/golang/protobuf/proto`). This wrapper does not provide 100% complete interop, so in some cases you may have to port your code over to the V2 API's `dynamicpb` package. (Sorry!)

Later this year (2023), we expect to cut a v2 of this whole repo. A lot of what's in this repo is no longer necessary, but some features still are. The v2 will _drop_ functionality now provided by the V2 Protobuf API. The remaining packages will be updated to make direct use of the V2 Protobuf API and have no more references to the old V1 API. One exception is that a v2 of this repo will _not_ include a new version of the `desc/protoparse` package in this repo -- that is already available in a brand new module named [`protocompile`](https://pkg.go.dev/github.com/bufbuild/protocompile).

----
## Descriptors: The Language Model of Protocol Buffers

```go
import "github.com/jhump/protoreflect/desc"
```

The `desc` package herein introduces a `Descriptor` interface and implementations of it that
correspond to each of the descriptor types. These new types are effectively smart wrappers around
the [generated protobuf types](https://github.com/golang/protobuf/blob/master/protoc-gen-go/descriptor/descriptor.pb.go)
that make them *much* more useful and easier to use.

You can construct descriptors from file descriptor sets (which can be generated by `protoc`), and
you can also load descriptors for messages and services that are linked into the current binary.
"What does it mean for messages and services to be linked in?" you may ask. It means your binary
imports a package that was generated by `protoc`. When you generate Go code from your `.proto`
sources, the resulting package has descriptor information embedded in it. The `desc` package allows
you to easily extract those embedded descriptors.

Descriptors can also be acquired directly from `.proto` source files (using the `protoparse` sub-package)
or by programmatically constructing them (using the `builder` sub-package).

*[Read more ≫](https://godoc.org/github.com/jhump/protoreflect/desc)*

```go
import "github.com/jhump/protoreflect/desc/protoparse"
```

The `protoparse` package allows for parsing of `.proto` source files into rich descriptors. Without
this package, you must invoke `protoc` to either generate a file descriptor set file or to generate
Go code (which has descriptor information embedded in it). This package allows reading the source
directly without having to invoke `protoc`.

*[Read more ≫](https://godoc.org/github.com/jhump/protoreflect/desc/protoparse)*

```go
import "github.com/jhump/protoreflect/desc/protoprint"
```

The `protoprint` package allows for printing of descriptors to `.proto` source files. This is
effectively the inverse of the `protoparse` package. Combined with the `builder` package, this
is a useful tool for programmatically generating protocol buffer sources.

*[Read more ≫](https://godoc.org/github.com/jhump/protoreflect/desc/protoprint)*

```go
import "github.com/jhump/protoreflect/desc/builder"
```

The `builder` package allows for programmatic construction of rich descriptors. Descriptors can
be constructed programmatically by creating trees of descriptor protos and using the `desc` package
to link those into rich descriptors. But constructing a valid tree of descriptor protos is far from
trivial.

So this package provides generous API to greatly simplify that task. It also allows for converting
rich descriptors into builders, which means you can programmatically modify/tweak existing
descriptors.

*[Read more ≫](https://godoc.org/github.com/jhump/protoreflect/desc/builder)*

----
## Dynamic Messages and Stubs

```go
import "github.com/jhump/protoreflect/dynamic"
```

The `dynamic` package provides a dynamic message implementation. It implements `proto.Message` but
is backed by a message descriptor and a map of fields->values, instead of a generated struct. This
is useful for acting generically with protocol buffer messages, without having to generate and link
in Go code for every kind of message. This is particularly useful for general-purpose tools that
need to operate on arbitrary protocol buffer schemas. This is made possible by having the tools load
descriptors at runtime.

*[Read more ≫](https://godoc.org/github.com/jhump/protoreflect/dynamic)*

```go
import "github.com/jhump/protoreflect/dynamic/grpcdynamic"
```

There is also sub-package named `grpcdynamic`, which provides a dynamic stub implementation. The stub can
be used to issue RPC methods using method descriptors instead of generated client interfaces.

*[Read more ≫](https://godoc.org/github.com/jhump/protoreflect/dynamic/grpcdynamic)*

----
## gRPC Server Reflection

```go
import "github.com/jhump/protoreflect/grpcreflect"
```

The `grpcreflect` package provides an easy-to-use client for the
[gRPC reflection service](https://github.com/grpc/grpc-go/blob/6bd4f6eb1ea9d81d1209494242554dcde44429a4/reflection/grpc_reflection_v1alpha/reflection.proto#L36),
making it much easier to query for and work with the schemas of remote services.

It also provides some helper methods for querying for rich service descriptors for the
services registered in a gRPC server.

*[Read more ≫](https://godoc.org/github.com/jhump/protoreflect/grpcreflect)*
