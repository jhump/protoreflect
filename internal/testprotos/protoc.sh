#!/usr/bin/env bash

# Downloads and caches the version of protoc pinned below, then prints the path
# of the protoc binary to stdout. Everything else this script emits goes to
# stderr, so callers can simply do:
#
#     PROTOC="$(/path/to/internal/testprotos/protoc.sh)"
#
# The download is cached in internal/testprotos/protoc, which is git-ignored.
# This is the single place where the protoc version used to generate any of the
# checked-in test data in this repo is defined. Bump it here and then re-run
# internal/testprotos/make_protos.sh.

set -euo pipefail

DIR="$(CDPATH= cd "$(dirname "${0}")" && pwd)"
cd "${DIR}"

PROTOC_VERSION="35.1"
# For release candidates, the download artifact has a dash between "rc" and the
# number even though the version tag does not :(
PROTOC_ARTIFACT_VERSION="$(echo "${PROTOC_VERSION}" | sed -E 's/-rc([0-9]+)$/-rc-\1/')"
PROTOC_OS="$(uname -s)"
PROTOC_ARCH="$(uname -m)"
case "${PROTOC_OS}" in
  Darwin) PROTOC_OS="osx" ;;
  Linux) PROTOC_OS="linux" ;;
  *)
    echo "Invalid value for uname -s: ${PROTOC_OS}" >&2
    exit 1
    ;;
esac

# This is for macs with M1 chips. Precompiled binaries for osx/amd64 are not available for download, so for that case
# we download the x86_64 version instead. This will work as long as rosetta2 is installed.
if [ "${PROTOC_OS}" = "osx" ] && [ "${PROTOC_ARCH}" = "arm64" ]; then
  PROTOC_ARCH="x86_64"
fi

PROTOC="${PWD}/protoc/bin/protoc"

if [[ "$("${PROTOC}" --version 2>/dev/null)" != "libprotoc ${PROTOC_VERSION}" ]]; then
  rm -rf ./protoc
  mkdir -p protoc
  curl -fSL -o protoc/protoc.zip \
    "https://github.com/google/protobuf/releases/download/v${PROTOC_VERSION}/protoc-${PROTOC_ARTIFACT_VERSION}-${PROTOC_OS}-${PROTOC_ARCH}.zip" >&2
  unzip -q protoc/protoc.zip -d protoc
fi

echo "${PROTOC}"
