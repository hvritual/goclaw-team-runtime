#!/usr/bin/env bash
set -euo pipefail

base_ref="${1:-}"
if [[ -z "${base_ref}" ]]; then
  echo "usage: $0 BASE_COMMIT_OR_REF" >&2
  exit 2
fi

git rev-parse --verify "${base_ref}^{commit}" >/dev/null
mapfile -t commits < <(git rev-list --no-merges "${base_ref}..HEAD")
if [[ "${#commits[@]}" -eq 0 ]]; then
  echo "No non-merge commits to inspect."
  exit 0
fi

required_trailers=(
  "Task-ID"
  "Project-ID"
  "Task-Revision"
  "Work-Item"
)
failed=0
for commit in "${commits[@]}"; do
  message="$(git show -s --format=%B "${commit}")"
  for trailer in "${required_trailers[@]}"; do
    if ! grep -Eq "^${trailer}: .+" <<<"${message}"; then
      echo "${commit}: missing ${trailer} trailer" >&2
      failed=1
    fi
  done
done

if [[ "${failed}" -ne 0 ]]; then
  echo "Commit traceability check failed." >&2
  exit 1
fi
echo "Commit traceability check passed for ${#commits[@]} commit(s)."
