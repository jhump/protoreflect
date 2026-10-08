package remotereg_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/known/typepb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/jhump/protoreflect/v2/internal/testprotos"
	"github.com/jhump/protoreflect/v2/internal/testprotos/pkg"
	. "github.com/jhump/protoreflect/v2/protoresolve/remotereg"
)

func TestRegisterPackageBaseURL(t *testing.T) {
	t.Parallel()
	reg := &Registry{}
	msg := testprotos.File_desc_test1_proto.Messages().ByName("TestMessage")
	nested := msg.Messages().Get(0)
	pkgMsg := pkg.File_pkg_desc_test_pkg_proto.Messages().Get(0)

	prev, registered := reg.RegisterPackageBaseURL("testprotos", "foo.com")
	assert.Equal(t, "type.googleapis.com", prev)
	assert.False(t, registered)
	assert.Equal(t, "https://foo.com/testprotos.TestMessage", reg.URLForType(msg))
	assert.Equal(t, "https://foo.com/"+string(nested.FullName()), reg.URLForType(nested))

	// A registration applies to sub-packages...
	_, _ = reg.RegisterPackageBaseURL("jhump", "bar.com")
	assert.Equal(t, "https://bar.com/"+string(pkgMsg.FullName()), reg.URLForType(pkgMsg))
	// ...unless they have their own.
	prev, registered = reg.RegisterPackageBaseURL("jhump.protoreflect", "baz.com")
	assert.Equal(t, "https://bar.com", prev)
	assert.False(t, registered)
	assert.Equal(t, "https://baz.com/"+string(pkgMsg.FullName()), reg.URLForType(pkgMsg))
	prev, registered = reg.RegisterPackageBaseURL("jhump.protoreflect", "qux.com")
	assert.Equal(t, "https://baz.com", prev)
	assert.True(t, registered)
}

func TestFindByNameUsesPackageBaseURL(t *testing.T) {
	t.Parallel()
	var requested []string
	reg := &Registry{
		TypeFetcher: TypeFetcherFunc(func(_ context.Context, url string, _ bool) (proto.Message, error) {
			requested = append(requested, url)
			return nil, protoregistry.NotFound
		}),
		Fallback: &protoregistry.Files{},
	}
	reg.RegisterPackageBaseURL("testprotos", "foo.com")
	// Only the name is known, so the package of a nested type is not known.
	// But the base URL for its package still applies.
	_, err := reg.FindMessageByName("testprotos.TestMessage.NestedMessage")
	require.ErrorIs(t, err, protoregistry.NotFound)
	assert.Equal(t, []string{"https://foo.com/testprotos.TestMessage.NestedMessage"}, requested)
}

func TestFetchDoesNotReplaceRegisteredTypes(t *testing.T) {
	t.Parallel()
	const valueURL = "https://example.com/google.protobuf.StringValue"
	reg := &Registry{
		TypeFetcher: TypeFetcherFunc(func(_ context.Context, url string, enum bool) (proto.Message, error) {
			if enum || url != "https://example.com/test.Wrapper" {
				return nil, protoregistry.NotFound
			}
			return &typepb.Type{
				Name:   "test.Wrapper",
				Syntax: typepb.Syntax_SYNTAX_PROTO3,
				Fields: []*typepb.Field{{
					Name:        "value",
					Number:      1,
					Cardinality: typepb.Field_CARDINALITY_OPTIONAL,
					Kind:        typepb.Field_TYPE_MESSAGE,
					TypeUrl:     valueURL,
				}},
			}, nil
		}),
	}
	registered := (&wrapperspb.StringValue{}).ProtoReflect().Descriptor()
	require.NoError(t, reg.RegisterMessageWithURL(registered, valueURL))

	wrapper, err := reg.FindMessageByURL("https://example.com/test.Wrapper")
	require.NoError(t, err)
	assert.Equal(t, registered.FullName(), wrapper.Fields().ByName("value").Message().FullName())

	// The registered type is unchanged by the fetch of a type that uses it.
	found, err := reg.FindMessageByURL(valueURL)
	require.NoError(t, err)
	assert.Same(t, registered, found)
}
