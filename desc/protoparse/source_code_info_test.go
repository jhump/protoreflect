package protoparse

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"testing"

	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"

	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/desc/internal"
	"github.com/jhump/protoreflect/internal/testutil"
)

// If true, re-generates the golden output file
const regenerateMode = false

func TestSourceCodeInfo(t *testing.T) {
	p := Parser{ImportPaths: []string{"../../internal/testprotos"}, IncludeSourceCodeInfo: true}
	fds, err := p.ParseFiles("desc_test_comments.proto", "desc_test_complex.proto")
	testutil.Ok(t, err)
	// also test that imported files have source code info
	// (desc_test_comments.proto imports desc_test_options.proto)
	var importedFd *desc.FileDescriptor
	for _, dep := range fds[0].GetDependencies() {
		if dep.GetName() == "desc_test_options.proto" {
			importedFd = dep
			break
		}
	}
	testutil.Require(t, importedFd != nil)

	// create description of source code info
	// (human readable so diffs in source control are comprehensible)
	var buf bytes.Buffer
	for _, fd := range fds {
		printSourceCodeInfo(fd, &buf)
	}
	printSourceCodeInfo(importedFd, &buf)
	actual := buf.String()

	if regenerateMode {
		// re-generate the file
		err = ioutil.WriteFile("test-source-info.txt", buf.Bytes(), 0666)
		testutil.Ok(t, err)
	}

	b, err := ioutil.ReadFile("test-source-info.txt")
	testutil.Ok(t, err)
	golden := string(b)

	testutil.Eq(t, golden, actual, "wrong source code info")
}

// NB: this function can be used to manually inspect the source code info for a
// descriptor, in a manner that is much easier to read and check than raw
// descriptor form.
func printSourceCodeInfo(fd *desc.FileDescriptor, out io.Writer) {
	_, _ = fmt.Fprintf(out, "---- %s ----\n", fd.GetName())
	msg := fd.AsFileDescriptorProto().ProtoReflect()
	var reg protoregistry.Types
	internal.RegisterExtensionsVisibleToFile(&reg, fd.UnwrapFile())

	for _, loc := range fd.AsFileDescriptorProto().GetSourceCodeInfo().GetLocation() {
		var buf bytes.Buffer
		findLocation(msg, &reg, loc.Path, &buf)
		_, _ = fmt.Fprintf(out, "\n\n%s:\n", buf.String())
		if len(loc.Span) == 3 {
			_, _ = fmt.Fprintf(out, "%s:%d:%d\n", fd.GetName(), loc.Span[0]+1, loc.Span[1]+1)
			_, _ = fmt.Fprintf(out, "%s:%d:%d\n", fd.GetName(), loc.Span[0]+1, loc.Span[2]+1)
		} else {
			_, _ = fmt.Fprintf(out, "%s:%d:%d\n", fd.GetName(), loc.Span[0]+1, loc.Span[1]+1)
			_, _ = fmt.Fprintf(out, "%s:%d:%d\n", fd.GetName(), loc.Span[2]+1, loc.Span[3]+1)
		}
		if len(loc.LeadingDetachedComments) > 0 {
			for i, comment := range loc.LeadingDetachedComments {
				_, _ = fmt.Fprintf(out, "    Leading detached comment [%d]:\n%s\n", i, comment)
			}
		}
		if loc.LeadingComments != nil {
			_, _ = fmt.Fprintf(out, "    Leading comments:\n%s\n", loc.GetLeadingComments())
		}
		if loc.TrailingComments != nil {
			_, _ = fmt.Fprintf(out, "    Trailing comments:\n%s\n", loc.GetTrailingComments())
		}
	}
}

func findLocation(msg protoreflect.Message, reg protoregistry.ExtensionTypeResolver, path []int32, buf *bytes.Buffer) {
	if len(path) == 0 {
		return
	}

	fieldNumber := protoreflect.FieldNumber(path[0])
	md := msg.Descriptor()
	fld := md.Fields().ByNumber(fieldNumber)
	if fld == nil {
		xt, err := reg.FindExtensionByNumber(md.FullName(), fieldNumber)
		if err == nil {
			fld = xt.TypeDescriptor()
		}
	}
	if fld == nil {
		panic(fmt.Sprintf("could not find field with tag %d in message of type %s", path[0], md.FullName()))
	}

	var name string
	if fld.IsExtension() {
		name = "(" + string(fld.FullName()) + ")"
	} else {
		name = string(fld.Name())
	}
	_, _ = fmt.Fprintf(buf, " > %s", name)
	path = path[1:]
	idx := -1
	if fld.Cardinality() == protoreflect.Repeated && len(path) > 0 {
		idx = int(path[0])
		_, _ = fmt.Fprintf(buf, "[%d]", path[0])
		path = path[1:]
	}

	if len(path) > 0 {
		if fld.Kind() != protoreflect.MessageKind && fld.Kind() != protoreflect.GroupKind {
			panic(fmt.Sprintf("path indicates tag %d, but field %v is %v, not a message", path[0], name, fld.Kind()))
		}
		var present bool
		var next protoreflect.Message
		if idx == -1 {
			present = msg.Has(fld)
			next = msg.Get(fld).Message()
		} else {
			list := msg.Get(fld).List()
			present = idx < list.Len()
			if present {
				next = list.Get(idx).Message()
			} else {
				next = list.NewElement().Message()
			}
		}

		if !present {
			buf.WriteString(" !!! ")
		}

		findLocation(next, reg, path, buf)
	}
}
