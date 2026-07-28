#!/usr/bin/env bash
set -euo pipefail

repo_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
dist_dir="${repo_dir}/dist"
release_root="${dist_dir}/releases"
release_version="${RELEASE_VERSION:-0.8.0-pilot.1-recovered.1}"
npm_cache_dir="${GOCLAW_NPM_CACHE:-${TMPDIR:-/tmp}/goclaw-npm-cache}"
include_obsidian_plugin="${INCLUDE_OBSIDIAN_PLUGIN:-0}"
source_only="${SOURCE_ONLY:-0}"

# shellcheck source=scripts/recovery/release-archive-lib.sh
source "${repo_dir}/scripts/recovery/release-archive-lib.sh"

fail_release() {
  echo "release validation failed: $*" >&2
  exit 1
}

for command_name in \
  awk chmod cp diff find flock git go grep gzip install mkdir mktemp mv node npm \
  od rm sha256sum sort tar tr uniq; do
  command -v "${command_name}" >/dev/null ||
    fail_release "required command is unavailable: ${command_name}"
done

[[ "${release_version}" =~ ^[0-9A-Za-z][0-9A-Za-z.+-]*$ ]] ||
  fail_release "RELEASE_VERSION contains unsafe characters"
[[ "${release_version}" != "0.8.0-pilot.1" ]] ||
  fail_release \
    "0.8.0-pilot.1 is reserved for the immutable original input archives"
[[ "${include_obsidian_plugin}" == "0" ||
  "${include_obsidian_plugin}" == "1" ]] ||
  fail_release "INCLUDE_OBSIDIAN_PLUGIN must be 0 or 1"
[[ "${source_only}" == "0" || "${source_only}" == "1" ]] ||
  fail_release "SOURCE_ONLY must be 0 or 1"

expected_go_version="go1.25.5"
expected_node_version="v24.14.0"
expected_npm_version="11.9.0"
[[ "$(go version | awk '{print $3}')" == "${expected_go_version}" ]] ||
  fail_release "Go must be ${expected_go_version}"
[[ "$(node --version)" == "${expected_node_version}" ]] ||
  fail_release "Node must be ${expected_node_version}"
[[ "$(npm --version)" == "${expected_npm_version}" ]] ||
  fail_release "npm must be ${expected_npm_version}"

release_commit="$(git -C "${repo_dir}" rev-parse --verify HEAD^{commit})"
release_tree="$(git -C "${repo_dir}" rev-parse "${release_commit}^{tree}")"
commit_epoch="$(git -C "${repo_dir}" show -s --format=%ct "${release_commit}")"
if [[ -n "${SOURCE_DATE_EPOCH:-}" &&
  "${SOURCE_DATE_EPOCH}" != "${commit_epoch}" ]]; then
  fail_release \
    "SOURCE_DATE_EPOCH must equal the release commit timestamp ${commit_epoch}"
fi
source_date_epoch="${commit_epoch}"
export SOURCE_DATE_EPOCH="${source_date_epoch}"

if [[ -n "$(git -C "${repo_dir}" status --porcelain --untracked-files=all)" ]]; then
  fail_release "release builds require a clean Git worktree"
fi

mkdir -p "${dist_dir}" "${release_root}"
exec 9>"${dist_dir}/.release.lock"
flock -n 9 ||
  fail_release "another release build already holds ${dist_dir}/.release.lock"

stage_dir="$(mktemp -d "${dist_dir}/.release-${release_version}.XXXXXX")"
work_dir="${stage_dir}/work"
publish_stage="${stage_dir}/release"
mkdir -p "${work_dir}" "${publish_stage}"
cleanup_release_stage() {
  if [[ -n "${stage_dir:-}" && -d "${stage_dir}" ]]; then
    rm -rf -- "${stage_dir}"
  fi
}
trap cleanup_release_stage EXIT

# Keep the source release intentionally narrower than the working tree. The
# predicate is applied while building the manifest and again after extraction.
source_path_is_forbidden() {
  local path="${1#./}"
  local base="${path##*/}"

  case "/${path}/" in
    */.git/* | */.agents/* | */.codex/* | */.claude/* | */.idea/* | \
      */.vscode/* | */.env/* | */.env.*/* | */.dSYM/* | \
      */node_modules/* | */dist/* | */build/* | */out/* | */target/* | \
      */coverage/* | */.cache/* | */.next/* | */.turbo/*)
      return 0
      ;;
  esac

  case "${base}" in
    .env | .env.* | .npmrc | .netrc | .pypirc | .DS_Store | Thumbs.db | \
      auth.json | credentials.json | data.json | goclaw | goclaw_test | \
      coverage.out | *.coverprofile | *.prof | *.trace | \
      *.pem | *.key | *.p12 | *.pfx | *.jks | \
      *.db | *.db-* | *.sqlite | *.sqlite-* | *.sqlite3 | *.sqlite3-* | \
      *.log | *.tar | *.tar.gz | *.tgz | *.zip | *.7z | *.rar | *.gz | \
      *.bz2 | *.xz | *.test | *.exe | *.dll | *.dylib | *.so | *.so.* | \
      *.a | *.o | *.obj | *.wasm | *.bin | *.class | *.jar | *.war | \
      *.pyc | *.tmp | *.swp | *.bak | *~ | .eslintcache)
      return 0
      ;;
  esac

  return 1
}

file_has_common_binary_magic() {
  local file="$1"
  local magic
  magic="$(LC_ALL=C od -An -tx1 -N8 -- "${file}" | tr -d '[:space:]')"
  case "${magic}" in
    7f454c46* | 4d5a* | cafebabe* | feedface* | feedfacf* | \
      cefaedfe* | cffaedfe* | 213c617263683e0a* | 0061736d*)
      return 0
      ;;
  esac
  return 1
}

file_has_raw_credential_assignment() {
  local file="$1"
  LC_ALL=C awk '
    function trim(value) {
      sub(/^[[:space:]]+/, "", value)
      sub(/[[:space:]]+$/, "", value)
      return value
    }
    {
      line = $0
      sub(/^[[:space:]]+/, "", line)
      sub(/^export[[:space:]]+/, "", line)
      if (line !~ /^["\047]?(FEISHU_(APP_SECRET|ENCRYPT_KEY|VERIFICATION_TOKEN)|LARK_(APP_SECRET|ENCRYPT_KEY|VERIFICATION_TOKEN)|OPENAI_API_KEY|AZURE_OPENAI_API_KEY|ANTHROPIC_API_KEY|GEMINI_API_KEY|GOOGLE_API_KEY|GITHUB_TOKEN|GH_TOKEN|AWS_SECRET_ACCESS_KEY|AWS_SESSION_TOKEN|SLACK_(BOT|APP|USER)_TOKEN|TELEGRAM_BOT_TOKEN|DISCORD_BOT_TOKEN|NPM_TOKEN|PYPI_TOKEN|HF_TOKEN|HUGGINGFACE_TOKEN|CODEX_(ACCESS|REFRESH)_TOKEN|GOCLAW_[A-Z0-9_]*(TOKEN|SECRET|PASSWORD)|GOCLAW_RUNNER_DEVICE_KEY)["\047]?[[:space:]]*[:=]/) {
        next
      }

      sub(/^[^:=]+[:=][[:space:]]*/, "", line)
      sub(/[[:space:]]+(#|\/\/).*/, "", line)
      sub(/[[:space:]]*,[[:space:]]*$/, "", line)
      value = trim(line)
      sub(/^["\047]/, "", value)
      sub(/["\047]$/, "", value)
      value = trim(value)
      lower = tolower(value)

      if (value == "" ||
          value ~ /\$/ ||
          value ~ /^<.*>$/ ||
          value ~ /^\{\{.*\}\}$/ ||
          value ~ /(\.\.\.|[*][*][*])/ ||
          lower ~ /^(task-|env:|os\.getenv|os\.environ|getenv|secretref|vault:)/ ||
          lower ~ /^(replace|change[-_]?me|redacted|example|your[-_]|dummy|fake|sensitive|test[-_]?only|secure-token)/ ||
          lower ~ /[[:space:]]token$/ ||
          lower ~ /^(user-secret|gateway-secret|github-secret|device-secret)$/ ||
          value ~ /^[Xx*_-]+$/ ||
          length(value) < 12) {
        next
      }

      found = 1
      exit
    }
    END {
      exit(found ? 0 : 1)
    }
  ' "${file}"
}

file_has_credential_material() {
  local file="$1"
  local literal_pattern
  literal_pattern='-----BEGIN ([A-Z0-9]+[[:space:]])*PRIVATE KEY-----|xox[baprs]-[A-Za-z0-9-]{10,}|(^|[^A-Za-z0-9])sk-(proj-|svcacct-)?[A-Za-z0-9_-]{20,}|(AKIA|ASIA)[0-9A-Z]{16}|gh[pousr]_[A-Za-z0-9]{20,}|github_pat_[A-Za-z0-9_]{20,}'

  LC_ALL=C grep -aEq -- "${literal_pattern}" "${file}" ||
    file_has_raw_credential_assignment "${file}"
}

validate_source_file() {
  local file="$1"
  local display_path="$2"
  local findings_file="$3"

  if [[ -L "${file}" ]]; then
    printf '%s (symlink)\n' "${display_path}" >> "${findings_file}"
    return
  fi
  if [[ ! -f "${file}" ]]; then
    printf '%s (not a regular file)\n' "${display_path}" >> "${findings_file}"
    return
  fi
  if file_has_common_binary_magic "${file}"; then
    printf '%s (common executable/object binary)\n' \
      "${display_path}" >> "${findings_file}"
    return
  fi
  if file_has_credential_material "${file}"; then
    printf '%s (credential-like material)\n' \
      "${display_path}" >> "${findings_file}"
  fi
}

read_json_version() {
  node -e \
    'const fs=require("fs");console.log(JSON.parse(fs.readFileSync(process.argv[1],"utf8")).version)' \
    "$1"
}

obsidian_package="${repo_dir}/plugins/obsidian-goclaw/package.json"
obsidian_manifest="${repo_dir}/plugins/obsidian-goclaw/manifest.json"
obsidian_versions="${repo_dir}/plugins/obsidian-goclaw/versions.json"
obsidian_version="$(read_json_version "${obsidian_package}")"
[[ "$(read_json_version "${obsidian_manifest}")" == "${obsidian_version}" ]] ||
  fail_release "Obsidian package.json and manifest.json versions differ"
node -e \
  'const fs=require("fs");const v=process.argv[2];const p=JSON.parse(fs.readFileSync(process.argv[1],"utf8"));if(!Object.prototype.hasOwnProperty.call(p,v))process.exit(1)' \
  "${obsidian_versions}" "${obsidian_version}" ||
  fail_release "Obsidian versions.json does not contain ${obsidian_version}"

(
  cd "${repo_dir}/ui"
  NPM_CONFIG_CACHE="${npm_cache_dir}" npm ci \
    --registry=https://registry.npmjs.org \
    --replace-registry-host=always
  NPM_CONFIG_CACHE="${npm_cache_dir}" npm test
  NPM_CONFIG_CACHE="${npm_cache_dir}" npm run build
)
find "${repo_dir}/gateway/ui_dist" -mindepth 1 -type f -delete
find "${repo_dir}/gateway/ui_dist" -mindepth 1 -type d -empty -delete
cp -R "${repo_dir}/ui/dist/." "${repo_dir}/gateway/ui_dist/"

if [[ "${source_only}" == "0" ]]; then
  (
    cd "${repo_dir}"
    # The upstream all-package suite contains channel integration tests with
    # external network side effects. Keep release verification deterministic.
    go test -count=1 \
      ./memory ./memory/catalog ./governance ./ouroboros ./orchestratorlite \
      ./harness ./teamcontrol ./workstation ./providers ./gateway ./agent \
      ./agent/tools ./config ./cli ./cli/commands ./internal/start

    mkdir -p "${work_dir}/bin"
    for release_arch in amd64 arm64; do
      binary="${work_dir}/bin/goclaw-linux-${release_arch}"
      CGO_ENABLED=0 GOOS=linux GOARCH="${release_arch}" \
        go build -buildvcs=false -trimpath \
        -ldflags="-s -w -X main.Version=${release_version}" \
        -o "${binary}" .
      go version -m "${binary}" \
        > "${work_dir}/linux-${release_arch}.buildinfo"
      grep -Fq "GOOS=linux" \
        "${work_dir}/linux-${release_arch}.buildinfo" ||
        fail_release "linux/${release_arch} binary reports the wrong GOOS"
      grep -Fq "GOARCH=${release_arch}" \
        "${work_dir}/linux-${release_arch}.buildinfo" ||
        fail_release "linux/${release_arch} binary reports the wrong GOARCH"
    done

    # Native Windows/macOS are control-CLI-only targets during the pilot.
    for control_target in \
      darwin/amd64 darwin/arm64 windows/amd64 windows/arm64; do
      control_os="${control_target%/*}"
      control_arch="${control_target#*/}"
      control_suffix=""
      if [[ "${control_os}" == "windows" ]]; then
        control_suffix=".exe"
      fi
      CGO_ENABLED=0 GOOS="${control_os}" GOARCH="${control_arch}" \
        go build -buildvcs=false -trimpath \
        -ldflags="-s -w -X main.Version=${release_version}" \
        -o \
        "${work_dir}/control-${control_os}-${control_arch}${control_suffix}" .
    done
  )
fi

if [[ "${source_only}" == "0" &&
  "${include_obsidian_plugin}" == "1" ]]; then
  obsidian_stage="${work_dir}/obsidian-package"
  obsidian_expected="${work_dir}/obsidian-expected.txt"
  obsidian_archive="$(
    printf '%s/obsidian-goclaw-plugin-%s.tar.gz' \
      "${publish_stage}" "${obsidian_version}"
  )"
  mkdir -p "${obsidian_stage}/obsidian-goclaw"
  (
    cd "${repo_dir}/plugins/obsidian-goclaw"
    NPM_CONFIG_CACHE="${npm_cache_dir}" npm ci \
      --registry=https://registry.npmjs.org \
      --replace-registry-host=always
    NPM_CONFIG_CACHE="${npm_cache_dir}" npm test
    NPM_CONFIG_CACHE="${npm_cache_dir}" npm run build
    cp manifest.json main.js styles.css versions.json \
      "${obsidian_stage}/obsidian-goclaw/"
  )
  printf '%s\n' \
    obsidian-goclaw/main.js \
    obsidian-goclaw/manifest.json \
    obsidian-goclaw/styles.css \
    obsidian-goclaw/versions.json \
    > "${obsidian_expected}"
  goclaw_create_normalized_archive \
    "${obsidian_archive}" \
    "${obsidian_stage}" \
    "${obsidian_expected}" \
    "" \
    "${source_date_epoch}"
  goclaw_validate_archive_contract \
    "${obsidian_archive}" \
    "${obsidian_expected}" \
    "obsidian-goclaw" \
    "${work_dir}/obsidian-archive-check"
fi

if [[ "${source_only}" == "0" ]]; then
  for release_arch in amd64 arm64; do
    binary_stage="${work_dir}/linux-${release_arch}-package"
    binary_expected="${work_dir}/linux-${release_arch}-expected.txt"
    binary_check="${work_dir}/linux-${release_arch}-archive-check"
    binary_archive="$(
      printf '%s/goclaw-team-runtime-linux-%s-%s.tar.gz' \
        "${publish_stage}" "${release_arch}" "${release_version}"
    )"
    mkdir -p \
      "${binary_stage}/scripts" \
      "${binary_stage}/deploy/systemd" \
      "${binary_stage}/deploy/wsl2" \
      "${binary_stage}/deploy/lima"
    install -m 0755 \
      "${work_dir}/bin/goclaw-linux-${release_arch}" \
      "${binary_stage}/goclaw"
    install -m 0755 \
      "${repo_dir}/scripts/verify-sandbox-bwrap.sh" \
      "${binary_stage}/scripts/verify-sandbox-bwrap.sh"
    install -m 0644 \
      "${repo_dir}/deploy/runner.env.example" \
      "${binary_stage}/deploy/runner.env.example"
    install -m 0644 \
      "${repo_dir}/deploy/systemd/goclaw-runner.service.example" \
      "${binary_stage}/deploy/systemd/goclaw-runner.service.example"
    install -m 0644 \
      "${repo_dir}/deploy/wsl2/README_CN.md" \
      "${binary_stage}/deploy/wsl2/README_CN.md"
    install -m 0644 \
      "${repo_dir}/deploy/wsl2/runner.env.example" \
      "${binary_stage}/deploy/wsl2/runner.env.example"
    install -m 0644 \
      "${repo_dir}/deploy/wsl2/wsl.conf.example" \
      "${binary_stage}/deploy/wsl2/wsl.conf.example"
    install -m 0644 \
      "${repo_dir}/deploy/lima/README_CN.md" \
      "${binary_stage}/deploy/lima/README_CN.md"
    install -m 0644 \
      "${repo_dir}/deploy/lima/goclaw-runner.yaml.example" \
      "${binary_stage}/deploy/lima/goclaw-runner.yaml.example"
    install -m 0644 \
      "${repo_dir}/deploy/lima/runner.env.example" \
      "${binary_stage}/deploy/lima/runner.env.example"

    printf '%s\n' \
      deploy/lima/README_CN.md \
      deploy/lima/goclaw-runner.yaml.example \
      deploy/lima/runner.env.example \
      deploy/runner.env.example \
      deploy/systemd/goclaw-runner.service.example \
      deploy/wsl2/README_CN.md \
      deploy/wsl2/runner.env.example \
      deploy/wsl2/wsl.conf.example \
      goclaw \
      scripts/verify-sandbox-bwrap.sh \
      > "${binary_expected}"
    goclaw_create_normalized_archive \
      "${binary_archive}" \
      "${binary_stage}" \
      "${binary_expected}" \
      "" \
      "${source_date_epoch}"
    goclaw_validate_archive_contract \
      "${binary_archive}" \
      "${binary_expected}" \
      "-" \
      "${binary_check}"

    [[ -x "${binary_check}/goclaw" ]] ||
      fail_release "Linux package lost the goclaw executable mode"
    [[ -x "${binary_check}/scripts/verify-sandbox-bwrap.sh" ]] ||
      fail_release "Linux package sandbox wrapper is not executable"
    go version -m "${binary_check}/goclaw" \
      > "${binary_check}/buildinfo.txt"
    grep -Fq "GOARCH=${release_arch}" "${binary_check}/buildinfo.txt" ||
      fail_release "packaged Linux binary architecture mismatch"
  done
fi

(
  cd "${repo_dir}"
  source_paths=(
    .editorconfig .gitattributes .github .gitignore .golangci.yml
    .goreleaser.yaml .tool-versions
    AGENTS.md CHANGE-HANDOFF.md CHANGELOG.md Dockerfile LICENSE Makefile
    README.md THIRD_PARTY_NOTICES.md config.json.example go.mod go.sum main.go
    signal_default.go signal_windows.go
    agent bus channels cli config cron deploy docker docs errors gateway
    governance harness integration internal memory orchestratorlite ouroboros
    pairing plugins providers scripts session src-tauri teamcontrol third_party
    ui workstation
  )
  source_manifest="${work_dir}/source-files.txt"
  source_expected="${work_dir}/source-expected.txt"
  source_findings="${work_dir}/source-findings.txt"
  source_archive="$(
    printf '%s/goclaw-team-runtime-source-%s.tar.gz' \
      "${publish_stage}" "${release_version}"
  )"
  : > "${source_manifest}"
  : > "${source_findings}"

  for source_path in "${source_paths[@]}"; do
    [[ -e "${source_path}" || -L "${source_path}" ]] ||
      fail_release "allowlisted source path does not exist: ${source_path}"
    while IFS= read -r -d '' candidate; do
      candidate="${candidate#./}"
      if [[ "${candidate}" == *$'\n'* || "${candidate}" == *$'\r'* ]]; then
        fail_release "source filename contains a line break"
      fi
      if source_path_is_forbidden "${candidate}"; then
        continue
      fi
      if [[ -L "${candidate}" ]]; then
        fail_release "allowlisted source contains a symlink: ${candidate}"
      fi
      [[ -f "${candidate}" ]] ||
        fail_release "allowlisted source is not a regular file: ${candidate}"
      printf '%s\n' "${candidate}" >> "${source_manifest}"
    done < <(
      find "${source_path}" \
        \( -type d \( \
          -name .git -o -name .agents -o -name .codex -o -name .claude -o \
          -name .idea -o -name .vscode -o -name node_modules -o -name dist -o \
          -name build -o -name out -o -name target -o -name coverage -o \
          -name .cache -o -name .next -o -name .turbo \
        \) -prune \) -o \
        \( -type f -o -type l \) -print0
    )
  done

  LC_ALL=C sort -u -o "${source_manifest}" "${source_manifest}"
  [[ -s "${source_manifest}" ]] ||
    fail_release "source manifest is empty"
  while IFS= read -r source_file; do
    validate_source_file \
      "${source_file}" "${source_file}" "${source_findings}"
    printf 'goclaw-%s/%s\n' \
      "${release_version}" "${source_file}"
  done < "${source_manifest}" > "${source_expected}"
  if [[ -s "${source_findings}" ]]; then
    echo "Refusing to package allowlisted source files:" >&2
    LC_ALL=C sort -u "${source_findings}" >&2
    exit 1
  fi

  goclaw_create_normalized_archive \
    "${source_archive}" \
    "${repo_dir}" \
    "${source_manifest}" \
    "goclaw-${release_version}" \
    "${source_date_epoch}"
  goclaw_validate_archive_contract \
    "${source_archive}" \
    "${source_expected}" \
    "goclaw-${release_version}" \
    "${work_dir}/source-archive-check"

  while IFS= read -r source_file; do
    validate_source_file \
      "${work_dir}/source-archive-check/goclaw-${release_version}/${source_file}" \
      "goclaw-${release_version}/${source_file}" \
      "${source_findings}"
  done < "${source_manifest}"
  if [[ -s "${source_findings}" ]]; then
    echo "Refusing recovered source archive:" >&2
    LC_ALL=C sort -u "${source_findings}" >&2
    exit 1
  fi
)

if ! git -C "${repo_dir}" diff --quiet -- gateway/ui_dist; then
  fail_release "Web build does not match tracked gateway/ui_dist"
fi
if [[ "${include_obsidian_plugin}" == "1" ]] &&
  ! git -C "${repo_dir}" diff --quiet -- plugins/obsidian-goclaw/main.js; then
  fail_release "Obsidian build does not match tracked main.js"
fi
if [[ -n "$(git -C "${repo_dir}" status --porcelain --untracked-files=all)" ]]; then
  fail_release "release verification changed the Git worktree"
fi

include_obsidian_json=false
if [[ "${source_only}" == "0" &&
  "${include_obsidian_plugin}" == "1" ]]; then
  include_obsidian_json=true
fi
release_manifest="${publish_stage}/release-manifest-${release_version}.json"
cat > "${release_manifest}" <<EOF
{
  "schema": "goclaw.release/v1",
  "runtime_version": "${release_version}",
  "source_commit": "${release_commit}",
  "source_tree": "${release_tree}",
  "source_date_epoch": ${source_date_epoch},
  "toolchain": {
    "go": "${expected_go_version}",
    "node": "${expected_node_version}",
    "npm": "${expected_npm_version}"
  },
  "components": {
    "obsidian_goclaw": {
      "version": "${obsidian_version}",
      "included": ${include_obsidian_json}
    }
  }
}
EOF

(
  cd "${publish_stage}"
  mapfile -t checksum_artifacts < <(
    find . -maxdepth 1 -type f ! -name 'SHA256SUMS-*' \
      -printf '%f\n' | LC_ALL=C sort
  )
  [[ "${#checksum_artifacts[@]}" -gt 0 ]] ||
    fail_release "no release artifacts were produced"
  LC_ALL=C sha256sum "${checksum_artifacts[@]}" \
    > "SHA256SUMS-${release_version}.txt"
  sha256sum -c "SHA256SUMS-${release_version}.txt"
  chmod 0644 ./*
)

final_release_dir="${release_root}/${release_version}"
if [[ -e "${final_release_dir}" ]]; then
  [[ -d "${final_release_dir}" && ! -L "${final_release_dir}" ]] ||
    fail_release "existing release target is not a regular directory"
  if ! diff --recursive --brief --no-dereference \
    "${publish_stage}" "${final_release_dir}"; then
    fail_release \
      "release ${release_version} already exists with different content"
  fi
  if find "${final_release_dir}" -mindepth 1 \
    \( ! -type f -o ! -perm 0644 \) -print -quit | grep -q .; then
    fail_release "existing release directory has an invalid type or mode"
  fi
  echo "Verified identical existing release:"
else
  mv -- "${publish_stage}" "${final_release_dir}"
  echo "Published atomically:"
fi

(
  cd "${final_release_dir}"
  sha256sum -c "SHA256SUMS-${release_version}.txt"
)
find "${final_release_dir}" -maxdepth 1 -type f -printf '  %p\n' |
  LC_ALL=C sort

cleanup_release_stage
stage_dir=""
