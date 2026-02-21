#!/usr/bin/env bash
set -euo pipefail

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$root_dir"

mkdir -p .tmp/dev

pid_file=".tmp/dev/watch-services.pid"

if [[ -f "$pid_file" ]]; then
  pid="$(cat "$pid_file")"
  if [[ -n "$pid" ]] && kill -0 "$pid" 2>/dev/null; then
    printf 'watch-services already running (pid %s)\n' "$pid"
    exit 0
  fi
  rm -f "$pid_file"
fi

bash .devcontainer/scripts/watch-services.sh >> .tmp/dev/watch-services.log 2>&1 &
pid="$!"
echo "$pid" > "$pid_file"
sleep 1
if kill -0 "$pid" 2>/dev/null; then
  printf 'started watch-services (pid %s)\n' "$pid"
  exit 0
fi

rm -f "$pid_file"
printf 'watch-services failed to stay running; check .tmp/dev/watch-services.log\n' >&2
exit 1
