// Package pinv1 is not used. It's sole purpose is to create a faux import
// of the v1 of this module. That way if someone links in both v1 and v2 in
// their program, it will pull in a later v1 version that shares sourceinfo
// with v2. That way, regardless of what version of the sourceinfo package
// was used, v1 or v2, all sourceinfo for all files is available.
package pinv1

import "github.com/jhump/protoreflect/desc"

var _ desc.Descriptor = nil
