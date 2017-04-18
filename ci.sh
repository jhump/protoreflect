#!/usr/bin/env bash
set -e

go get -d github.com/golang/protobuf/{proto,ptypes,protoc-gen-go} google.golang.org/grpc golang.org/x/net/context

fmtdiff="$(gofmt -s -l ./)"
if [[ -n "$fmtdiff" ]]; then
  gofmt -s -l ./ >&2
  echo "Run gofmt on the above files!" >&2
  exit 1
fi

go test -v -race ./...
