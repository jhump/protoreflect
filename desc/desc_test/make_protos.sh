#!/usr/bin/env bash

set -e

cd $(dirname $0)

# Output directory will effectively be GOPATH/src.
outdir="../../../../.."
protoc "--go_out=plugins=grpc:$outdir" -I. *.proto
protoc "--go_out=plugins=grpc:$outdir" -I. nopkg/*.proto
protoc "--go_out=plugins=grpc:$outdir" -I. pkg/*.proto

# And make descriptor set (with source info) for one file
protoc --descriptor_set_out=./desc_test1.protoset --include_source_info --include_imports -I. desc_test1.proto
# And then process that file into Go code
go run util/fileset_to_go.go < ./desc_test1.protoset > desc_test1_protoset.go

