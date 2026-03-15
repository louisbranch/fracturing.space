#!/usr/bin/env bash
set -euo pipefail

MODE="${1:-smoke}"
OUT_DIR="${OUT_DIR:-.tmp/test-runtime}"
INTEGRATION_SMOKE_FULL_PATTERN='^(TestMCPEndToEnd|TestMCPHTTPBlackbox)$'
INTEGRATION_SMOKE_PR_PATTERN='^(TestMCPEndToEnd|TestMCPHTTPBlackboxSmoke)$'
SCENARIO_SMOKE_MANIFEST="${SCENARIO_SMOKE_MANIFEST:-internal/test/game/scenarios/manifests/smoke.txt}"
SCENARIO_PARALLELISM="${SCENARIO_PARALLELISM:-4}"
INTEGRATION_SHARED_FIXTURE="${INTEGRATION_SHARED_FIXTURE:-true}"
RUNTIME_BUDGET_FILE="${RUNTIME_BUDGET_FILE:-.github/test-runtime-budgets.json}"
RUNTIME_BUDGET_ENFORCE="${RUNTIME_BUDGET_ENFORCE:-false}"

mkdir -p "$OUT_DIR"
rm -f "$OUT_DIR"/*.jsonl "$OUT_DIR"/summary.json "$OUT_DIR"/summary.csv

run_capture() {
	local label="$1"
	shift
	echo "[runtime] running ${label}"
	"$@" > "${OUT_DIR}/${label}.jsonl"
}

case "$MODE" in
smoke)
	run_capture integration-smoke env INTEGRATION_SHARED_FIXTURE="$INTEGRATION_SHARED_FIXTURE" go test -count=1 -tags=integration ./internal/test/integration -run "$INTEGRATION_SMOKE_FULL_PATTERN" -json
	run_capture scenario-smoke env SCENARIO_MANIFEST="$SCENARIO_SMOKE_MANIFEST" go test -count=1 -parallel="$SCENARIO_PARALLELISM" -tags=scenario ./internal/test/game -json
	;;
smoke-pr)
	run_capture integration-smoke-pr env INTEGRATION_SHARED_FIXTURE="$INTEGRATION_SHARED_FIXTURE" go test -count=1 -tags=integration ./internal/test/integration -run "$INTEGRATION_SMOKE_PR_PATTERN" -json
	run_capture scenario-smoke env SCENARIO_MANIFEST="$SCENARIO_SMOKE_MANIFEST" go test -count=1 -parallel="$SCENARIO_PARALLELISM" -tags=scenario ./internal/test/game -json
	;;
integration-full)
	run_capture integration-full env INTEGRATION_SHARED_FIXTURE="$INTEGRATION_SHARED_FIXTURE" go test -count=1 -tags=integration ./internal/test/integration -json
	;;
integration-shard)
	: "${INTEGRATION_SHARD_TOTAL:?set INTEGRATION_SHARD_TOTAL}"
	: "${INTEGRATION_SHARD_INDEX:?set INTEGRATION_SHARD_INDEX}"
	run_capture "integration-shard-${INTEGRATION_SHARD_INDEX}-of-${INTEGRATION_SHARD_TOTAL}" \
		env INTEGRATION_SHARED_FIXTURE="$INTEGRATION_SHARED_FIXTURE" INTEGRATION_SHARD_TOTAL="${INTEGRATION_SHARD_TOTAL}" INTEGRATION_SHARD_INDEX="${INTEGRATION_SHARD_INDEX}" \
		bash ./scripts/integration-shard.sh -count=1 -json
	;;
scenario-full)
	run_capture scenario-full go test -count=1 -parallel="$SCENARIO_PARALLELISM" -tags=scenario ./internal/test/game -json
	;;
scenario-shard)
	: "${SCENARIO_SHARD_TOTAL:?set SCENARIO_SHARD_TOTAL}"
	: "${SCENARIO_SHARD_INDEX:?set SCENARIO_SHARD_INDEX}"
	run_capture "scenario-shard-${SCENARIO_SHARD_INDEX}-of-${SCENARIO_SHARD_TOTAL}" \
		env SCENARIO_SHARD_TOTAL="${SCENARIO_SHARD_TOTAL}" SCENARIO_SHARD_INDEX="${SCENARIO_SHARD_INDEX}" \
		go test -count=1 -parallel="$SCENARIO_PARALLELISM" -tags=scenario ./internal/test/game -json
	;;
*)
	echo "unknown runtime report mode: ${MODE}" >&2
	exit 1
	;;
esac

tool_args=(
	-input-dir "$OUT_DIR"
	-out-json "${OUT_DIR}/summary.json"
	-out-csv "${OUT_DIR}/summary.csv"
)

if [[ -f "$RUNTIME_BUDGET_FILE" ]]; then
	tool_args+=(-budget-file "$RUNTIME_BUDGET_FILE")
	if [[ "$RUNTIME_BUDGET_ENFORCE" == "true" ]]; then
		tool_args+=(-enforce-budget)
	fi
fi

go run ./internal/tools/testruntimereport "${tool_args[@]}"

echo "[runtime] summary json: ${OUT_DIR}/summary.json"
echo "[runtime] summary csv: ${OUT_DIR}/summary.csv"
