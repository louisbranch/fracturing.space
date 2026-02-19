#!/usr/bin/env bash

set -euo pipefail

if ! git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  echo "pre-commit formatter must run inside a git worktree" >&2
  exit 1
fi

ROOT="$(git rev-parse --show-toplevel)"
cd "$ROOT"

mapfile -t staged_go_files < <(git diff --cached --name-only --diff-filter=ACMR -- '*.go')
if [[ ${#staged_go_files[@]} -eq 0 ]]; then
  exit 0
fi

mapfile -t unstaged_go_files < <(git diff --name-only --diff-filter=ACMR -- '*.go')
declare -A has_unstaged=()
for file in "${unstaged_go_files[@]}"; do
  has_unstaged["$file"]=1
done

partially_staged=()
for file in "${staged_go_files[@]}"; do
  if [[ -n "${has_unstaged[$file]:-}" ]]; then
    partially_staged+=("$file")
  fi
done

if [[ ${#partially_staged[@]} -gt 0 ]]; then
  echo "pre-commit formatting aborted: staged Go files are partially staged:" >&2
  printf '  %s\n' "${partially_staged[@]}" >&2
  echo "Stage or unstage complete file contents and retry the commit." >&2
  exit 1
fi

MAKE_CMD="${MAKE_CMD:-make}"
for file in "${staged_go_files[@]}"; do
  "$MAKE_CMD" fmt "FILE=$file"
done
git add -- "${staged_go_files[@]}"
