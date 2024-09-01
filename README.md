# Protocol Buffer and gRPC Reflection
[![Build Status](https://circleci.com/gh/jhump/protoreflect/tree/v2.svg?style=svg)](https://circleci.com/gh/jhump/protoreflect/tree/v2)
[![Go Report Card](https://goreportcard.com/badge/github.com/jhump/protoreflect)](https://goreportcard.com/report/github.com/jhump/protoreflect)

This repo builds on top of the reflection capabilities in the [Protobuf runtime for Go](https://pkg.go.dev/google.golang.org/protobuf/reflect/protoreflect)
and also provides reflection APIs for [gRPC](https://grpc.io/) as well.

[![GoDoc](https://pkg.go.dev/github.com/jhump/protoreflect/v2?status.svg)](https://pkg.go.dev/github.com/jhump/protoreflect/v2)

> [!NOTE]
> This v2 branch is a work in progress. It is basically feature complete, but still needs more tests for the new functionality
> and may also need some changes to the API to ensure long-term compatibility with interfaces in the Protobuf Go runtime.
>
> You can try it out by getting a pre-release version:
> ```
> go get github.com/jhump/protoreflect/v2@c9ae7caed596cda2e3c4a90f5973a46081a371a
> ```
>
> Note that the APIs are likely to change a little bit between now and a formal v2 release. Also note that some packages in the v2 still need more tests, so you may find some bugs, but that is mostly for new functionality. If you're just trying to update your code from v1 of this repo, those packages should be rock-solid and least likely to see any further API changes.

----
## Descriptors and Reflection Utilities

The [`protoreflect`](https://pkg.go.dev/google.golang.org/protobuf/reflect/protoreflect) package in
the Protobuf Go runtime provides the `Descriptor` interface and implementations of it that correspond
to each of the descriptor types. These types are effectively smart wrappers around the generated Protobuf
types in the [`descriptorpb`](https://pkg.go.dev/google.golang.org/protobuf/types/descriptorpb) package.
These wrappers make descriptors *much* more useful and easier to use.

This repo provides some additional packages for using and interacting with descriptors.

```go
import "github.com/jhump/protoreflect/v2/protoprint"
```

The `protoprint` package allows for printing of descriptors to `.proto` source files. This is
effectively the inverse of a parser/compiler (such as the [`protocompile`](https://pkg.go.dev/github.com/bufbuild/protocompile)
package.) Combined with the `protobuilder` package, this is a useful tool for programmatically
generating protocol buffer sources.

*[Read more ≫](https://pkg.go.dev/github.com/jhump/protoreflect/v2/protoprint)*

```go
import "github.com/jhump/protoreflect/v2/protobuilder"
```

The `protobuilder` package allows for programmatic construction of rich descriptors. Descriptors can
be constructed programmatically by creating trees of descriptor protos and using the [`protodesc`](https://pkg.go.dev/google.golang.org/protobuf/reflect/protodesc)
package to link those into rich descriptors. But constructing a valid tree of descriptor protos is far
from trivial.

So this package provides generous API to greatly simplify that task. It also allows for converting
rich descriptors into builders, which means you can programmatically modify/tweak existing
descriptors.

*[Read more ≫](https://pkg.go.dev/github.com/jhump/protoreflect/v2/protobuilder)*

```go
import "github.com/jhump/protoreflect/v2/protoresolve"
```

The `protoresolve` package provides named interfaces for many kinds of resolvers. It also provides
a `Resolver` interface that acts like a union of the various resolver interfaces and unifies both
_descriptor_ resolvers and _type_ resolvers. The former returns descriptor instances; the latter
returns types (often implemented by the `dynamicpb` package). These interfaces provide a comprehensive
set of types for resolving elements in Protobuf schemas and effectively _extend_ the APIs in
the [`protoregistry`](https://pkg.go.dev/google.golang.org/protobuf/reflect/protoregistry) package
provided by the Protobuf Go runtime.

*[Read more ≫](https://pkg.go.dev/github.com/jhump/protoreflect/v2/protoresolve)*

```go
import "github.com/jhump/protoreflect/v2/protomessage"
```

The `protomessage` package contains helpers for work with `proto.Message` instances from generic
and/or dynamic code.

*[Read more ≫](https://pkg.go.dev/github.com/jhump/protoreflect/v2/protomessage)*

```go
import "github.com/jhump/protoreflect/v2/protowrap"
```

The `protowrap` package defines interfaces for _wrapping_ the `protoreflect.Descriptor` interfaces.
This is **experimental**. Implementations of the `protoreflect.Descriptor` interfaces outside of the
Protobuf Go runtime are not officially supported by the Protobuf Go runtime. So though this package
works as of this wrigin, it will likely be replaced with an alternate formulation for exposing the
same functionality, to ensure long-term compatibility with the Protobuf Go runtime.

The main functionality provided is an efficient way to associate `descriptorpb` descriptor protos
with their "richer" `protoreflect` descriptor cousins. Without this ability, acquiring a descriptor
proto that is the underlying data for a descriptor requires a non-trivial conversion and likely
application logic to memoize the results. This package provided the ability by _wrapping_ the
descriptors and exposing an extra exported method for efficiently recovering the descriptor proto.

*[Read more ≫](https://pkg.go.dev/github.com/jhump/protoreflect/v2/protowrap)*

----
## Source Code Info

Generated Protobuf types in Go do not include "source code information". Source code information
is data that comes from the original Protobuf source file that defined messages and includes things
like position information (i.e. the filename, line, and column on which a message, enum, or service
was defined) and comments.

This repo includes some APIs to help work with source code info and also a mechanism (and Proto plugin)
for restoring the source code information to the descriptors embedded in generated Go code.

```go
import "github.com/jhump/protoreflect/v2/sourceinfo"
```

The `sourceinfo` package contains APIs that for retreiving descriptors for generated types that include
source code info. When generating Go code, source code information is not preserved. But if you also
generate code using the included `protoc-gen-gosrcinfo` plugin and query for the descriptors using this
package, you can access that information. The most immediate use of this information is to provide
comments for services, methods, and types to dynamic RPC clients that use the gRPC server reflection
service.

*[Read more ≫](https://pkg.go.dev/github.com/jhump/protoreflect/v2/sourceinfo)*

```go
import "github.com/jhump/protoreflect/v2/sourceloc"
```

The `sourceloc` package contains helpers for working with instances of `protoreflect.SourceLocation`
and `protoreflect.SourcePath`.

*[Read more ≫](https://pkg.go.dev/github.com/jhump/protoreflect/v2/sourceloc)*

----
## Dynamic RPC Stubs

The [`dynamicpb`](https://pkg.go.dev/google.golang.org/protobuf/types/dynamicpb) package in the Protobuf
Go runtime provides a dynamic message implementation. It implements `proto.Message` but is backed by a
message descriptor and a map of fields->values, instead of a generated struct. This is useful for acting
generically with protocol buffer messages, without having to generate and link in Go code for every kind
of message. This is particularly useful for general-purpose tools that need to operate on arbitrary
Protobuf schemas. This is made possible by having the tools load descriptors at runtime.

This repo provides capabilities on top of `dynamicpb` to not only use message schemas dynamically but to
also use RPC schemas dynamically. This enables invoking RPCs without having any generated code for the
RPC service to be used.

```go
import "github.com/jhump/protoreflect/v2/grpcdynamic"
```

The `grpcdynamic` package provides the dynamic stub implementation. The stub can be used to issue
RPC methods using method descriptors instead of generated client interfaces.

*[Read more ≫](https://pkg.go.dev/github.com/jhump/protoreflect/v2/grpcdynamic)*

----
## gRPC Server Reflection

```go
import "github.com/jhump/protoreflect/v2/grpcreflect"
```

The `grpcreflect` package provides an easy-to-use client for the
[gRPC reflection service](https://github.com/grpc/grpc-go/blob/6bd4f6eb1ea9d81d1209494242554dcde44429a4/reflection/grpc_reflection_v1alpha/reflection.proto#L36),
making it much easier to query for and work with the schemas of remote services.

It also provides some helper methods for querying for rich service descriptors for the
services registered in a gRPC server.

*[Read more ≫](https://pkg.go.dev/github.com/jhump/protoreflect/v2/grpcreflect)*
