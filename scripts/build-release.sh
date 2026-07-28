#!/usr/bin/env bash
set -euo pipefail

repo_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
dist_dir="${repo_dir}/dist"
release_version="0.8.0-pilot.1"
npm_cache_dir="${GOCLAW_NPM_CACHE:-${TMPDIR:-/tmp}/goclaw-npm-cache}"
include_obsidian_plugin="${INCLUDE_OBSIDIAN_PLUGIN:-0}"
source_only="${SOURCE_ONLY:-0}"

fail_release() {
  echo "release validation failed: $*" >&2
  exit 1
}

# Keep the source release intentionally narrower than the working tree. This
# predicate is applied while building the manifest and again to every archive
# member so a future allowlist expansion cannot accidentally ship local state
# or build output.
source_path_is_forbidden() {
  local path="${1#./}"
  local base="${path##*/}"

  case "/${path}/" in
    */.git/* | */.agents/* | */.codex/* | */.claude/* | */.idea/* | */.vscode/* | \
      */.env/* | */.env.*/* | */.dSYM/* | \
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
      *.db | *.db-* | *.sqlite | *.sqlite-* | *.sqlite3 | *.sqlite3-* | *.log | \
      *.tar | *.tar.gz | *.tgz | *.zip | *.7z | *.rar | *.gz | *.bz2 | *.xz | \
      *.test | *.exe | *.dll | *.dylib | *.so | *.so.* | *.a | *.o | *.obj | \
      *.wasm | *.bin | *.class | *.jar | *.war | *.pyc | *.tmp | *.swp | \
      *.bak | *~ | .eslintcache)
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

      # Environment lookups, templates, and conspicuous documentation/test
      # values are references, not raw credentials. In particular, task- IDs
      # used throughout the development docs must never be classified as a
      # token merely because the setting name ends in TOKEN.
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
    printf '%s (common executable/object binary)\n' "${display_path}" >> "${findings_file}"
    return
  fi
  if file_has_credential_material "${file}"; then
    printf '%s (credential-like material)\n' "${display_path}" >> "${findings_file}"
  fi
}

validate_source_archive() {
  local archive="$1"
  local expected_count="$2"
  local validation_root="$3"
  local member_list="${validation_root}/members.txt"
  local type_list="${validation_root}/types.txt"
  local extract_root="${validation_root}/extracted"
  local findings_file="${validation_root}/findings.txt"
  local member member_path type actual_count

  mkdir -p "${validation_root}" "${extract_root}"
  : > "${findings_file}"

  tar -tzf "${archive}" > "${member_list}" ||
    fail_release "cannot list source archive ${archive}"
  tar -tvzf "${archive}" > "${type_list}" ||
    fail_release "cannot inspect source archive entry types"
  [[ -s "${member_list}" ]] || fail_release "source archive is empty"

  while IFS= read -r member; do
    [[ -n "${member}" ]] || fail_release "source archive contains an empty member name"
    member_path="${member%/}"
    case "${member_path}" in
      /* | .. | ../* | */../* | */.. | *\\*)
        fail_release "unsafe source archive member: ${member}"
        ;;
    esac
    case "${member_path}" in
      "goclaw-${release_version}" | "goclaw-${release_version}/"*)
        ;;
      *)
        fail_release "source archive member is outside the release root: ${member}"
        ;;
    esac
    if source_path_is_forbidden "${member_path}"; then
      fail_release "forbidden source archive member: ${member}"
    fi
  done < "${member_list}"

  while IFS= read -r member; do
    [[ -n "${member}" ]] || fail_release "cannot parse source archive entry type"
    type="${member:0:1}"
    case "${type}" in
      - | d)
        ;;
      l | h)
        fail_release "source archive contains a link entry"
        ;;
      *)
        fail_release "source archive contains unsupported entry type ${type}"
        ;;
    esac
  done < "${type_list}"

  actual_count="$(wc -l < "${member_list}")"
  actual_count="${actual_count//[[:space:]]/}"
  [[ "${actual_count}" == "${expected_count}" ]] ||
    fail_release "source archive member count changed (${actual_count}, expected ${expected_count})"

  tar \
    --extract \
    --gzip \
    --file "${archive}" \
    --directory "${extract_root}" \
    --no-same-owner \
    --no-same-permissions ||
    fail_release "cannot recover source archive for credential validation"

  if find "${extract_root}" -type l -print -quit | grep -q .; then
    fail_release "recovered source archive contains a symlink"
  fi

  while IFS= read -r -d '' member; do
    member_path="${member#"${extract_root}/"}"
    validate_source_file "${member}" "${member_path}" "${findings_file}"
  done < <(find "${extract_root}" -type f -print0)

  if [[ -s "${findings_file}" ]]; then
    echo "Refusing source archive after recoverable-content validation:" >&2
    LC_ALL=C sort -u "${findings_file}" >&2
    exit 1
  fi
}

mkdir -p "${dist_dir}"
stage_dir="$(mktemp -d "${dist_dir}/.release-${release_version}.XXXXXX")"
cleanup_release_stage() {
  if [[ -n "${stage_dir:-}" && -d "${stage_dir}" ]]; then
    rm -rf -- "${stage_dir}"
  fi
}
trap cleanup_release_stage EXIT

[[ "${include_obsidian_plugin}" == "0" || "${include_obsidian_plugin}" == "1" ]] ||
  fail_release "INCLUDE_OBSIDIAN_PLUGIN must be 0 or 1"
[[ "${source_only}" == "0" || "${source_only}" == "1" ]] ||
  fail_release "SOURCE_ONLY must be 0 or 1"

(
  cd "${repo_dir}/ui"
  NPM_CONFIG_CACHE="${npm_cache_dir}" npm ci \
    --registry=https://registry.npmjs.org \
    --replace-registry-host=always
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
    go test -count=1 ./memory ./memory/catalog ./governance ./ouroboros ./orchestratorlite ./harness ./teamcontrol ./workstation ./providers ./gateway ./agent ./agent/tools ./config ./cli ./cli/commands ./internal/start
    for release_arch in amd64 arm64; do
      CGO_ENABLED=0 GOOS=linux GOARCH="${release_arch}" \
        go build -buildvcs=false -trimpath \
        -ldflags="-s -w -X main.Version=${release_version}" \
        -o "${dist_dir}/goclaw-linux-${release_arch}" .
      go version -m "${dist_dir}/goclaw-linux-${release_arch}" \
        > "${stage_dir}/linux-${release_arch}.buildinfo"
      grep -Fq "GOOS=linux" "${stage_dir}/linux-${release_arch}.buildinfo" ||
        fail_release "linux/${release_arch} binary reports the wrong GOOS"
      grep -Fq "GOARCH=${release_arch}" "${stage_dir}/linux-${release_arch}.buildinfo" ||
        fail_release "linux/${release_arch} binary reports the wrong GOARCH"
    done
    install -m 0755 "${dist_dir}/goclaw-linux-amd64" "${dist_dir}/goclaw"

    # Native Windows/macOS are control-CLI-only targets during the pilot.
    # Compile them to prove platform helpers remain portable, but do not ship
    # them as execution Runner packages.
    for control_target in darwin/amd64 darwin/arm64 windows/amd64 windows/arm64; do
      control_os="${control_target%/*}"
      control_arch="${control_target#*/}"
      control_suffix=""
      if [[ "${control_os}" == "windows" ]]; then
        control_suffix=".exe"
      fi
      CGO_ENABLED=0 GOOS="${control_os}" GOARCH="${control_arch}" \
        go build -buildvcs=false -trimpath \
        -ldflags="-s -w -X main.Version=${release_version}" \
        -o "${stage_dir}/control-${control_os}-${control_arch}${control_suffix}" .
    done
  )
fi

if [[ "${source_only}" == "0" && "${include_obsidian_plugin}" == "1" ]]; then
  obsidian_stage="${stage_dir}/obsidian-goclaw"
  mkdir -p "${obsidian_stage}"
  (
    cd "${repo_dir}/plugins/obsidian-goclaw"
    NPM_CONFIG_CACHE="${npm_cache_dir}" npm ci \
      --registry=https://registry.npmjs.org \
      --replace-registry-host=always
    NPM_CONFIG_CACHE="${npm_cache_dir}" npm test
    NPM_CONFIG_CACHE="${npm_cache_dir}" npm run build
    cp manifest.json main.js styles.css versions.json "${obsidian_stage}/"
  )
  (
    cd "${stage_dir}"
    tar -czf "obsidian-goclaw-plugin-${release_version}.tar.gz" obsidian-goclaw
  )
  rm -rf -- "${obsidian_stage}"
fi

if [[ "${source_only}" == "0" ]]; then
  for release_arch in amd64 arm64; do
    (
    binary_stage="${stage_dir}/linux-${release_arch}-package"
    binary_check="${stage_dir}/linux-${release_arch}-package-check"
    binary_archive="${stage_dir}/goclaw-team-runtime-linux-${release_arch}-${release_version}.tar.gz"
    mkdir -p \
      "${binary_stage}/scripts" \
      "${binary_stage}/deploy/systemd" \
      "${binary_stage}/deploy/wsl2" \
      "${binary_stage}/deploy/lima" \
      "${binary_check}"
    install -m 0755 \
      "${dist_dir}/goclaw-linux-${release_arch}" \
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
    cp -R "${repo_dir}/deploy/wsl2/." "${binary_stage}/deploy/wsl2/"
    cp -R "${repo_dir}/deploy/lima/." "${binary_stage}/deploy/lima/"

    (
      cd "${binary_stage}"
      tar -czf "${binary_archive}" \
        goclaw \
        scripts/verify-sandbox-bwrap.sh \
        deploy/runner.env.example \
        deploy/systemd/goclaw-runner.service.example \
        deploy/wsl2 \
        deploy/lima
    )

    tar -xzf "${binary_archive}" -C "${binary_check}"
    [[ -x "${binary_check}/goclaw" ]] ||
      fail_release "Linux package lost the goclaw executable mode"
    [[ -x "${binary_check}/scripts/verify-sandbox-bwrap.sh" ]] ||
      fail_release "Linux package sandbox wrapper is not executable"
    [[ -f "${binary_check}/deploy/runner.env.example" ]] ||
      fail_release "Linux package is missing runner.env.example"
    [[ -f "${binary_check}/deploy/systemd/goclaw-runner.service.example" ]] ||
      fail_release "Linux package is missing the runner systemd example"
    [[ -f "${binary_check}/deploy/wsl2/wsl.conf.example" ]] ||
      fail_release "Linux package is missing the WSL2 profile"
    [[ -f "${binary_check}/deploy/lima/goclaw-runner.yaml.example" ]] ||
      fail_release "Linux package is missing the Lima profile"
    go version -m "${binary_check}/goclaw" \
      > "${binary_check}/buildinfo.txt"
    grep -Fq "GOARCH=${release_arch}" "${binary_check}/buildinfo.txt" ||
      fail_release "packaged Linux binary architecture mismatch"
    )
  done
fi

(
  cd "${repo_dir}"
  source_paths=(
    .editorconfig .gitattributes .github .gitignore .golangci.yml .goreleaser.yaml
    AGENTS.md CHANGE-HANDOFF.md CHANGELOG.md Dockerfile LICENSE Makefile README.md
    THIRD_PARTY_NOTICES.md config.json.example go.mod go.sum main.go
    signal_default.go signal_windows.go
    agent bus channels cli config cron deploy docker docs errors gateway governance
    harness integration internal memory orchestratorlite ouroboros pairing plugins
    providers scripts session src-tauri teamcontrol third_party ui workstation
  )
  source_manifest_unsorted="${stage_dir}/source-files.unsorted.nul"
  source_manifest="${stage_dir}/source-files.nul"
  source_findings="${stage_dir}/source-findings.txt"
  : > "${source_manifest_unsorted}"
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
      printf '%s\0' "${candidate}" >> "${source_manifest_unsorted}"
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

  LC_ALL=C sort -zu "${source_manifest_unsorted}" > "${source_manifest}"
  source_count="$(LC_ALL=C tr '\0' '\n' < "${source_manifest}" | wc -l)"
  source_count="${source_count//[[:space:]]/}"
  [[ "${source_count}" =~ ^[1-9][0-9]*$ ]] ||
    fail_release "source manifest is empty"

  while IFS= read -r -d '' source_file; do
    validate_source_file "${source_file}" "${source_file}" "${source_findings}"
  done < "${source_manifest}"
  if [[ -s "${source_findings}" ]]; then
    echo "Refusing to package allowlisted source files:" >&2
    LC_ALL=C sort -u "${source_findings}" >&2
    exit 1
  fi

  tar \
    --create \
    --gzip \
    --file="${stage_dir}/goclaw-team-runtime-source-${release_version}.tar.gz" \
    --null \
    --no-recursion \
    --hard-dereference \
    --transform="s#^#goclaw-${release_version}/#" \
    --files-from="${source_manifest}"

  validate_source_archive \
    "${stage_dir}/goclaw-team-runtime-source-${release_version}.tar.gz" \
    "${source_count}" \
    "${stage_dir}/source-archive-validation"
)

(
  cd "${stage_dir}"
  checksum_artifacts=(
    "goclaw-team-runtime-source-${release_version}.tar.gz"
  )
  if [[ "${source_only}" == "0" ]]; then
    checksum_artifacts=(
      "goclaw-team-runtime-linux-amd64-${release_version}.tar.gz"
      "goclaw-team-runtime-linux-arm64-${release_version}.tar.gz"
      "${checksum_artifacts[@]}"
    )
  fi
  if [[ "${source_only}" == "0" && "${include_obsidian_plugin}" == "1" ]]; then
    checksum_artifacts+=("obsidian-goclaw-plugin-${release_version}.tar.gz")
  fi
  sha256sum "${checksum_artifacts[@]}" > "SHA256SUMS-${release_version}.txt"
)

release_artifacts=(
  "goclaw-team-runtime-source-${release_version}.tar.gz"
  "SHA256SUMS-${release_version}.txt"
)
if [[ "${source_only}" == "0" ]]; then
  release_artifacts=(
    "goclaw-team-runtime-linux-amd64-${release_version}.tar.gz"
    "goclaw-team-runtime-linux-arm64-${release_version}.tar.gz"
    "${release_artifacts[@]}"
  )
fi
if [[ "${source_only}" == "0" && "${include_obsidian_plugin}" == "1" ]]; then
  release_artifacts+=("obsidian-goclaw-plugin-${release_version}.tar.gz")
fi
for artifact in "${release_artifacts[@]}"; do
  mv -f "${stage_dir}/${artifact}" "${dist_dir}/${artifact}"
done
cleanup_release_stage
stage_dir=""

echo "Built:"
if [[ "${source_only}" == "0" ]]; then
  echo "  ${dist_dir}/goclaw-linux-amd64"
  echo "  ${dist_dir}/goclaw-linux-arm64"
  echo "  ${dist_dir}/goclaw-team-runtime-linux-amd64-${release_version}.tar.gz"
  echo "  ${dist_dir}/goclaw-team-runtime-linux-arm64-${release_version}.tar.gz"
fi
echo "  ${dist_dir}/goclaw-team-runtime-source-${release_version}.tar.gz"
if [[ "${source_only}" == "0" && "${include_obsidian_plugin}" == "1" ]]; then
  echo "  ${dist_dir}/obsidian-goclaw-plugin-${release_version}.tar.gz"
elif [[ "${source_only}" == "0" ]]; then
  echo "  Obsidian adapter skipped (set INCLUDE_OBSIDIAN_PLUGIN=1 to package it)"
fi
echo "  ${dist_dir}/SHA256SUMS-${release_version}.txt"
