#!/usr/bin/env bash
set -e

gover="$(go version | awk '{ print $3 }')"

# We don't run gofmt for devel versions because it
# changed circa 11/2017. So code that passes the gofmt
# check for other versions will fail for devel version.
# For now, just skip the check for devel versions.

# The second term removes "devel" prefix, so if the two
# strings are equal, it does not have that prefix, and
# thus this is not a devel version.
if [[ ${gover} == ${gover#devel*} ]]; then
  fmtdiff="$(gofmt -s -l ./)"
  if [[ -n "$fmtdiff" ]]; then
    gofmt -s -l ./ >&2
    echo "Run gofmt on the above files!" >&2
    exit 1
  fi
fi

go test -v -race ./...
