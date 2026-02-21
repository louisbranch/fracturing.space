#!/usr/bin/env bash

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

proto_plan="$(make -n proto)"
templ_plan="$(make -n templ-generate)"
setup_plan="$(make -n setup-hooks)"

if grep -Fq 'goimports -w' <<<"$proto_plan"; then
  echo "expected proto target to avoid implicit formatting commands" >&2
  exit 1
fi

if grep -Fq 'goimports -w' <<<"$templ_plan"; then
  echo "expected templ-generate target to avoid implicit formatting commands" >&2
  exit 1
fi

if ! grep -Fq 'git config --local --get core.hooksPath' <<<"$setup_plan"; then
  echo "expected setup-hooks to check existing hooks path before writing" >&2
  exit 1
fi

if ! grep -Fq 'already configured' <<<"$setup_plan"; then
  echo "expected setup-hooks to no-op when already configured" >&2
  exit 1
fi

if ! grep -Fq 'chmod +x .githooks/pre-commit' <<<"$setup_plan"; then
  echo "expected setup-hooks to ensure executable pre-commit hook" >&2
  exit 1
fi

echo "PASS"
