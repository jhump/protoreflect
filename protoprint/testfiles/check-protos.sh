#!/usr/bin/env bash

set -e

cd $(dirname $0)

for f in *.proto; do
  echo -n "Checking $f..."
  ../../internal/testdata/protoc/bin/protoc $f -o /dev/null -I . -I ../../internal/testdata
  echo "  good"
done
