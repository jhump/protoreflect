// Package protoparse provides functionality for parsing *.proto source files
// into descriptors that can be used with other protoreflect packages, like
// dynamic messages and dynamic GRPC clients.
//
// The file descriptors produced by this package do not include the
// source_code_info field. As such, comments and location information (like
// where in the file an element is declared) are not available. This also means
// that many errors encountered will not have location information. The errors
// try to be as specific as possible for pinpointing their source. But the only
// errors that include locations (like line numbers) are those generated during
// parsing (e.g. syntax errors). Other validation and link errors will instead
// refer to symbols in the parsed files, not their location/line number.
//
// This package links in various packages that include compiled descriptors for
// the various "google/protobuf/*.proto" files that are included with protoc.
// That way, like when invoking protoc, programs need not supply copies of these
// "builtin" files. Though if copies of the files are provided, they will be
// used instead of the builtin descriptors.
package protoparse
