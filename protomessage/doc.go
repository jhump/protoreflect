// Package protomessage contains helpers for working with proto.Message values
// from generic and/or dynamic code.
//
// This includes converting a message to a particular concrete type (like
// converting a dynamic message to a generated type), re-parsing unrecognized
// fields once the extensions they represent are known, and walking a message
// to visit all of the messages nested inside it.
package protomessage
