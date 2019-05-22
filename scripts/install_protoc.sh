#!/usr/bin/env bash

set -euo pipefail

DIR="$(cd "$(dirname "${0}")" && pwd)"
cd "${DIR}"

PROTOC_VERSION="3.5.1"
PROTOC_OS="$(uname -s)"
PROTOC_ARCH="$(uname -m)"
case "${PROTOC_OS}" in
  Darwin) PROTOC_OS="osx" ;;
  Linux) PROTOC_OS="linux" ;;
  *)
    echo "Invalid value for uname -s: ${PROTOC_OS}" >&2
    exit 1
esac

PROTOC_DIR="../.tmp/protoc"
PROTOC="${PROTOC_DIR}/bin/protoc"

if [[ "$(${PROTOC} --version 2>/dev/null)" != "libprotoc ${PROTOC_VERSION}" ]]; then
  rm -rf "${PROTOC_DIR}"
  mkdir -p "${PROTOC_DIR}"
  curl -L "https://github.com/google/protobuf/releases/download/v${PROTOC_VERSION}/protoc-${PROTOC_VERSION}-${PROTOC_OS}-${PROTOC_ARCH}.zip" > "${PROTOC_DIR}/protoc.zip"
  cd "${PROTOC_DIR}" && unzip protoc.zip && cd -
fi
