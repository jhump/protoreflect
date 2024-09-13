#!/usr/bin/env bash

set -e

cd $(dirname $0)

for f in *.proto; do
  echo -n "Checking $f..."
  ../../internal/testprotos/protoc/bin/protoc $f -o /dev/null -I . -I ../../internal/testprotos
  echo "  good"
done
