#!/usr/bin/env bash

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

export BASE_URL="${BASE_URL:-http://localhost:8082}"
export ARTIFACT_ROOT="${ARTIFACT_ROOT:-artifacts/playwright}"

exec "$ROOT/scripts/playwright-run-spec.sh" "$ROOT/docs/specs/admin-smoke.md"
