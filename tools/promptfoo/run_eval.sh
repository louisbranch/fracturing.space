#!/usr/bin/env bash
set -euo pipefail

: "${INTEGRATION_OPENAI_API_KEY:?INTEGRATION_OPENAI_API_KEY is required}"

preset="${AI_EVAL_PRESET:-manual}"
scenario_set="${PROMPTFOO_SCENARIO_SET:-core}"
repeat_count="${PROMPTFOO_REPEAT:-1}"
run_id="${PROMPTFOO_RUN_ID:-$(date -u +%Y%m%dT%H%M%SZ)-${preset}}"
out_dir="${PROMPTFOO_OUT_DIR:-.tmp/promptfoo/${run_id}}"
promptfoo_npx_spec="${PROMPTFOO_NPX_SPEC:-promptfoo@latest}"
capture_dir="${out_dir}/cases"
promptfoo_config_dir="${PROMPTFOO_CONFIG_DIR:-.tmp/promptfoo-home}"
npm_cache_dir="${NPM_CONFIG_CACHE:-$PWD/.tmp/npm-cache}"
go_cache_dir="${GOCACHE:-$PWD/.tmp/go-cache}"
go_tmp_dir="${GOTMPDIR:-$PWD/.tmp/go-build}"

mkdir -p "${out_dir}"
mkdir -p "${promptfoo_config_dir}"
mkdir -p "${npm_cache_dir}"
mkdir -p "${go_cache_dir}"
mkdir -p "${go_tmp_dir}"

results_path="${out_dir}/results.json"
scorecard_path="${out_dir}/scorecard.md"

echo "Promptfoo preset: ${preset}"
echo "Promptfoo scenario set: ${scenario_set}"
echo "Promptfoo repeats: ${repeat_count}"
echo "Promptfoo package: ${promptfoo_npx_spec}"

marker_path="${promptfoo_config_dir}/evalLastWritten"
before_marker=""
if [[ -f "${marker_path}" ]]; then
  before_marker="$(cat "${marker_path}")"
fi

status=0
export PROMPTFOO_CONFIG_DIR="${promptfoo_config_dir}"
export PROMPTFOO_RUN_ID="${run_id}"
export NPM_CONFIG_CACHE="${npm_cache_dir}"
export GOCACHE="${go_cache_dir}"
export GOTMPDIR="${go_tmp_dir}"
PROMPTFOO_CAPTURE_DIR="${capture_dir}" \
npx --yes "${promptfoo_npx_spec}" eval \
  -c tools/promptfoo/promptfooconfig.js \
  "$@" || status=$?

after_marker=""
if [[ -f "${marker_path}" ]]; then
  after_marker="$(cat "${marker_path}")"
fi

if [[ -n "${after_marker}" && "${after_marker}" != "${before_marker}" ]]; then
  node tools/promptfoo/scripts/export_latest_eval.js \
    --output "${results_path}"
fi

if [[ ! -f "${results_path}" ]]; then
  node tools/promptfoo/scripts/synthesize_results.js \
    --config tools/promptfoo/promptfooconfig.js \
    --capture-dir "${capture_dir}" \
    --output "${results_path}" \
    --run-id "${run_id}"
  if ! npx --yes "${promptfoo_npx_spec}" import "${results_path}" >/dev/null 2>&1; then
    echo "Promptfoo viewer import skipped for synthesized results"
  fi
fi

if [[ -f "${results_path}" ]]; then
  node tools/promptfoo/scripts/summarize_results.js \
    --input "${results_path}" \
    --output "${scorecard_path}"
fi

echo "Promptfoo artifacts: ${out_dir}"

exit "${status}"
