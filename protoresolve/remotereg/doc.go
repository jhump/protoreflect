// Package remotereg provides a registry of message and enum types that are
// identified by type URL, such as the URLs found in google.protobuf.Any
// messages.
//
// A Registry can be configured with a TypeFetcher, so that types which have not
// been explicitly registered can be downloaded from a remote source when they
// are first needed. Fetched types are represented using google.protobuf.Type
// and google.protobuf.Enum messages, which are then converted to descriptors.
// The same conversion logic is also exposed directly, for converting between
// descriptors and these alternate forms (including google.protobuf.Api, for
// services). See Registry.AsDescriptorConverter.
//
// A Registry can also be used as a protoresolve.TypeResolver (see
// Registry.AsTypeResolver), for marshalling and unmarshalling messages that
// contain google.protobuf.Any values whose types are not known at compile time.
package remotereg
