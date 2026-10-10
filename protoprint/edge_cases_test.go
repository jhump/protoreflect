package protoprint

import (
	"math"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	_ "google.golang.org/protobuf/types/known/emptypb"

	"github.com/jhump/protoreflect/v2/internal/testprotos"
)

func TestPrinterIsNotModified(t *testing.T) {
	t.Parallel()
	printer := &Printer{Indent: "\t-"}
	_, err := printer.PrintProtoToString(testprotos.File_desc_test1_proto)
	require.NoError(t, err)
	assert.Equal(t, "\t-", printer.Indent)
}

func TestPrintDoesNotModifyOptions(t *testing.T) {
	t.Parallel()
	// Round-trip the proto through the binary format without knowledge of
	// the file's custom options, so they are all unrecognized fields.
	data, err := proto.Marshal(protodesc.ToFileDescriptorProto(testprotos.File_desc_test_complex_proto))
	require.NoError(t, err)
	var fileProto descriptorpb.FileDescriptorProto
	require.NoError(t, proto.UnmarshalOptions{Resolver: &protoregistry.Types{}}.Unmarshal(data, &fileProto))
	file, err := protodesc.NewFile(&fileProto, protoregistry.GlobalFiles)
	require.NoError(t, err)
	msg := file.Messages().ByName("Test").Messages().ByName("Nested").Messages().ByName("_NestedNested")
	require.NotNil(t, msg)
	require.NotEmpty(t, msg.Options().ProtoReflect().GetUnknown())

	output, err := (&Printer{}).PrintProtoToString(file)
	require.NoError(t, err)
	// The printer recognizes the custom options...
	assert.Contains(t, output, "option (fooblez) = 10101;")
	// ...but does not modify the descriptor's options.
	assert.NotEmpty(t, msg.Options().ProtoReflect().GetUnknown())
}

func TestPrintCommentOfBlankLines(t *testing.T) {
	t.Parallel()
	file, err := protodesc.NewFile(&descriptorpb.FileDescriptorProto{
		Name:        proto.String("test.proto"),
		Syntax:      proto.String("proto3"),
		MessageType: []*descriptorpb.DescriptorProto{{Name: proto.String("Msg")}},
		SourceCodeInfo: &descriptorpb.SourceCodeInfo{
			Location: []*descriptorpb.SourceCodeInfo_Location{{
				Path: []int32{4, 0},
				Span: []int32{3, 0, 4, 1},
				// Two comment lines with no text: "//\n//\n"
				LeadingComments: proto.String("\n\n"),
			}},
		},
	}, nil)
	require.NoError(t, err)
	for _, multiLine := range []bool{false, true} {
		_, err := (&Printer{PreferMultiLineStyleComments: multiLine}).PrintProtoToString(file)
		require.NoError(t, err)
	}
}

func TestPrintNonFiniteFloatOptions(t *testing.T) {
	t.Parallel()
	opts := &descriptorpb.MethodOptions{}
	proto.SetExtension(opts, testprotos.E_Mtfubar, []float32{float32(math.Inf(1)), float32(math.Inf(-1)), 1e-7})
	proto.SetExtension(opts, testprotos.E_Mtfubard, math.NaN())
	file, err := protodesc.NewFile(&descriptorpb.FileDescriptorProto{
		Name:       proto.String("test.proto"),
		Package:    proto.String("test"),
		Syntax:     proto.String("proto3"),
		Dependency: []string{testprotos.File_desc_test_options_proto.Path(), "google/protobuf/empty.proto"},
		Service: []*descriptorpb.ServiceDescriptorProto{{
			Name: proto.String("Svc"),
			Method: []*descriptorpb.MethodDescriptorProto{{
				Name:       proto.String("Method"),
				InputType:  proto.String(".google.protobuf.Empty"),
				OutputType: proto.String(".google.protobuf.Empty"),
				Options:    opts,
			}},
		}},
	}, protoregistry.GlobalFiles)
	require.NoError(t, err)

	output, err := (&Printer{}).PrintProtoToString(file)
	require.NoError(t, err)
	var optionLines []string
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, "mtfubar") {
			optionLines = append(optionLines, strings.TrimSpace(line))
		}
	}
	assert.ElementsMatch(t, []string{
		"option (testprotos.mtfubar) = inf;",
		"option (testprotos.mtfubar) = -inf;",
		"option (testprotos.mtfubar) = 1e-07;",
		"option (testprotos.mtfubard) = nan;",
	}, optionLines)
}
