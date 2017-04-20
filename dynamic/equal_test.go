package dynamic

import (
	"github.com/golang/protobuf/proto"
)

func eqm(a, b interface{}) bool {
	return MessagesEqual(a.(proto.Message), b.(proto.Message))
}

func eqdm(a, b interface{}) bool {
	return Equal(a.(*Message), b.(*Message))
}

func eqpm(a, b interface{}) bool {
	return proto.Equal(a.(proto.Message), b.(proto.Message))
}
