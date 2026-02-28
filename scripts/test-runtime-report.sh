#!/usr/bin/env bash
set -euo pipefail

MODE="${1:-smoke}"
OUT_DIR="${OUT_DIR:-.tmp/test-runtime}"
INTEGRATION_SMOKE_FULL_PATTERN='^(TestMCPStdioEndToEnd|TestMCPHTTPBlackbox)$'
INTEGRATION_SMOKE_PR_PATTERN='^(TestMCPStdioEndToEnd|TestMCPHTTPBlackboxSmoke)$'
SCENARIO_SMOKE_MANIFEST="${SCENARIO_SMOKE_MANIFEST:-internal/test/game/scenarios/smoke.txt}"
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
	run_capture integration-smoke go test -tags=integration ./internal/test/integration -run "$INTEGRATION_SMOKE_FULL_PATTERN" -json
	run_capture scenario-smoke env SCENARIO_MANIFEST="$SCENARIO_SMOKE_MANIFEST" go test -tags=scenario ./internal/test/game -json
	;;
smoke-pr)
	run_capture integration-smoke-pr go test -tags=integration ./internal/test/integration -run "$INTEGRATION_SMOKE_PR_PATTERN" -json
	run_capture scenario-smoke env SCENARIO_MANIFEST="$SCENARIO_SMOKE_MANIFEST" go test -tags=scenario ./internal/test/game -json
	;;
integration-full)
	run_capture integration-full go test -tags=integration ./internal/test/integration -json
	;;
integration-shard)
	: "${INTEGRATION_SHARD_TOTAL:?set INTEGRATION_SHARD_TOTAL}"
	: "${INTEGRATION_SHARD_INDEX:?set INTEGRATION_SHARD_INDEX}"
	run_capture "integration-shard-${INTEGRATION_SHARD_INDEX}-of-${INTEGRATION_SHARD_TOTAL}" \
		env INTEGRATION_SHARD_TOTAL="${INTEGRATION_SHARD_TOTAL}" INTEGRATION_SHARD_INDEX="${INTEGRATION_SHARD_INDEX}" \
		bash ./scripts/integration-shard.sh -json
	;;
scenario-full)
	run_capture scenario-full go test -tags=scenario ./internal/test/game -json
	;;
scenario-shard)
	: "${SCENARIO_SHARD_TOTAL:?set SCENARIO_SHARD_TOTAL}"
	: "${SCENARIO_SHARD_INDEX:?set SCENARIO_SHARD_INDEX}"
	run_capture "scenario-shard-${SCENARIO_SHARD_INDEX}-of-${SCENARIO_SHARD_TOTAL}" \
		env SCENARIO_SHARD_TOTAL="${SCENARIO_SHARD_TOTAL}" SCENARIO_SHARD_INDEX="${SCENARIO_SHARD_INDEX}" \
		go test -tags=scenario ./internal/test/game -json
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
