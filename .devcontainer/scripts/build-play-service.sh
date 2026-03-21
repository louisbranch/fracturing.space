#!/usr/bin/env bash

set -euo pipefail

ui_root="internal/services/play/ui"
ui_manifest="$ui_root/dist/manifest.json"

if [ -z "${FRACTURING_SPACE_PLAY_UI_DEV_SERVER_URL:-}" ] && [ ! -f "$ui_manifest" ]; then
  echo "[build-play-service] play UI manifest is missing at $ui_manifest; run 'make play-ui-dist' or set FRACTURING_SPACE_PLAY_UI_DEV_SERVER_URL" >&2
  exit 1
fi

go build -o .tmp/dev/bin/play ./cmd/play
