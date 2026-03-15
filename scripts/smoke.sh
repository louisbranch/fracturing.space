#!/usr/bin/env bash
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
source "$repo_root/scripts/test-progress-lib.sh"

make_cmd="${MAKE:-make}"
status_dir="$(progress_root_dir)/smoke"
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
		"smoke" \
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
		"smoke" \
		"running" \
		"$started_at" \
		"$start_epoch" \
		"$current_stage" \
		"$stages_completed" \
		"$stages_total" \
		""
	echo "[smoke] stage ${stage_index}/${stages_total}: ${stage_name}"
	"$@"
	stages_completed="$stage_index"
}

run_stage 1 "event-catalog-check" "$make_cmd" event-catalog-check
run_stage 2 "topology-check" "$make_cmd" topology-check
run_stage 3 "integration-smoke" "$make_cmd" smoke-integration
run_stage 4 "scenario-smoke" "$make_cmd" smoke-scenario

current_stage=""
