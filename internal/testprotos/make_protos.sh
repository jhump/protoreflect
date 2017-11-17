#!/usr/bin/env bash

set -e

cd $(dirname $0)

# Output directory will effectively be GOPATH/src.
outdir="../../../../.."
protoc "--go_out=plugins=grpc:$outdir" -I. *.proto
protoc "--go_out=plugins=grpc:$outdir" -I. nopkg/*.proto
protoc "--go_out=plugins=grpc:$outdir" -I. pkg/*.proto

# And make descriptor set (with source info) for several files
protoc --descriptor_set_out=./desc_test1.protoset --include_source_info --include_imports -I. desc_test1.proto
protoc --descriptor_set_out=./desc_test_comments.protoset --include_source_info --include_imports -I. desc_test_comments.proto
protoc --descriptor_set_out=./desc_test_complex.protoset -I. desc_test_complex.proto
protoc --descriptor_set_out=./desc_test_complex_source_info.protoset --include_source_info --include_imports -I. desc_test_complex.proto
protoc --descriptor_set_out=./descriptor.protoset --include_source_info --include_imports -I../../../../ ../../../../golang/protobuf/protoc-gen-go/descriptor/descriptor.proto
