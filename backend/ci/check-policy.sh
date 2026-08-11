#!/bin/sh
set -eu

repo_root=$(git rev-parse --show-toplevel)
base_ref=${BASE_REF:-codex/multica-six-domain-baseline}

cd "$repo_root"

if ! git rev-parse --verify "$base_ref^{commit}" >/dev/null 2>&1; then
  echo "policy-check: base ref $base_ref is unavailable" >&2
  exit 2
fi

changed=$(git diff --name-only "$base_ref...HEAD")

if printf '%s\n' "$changed" | grep -E '^server/' >/dev/null 2>&1; then
  echo "policy-check: server/** is permanently read-only" >&2
  printf '%s\n' "$changed" | grep -E '^server/' >&2
  exit 1
fi

if printf '%s\n' "$changed" | grep -Ev '^(backend/|\.github/workflows/backend\.yml$|$)' >/dev/null 2>&1; then
  echo "policy-check: P0P2-BACKEND-PLATFORM-001 v2 permits only backend/** and .github/workflows/backend.yml" >&2
  printf '%s\n' "$changed" | grep -Ev '^(backend/|\.github/workflows/backend\.yml$|$)' >&2
  exit 1
fi

if grep -R --include='*.go' -n 'github.com/hvritual/workspace/server\|/server/internal/' backend >/dev/null 2>&1; then
  echo "policy-check: canonical backend imports legacy server code" >&2
  exit 1
fi

echo "policy-check: backend path and dependency boundaries passed"
