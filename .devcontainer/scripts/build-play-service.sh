#!/usr/bin/env bash

set -euo pipefail

ui_root="internal/services/play/ui"
ui_manifest="$ui_root/dist/manifest.json"

should_build_ui() {
  if [ -n "${FRACTURING_SPACE_PLAY_UI_DEV_SERVER_URL:-}" ]; then
    return 1
  fi

  if [ ! -f "$ui_manifest" ]; then
    return 0
  fi

  find "$ui_root" \
    \( -path "$ui_root/dist" -o -path "$ui_root/node_modules" \) -prune \
    -o -type f -newer "$ui_manifest" -print -quit | grep -q .
}

if should_build_ui; then
  npm --prefix "$ui_root" run build
fi

go build -o .tmp/dev/bin/play ./cmd/play
