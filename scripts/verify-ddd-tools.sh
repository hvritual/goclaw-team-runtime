#!/usr/bin/env bash

set -euo pipefail

if [[ $# -ne 3 ]]; then
  echo "usage: $0 <tools-bin> <expected-module> <expected-version>" >&2
  exit 2
fi

tools_bin=$1
expected_module=$2
expected_version=$3

verify_tool() {
  local name=$1
  local expected_path=$2
  local binary="${tools_bin}/${name}"
  local metadata
  local actual_path
  local actual_module
  local actual_version

  if [[ ! -x "${binary}" ]]; then
    echo "missing ${name}: run 'make bootstrap' to install the pinned DDD toolchain" >&2
    return 1
  fi

  metadata=$(go version -m "${binary}")
  actual_path=$(awk '$1 == "path" { print $2 }' <<<"${metadata}")
  actual_module=$(awk '$1 == "mod" { print $2 }' <<<"${metadata}")
  actual_version=$(awk '$1 == "mod" { print $3 }' <<<"${metadata}")

  if [[ "${actual_path}" != "${expected_path}" ]]; then
    echo "${name} package mismatch: expected ${expected_path}, got ${actual_path:-unknown}" >&2
    return 1
  fi
  if [[ "${actual_module}" != "${expected_module}" || "${actual_version}" != "${expected_version}" ]]; then
    echo "${name} version mismatch: expected ${expected_module}@${expected_version}, got ${actual_module:-unknown}@${actual_version:-unknown}" >&2
    return 1
  fi
}

verify_tool dddgen "${expected_module}/cmd/dddgen"
verify_tool protoc-gen-access "${expected_module}/cmd/protoc-gen-access"

echo "verified dddgen toolchain ${expected_module}@${expected_version}"
