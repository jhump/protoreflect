// Package protowrap contains types and utilities related to wrapping descriptors
// and descriptor protos. It has two categories of wrappers, described below.
//
// # Wrapping descriptor protos
//
// The main implementations of protoreflect.Descriptor in the core protobuf
// runtime API are a separate model from the descriptor proto messages from
// which they were created. One can use the [google.golang.org/protobuf/reflect/protodesc]
// package to convert back to a proto, but the transformation is lossy: the
// resulting proto will not likely be identical to the one originally used
// to create the descriptor. Also, this is not a cheap operation as the entire
// FileDescriptorProto hierarchy must be re-created with each conversion.
//
// The [ProtoWrapper] and related types (FileWrapper, MessageWrapper, etc) in
// this package are different. They *wrap* the descriptor proto from which they
// were created and provide methods to extract it. This extraction is cheap since
// nothing must be re-created. (Caution should be used so that callers do not
// MUTATE the returned protos, as this package does not make defensive copies.)
// Use the various ProtoFrom*Descriptor functions to extract the proto. If the
// given descriptor is not a wrapper of any sort, then these functions fall back
// to using the protodesc package to convert to a proto.
//
// # Wrapping descriptors
//
// The second kind of wrapper is a value that wraps a [protoreflect.Descriptor] and
// also implements that same interface. This kind of implementation is often an
// "interceptor" or "decorator", that delegates methods to the wrapped descriptor,
// but provides some special or additional handling for one or more of the methods.
// This kind of wrapper implements [WrappedDescriptor] and provides an Unwrap
// method from which the underlying descriptor can be recovered.
//
// The [FromFileDescriptorProto] and [FromFileDescriptorSet] functions in this package
// create wrappers of both kinds: the returned descriptors wrap both the core runtime
// implementation of descriptors as well as the protos from which they are created. So
// the returned descriptors implement ProtoWrapper, FileWrapper, and WrappedDescriptor.
package protowrap
