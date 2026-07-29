#!/usr/bin/env bash
set -euo pipefail

repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
output_dir="${repo_dir}/dist/apps"
release_version="${VERSION:-dev}"
release_commit="${COMMIT:-unknown}"
release_date="${BUILD_DATE:-unknown}"

usage() {
  printf '%s\n' \
    "Usage: scripts/build-apps.sh [--output DIR] [--version VERSION]" \
    "" \
    "Builds Team Control, Runner, and the compatibility goclaw entrypoint for" \
    "linux, darwin, and windows on amd64 and arm64."
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --output)
      output_dir="$2"
      shift 2
      ;;
    --version)
      release_version="$2"
      shift 2
      ;;
    --help|-h)
      usage
      exit 0
      ;;
    *)
      printf 'unknown argument: %s\n' "$1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

output_parent="$(dirname "${output_dir}")"
output_name="$(basename "${output_dir}")"
mkdir -p "${output_parent}"
output_parent="$(cd "${output_parent}" && pwd)"
output_dir="${output_parent}/${output_name}"
if [[ -e "${output_dir}" ]]; then
  [[ -d "${output_dir}" && ! -L "${output_dir}" ]] || {
    printf 'output path is not a regular directory: %s\n' "${output_dir}" >&2
    exit 1
  }
  if find "${output_dir}" -mindepth 1 -print -quit | grep -q .; then
    printf 'refusing non-empty output directory: %s\n' "${output_dir}" >&2
    exit 1
  fi
  rmdir "${output_dir}"
fi

stage_dir="$(mktemp -d "${output_parent}/.${output_name}.stage.XXXXXX")"
artifact_dir="${stage_dir}/artifacts"
work_dir="${stage_dir}/work"
expected_manifest="${work_dir}/expected-artifacts.txt"
mkdir -p "${artifact_dir}" "${work_dir}"
published=0
cleanup_build_stage() {
  if [[ "${published}" == "0" && -d "${stage_dir}" ]]; then
    rm -rf -- "${stage_dir}"
  fi
}
trap cleanup_build_stage EXIT

build_one() {
  local app_name="$1"
  local package_path="$2"
  local target_os="$3"
  local target_arch="$4"
  local suffix=""
  local output

  if [[ "${target_os}" == "windows" ]]; then
    suffix=".exe"
  fi
  output="${artifact_dir}/${app_name}-${target_os}-${target_arch}${suffix}"
  (
    cd "${repo_dir}"
    CGO_ENABLED=0 GOOS="${target_os}" GOARCH="${target_arch}" \
      go build -buildvcs=false -trimpath \
      -ldflags="-s -w -X main.Version=${release_version} -X main.Commit=${release_commit} -X main.Date=${release_date}" \
      -o "${output}" "${package_path}"
  )
  go version -m "${output}" > "${work_dir}/${app_name}-${target_os}-${target_arch}.buildinfo"
  grep -Fq "GOOS=${target_os}" "${work_dir}/${app_name}-${target_os}-${target_arch}.buildinfo"
  grep -Fq "GOARCH=${target_arch}" "${work_dir}/${app_name}-${target_os}-${target_arch}.buildinfo"
}

: > "${expected_manifest}"
for target in \
  linux/amd64 linux/arm64 \
  darwin/amd64 darwin/arm64 \
  windows/amd64 windows/arm64; do
  target_os="${target%/*}"
  target_arch="${target#*/}"
  target_suffix=""
  if [[ "${target_os}" == "windows" ]]; then
    target_suffix=".exe"
  fi
  printf '%s\n' \
    "goclaw-${target_os}-${target_arch}${target_suffix}" \
    "goclaw-runner-${target_os}-${target_arch}${target_suffix}" \
    "goclaw-team-control-${target_os}-${target_arch}${target_suffix}" \
    >> "${expected_manifest}"
  build_one "goclaw-team-control" "./cmd/team-control" "${target_os}" "${target_arch}"
  build_one "goclaw-runner" "./cmd/runner" "${target_os}" "${target_arch}"
  build_one "goclaw" "." "${target_os}" "${target_arch}"
done
LC_ALL=C sort -u -o "${expected_manifest}" "${expected_manifest}"

actual_manifest="${work_dir}/actual-artifacts.txt"
find "${artifact_dir}" -maxdepth 1 -type f -printf '%f\n' |
  LC_ALL=C sort > "${actual_manifest}"
if ! diff -u "${expected_manifest}" "${actual_manifest}"; then
  printf 'cross-build artifact contract mismatch\n' >&2
  exit 1
fi
[[ "$(wc -l < "${actual_manifest}")" -eq 18 ]] || {
  printf 'cross-build must produce exactly 18 binaries\n' >&2
  exit 1
}

for secret_name in \
  GOCLAW_USER_TOKEN GOCLAW_GATEWAY_TOKEN GOCLAW_REVIEWER_TOKEN \
  GOCLAW_RUNNER_DEVICE_KEY CODEX_ACCESS_TOKEN CODEX_REFRESH_TOKEN \
  GH_TOKEN GITHUB_TOKEN; do
  secret_value="${!secret_name:-}"
  if [[ -z "${secret_value}" ]]; then
    continue
  fi
  while IFS= read -r artifact; do
    if grep -aFq -- "${secret_value}" "${artifact}"; then
      printf 'refusing artifact containing %s: %s\n' \
        "${secret_name}" "${artifact}" >&2
      exit 1
    fi
  done < <(find "${artifact_dir}" -maxdepth 1 -type f -print)
done

(
  cd "${artifact_dir}"
  mapfile -t expected_artifacts < "${expected_manifest}"
  sha256sum -- "${expected_artifacts[@]}" > SHA256SUMS
  sha256sum -c SHA256SUMS
)

[[ ! -e "${output_dir}" ]] || {
  printf 'output path appeared during build: %s\n' "${output_dir}" >&2
  exit 1
}
mv -- "${artifact_dir}" "${output_dir}"
published=1
rm -rf -- "${stage_dir}"
printf 'Built application artifacts in %s\n' "${output_dir}"
