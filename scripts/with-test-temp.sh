#!/usr/bin/env bash
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
cd "$repo_root"

if [[ $# -lt 3 ]]; then
	echo "usage: $0 <label> -- <command> [args...]" >&2
	exit 1
fi

label="$1"
shift

if [[ "$1" != "--" ]]; then
	echo "expected '--' before wrapped command" >&2
	exit 1
fi
shift

if [[ $# -eq 0 ]]; then
	echo "wrapped command is required" >&2
	exit 1
fi

if [[ "${TEST_TEMP_HELD:-}" == "true" ]]; then
	"$@"
	exit $?
fi

test_tmp_root="${TEST_TMP_ROOT:-$repo_root/.tmp/test-tmp}"
go_test_cache_dir="${GO_TEST_CACHE_DIR:-$repo_root/.tmp/go-cache}"

mkdir -p "$test_tmp_root" "$go_test_cache_dir"

run_root="$(mktemp -d "$test_tmp_root/${label}.XXXXXX")"
tmp_dir="$run_root/tmp"
go_tmp_dir="$run_root/go-build"

mkdir -p "$tmp_dir" "$go_tmp_dir"

cleanup() {
	rm -rf "$run_root"
}

trap cleanup EXIT INT TERM

TEST_TEMP_HELD=true \
TEST_TEMP_RUN_ROOT="$run_root" \
TMPDIR="$tmp_dir" \
TMP="$tmp_dir" \
TEMP="$tmp_dir" \
GOTMPDIR="$go_tmp_dir" \
GO_TEST_TMP_DIR="$go_tmp_dir" \
GOCACHE="${GOCACHE:-$go_test_cache_dir}" \
	"$@"
