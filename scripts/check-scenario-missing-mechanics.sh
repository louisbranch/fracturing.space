#!/usr/bin/env bash
set -euo pipefail

DOC="docs/project/scenario-missing-mechanics.md"
SCENARIO_DIR="internal/test/game/scenarios"
MARKER="-- Missing DSL:"

tmp_markers="$(mktemp)"
tmp_doc="$(mktemp)"
tmp_missing="$(mktemp)"
tmp_stale="$(mktemp)"
trap 'rm -f "$tmp_markers" "$tmp_doc" "$tmp_missing" "$tmp_stale"' EXIT

while IFS= read -r -d '' file; do
  if rg --fixed-strings --quiet -- "$MARKER" "$file"; then
    echo "${file##*/}" | sed 's/\.lua$//' >> "$tmp_markers"
  fi
done < <(find "$SCENARIO_DIR" -name '*.lua' -print0 | sort -zV)

sort -u "$tmp_markers" -o "$tmp_markers"

  rg -o 'internal/test/game/scenarios/[A-Za-z0-9_-]+\.lua' "$DOC" \
  | sed -e 's#internal/test/game/scenarios/##' -e 's/\.lua$//' \
  | sort -u > "$tmp_doc"

comm -23 "$tmp_markers" "$tmp_doc" > "$tmp_missing" || true
comm -13 "$tmp_markers" "$tmp_doc" > "$tmp_stale" || true

issues=0
if [ -s "$tmp_missing" ]; then
  issues=1
  echo "Missing doc entries for scenario markers:"
  cat "$tmp_missing"
fi

if [ -s "$tmp_stale" ]; then
  issues=1
  echo "Stale doc entries no longer marked in scenarios:"
  cat "$tmp_stale"
fi

if [ "$issues" -eq 1 ]; then
  echo "Mismatch detected between scenario markers and docs."
  echo "Run: rg -l --fixed-strings -- \"$MARKER\" internal/test/game/scenarios/*.lua"
  exit 1
fi

echo "Scenario missing-mechanics doc coverage is in sync."
