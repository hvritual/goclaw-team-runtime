#!/usr/bin/env bash
set -euo pipefail

repo_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../.." && pwd)"
# shellcheck source=scripts/recovery/release-archive-lib.sh
source "${repo_dir}/scripts/recovery/release-archive-lib.sh"

test_root="$(mktemp -d "${TMPDIR:-/tmp}/goclaw-archive-lib-test.XXXXXX")"
cleanup() {
  rm -rf -- "${test_root}"
}
trap cleanup EXIT

expect_rejected() {
  local name="$1"
  shift
  if "$@" > "${test_root}/${name}.stdout" 2> "${test_root}/${name}.stderr"; then
    echo "archive contract unexpectedly accepted ${name}" >&2
    exit 1
  fi
}

mkdir -p "${test_root}/input"
printf 'payload\n' > "${test_root}/input/payload.txt"
printf 'payload.txt\n' > "${test_root}/manifest.txt"
printf 'root/payload.txt\n' > "${test_root}/expected.txt"

goclaw_create_normalized_archive \
  "${test_root}/first.tar.gz" \
  "${test_root}/input" \
  "${test_root}/manifest.txt" \
  "root" \
  "1700000000"
goclaw_create_normalized_archive \
  "${test_root}/second.tar.gz" \
  "${test_root}/input" \
  "${test_root}/manifest.txt" \
  "root" \
  "1700000000"
cmp "${test_root}/first.tar.gz" "${test_root}/second.tar.gz"
goclaw_validate_archive_contract \
  "${test_root}/first.tar.gz" \
  "${test_root}/expected.txt" \
  "root" \
  "${test_root}/valid-extract"

printf 'extra\n' > "${test_root}/input/extra.txt"
printf '%s\n' payload.txt extra.txt > "${test_root}/extra-manifest.txt"
goclaw_create_normalized_archive \
  "${test_root}/extra.tar.gz" \
  "${test_root}/input" \
  "${test_root}/extra-manifest.txt" \
  "root" \
  "1700000000"
expect_rejected extra-member \
  goclaw_validate_archive_contract \
  "${test_root}/extra.tar.gz" \
  "${test_root}/expected.txt" \
  "root" \
  "${test_root}/extra-extract"

(
  cd "${test_root}/input"
  tar -czf "${test_root}/duplicate.tar.gz" payload.txt payload.txt
)
printf 'payload.txt\n' > "${test_root}/plain-expected.txt"
expect_rejected duplicate-member \
  goclaw_validate_archive_contract \
  "${test_root}/duplicate.tar.gz" \
  "${test_root}/plain-expected.txt" \
  "-" \
  "${test_root}/duplicate-extract"

ln -s payload.txt "${test_root}/input/link"
(
  cd "${test_root}/input"
  tar -czf "${test_root}/link.tar.gz" link
)
printf 'link\n' > "${test_root}/link-expected.txt"
expect_rejected link-member \
  goclaw_validate_archive_contract \
  "${test_root}/link.tar.gz" \
  "${test_root}/link-expected.txt" \
  "-" \
  "${test_root}/link-extract"

(
  cd "${test_root}/input"
  tar \
    --create \
    --gzip \
    --file="${test_root}/traversal.tar.gz" \
    --transform='s#^#../#' \
    payload.txt
)
printf '../payload.txt\n' > "${test_root}/traversal-expected.txt"
expect_rejected traversal-member \
  goclaw_validate_archive_contract \
  "${test_root}/traversal.tar.gz" \
  "${test_root}/traversal-expected.txt" \
  "-" \
  "${test_root}/traversal-extract"

echo "release archive library tests: PASS"
