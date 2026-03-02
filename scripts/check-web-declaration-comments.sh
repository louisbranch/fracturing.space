#!/usr/bin/env bash

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

BASELINE="${WEB_DECLARATION_COMMENT_BASELINE:-docs/reference/web-declaration-comment-baseline.txt}"

go run ./internal/tools/webdoccheck -mode declarations -baseline "$BASELINE"
