package protoresolve

import (
	"context"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/typepb"
)

type TypeFetcher interface {
	FetchMessageType(ctx context.Context, url string) (*typepb.Type, error)
	FetchEnumType(ctx context.Context, url string) (*typepb.Enum, error)
}

type TypeFetcherFunc func(ctx context.Context, url string, dest proto.Message) error

var _ TypeFetcher = TypeFetcherFunc(nil)

func (t TypeFetcherFunc) FetchMessageType(ctx context.Context, url string) (*typepb.Type, error) {
	var dest typepb.Type
	if err := t(ctx, url, &dest); err != nil {
		return nil, err
	}
	return &dest, nil
}

func (t TypeFetcherFunc) FetchEnumType(ctx context.Context, url string) (*typepb.Enum, error) {
	var dest typepb.Enum
	if err := t(ctx, url, &dest); err != nil {
		return nil, err
	}
	return &dest, nil
}
