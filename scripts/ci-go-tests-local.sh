#!/usr/bin/env bash
set -euo pipefail
#
# Local parity script for .github/workflows/go-tests.yml (test job).
# Keep steps in sync; intentional divergences:
#   - coverage-badge job is CI-only (runs on main push, not locally)
#   - coverage regression checks use local git fetch instead of CI artifact download

run_step() {
  echo "==> $*"
  "$@"
}

run_step make docs-check
run_step make fmt-check
run_step make event-catalog-check
run_step make i18n-check
run_step make i18n-status-check
run_step make topology-check
run_step make negative-test-assertion-check
run_step make web-architecture-check
run_step make game-architecture-check
run_step make admin-architecture-check
run_step make cover
run_step make cover-critical-domain

CURRENT=$(go tool cover -func=coverage.out | awk '/total/ {print substr($3, 1, length($3)-1)}')
if [ -z "$CURRENT" ] || ! printf '%s\n' "$CURRENT" | grep -Eq '^[0-9]+(\.[0-9]+)?$'; then
  echo "Failed to extract valid coverage percentage."
  exit 1
fi

if git ls-remote --exit-code --heads origin badges >/dev/null 2>&1; then
  git fetch origin badges:refs/remotes/origin/badges
  if git show origin/badges:coverage-baseline.txt > /tmp/coverage-baseline.txt; then
    BASELINE=$(cat /tmp/coverage-baseline.txt)
    if [ -z "$BASELINE" ] || ! printf '%s\n' "$BASELINE" | grep -Eq '^[0-9]+(\.[0-9]+)?$'; then
      echo "Invalid coverage baseline value: $BASELINE"
      exit 1
    fi
    ALLOW_DROP=0.5
    if ! awk "BEGIN {exit !($CURRENT + 0 >= $BASELINE - $ALLOW_DROP)}"; then
      echo "Coverage regression: current $CURRENT% vs baseline $BASELINE% (allow drop $ALLOW_DROP%)."
      exit 1
    fi
  else
    echo "No coverage-baseline.txt found on badges; skipping coverage regression check."
  fi
else
  echo "No badges branch found; skipping coverage regression check."
fi

CRITICAL_CURRENT=$(go tool cover -func=coverage-critical-domain.out | awk '/total/ {print substr($3, 1, length($3)-1)}')
if [ -z "$CRITICAL_CURRENT" ] || ! printf '%s\n' "$CRITICAL_CURRENT" | grep -Eq '^[0-9]+(\.[0-9]+)?$'; then
  echo "Failed to extract valid critical-domain coverage percentage."
  exit 1
fi

if git ls-remote --exit-code --heads origin badges >/dev/null 2>&1; then
  git fetch origin badges:refs/remotes/origin/badges
  if git show origin/badges:coverage-critical-domain-baseline.txt > /tmp/coverage-critical-domain-baseline.txt; then
    CRITICAL_BASELINE=$(cat /tmp/coverage-critical-domain-baseline.txt)
    if [ -z "$CRITICAL_BASELINE" ] || ! printf '%s\n' "$CRITICAL_BASELINE" | grep -Eq '^[0-9]+(\.[0-9]+)?$'; then
      echo "Invalid critical-domain coverage baseline value: $CRITICAL_BASELINE"
      exit 1
    fi
    ALLOW_DROP=1.2
    if ! awk "BEGIN {exit !($CRITICAL_CURRENT + 0 >= $CRITICAL_BASELINE - $ALLOW_DROP)}"; then
      echo "Critical-domain coverage regression: current $CRITICAL_CURRENT% vs baseline $CRITICAL_BASELINE% (allow drop $ALLOW_DROP%)."
      exit 1
    fi
  else
    echo "No coverage-critical-domain-baseline.txt found on badges; skipping critical-domain coverage regression check."
  fi
else
  echo "No badges branch found; skipping critical-domain coverage regression check."
fi

FLOORS=docs/reference/coverage-floors.json
if git ls-remote --exit-code --heads origin badges >/dev/null 2>&1; then
  git fetch origin badges:refs/remotes/origin/badges
  if git show origin/badges:coverage-package-floors.json > /tmp/coverage-package-floors.json; then
    FLOORS=/tmp/coverage-package-floors.json
  fi
fi

run_step go run ./internal/tools/coveragefloors check -profile=coverage.out -floors="$FLOORS"

echo "Local CI Go Tests parity checks passed."
