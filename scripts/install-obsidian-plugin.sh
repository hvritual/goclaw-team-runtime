#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 1 ]]; then
  echo "Usage: $0 /absolute/path/to/ObsidianVault" >&2
  exit 2
fi

repo_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
vault_dir="$1"
plugin_dir="${vault_dir}/.obsidian/plugins/goclaw-project-console"

if [[ ! -d "${vault_dir}/.obsidian" ]]; then
  echo "Not an Obsidian vault: ${vault_dir}" >&2
  exit 1
fi

(
  cd "${repo_dir}/plugins/obsidian-goclaw"
  npm ci
  npm run build
)

mkdir -p "${plugin_dir}"
cp \
  "${repo_dir}/plugins/obsidian-goclaw/manifest.json" \
  "${repo_dir}/plugins/obsidian-goclaw/main.js" \
  "${repo_dir}/plugins/obsidian-goclaw/styles.css" \
  "${repo_dir}/plugins/obsidian-goclaw/versions.json" \
  "${plugin_dir}/"

echo "Installed GoClaw Project Console to ${plugin_dir}"
echo "Restart Obsidian, then enable it under Settings → Community plugins."
