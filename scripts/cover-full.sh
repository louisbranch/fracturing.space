#!/usr/bin/env bash
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
source "$repo_root/scripts/test-progress-lib.sh"

status_dir="$(progress_root_dir)/cover"
status_json="$status_dir/status.json"
log_dir="$status_dir/logs"
started_at="$(progress_now_utc)"
start_epoch="$(date +%s)"
current_stage=""
stages_completed=0
stages_total=5

go_test_cache_dir="${GO_TEST_CACHE_DIR:-$repo_root/.tmp/go-cache}"
go_test_tmp_dir="${GO_TEST_TMP_DIR:-$repo_root/.tmp/go-build}"
integration_shared_fixture="${INTEGRATION_SHARED_FIXTURE:-true}"
integration_shards="${INTEGRATION_COVERAGE_SHARDS:-4}"
integration_parallelism="${INTEGRATION_COVERAGE_PARALLELISM:-1}"
scenario_parallelism="${SCENARIO_COVERAGE_PARALLELISM:-4}"
cov_root="$repo_root/.tmp/cover-covdata"

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
		"cover" \
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
		"cover" \
		"running" \
		"$started_at" \
		"$start_epoch" \
		"$current_stage" \
		"$stages_completed" \
		"$stages_total" \
		""
	echo "[cover] stage ${stage_index}/${stages_total}: ${stage_name}"
	"$@"
	stages_completed="$stage_index"
}

join_by_comma() {
	local value
	local first=true
	for value in "$@"; do
		if [[ "$first" == true ]]; then
			printf '%s' "$value"
			first=false
		else
			printf ',%s' "$value"
		fi
	done
}

verify_shards() {
	env INTEGRATION_VERIFY_SHARDS_TOTAL="$integration_shards" bash ./scripts/integration-shard.sh --check >/dev/null
}

run_non_integration_coverage() {
	mkdir -p "$cov_root/non-integration"
	bash ./scripts/go-test-progress.sh \
		--label "cover-non-integration" \
		--status-dir "$status_dir/non-integration" \
		-- \
		env GOCACHE="$go_test_cache_dir" GOTMPDIR="$go_test_tmp_dir" \
		go test -json -count=1 -tags=integration -cover -covermode=set "${non_integration_packages[@]}" -args "-test.gocoverdir=$cov_root/non-integration" \
		> >(tee "$log_dir/non-integration.log") 2>&1
}

run_integration_shards() {
	local -a pids=()
	local shard_index
	if ! [[ "$integration_parallelism" =~ ^[0-9]+$ ]] || (( integration_parallelism <= 0 )); then
		echo "invalid INTEGRATION_COVERAGE_PARALLELISM=$integration_parallelism" >&2
		exit 1
	fi
	for (( shard_index = 0; shard_index < integration_shards; shard_index++ )); do
		local shard_status_dir="$status_dir/integration-shards/${shard_index}"
		local shard_covdir="$cov_root/integration-shard-${shard_index}"
		local shard_log="$log_dir/integration-shard-${shard_index}.log"
		local shard_go_tmp_dir="$go_test_tmp_dir/integration-shard-${shard_index}"
		mkdir -p "$shard_covdir"
		mkdir -p "$shard_go_tmp_dir"
		(
			bash ./scripts/go-test-progress.sh \
				--label "cover-integration-shard-${shard_index}" \
				--status-dir "$shard_status_dir" \
				-- \
				env GOCACHE="$go_test_cache_dir" GOTMPDIR="$shard_go_tmp_dir" \
					INTEGRATION_SHARED_FIXTURE="$integration_shared_fixture" \
					INTEGRATION_SHARD_TOTAL="$integration_shards" \
					INTEGRATION_SHARD_INDEX="$shard_index" \
					bash ./scripts/integration-shard.sh -count=1 -json -cover -covermode=set -coverpkg="$integration_coverpkg" -args "-test.gocoverdir=$shard_covdir"
		) > >(tee "$shard_log") 2>&1 &
		pids+=("$!")
		if (( ${#pids[@]} >= integration_parallelism )); then
			local pid
			for pid in "${pids[@]}"; do
				wait "$pid"
			done
			pids=()
		fi
	done

	local pid
	for pid in "${pids[@]}"; do
		wait "$pid"
	done
}

run_scenario_coverage() {
	mkdir -p "$cov_root/scenario"
	local scenario_go_tmp_dir="$go_test_tmp_dir/scenario"
	mkdir -p "$scenario_go_tmp_dir"
	bash ./scripts/go-test-progress.sh \
		--label "cover-scenario" \
		--status-dir "$status_dir/scenario" \
		-- \
		env GOCACHE="$go_test_cache_dir" GOTMPDIR="$scenario_go_tmp_dir" \
		go test -json -count=1 -parallel="$scenario_parallelism" -tags=scenario \
			-cover -covermode=set -coverpkg="$scenario_coverpkg" \
			./internal/test/game \
			-args "-test.gocoverdir=$cov_root/scenario" \
		> >(tee "$log_dir/scenario.log") 2>&1
}

merge_coverage_artifacts() {
	local -a inputs=("$cov_root/non-integration")
	local shard_index
	for (( shard_index = 0; shard_index < integration_shards; shard_index++ )); do
		inputs+=("$cov_root/integration-shard-${shard_index}")
	done
	inputs+=("$cov_root/scenario")

	rm -rf "$cov_root/merged"
	mkdir -p "$cov_root/merged"
	go tool covdata merge -i="$(join_by_comma "${inputs[@]}")" -o "$cov_root/merged"
	go tool covdata textfmt -i="$cov_root/merged" -o coverage.raw
	{
		head -n 1 coverage.raw
		tail -n +2 coverage.raw | grep -E '^[^[:space:]]+:[0-9]+\.[0-9]+,[0-9]+\.[0-9]+ [0-9]+ [0-9]+$' | grep -Ev "$COVER_EXCLUDE_REGEX"
	} > coverage.out
	go tool cover -func coverage.out > coverage.func
	awk '/^total:/{print}' coverage.func
	go tool cover -html=coverage.out -o coverage.html
	cat "$log_dir"/non-integration.log "$log_dir"/integration-shard-*.log "$log_dir"/scenario.log > coverage.log
}

mkdir -p "$go_test_cache_dir" "$go_test_tmp_dir" "$log_dir"
rm -rf "$cov_root"
mkdir -p "$cov_root"
rm -f coverage.raw coverage.out coverage.html coverage.func coverage.log

integration_pkg="$(go list -tags=integration ./internal/test/integration)"
mapfile -t all_packages < <(go list -tags=integration ./...)
non_integration_packages=()
for pkg in "${all_packages[@]}"; do
	if [[ "$pkg" == "$integration_pkg" ]]; then
		continue
	fi
	non_integration_packages+=("$pkg")
done

# Compute -coverpkg lists so integration and scenario tests contribute
# coverage to the service packages they exercise, not just their own
# test-only packages.
integration_coverpkg="$(COVER_EXCLUDE_REGEX="$COVER_EXCLUDE_REGEX" bash ./scripts/coverpkg-list.sh integration ./internal/services/... ./internal/platform/...)"
scenario_coverpkg="$(COVER_EXCLUDE_REGEX="$COVER_EXCLUDE_REGEX" bash ./scripts/coverpkg-list.sh scenario ./internal/services/game/... ./internal/services/auth/... ./internal/platform/...)"

run_stage 1 "discover-shards" verify_shards
run_stage 2 "non-integration-coverage" run_non_integration_coverage
run_stage 3 "integration-sharded-coverage" run_integration_shards
run_stage 4 "scenario-coverage" run_scenario_coverage
run_stage 5 "merge-coverage-artifacts" merge_coverage_artifacts

current_stage=""
