#!/usr/bin/env bash
set -euo pipefail

archive="${1:-}"
import_ref="${GOCLAW_IMPORT_REF:-}"
if [[ -z "${import_ref}" ]]; then
  import_ref='v0.8.0-pilot.1-import^{}'
fi
expected_archive_sha="cf327169e7654d2284c98482e4d885085ed6068152f5ae9cbd103ea5ffd78c8f"
expected_tree="38f798c2a652eaf99d5ad1ca145e50c176ee4c58"
expected_root="goclaw-0.8.0-pilot.1"
expected_files="611"

if [[ -z "${archive}" ]]; then
  echo "usage: $0 /immutable/path/goclaw-team-runtime-source-0.8.0-pilot.1.tar.gz" >&2
  exit 2
fi
if [[ ! -f "${archive}" ]]; then
  echo "source archive is not a regular file: ${archive}" >&2
  exit 2
fi

repo_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../.." && pwd)"
archive="$(cd -- "$(dirname -- "${archive}")" && pwd)/$(basename -- "${archive}")"

for command_name in git tar sha256sum diff find sort uniq awk mktemp; do
  command -v "${command_name}" >/dev/null ||
    {
      echo "required command is unavailable: ${command_name}" >&2
      exit 2
    }
done

actual_archive_sha="$(sha256sum -- "${archive}" | awk '{print $1}')"
if [[ "${actual_archive_sha}" != "${expected_archive_sha}" ]]; then
  echo "source archive SHA-256 mismatch" >&2
  echo "expected: ${expected_archive_sha}" >&2
  echo "actual:   ${actual_archive_sha}" >&2
  exit 1
fi

import_commit="$(git -C "${repo_dir}" rev-parse --verify "${import_ref}^{commit}")"
import_tree="$(git -C "${repo_dir}" rev-parse "${import_commit}^{tree}")"
if [[ "${import_tree}" != "${expected_tree}" ]]; then
  echo "import tree mismatch" >&2
  echo "expected: ${expected_tree}" >&2
  echo "actual:   ${import_tree}" >&2
  exit 1
fi

verify_root="$(mktemp -d "${TMPDIR:-/tmp}/goclaw-source-import.XXXXXX")"
cleanup() {
  rm -rf -- "${verify_root}"
}
trap cleanup EXIT

member_list="${verify_root}/archive-members.txt"
type_list="${verify_root}/archive-types.txt"
duplicate_list="${verify_root}/archive-duplicates.txt"
archive_root="${verify_root}/archive"
git_root="${verify_root}/git"
mkdir -p "${archive_root}" "${git_root}"

tar -tzf "${archive}" > "${member_list}"
tar -tvzf "${archive}" > "${type_list}"

member_count="$(wc -l < "${member_list}")"
member_count="${member_count//[[:space:]]/}"
if [[ "${member_count}" != "${expected_files}" ]]; then
  echo "archive member count mismatch: ${member_count}, expected ${expected_files}" >&2
  exit 1
fi

LC_ALL=C sort "${member_list}" | uniq -d > "${duplicate_list}"
if [[ -s "${duplicate_list}" ]]; then
  echo "source archive contains duplicate member names:" >&2
  cat "${duplicate_list}" >&2
  exit 1
fi

while IFS= read -r member; do
  [[ -n "${member}" ]] || {
    echo "source archive contains an empty member name" >&2
    exit 1
  }
  case "${member}" in
    /* | .. | ../* | */../* | */.. | *\\*)
      echo "unsafe source archive member: ${member}" >&2
      exit 1
      ;;
    "${expected_root}/"*)
      ;;
    *)
      echo "source archive member is outside ${expected_root}: ${member}" >&2
      exit 1
      ;;
  esac
  case "/${member}/" in
    */.git/*)
      echo "source archive contains Git metadata: ${member}" >&2
      exit 1
      ;;
  esac
done < "${member_list}"

while IFS= read -r verbose_member; do
  [[ "${verbose_member:0:1}" == "-" ]] || {
    echo "source archive contains a non-regular member: ${verbose_member}" >&2
    exit 1
  }
done < "${type_list}"

tar \
  --extract \
  --gzip \
  --file "${archive}" \
  --directory "${archive_root}" \
  --no-same-owner
git -C "${repo_dir}" archive \
  --format=tar \
  --prefix="${expected_root}/" \
  "${import_commit}" |
  tar --extract --file=- --directory "${git_root}" --no-same-owner

if find "${archive_root}" "${git_root}" -type l -print -quit | grep -q .; then
  echo "a recovered tree contains a symlink" >&2
  exit 1
fi

archive_file_count="$(
  find "${archive_root}/${expected_root}" -type f -printf '.' |
    wc -c
)"
git_file_count="$(
  find "${git_root}/${expected_root}" -type f -printf '.' |
    wc -c
)"
archive_file_count="${archive_file_count//[[:space:]]/}"
git_file_count="${git_file_count//[[:space:]]/}"
if [[ "${archive_file_count}" != "${expected_files}" ||
      "${git_file_count}" != "${expected_files}" ]]; then
  echo "recovered file count mismatch: archive=${archive_file_count} git=${git_file_count}" >&2
  exit 1
fi

diff \
  --recursive \
  --brief \
  --no-dereference \
  "${archive_root}/${expected_root}" \
  "${git_root}/${expected_root}"

(
  cd "${archive_root}/${expected_root}"
  find . -type f -perm /111 -printf '%P\n' | LC_ALL=C sort
) > "${verify_root}/archive-executable-files.txt"
(
  cd "${git_root}/${expected_root}"
  find . -type f -perm /111 -printf '%P\n' | LC_ALL=C sort
) > "${verify_root}/git-executable-files.txt"
diff \
  --brief \
  "${verify_root}/archive-executable-files.txt" \
  "${verify_root}/git-executable-files.txt"

printf 'source import verification: PASS\n'
printf 'archive_sha256=%s\n' "${actual_archive_sha}"
printf 'import_commit=%s\n' "${import_commit}"
printf 'import_tree=%s\n' "${import_tree}"
printf 'files=%s\n' "${expected_files}"
printf 'content_mismatches=0\n'
printf 'executable_bit_mismatches=0\n'
printf 'extra_files=0\n'
