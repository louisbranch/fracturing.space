#!/usr/bin/env bash
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
source "$repo_root/scripts/test-progress-lib.sh"

make_cmd="${MAKE:-make}"
status_dir="$(progress_root_dir)/check"
status_json="$status_dir/status.json"
started_at="$(progress_now_utc)"
start_epoch="$(date +%s)"
current_stage=""
stages_completed=0
stages_total=4

finish() {
	local code=$?
	local state="passed"
	local message=""
	if [[ "$code" -ne 0 ]]; then
		state="failed"
		message="stage ${current_stage:-unknown} failed"
	fi
	progress_write_stage_status \
		"$status_json" \
		"check" \
		"$state" \
		"$started_at" \
		"$start_epoch" \
		"$current_stage" \
		"$stages_completed" \
		"$stages_total" \
		"$message"
}
trap finish EXIT

run_stage() {
	local stage_index="$1"
	local stage_name="$2"
	shift 2
	current_stage="$stage_name"
	stages_completed=$((stage_index - 1))
	progress_write_stage_status \
		"$status_json" \
		"check" \
		"running" \
		"$started_at" \
		"$start_epoch" \
		"$current_stage" \
		"$stages_completed" \
		"$stages_total" \
		""
	echo "[check] stage ${stage_index}/${stages_total}: ${stage_name}"
	"$@"
	stages_completed="$stage_index"
}

run_stage 1 "check-core" "$make_cmd" check-core
run_stage 2 "check-focused" "$make_cmd" check-focused
run_stage 3 "check-runtime" "$make_cmd" check-runtime
run_stage 4 "check-coverage" "$make_cmd" check-coverage

current_stage=""
