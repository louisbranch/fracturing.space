#!/usr/bin/env bash

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

SPEC_FILE="${1:-}"
if [[ -z "$SPEC_FILE" ]]; then
  echo "Usage: scripts/playwright-run-spec.sh <spec.md>" >&2
  exit 1
fi

if [[ ! -f "$SPEC_FILE" ]]; then
  echo "Spec not found: $SPEC_FILE" >&2
  exit 1
fi

BASE_URL="${BASE_URL:-http://localhost:8082}"
ARTIFACT_ROOT="${ARTIFACT_ROOT:-artifacts/playwright}"
PLAYWRIGHT_CLI_PKG_DEFAULT="@playwright/cli@0.1.0"
PLAYWRIGHT_CLI_PKG="${PLAYWRIGHT_CLI_PKG:-$PLAYWRIGHT_CLI_PKG_DEFAULT}"
PLAYWRIGHT_CLI_CMD="${PLAYWRIGHT_CLI_CMD:-}"
SPEC_NAME="$(basename "$SPEC_FILE")"
FLOW_NAME="${FLOW_NAME:-${SPEC_NAME%.md}}"
TIMESTAMP="$(date -u +"%Y-%m-%dT%H%MZ")"
ARTIFACT_DIR="${ARTIFACT_ROOT}/${FLOW_NAME}__${TIMESTAMP}"
REPORT_FILE="${ARTIFACT_DIR}/report.txt"

mkdir -p "$ARTIFACT_DIR"
: > "$REPORT_FILE"

export BASE_URL

playwright_cli() {
  if [[ -n "$PLAYWRIGHT_CLI_CMD" ]]; then
    "$PLAYWRIGHT_CLI_CMD" "$@"
    return
  fi
  npx -y "$PLAYWRIGHT_CLI_PKG" "$@"
}

cleanup() {
  playwright_cli close >/dev/null 2>&1 || true
  if [[ -n "${tmp_script:-}" ]]; then
    rm -f "$tmp_script"
  fi
}

tmp_script="$(mktemp)"

trap cleanup EXIT

cat > "$tmp_script" <<'EOF'
#!/usr/bin/env bash

set -euo pipefail
CURRENT_STEP="playwright-cli"
PLAYWRIGHT_CLI_PKG="${PLAYWRIGHT_CLI_PKG:-@playwright/cli@0.1.0}"
PLAYWRIGHT_CLI_CMD="${PLAYWRIGHT_CLI_CMD:-}"

slugify() {
  echo "$1" | tr '[:upper:]' '[:lower:]' | tr ' /:' '---' | tr -cd 'a-z0-9-_'
}

report_line() {
  printf '%s|%s\n' "$1" "$2" >> "$REPORT_FILE"
}

playwright_cli() {
  if [[ -n "$PLAYWRIGHT_CLI_CMD" ]]; then
    "$PLAYWRIGHT_CLI_CMD" "$@"
    return
  fi
  npx -y "$PLAYWRIGHT_CLI_PKG" "$@"
}

cli() {
  local label="${CURRENT_STEP:-playwright-cli}"
  local slug
  slug="$(slugify "$label")"
  local log_file="${ARTIFACT_DIR}/${slug}.log"

  echo "==> ${label}"
  set +e
  output=$(playwright_cli "$@" 2>&1)
  status=$?
  set -e

  printf '%s\n' "$output" | tee "$log_file"

  if [[ $status -ne 0 || "$output" == *"### Error"* || "$output" == *"Error:"* || "$output" == *"TimeoutError"* ]]; then
    echo "FAIL: ${label}"
    report_line "FAIL" "$label"
    playwright_cli screenshot "${ARTIFACT_DIR}/${slug}-failure.png" >/dev/null 2>&1 || true
    exit 1
  fi

  echo "PASS: ${label}"
  report_line "PASS" "$label"
  CURRENT_STEP="playwright-cli"
}

step() {
  local label="$1"
  CURRENT_STEP="$label"
}

open_browser() {
  if [[ -n "${PLAYWRIGHT_OPEN_ARGS:-}" ]]; then
    read -r -a args <<< "${PLAYWRIGHT_OPEN_ARGS}"
    cli open "${args[@]}" "$BASE_URL"
  else
    cli open "$BASE_URL"
  fi
}

export -f slugify report_line playwright_cli cli step open_browser
EOF

chmod +x "$tmp_script"

in_block=0
while IFS= read -r line; do
  if [[ "$line" == '```playwright-cli' ]]; then
    in_block=1
    continue
  fi
  if [[ "$line" == '```' && $in_block -eq 1 ]]; then
    in_block=0
    continue
  fi
  if [[ $in_block -eq 1 ]]; then
    printf '%s\n' "$line" >> "$tmp_script"
  fi
done < "$SPEC_FILE"

if ! grep -q '```playwright-cli' "$SPEC_FILE"; then
  echo "No playwright-cli code blocks found in $SPEC_FILE" >&2
  exit 1
fi

REPORT_FILE="$REPORT_FILE" ARTIFACT_DIR="$ARTIFACT_DIR" BASE_URL="$BASE_URL" PLAYWRIGHT_OPEN_ARGS="${PLAYWRIGHT_OPEN_ARGS:-}" PLAYWRIGHT_CLI_PKG="$PLAYWRIGHT_CLI_PKG" PLAYWRIGHT_CLI_CMD="$PLAYWRIGHT_CLI_CMD" bash "$tmp_script"

echo "Report: ${REPORT_FILE}"
