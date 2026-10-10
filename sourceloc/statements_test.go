package sourceloc_test

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/jhump/protoreflect/v2/internal"
	prototesting "github.com/jhump/protoreflect/v2/internal/testing"
	"github.com/jhump/protoreflect/v2/internal/testprotos"
	. "github.com/jhump/protoreflect/v2/sourceloc"
)

func TestIsSubspanOf(t *testing.T) {
	t.Parallel()
	span := func(startLine, startCol, endLine, endCol int) protoreflect.SourceLocation {
		return protoreflect.SourceLocation{StartLine: startLine, StartColumn: startCol, EndLine: endLine, EndColumn: endCol}
	}
	loc := span(10, 5, 20, 15)
	testCases := []struct {
		name      string
		candidate protoreflect.SourceLocation
		expected  bool
	}{
		{"same", loc, true},
		{"strictly inside", span(11, 0, 19, 0), true},
		{"same lines, inner columns", span(10, 6, 20, 14), true},
		{"starts before", span(10, 4, 20, 15), false},
		{"starts on earlier line", span(9, 5, 20, 15), false},
		{"ends after", span(10, 5, 20, 16), false},
		{"ends on later line", span(10, 5, 21, 0), false},
		{"entirely after", span(30, 0, 31, 0), false},
	}
	for _, testCase := range testCases {
		assert.Equal(t, testCase.expected, IsSubspanOf(testCase.candidate, loc), testCase.name)
	}
}

func TestStatementLocations(t *testing.T) {
	t.Parallel()
	file, err := prototesting.LoadProtoset("../internal/testprotos/desc_test_complex_source_info.protoset")
	require.NoError(t, err)
	data, err := os.ReadFile("../internal/testprotos/" + file.Path())
	require.NoError(t, err)
	sourceLines := strings.Split(string(data), "\n")
	srcLocs := file.SourceLocations()

	// checkStatement verifies that the given statement location starts with the
	// given keyword and encloses the location of the element at elementPath.
	checkStatement := func(t *testing.T, stmt protoreflect.SourceLocation, keyword string, elementPath protoreflect.SourcePath) {
		t.Helper()
		if !assert.False(t, IsZero(stmt), "no statement found for element at %v", elementPath) {
			return
		}
		require.Less(t, stmt.StartLine, len(sourceLines))
		// Columns can't be used to index into the line since they account
		// for tab stops. But every statement in the file starts its line.
		line := strings.TrimSpace(sourceLines[stmt.StartLine])
		assert.True(t, strings.HasPrefix(line, keyword), "statement for element at %v should start with %q: %q", elementPath, keyword, line)
		element := srcLocs.ByPath(elementPath)
		require.False(t, IsZero(element), "no location for element at %v", elementPath)
		assert.True(t, IsSubspanOf(element, stmt), "statement for element at %v should enclose it", elementPath)
	}

	var numExtensions, numExtensionRanges, numReservedRanges, numReservedNames int
	var checkMessages func(t *testing.T, msgs protoreflect.MessageDescriptors)
	checkExtensions := func(t *testing.T, exts protoreflect.ExtensionDescriptors) {
		for i := range exts.Len() {
			ext := exts.Get(i)
			checkStatement(t, ForExtendBlock(ext), "extend", PathFor(ext))
			numExtensions++
		}
	}
	checkMessages = func(t *testing.T, msgs protoreflect.MessageDescriptors) {
		for i := range msgs.Len() {
			msg := msgs.Get(i)
			msgPath := PathFor(msg)
			for j := range msg.ExtensionRanges().Len() {
				path := append(append(protoreflect.SourcePath{}, msgPath...), internal.MessageExtensionRangeTag, int32(j))
				checkStatement(t, ForExtensionsStatement(msg, j), "extensions", path)
				numExtensionRanges++
			}
			for j := range msg.ReservedRanges().Len() {
				path := append(append(protoreflect.SourcePath{}, msgPath...), internal.MessageReservedRangeTag, int32(j))
				checkStatement(t, ForReservedRangesStatement(msg, j), "reserved", path)
				numReservedRanges++
			}
			for j := range msg.ReservedNames().Len() {
				path := append(append(protoreflect.SourcePath{}, msgPath...), internal.MessageReservedNameTag, int32(j))
				checkStatement(t, ForReservedNamesStatement(msg, j), "reserved", path)
				numReservedNames++
			}
			// Indexes out of range have no location.
			assert.True(t, IsZero(ForExtensionsStatement(msg, msg.ExtensionRanges().Len())))
			assert.True(t, IsZero(ForReservedRangesStatement(msg, msg.ReservedRanges().Len())))
			assert.True(t, IsZero(ForReservedNamesStatement(msg, msg.ReservedNames().Len())))

			checkExtensions(t, msg.Extensions())
			checkMessages(t, msg.Messages())
		}
	}
	checkExtensions(t, file.Extensions())
	checkMessages(t, file.Messages())
	// Make sure the test file exercised all the cases.
	assert.NotZero(t, numExtensions)
	assert.NotZero(t, numExtensionRanges)
	assert.NotZero(t, numReservedRanges)
	assert.NotZero(t, numReservedNames)

	// A single statement that declares several ranges is the location for all of them.
	msg := file.Messages().ByName("Test")
	require.NotNil(t, msg)
	require.Equal(t, 5, msg.ExtensionRanges().Len())
	first := ForExtensionsStatement(msg, 1)
	for i := 2; i < 5; i++ {
		assert.Equal(t, first, ForExtensionsStatement(msg, i))
	}
	assert.NotEqual(t, first, ForExtensionsStatement(msg, 0))

	// A normal field has no extend block.
	assert.True(t, IsZero(ForExtendBlock(msg.Fields().Get(0))))
}

func TestStatementLocationsWithoutSourceInfo(t *testing.T) {
	t.Parallel()
	file := testprotos.File_desc_test1_proto
	require.Zero(t, file.SourceLocations().Len())
	assert.True(t, IsZero(ForExtendBlock(file.Extensions().Get(0))))
	msg := file.Messages().ByName("AnotherTestMessage")
	require.NotZero(t, msg.ExtensionRanges().Len())
	assert.True(t, IsZero(ForExtensionsStatement(msg, 0)))
}

func TestForExtendBlockEdgeCases(t *testing.T) {
	t.Parallel()
	newFile := func(t *testing.T, locs ...*descriptorpb.SourceCodeInfo_Location) protoreflect.ExtensionDescriptor {
		t.Helper()
		file, err := protodesc.NewFile(&descriptorpb.FileDescriptorProto{
			Name:       proto.String("test.proto"),
			Dependency: []string{"google/protobuf/descriptor.proto"},
			Extension: []*descriptorpb.FieldDescriptorProto{{
				Name:     proto.String("ext"),
				Number:   proto.Int32(50000),
				Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
				Type:     descriptorpb.FieldDescriptorProto_TYPE_INT32.Enum(),
				Extendee: proto.String(".google.protobuf.MessageOptions"),
			}},
			SourceCodeInfo: &descriptorpb.SourceCodeInfo{Location: locs},
		}, protoregistry.GlobalFiles)
		require.NoError(t, err)
		return file.Extensions().Get(0)
	}
	extLoc := &descriptorpb.SourceCodeInfo_Location{Path: []int32{internal.FileExtensionsTag, 0}, Span: []int32{5, 2, 30}}

	// With no location for the enclosing block, the extension's own location is returned.
	ext := newFile(t, extLoc)
	loc := ForExtendBlock(ext)
	assert.Equal(t, 5, loc.StartLine)
	assert.Equal(t, 2, loc.StartColumn)

	// If no location for the enclosing block contains the extension, there is no result.
	ext = newFile(t, extLoc,
		&descriptorpb.SourceCodeInfo_Location{Path: []int32{internal.FileExtensionsTag}, Span: []int32{1, 0, 3, 1}},
		&descriptorpb.SourceCodeInfo_Location{Path: []int32{internal.FileExtensionsTag}, Span: []int32{10, 0, 12, 1}},
	)
	assert.True(t, IsZero(ForExtendBlock(ext)))
}
