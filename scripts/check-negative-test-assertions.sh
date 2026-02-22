#!/usr/bin/env bash

set -euo pipefail

repo_root="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
cd "$repo_root"

if [ "$#" -gt 0 ]; then
  mapfile -t files < <(printf '%s\n' "$@" | rg '_test[.]go$' || true)
else
  mapfile -t files < <(rg --files -g '*_test.go')
fi

if [ "${#files[@]}" -eq 0 ]; then
  echo "Negative test assertion check passed."
  exit 0
fi

failures=0

for file in "${files[@]}"; do
  if [ ! -f "$file" ]; then
    continue
  fi

  if ! awk '
    {
      lines[NR] = $0
    }
    END {
      bad = 0
      for (i = 1; i <= NR; i++) {
        line = lines[i]
        if (line ~ /^[[:space:]]*\/\//) {
          continue
        }
        if (line ~ /^[[:space:]]*func[[:space:]]+(assertNotContains|assertHTMLNotContains)[[:space:]]*\(/) {
          continue
        }
        if (line !~ /(assertNotContains|assertHTMLNotContains)[[:space:]]*\(/) {
          continue
        }

        allowed = (line ~ /Invariant:/)
        if (!allowed) {
          for (j = i - 1; j >= 1 && j >= i - 3; j--) {
            if (lines[j] ~ /^[[:space:]]*\/\/[[:space:]]*Invariant:/) {
              allowed = 1
              break
            }
          }
        }

        if (!allowed) {
          printf "%s:%d: low-value negative assertion missing Invariant: rationale\n", FILENAME, i > "/dev/stderr"
          bad = 1
        }
      }
      exit bad
    }
  ' "$file"; then
    failures=1
  fi
done

if [ "$failures" -ne 0 ]; then
  echo "Negative test assertion check failed." >&2
  exit 1
fi

echo "Negative test assertion check passed."
