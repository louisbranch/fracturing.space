#!/usr/bin/env bash

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
tmp_root="$repo_root/.tmp/play-ui-dist"
export_dir="$tmp_root/export"
dist_root="$repo_root/internal/services/play/ui/dist"

if ! command -v docker >/dev/null 2>&1; then
  echo "[refresh-play-ui-dist] docker is required" >&2
  exit 1
fi

rm -rf "$tmp_root"
mkdir -p "$export_dir"

echo "[refresh-play-ui-dist] exporting Docker/Linux play UI dist"
docker buildx build \
  --target export-play-ui-dist \
  --output "type=local,dest=$export_dir" \
  "$repo_root" >/dev/null

if [ ! -f "$export_dir/dist/manifest.json" ]; then
  echo "[refresh-play-ui-dist] exported dist is missing manifest.json" >&2
  exit 1
fi

rm -rf "$dist_root"
mkdir -p "$dist_root"
cp -R "$export_dir/dist/." "$dist_root/"

echo "[refresh-play-ui-dist] refreshed $dist_root"
