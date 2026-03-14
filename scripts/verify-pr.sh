#!/usr/bin/env bash
set -euo pipefail

run_step() {
  echo "==> $*"
  "$@"
}

run_step make verify-pr-fast
run_step make runtime-smoke
run_step make coverage-pr

echo "Local PR verification checks passed."
