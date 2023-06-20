// Package protoresolve contains named types for various kinds of resolvers for
// use with Protobuf reflection.
//
// The core protobuf runtime API (provided by the google.golang.org/protobuf module)
// accepts resolvers for a number of things, such as unmarshalling from binary, JSON,
// and text formats and for creating protoreflect.FileDescriptor instances from
// *descriptorpb.FileDescriptorProto instances. However, it uses anonymous interface
// types for many of these cases. This package provides named types, useful for more
// compact parameter and field declarations as well as type assertions.
//
// The core protobuf runtime API also includes two resolver implementations:
//   - protoregistry.Files: for resolving descriptors.
//   - protoregistry.Types: for resolving types.
//
// When using descriptors, such that all types are dynamic, using the above two
// types requires double the work to register everything with both. The first must
// be used in order to process FileDescriptorProtos into more useful FileDescriptor
// instances, and the second must be used to create a resolver that the other APIs
// accept.
//
// The Registry type in this package, on the other hand, allows callers to register
// descriptors once, and the result can automatically be used as a type registry,
// too, with all dynamic types.
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
