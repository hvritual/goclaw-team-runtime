#!/bin/sh
set -eu

repo_root=$(git rev-parse --show-toplevel)
base_ref=${BASE_REF:-codex/multica-six-domain-baseline}

cd "$repo_root"

case "$base_ref" in
  0000000000000000000000000000000000000000)
    base_ref=$(git rev-parse HEAD^)
    ;;
esac

if ! git rev-parse --verify "$base_ref^{commit}" >/dev/null 2>&1; then
  echo "policy-check: base ref $base_ref is unavailable" >&2
  exit 2
fi

changed=$(git diff --name-only --diff-filter=ACDMRTUXB "$base_ref...HEAD")
server_paths=$(printf '%s\n' "$changed" | grep -E '^server(/|$)' || true)

if [ -n "$server_paths" ]; then
  echo "policy-check: server/** is permanently read-only; no plan-level exception exists" >&2
  printf '%s\n' "$server_paths" >&2
  exit 1
fi

# This fast preflight scans production Go sources only. The authoritative
# parser-based policy check runs later in `make check` and covers test imports
# without mistaking negative-test fixture strings for real imports.
legacy_imports=$(grep -R --include='*.go' --exclude='*_test.go' -n 'github.com/hvritual/workspace/server\|/server/internal/' backend || true)
if [ -n "$legacy_imports" ]; then
  echo "policy-check: canonical backend imports legacy server code" >&2
  printf '%s\n' "$legacy_imports" >&2
  exit 1
fi

echo "policy-check: server boundary and canonical backend dependency boundary passed"
