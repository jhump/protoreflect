#!/usr/bin/env bash

set -e

cd $(dirname $0)

# Downloads protoc if it's not already cached. The version of protoc is
# pinned in this script.
PROTOC="$(./protoc.sh)"

go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.12
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.5.1
go install github.com/jhump/protoreflect/v2/sourceinfo/cmd/protoc-gen-gosrcinfo

# Output directory will effectively be GOPATH/src.
outdir="."
${PROTOC} "--go_out=paths=source_relative:$outdir" "--gosrcinfo_out=paths=source_relative,debug:$outdir" -I. *.proto
${PROTOC} "--go_out=paths=source_relative:$outdir" "--gosrcinfo_out=paths=source_relative,debug:$outdir" -I. nopkg/*.proto
${PROTOC} "--go_out=paths=source_relative:$outdir" "--gosrcinfo_out=paths=source_relative,debug:$outdir" -I. pkg/*.proto
${PROTOC} "--go_out=paths=source_relative:$outdir" "--gosrcinfo_out=paths=source_relative,debug:$outdir" -I. proto3_optional/desc_test_proto3_optional.proto
${PROTOC} "--go_out=paths=source_relative:$outdir" "--go-grpc_out=paths=source_relative:$outdir" "--gosrcinfo_out=paths=source_relative,debug:$outdir" -I. grpc/*.proto

# And make descriptor set (with source info) for several files
${PROTOC} --descriptor_set_out=./desc_test1.protoset --include_source_info --include_imports -I. desc_test1.proto
${PROTOC} --descriptor_set_out=./desc_test_comments.protoset --include_source_info --include_imports -I. desc_test_comments.proto
${PROTOC} --descriptor_set_out=./desc_test_complex.protoset -I. desc_test_complex.proto
${PROTOC} --descriptor_set_out=./desc_test_complex_source_info.protoset --include_source_info --include_imports -I. desc_test_complex.proto
${PROTOC} --descriptor_set_out=./desc_test_editions.protoset --include_source_info --include_imports -I. desc_test_editions.proto
${PROTOC} --descriptor_set_out=./desc_test_proto3.protoset --include_source_info --include_imports -I. desc_test_proto3.proto
${PROTOC} --descriptor_set_out=./descriptor.protoset --include_source_info --include_imports -I./protoc/include/ google/protobuf/descriptor.proto
${PROTOC} --descriptor_set_out=./duration.protoset -I./protoc/include/ google/protobuf/duration.proto
${PROTOC} --descriptor_set_out=./proto3_optional/desc_test_proto3_optional.protoset --include_source_info --include_imports -I. proto3_optional/desc_test_proto3_optional.proto

