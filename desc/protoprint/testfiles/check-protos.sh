#!/usr/bin/env bash

set -e

cd $(dirname $0)

for f in *.proto; do
  echo -n "Checking $f..."
  ../../../internal/testprotos/protoc/bin/protoc $f --experimental_editions -o /dev/null -I ../../../internal/testprotos -I .
  echo "  good"
done
