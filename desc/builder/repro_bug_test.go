package builder

import (
	"testing"

	dpb "github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/jhump/protoreflect/dynamic"
	"github.com/jhump/protoreflect/internal/testutil"
)

func TestMarshaledAndUnmarshaledMessageIsEqual(t *testing.T) {
	var (
		schemaBuilder = NewMessage("schema")
		fieldType     = FieldTypeScalar(dpb.FieldDescriptorProto_TYPE_DOUBLE)
		field         = NewField("test_repeated_field", fieldType).
				SetNumber(1).
				SetRepeated()
	)
	schemaBuilder.AddField(field)
	schema, err := schemaBuilder.Build()
	if err != nil {
		panic(err)
	}

	m := dynamic.NewMessage(schema)
	m.SetFieldByNumber(1, []float64{})

	marshaled, err := m.Marshal()
	if err != nil {
		panic(err)
	}

	unmarshaled := dynamic.NewMessage(schema)
	err = unmarshaled.Unmarshal(marshaled)
	if err != nil {
		panic(err)
	}

	testutil.Eq(t, true, dynamic.Equal(m, unmarshaled))
}
