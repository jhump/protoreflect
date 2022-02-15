#!/usr/bin/env bash

set -e

cd $(dirname $0)

PROTOC_VERSION="3.19.0"
PROTOC_OS="$(uname -s)"
PROTOC_ARCH="$(uname -m)"
case "${PROTOC_OS}" in
  Darwin) PROTOC_OS="osx" ;;
  Linux) PROTOC_OS="linux" ;;
  *)
    echo "Invalid value for uname -s: ${PROTOC_OS}" >&2
    exit 1
esac

# This is for macs with M1 chips. Precompiled binaries for osx/amd64 are not available for download, so for that case
# we download the x86_64 version instead. This will work as long as rosetta2 is installed.
if [ "$PROTOC_OS" = "osx" ] && [ "$PROTOC_ARCH" = "arm64" ]; then
  PROTOC_ARCH="x86_64"
fi

PROTOC="./protoc/bin/protoc"

if [[ "$(${PROTOC} --version 2>/dev/null)" != "libprotoc ${PROTOC_VERSION}" ]]; then
  rm -rf ./protoc
  mkdir -p protoc
  curl -L "https://github.com/google/protobuf/releases/download/v${PROTOC_VERSION}/protoc-${PROTOC_VERSION}-${PROTOC_OS}-${PROTOC_ARCH}.zip" > protoc/protoc.zip
  cd ./protoc && unzip protoc.zip && cd ..
fi

go install github.com/golang/protobuf/protoc-gen-go 
go install github.com/jhump/protoreflect/desc/sourceinfo/cmd/protoc-gen-gosrcinfo

# Output directory will effectively be GOPATH/src.
outdir="../../../../.."
${PROTOC} "--go_out=plugins=grpc:$outdir" "--gosrcinfo_out=debug:$outdir" -I. *.proto
${PROTOC} "--go_out=plugins=grpc:$outdir" "--gosrcinfo_out=debug:$outdir" -I. nopkg/*.proto
${PROTOC} "--go_out=plugins=grpc:$outdir" "--gosrcinfo_out=debug:$outdir" -I. pkg/*.proto
${PROTOC} "--go_out=plugins=grpc:$outdir" "--gosrcinfo_out=debug:$outdir" -I. grpc/*.proto

# And make descriptor set (with source info) for several files
${PROTOC} --descriptor_set_out=./desc_test1.protoset --include_source_info --include_imports -I. desc_test1.proto
${PROTOC} --descriptor_set_out=./desc_test_comments.protoset --include_source_info --include_imports -I. desc_test_comments.proto
${PROTOC} --descriptor_set_out=./desc_test_complex.protoset -I. desc_test_complex.proto
${PROTOC} --descriptor_set_out=./desc_test_complex_source_info.protoset --include_source_info --include_imports -I. desc_test_complex.proto
${PROTOC} --descriptor_set_out=./descriptor.protoset --include_source_info --include_imports -I./protoc/include/ google/protobuf/descriptor.proto
${PROTOC} --descriptor_set_out=./duration.protoset -I./protoc/include/ google/protobuf/duration.proto

# We are currently pinning an earlier version of Go protobuf runtime, and thus of protoc-gen-go.
# So it doesn't support proto3 optional fields yet. So we only create a descriptor for these, just
# for testing proto3 optional support in the desc and desc/protoparse packages.
${PROTOC} --descriptor_set_out=./proto3_optional/desc_test_proto3_optional.protoset --include_source_info --include_imports -I. proto3_optional/desc_test_proto3_optional.proto

