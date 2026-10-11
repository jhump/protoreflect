// Package protodescs contains miscellaneous helpers for working with
// descriptors, including both rich descriptors (protoreflect.Descriptor) and
// descriptor protos (descriptorpb.FileDescriptorProto and friends).
//
// This is home to the odds and ends that don't quite fit anywhere else, such
// as determining the edition of a file or sorting a set of file descriptor
// protos so that every file appears after its imports.
package protodescs
