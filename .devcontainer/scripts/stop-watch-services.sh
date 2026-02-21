#!/usr/bin/env bash
set -euo pipefail

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$root_dir"

pid_file=".tmp/dev/watch-services.pid"

if [[ ! -f "$pid_file" ]]; then
  echo "watch-services is not running"
  exit 0
fi

pid="$(cat "$pid_file")"
if [[ -n "$pid" ]] && kill -0 "$pid" 2>/dev/null; then
  kill "$pid"
  echo "stopped watch-services (pid $pid)"
else
  echo "watch-services pid file found but process is not running"
fi

rm -f "$pid_file"
