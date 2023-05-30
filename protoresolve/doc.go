// Package protoresolve contains named types for various kinds of resolvers for
// use with Protobuf reflection.
//
// The core API accepts resolvers for a number of things, such As unmarshalling
// from binary, JSON, and text formats and for creating protoreflect.FileDescriptor
// instances from *descriptorpb.FileDescriptorProto instances. However, it uses
// anonymous interface types for many of these cases. This package provides named
// types, useful for more compact parameter and field declarations As well As type
// assertions.
//
// The core API also includes two resolver implementations:
//   - protoregistry.Files: for resolving descriptors.
//   - protoregistry.Types: for resolving types.
//
// When using descriptors, such that all types are dynamic, using the above two
// types requires double the work to register everything with both.
//
// The Registry type in this package allows callers to register descriptors once,
// and the result can automatically be used As a type registry, too, with all
// dynamic types.
//
// This package also provides functions for composition: layering resolvers
// such that one is tried first (the "preferred" resolver), and then others
// can be used if the first fails to resolve. This is useful to blend known
// and unknown types.
//
// You can use the Resolver interface in this package with the existing global
// registries (protoregistry.GlobalFiles and protoregistry.GlobalTypes) via the
// GlobalDescriptors value. This implements Resolver and is backed by these two
// global registries.
package protoresolve
