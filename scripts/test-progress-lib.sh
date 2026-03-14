#!/usr/bin/env bash
set -euo pipefail

progress_repo_root() {
	git rev-parse --show-toplevel 2>/dev/null || pwd
}

progress_root_dir() {
	local repo_root
	repo_root="$(progress_repo_root)"
	printf '%s\n' "${TEST_PROGRESS_DIR:-$repo_root/.tmp/test-status}"
}

progress_now_utc() {
	date -u +"%Y-%m-%dT%H:%M:%SZ"
}

progress_elapsed_seconds() {
	local start_epoch="$1"
	local now_epoch
	now_epoch="$(date +%s)"
	echo $((now_epoch - start_epoch))
}

progress_write_stage_status() {
	local path="$1"
	local label="$2"
	local state="$3"
	local started_at="$4"
	local start_epoch="$5"
	local current_stage="$6"
	local stages_completed="$7"
	local stages_total="$8"
	local message="${9:-}"

	mkdir -p "$(dirname "$path")"
	cat >"$path" <<EOF
{
  "label": "$label",
  "state": "$state",
  "started_at_utc": "$started_at",
  "updated_at_utc": "$(progress_now_utc)",
  "elapsed_seconds": $(progress_elapsed_seconds "$start_epoch"),
  "current_stage": "$current_stage",
  "stages_completed": $stages_completed,
  "stages_total": $stages_total,
  "message": "$message"
}
EOF
}
