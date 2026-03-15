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
  if command -v npm >/dev/null 2>&1; then
    npm --prefix "$ui_root" run build
  elif [ -f "$ui_manifest" ]; then
    echo "[build-play-service] npm not found; using checked-in play UI bundle at $ui_manifest" >&2
  else
    echo "[build-play-service] npm not found and $ui_manifest is missing; cannot build embedded play UI" >&2
    exit 1
  fi
fi

go build -o .tmp/dev/bin/play ./cmd/play
