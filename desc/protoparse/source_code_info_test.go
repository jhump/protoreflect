package protoparse

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"

	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/dynamic"
	"github.com/jhump/protoreflect/internal/testutil"
)

// If true, re-generates the golden output file
const regenerateMode = false

func TestSourceCodeInfo(t *testing.T) {
	p := Parser{ImportPaths: []string{"../../internal/testprotos"}, IncludeSourceCodeInfo: true}
	fds, err := p.ParseFiles("desc_test_comments.proto")
	testutil.Ok(t, err)
	fd := fds[0]

	if regenerateMode {
		// re-generate the file
		b, err := proto.Marshal(fd.AsFileDescriptorProto().GetSourceCodeInfo())
		testutil.Ok(t, err)
		err = ioutil.WriteFile("test-source-info.bin", b, 0666)
		testutil.Ok(t, err)

		printSourceCodeInfo(t, fd)
	}

	b, err := ioutil.ReadFile("test-source-info.bin")
	testutil.Ok(t, err)
	var sci descriptor.SourceCodeInfo
	err = proto.Unmarshal(b, &sci)
	testutil.Ok(t, err)

	testutil.Require(t, proto.Equal(&sci, fd.AsFileDescriptorProto().GetSourceCodeInfo()), "wrong source code info")
}

// NB: this function can be used to manually inspect the source code info for a
// descriptor, in a manner that is much easier to read and check than raw
// descriptor form.
func printSourceCodeInfo(t *testing.T, fd *desc.FileDescriptor) {
	md, err := desc.LoadMessageDescriptorForMessage(fd.AsProto())
	testutil.Ok(t, err)
	er := &dynamic.ExtensionRegistry{}
	er.AddExtensionsFromFileRecursively(fd)
	mf := dynamic.NewMessageFactoryWithExtensionRegistry(er)
	dfd := mf.NewDynamicMessage(md)
	err = dfd.ConvertFrom(fd.AsProto())
	testutil.Ok(t, err)

	for _, loc := range fd.AsFileDescriptorProto().GetSourceCodeInfo().GetLocation() {
		var buf bytes.Buffer
		findLocation(mf, dfd, md, loc.Path, &buf)
		fmt.Printf("\n\n%s:\n", buf.String())
		if len(loc.Span) == 3 {
			fmt.Printf("%s:%d:%d\n", fd.GetName(), loc.Span[0]+1, loc.Span[1]+1)
			fmt.Printf("%s:%d:%d\n", fd.GetName(), loc.Span[0]+1, loc.Span[2]+1)
		} else {
			fmt.Printf("%s:%d:%d\n", fd.GetName(), loc.Span[0]+1, loc.Span[1]+1)
			fmt.Printf("%s:%d:%d\n", fd.GetName(), loc.Span[2]+1, loc.Span[3]+1)
		}
		if len(loc.LeadingDetachedComments) > 0 {
			for i, comment := range loc.LeadingDetachedComments {
				fmt.Printf("    Leading detached comment [%d]:\n%s\n", i, comment)
			}
		}
		if loc.LeadingComments != nil {
			fmt.Printf("    Leading comments:\n%s\n", loc.GetLeadingComments())
		}
		if loc.TrailingComments != nil {
			fmt.Printf("    Trailing comments:\n%s\n", loc.GetTrailingComments())
		}
	}
}

func findLocation(mf *dynamic.MessageFactory, msg *dynamic.Message, md *desc.MessageDescriptor, path []int32, buf *bytes.Buffer) {
	if len(path) == 0 {
		return
	}

	var fld *desc.FieldDescriptor
	if msg != nil {
		fld = msg.FindFieldDescriptor(path[0])
	} else {
		fld = md.FindFieldByNumber(path[0])
		if fld == nil {
			fld = mf.GetExtensionRegistry().FindExtension(md.GetFullyQualifiedName(), path[0])
		}
	}
	if fld == nil {
		panic(fmt.Sprintf("could not find field with tag %d in message of type %s", path[0], msg.XXX_MessageName()))
	}

	fmt.Fprintf(buf, " > %s", fld.GetName())
	path = path[1:]
	idx := -1
	if fld.IsRepeated() && len(path) > 0 {
		idx = int(path[0])
		fmt.Fprintf(buf, "[%d]", path[0])
		path = path[1:]
	}

	if len(path) > 0 {
		var next proto.Message
		if msg != nil {
			if idx >= 0 {
				if idx < msg.FieldLength(fld) {
					next = msg.GetRepeatedField(fld, idx).(proto.Message)
				}
			} else {
				if m, ok := msg.GetField(fld).(proto.Message); ok {
					next = m
				} else {
					panic(fmt.Sprintf("path traverses into non-message type %T: %s -> %v", msg.GetField(fld), buf.String(), path))
				}
			}
		}

		if next == nil && msg != nil {
			buf.WriteString(" !!! ")
		}

		if dm, ok := next.(*dynamic.Message); ok || next == nil {
			findLocation(mf, dm, fld.GetMessageType(), path, buf)
		} else {
			dm := mf.NewDynamicMessage(fld.GetMessageType())
			err := dm.ConvertFrom(next)
			if err != nil {
				panic(err.Error())
			}
			findLocation(mf, dm, fld.GetMessageType(), path, buf)
		}
	}
}
