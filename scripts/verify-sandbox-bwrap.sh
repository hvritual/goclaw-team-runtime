#!/usr/bin/env bash
set -euo pipefail

readonly safe_path="/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
readonly bwrap_bin="/usr/bin/bwrap"
readonly stat_bin="/usr/bin/stat"

validate_bwrap() {
  local owner mode
  if [[ ! -f "${bwrap_bin}" || -L "${bwrap_bin}" || ! -x "${bwrap_bin}" ]]; then
    echo "fixed bubblewrap executable ${bwrap_bin} is required" >&2
    return 127
  fi
  owner="$("${stat_bin}" -c '%u' -- "${bwrap_bin}")"
  mode="$("${stat_bin}" -c '%a' -- "${bwrap_bin}")"
  if [[ "${owner}" != "0" || $((8#${mode} & 8#022)) -ne 0 ]]; then
    echo "${bwrap_bin} must be root-owned and not writable by group or others" >&2
    return 126
  fi
}

if [[ "$#" -eq 1 && "$1" == "--goclaw-doctor" ]]; then
  validate_bwrap
  "${bwrap_bin}" \
    --die-with-parent \
    --new-session \
    --unshare-user \
    --unshare-pid \
    --unshare-net \
    --unshare-ipc \
    --unshare-uts \
    --proc /proc \
    --dev /dev \
    --tmpfs /tmp \
    --clearenv \
    --setenv PATH "${safe_path}" \
    -- /bin/true
  echo "goclaw-verifier/linux-bwrap-v1"
  exit 0
fi

if [[ "$#" -lt 4 ]]; then
  echo "usage: $0 WORKTREE SANDBOX_HOME -- COMMAND [ARG...]" >&2
  exit 2
fi

worktree="$1"
sandbox_home="$2"
shift 2
if [[ "$1" != "--" ]]; then
  echo "verification sandbox requires -- before the command" >&2
  exit 2
fi
shift
if [[ "$#" -lt 1 || -z "$1" ]]; then
  echo "verification command is required" >&2
  exit 2
fi
if [[ "${worktree}" != /* || "${sandbox_home}" != /* ]]; then
  echo "worktree and sandbox home must be absolute paths" >&2
  exit 2
fi
if [[ ! -d "${worktree}" || ! -d "${sandbox_home}" ]]; then
  echo "worktree and sandbox home must already exist" >&2
  exit 2
fi
validate_bwrap

# Project tests are arbitrary code. Run them without network, host homes,
# credential sockets, or writable host paths. Toolchains remain readable from
# the host; the task worktree and an ephemeral HOME are the only writable
# mounts. Teams needing language caches should provide a reviewed wrapper that
# adds narrowly scoped cache mounts instead of weakening this baseline.
exec "${bwrap_bin}" \
  --die-with-parent \
  --new-session \
  --unshare-user \
  --unshare-pid \
  --unshare-net \
  --unshare-ipc \
  --unshare-uts \
  --ro-bind / / \
  --proc /proc \
  --dev /dev \
  --tmpfs /home \
  --tmpfs /root \
  --tmpfs /run \
  --tmpfs /tmp \
  --bind "${worktree}" /workspace \
  --bind "${sandbox_home}" /sandbox-home \
  --chdir /workspace \
  --clearenv \
  --setenv HOME /sandbox-home \
  --setenv XDG_CACHE_HOME /sandbox-home/.cache \
  --setenv XDG_CONFIG_HOME /sandbox-home/.config \
  --setenv XDG_RUNTIME_DIR /sandbox-home/.runtime \
  --setenv TMPDIR /tmp \
  --setenv TMP /tmp \
  --setenv TEMP /tmp \
  --setenv PATH "${safe_path}" \
  --setenv LANG "${LANG:-C.UTF-8}" \
  --setenv GIT_TERMINAL_PROMPT 0 \
  -- "$@"
