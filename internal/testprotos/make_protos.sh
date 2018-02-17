#!/usr/bin/env bash

set -e

cd $(dirname $0)

PROTOC_VERSION="3.5.1"
PROTOC_OS="$(uname -s)"
PROTOC_ARCH="$(uname -m)"
case "${PROTOC_OS}" in
  Darwin) PROTOC_OS="osx" ;;
  Linux) PROTOC_OS=linux ;;
  *)
    echo "Invalid value for uname -s: ${PROTOC_OS}" >&2
    exit 1
esac

rm -rf ./protoc
mkdir -p protoc

curl -L "https://github.com/google/protobuf/releases/download/v${PROTOC_VERSION}/protoc-${PROTOC_VERSION}-${PROTOC_OS}-${PROTOC_ARCH}.zip" > protoc/protoc.zip
cd ./protoc && unzip protoc.zip && cd ..

PROTOC="./protoc/bin/protoc"

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
${PROTOC} --descriptor_set_out=./descriptor.protoset --include_source_info --include_imports -I../../../../ ../../../../golang/protobuf/protoc-gen-go/descriptor/descriptor.proto
