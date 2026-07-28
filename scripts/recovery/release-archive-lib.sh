#!/usr/bin/env bash

# Shared deterministic archive creation and pre-extraction contract checks.
# Callers are expected to enable `set -euo pipefail`.

goclaw_archive_fail() {
  echo "release archive validation failed: $*" >&2
  return 1
}

goclaw_create_normalized_archive() {
  local archive="$1"
  local source_root="$2"
  local manifest="$3"
  local prefix="$4"
  local source_date_epoch="$5"
  local transform_args=()

  if [[ ! -d "${source_root}" ]]; then
    goclaw_archive_fail "source root is not a directory: ${source_root}"
    return 1
  fi
  if [[ ! -s "${manifest}" ]]; then
    goclaw_archive_fail "archive manifest is empty: ${manifest}"
    return 1
  fi
  if [[ ! "${source_date_epoch}" =~ ^[0-9]+$ ]]; then
    goclaw_archive_fail "SOURCE_DATE_EPOCH is not numeric"
    return 1
  fi

  if [[ -n "${prefix}" ]]; then
    transform_args=(--transform="s#^#${prefix}/#")
  fi

  (
    cd "${source_root}"
    LC_ALL=C tar \
      --create \
      --format=gnu \
      --sort=name \
      --mtime="@${source_date_epoch}" \
      --owner=0 \
      --group=0 \
      --numeric-owner \
      --mode='u+rwX,go+rX,go-w' \
      --no-recursion \
      "${transform_args[@]}" \
      --files-from="${manifest}" |
      gzip --no-name --best > "${archive}"
  )
}

goclaw_validate_archive_contract() {
  local archive="$1"
  local expected_manifest="$2"
  local expected_root="$3"
  local extract_root="$4"
  local validation_root actual_manifest type_manifest duplicates
  local member member_path type

  if [[ ! -f "${archive}" ]]; then
    goclaw_archive_fail "archive is not a regular file: ${archive}"
    return 1
  fi
  if [[ ! -s "${expected_manifest}" ]]; then
    goclaw_archive_fail "expected member manifest is empty"
    return 1
  fi
  if [[ -e "${extract_root}" ]]; then
    goclaw_archive_fail \
      "extract destination already exists: ${extract_root}"
    return 1
  fi

  validation_root="${extract_root}.contract"
  if [[ -e "${validation_root}" ]]; then
    goclaw_archive_fail \
      "archive validation directory already exists: ${validation_root}"
    return 1
  fi
  mkdir -p "${validation_root}"
  actual_manifest="${validation_root}/actual.txt"
  type_manifest="${validation_root}/types.txt"
  duplicates="${validation_root}/duplicates.txt"

  if ! tar -tzf "${archive}" > "${actual_manifest}"; then
    goclaw_archive_fail "cannot list ${archive}"
    return 1
  fi
  if ! tar -tvzf "${archive}" > "${type_manifest}"; then
    goclaw_archive_fail "cannot inspect entry types in ${archive}"
    return 1
  fi
  if [[ ! -s "${actual_manifest}" ]]; then
    goclaw_archive_fail "archive is empty: ${archive}"
    return 1
  fi

  LC_ALL=C sort "${actual_manifest}" | uniq -d > "${duplicates}"
  if [[ -s "${duplicates}" ]]; then
    echo "duplicate archive members:" >&2
    cat "${duplicates}" >&2
    return 1
  fi
  LC_ALL=C sort "${expected_manifest}" | uniq -d > "${duplicates}"
  if [[ -s "${duplicates}" ]]; then
    echo "duplicate expected members:" >&2
    cat "${duplicates}" >&2
    return 1
  fi

  while IFS= read -r member; do
    if [[ -z "${member}" ]]; then
      goclaw_archive_fail "archive contains an empty member name"
      return 1
    fi
    member_path="${member%/}"
    case "${member_path}" in
      /* | .. | ../* | */../* | */.. | *\\*)
        goclaw_archive_fail "unsafe archive member: ${member}"
        return 1
        ;;
    esac
    if [[ "${expected_root}" != "-" ]]; then
      case "${member_path}" in
        "${expected_root}/"*)
          ;;
        *)
          goclaw_archive_fail \
            "archive member is outside ${expected_root}: ${member}"
          return 1
          ;;
      esac
    fi
  done < "${actual_manifest}"

  while IFS= read -r member; do
    if [[ -z "${member}" ]]; then
      goclaw_archive_fail "cannot parse archive entry type"
      return 1
    fi
    type="${member:0:1}"
    if [[ "${type}" != "-" ]]; then
      goclaw_archive_fail "unsupported archive entry type ${type}"
      return 1
    fi
  done < "${type_manifest}"

  if ! diff -u \
    <(LC_ALL=C sort "${expected_manifest}") \
    <(LC_ALL=C sort "${actual_manifest}"); then
    goclaw_archive_fail "archive members do not match the exact contract"
    return 1
  fi

  mkdir -p "${extract_root}"
  if ! tar \
    --extract \
    --gzip \
    --file="${archive}" \
    --directory="${extract_root}" \
    --no-same-owner \
    --no-same-permissions; then
    goclaw_archive_fail "cannot extract validated archive"
    return 1
  fi

  if find "${extract_root}" ! -type d ! -type f -print -quit | grep -q .; then
    goclaw_archive_fail "extracted archive contains a non-regular object"
    return 1
  fi

  rm -rf -- "${validation_root}"
}
