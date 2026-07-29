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

mkdir -p "${output_dir}"
output_dir="$(cd "${output_dir}" && pwd)"
work_dir="$(mktemp -d)"
trap 'rm -rf "${work_dir}"' EXIT

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
  output="${output_dir}/${app_name}-${target_os}-${target_arch}${suffix}"
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

for target in \
  linux/amd64 linux/arm64 \
  darwin/amd64 darwin/arm64 \
  windows/amd64 windows/arm64; do
  target_os="${target%/*}"
  target_arch="${target#*/}"
  build_one "goclaw-team-control" "./cmd/team-control" "${target_os}" "${target_arch}"
  build_one "goclaw-runner" "./cmd/runner" "${target_os}" "${target_arch}"
  build_one "goclaw" "." "${target_os}" "${target_arch}"
done

for secret_name in \
  GOCLAW_USER_TOKEN GOCLAW_GATEWAY_TOKEN GH_TOKEN GITHUB_TOKEN; do
  secret_value="${!secret_name:-}"
  if [[ -z "${secret_value}" ]]; then
    continue
  fi
  while IFS= read -r artifact; do
    if grep -aFq "${secret_value}" "${artifact}"; then
      printf 'refusing artifact containing %s: %s\n' \
        "${secret_name}" "${artifact}" >&2
      exit 1
    fi
  done < <(find "${output_dir}" -maxdepth 1 -type f ! -name SHA256SUMS -print)
done

(
  cd "${output_dir}"
  find . -maxdepth 1 -type f ! -name SHA256SUMS -printf '%f\n' |
    LC_ALL=C sort |
    xargs sha256sum > SHA256SUMS
)

printf 'Built application artifacts in %s\n' "${output_dir}"
