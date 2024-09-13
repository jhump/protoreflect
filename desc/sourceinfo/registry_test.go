package sourceinfo_test

import (
	"fmt"
	"testing"

	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/jhump/protoreflect/desc/sourceinfo"
	_ "github.com/jhump/protoreflect/internal/testprotos"
	"github.com/jhump/protoreflect/internal/testutil"
)

func TestRegistry(t *testing.T) {
	fd, err := sourceinfo.GlobalFiles.FindFileByPath("desc_test1.proto")
	testutil.Ok(t, err)
	checkFileComments(t, fd)
}

func checkFileComments(t *testing.T, fd protoreflect.FileDescriptor) {
	srcLocs := fd.SourceLocations()
	for i := 0; i < fd.Messages().Len(); i++ {
		checkMessageComments(t, srcLocs, fd.Messages().Get(i))
	}
	for i := 0; i < fd.Enums().Len(); i++ {
		checkEnumComments(t, srcLocs, fd.Enums().Get(i))
	}
	for i := 0; i < fd.Extensions().Len(); i++ {
		checkComment(t, srcLocs, fd.Extensions().Get(i))
	}
	for i := 0; i < fd.Services().Len(); i++ {
		sd := fd.Services().Get(i)
		checkComment(t, srcLocs, sd)
		for j := 0; j < sd.Methods().Len(); j++ {
			mtd := sd.Methods().Get(j)
			checkComment(t, srcLocs, mtd)
		}
	}
}

func checkMessageComments(t *testing.T, srcLocs protoreflect.SourceLocations, md protoreflect.MessageDescriptor) {
	checkComment(t, srcLocs, md)

	for i := 0; i < md.Fields().Len(); i++ {
		fld := md.Fields().Get(i)
		if fld.Kind() == protoreflect.GroupKind {
			continue // comment is attributed to group message, not field
		}
		checkComment(t, srcLocs, fld)
	}
	for i := 0; i < md.Oneofs().Len(); i++ {
		checkComment(t, srcLocs, md.Oneofs().Get(i))
	}

	for i := 0; i < md.Messages().Len(); i++ {
		nmd := md.Messages().Get(i)
		if nmd.IsMapEntry() {
			// synthetic map entry messages won't have comments
			continue
		}
		checkMessageComments(t, srcLocs, nmd)
	}
	for i := 0; i < md.Enums().Len(); i++ {
		checkEnumComments(t, srcLocs, md.Enums().Get(i))
	}
	for i := 0; i < md.Extensions().Len(); i++ {
		checkComment(t, srcLocs, md.Extensions().Get(i))
	}
}

func checkEnumComments(t *testing.T, srcLocs protoreflect.SourceLocations, ed protoreflect.EnumDescriptor) {
	checkComment(t, srcLocs, ed)
	for i := 0; i < ed.Values().Len(); i++ {
		evd := ed.Values().Get(i)
		checkComment(t, srcLocs, evd)
	}
}

func checkComment(t *testing.T, srcLocs protoreflect.SourceLocations, d protoreflect.Descriptor) {
	cmt := fmt.Sprintf(" Comment for %s\n", d.Name())
	testutil.Eq(t, cmt, srcLocs.ByDescriptor(d).LeadingComments)
}
