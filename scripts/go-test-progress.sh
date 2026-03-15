#!/usr/bin/env bash
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"

label=""
status_dir=""
artifacts_csv=""
heartbeat_interval="${TEST_PROGRESS_HEARTBEAT:-10s}"

while [[ $# -gt 0 ]]; do
	case "$1" in
	--label)
		label="$2"
		shift 2
		;;
	--status-dir)
		status_dir="$2"
		shift 2
		;;
	--artifacts)
		artifacts_csv="$2"
		shift 2
		;;
	--heartbeat-interval)
		heartbeat_interval="$2"
		shift 2
		;;
	--)
		shift
		break
		;;
	*)
		echo "unknown argument: $1" >&2
		exit 1
		;;
	esac
done

if [[ -z "$label" ]]; then
	echo "--label is required" >&2
	exit 1
fi
if [[ -z "$status_dir" ]]; then
	echo "--status-dir is required" >&2
	exit 1
fi
if [[ $# -eq 0 ]]; then
	echo "a command is required after --" >&2
	exit 1
fi

mkdir -p "$status_dir"

status_json="$status_dir/status.json"
raw_jsonl="$status_dir/raw.jsonl"

artifacts_csv="${artifacts_csv:+$artifacts_csv,}$status_json,$raw_jsonl"

set +e
"$@" | go run ./internal/tools/testruntimereport stream \
	-label "$label" \
	-status-json "$status_json" \
	-raw-jsonl "$raw_jsonl" \
	-heartbeat-interval "$heartbeat_interval" \
	-artifacts "$artifacts_csv"
pipeline_status=$?
set -e

exit "$pipeline_status"
