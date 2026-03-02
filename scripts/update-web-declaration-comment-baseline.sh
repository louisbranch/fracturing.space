#!/usr/bin/env bash

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

{
  echo "# Web declaration comment baseline for staged rollout."
  echo "# To ratchet this file after adding comments, run:"
  echo "#   make web-doc-baseline-update"
  go run ./internal/tools/webdoccheck -mode declarations -write-baseline
} > docs/reference/web-declaration-comment-baseline.txt
