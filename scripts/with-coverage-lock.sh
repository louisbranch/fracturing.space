#!/usr/bin/env bash
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
cd "$repo_root"

if [[ "${COVERAGE_LOCK_HELD:-}" == "true" ]]; then
	"$@"
	exit $?
fi

mkdir -p "$repo_root/.tmp"

lock_dir="$repo_root/.tmp/coverage.lock"
owner_file="$lock_dir/owner"
pid_file="$lock_dir/pid"
owner_label="${COVERAGE_LOCK_LABEL:-coverage command}"

acquire_lock() {
	mkdir "$lock_dir" 2>/dev/null
}

if ! acquire_lock; then
	active_owner="another top-level coverage command"
	active_pid=""
	if [[ -f "$owner_file" ]]; then
		active_owner="$(cat "$owner_file")"
	fi
	if [[ -f "$pid_file" ]]; then
		active_pid="$(cat "$pid_file")"
	fi
	if [[ -z "$active_pid" ]]; then
		sleep 0.1
		if [[ -f "$pid_file" ]]; then
			active_pid="$(cat "$pid_file")"
		fi
	fi

	if [[ -z "$active_pid" ]] || ! kill -0 "$active_pid" 2>/dev/null; then
		rm -f "$owner_file" "$pid_file"
		rmdir "$lock_dir" 2>/dev/null || true
		if ! acquire_lock; then
			if [[ -f "$owner_file" ]]; then
				active_owner="$(cat "$owner_file")"
			fi
		else
			active_owner=""
			active_pid=""
		fi
	fi

	if [[ -n "$active_owner" ]]; then
		cat >&2 <<EOF
Coverage artifacts are already in use by $active_owner.
$owner_label cannot run in parallel with another top-level coverage command.
If 'make check' is already running, let it finish; it already generates the full coverage artifacts.
Use 'make cover' and 'make cover-critical-domain' only for focused diagnostics.
EOF
		exit 1
	fi
fi

printf '%s\n' "$owner_label" > "$owner_file"
printf '%s\n' "$$" > "$pid_file"

cleanup() {
	rm -f "$owner_file" "$pid_file"
	rmdir "$lock_dir" 2>/dev/null || true
}

trap cleanup EXIT INT TERM

COVERAGE_LOCK_HELD=true "$@"
