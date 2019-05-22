#!/usr/bin/env bash

set -euo pipefail

DIR="$(cd "$(dirname "${0}")" && pwd)"
cd "${DIR}"

../../scripts/install_protoc.sh

PROTOC="../../.tmp/protoc/bin/protoc"

go install github.com/golang/protobuf/protoc-gen-go

# Output directory will effectively be GOPATH/src.
outdir="../../../../.."
${PROTOC} "--go_out=plugins=grpc:$outdir" -I. *.proto
${PROTOC} "--go_out=plugins=grpc:$outdir" -I. nopkg/*.proto
${PROTOC} "--go_out=plugins=grpc:$outdir" -I. pkg/*.proto

# And make descriptor set (with source info) for several files
${PROTOC} --descriptor_set_out=./desc_test1.protoset --include_source_info --include_imports -I. desc_test1.proto
${PROTOC} --descriptor_set_out=./desc_test_comments.protoset --include_source_info --include_imports -I. desc_test_comments.proto
${PROTOC} --descriptor_set_out=./desc_test_complex.protoset -I. desc_test_complex.proto
${PROTOC} --descriptor_set_out=./desc_test_complex_source_info.protoset --include_source_info --include_imports -I. desc_test_complex.proto
${PROTOC} --descriptor_set_out=./descriptor.protoset --include_source_info --include_imports -I../../.tmp/protoc/include/ google/protobuf/descriptor.proto
${PROTOC} --descriptor_set_out=./duration.protoset -I../../.tmp/protoc/include/ google/protobuf/duration.proto
