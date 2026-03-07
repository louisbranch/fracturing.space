#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR=$(git rev-parse --show-toplevel)
cd "$ROOT_DIR"

fail() {
  echo "tool-cli-contract-check: $1" >&2
  exit 1
}

check_regex() {
  local file=$1
  local pattern=$2
  local message=$3
  if ! rg -q "$pattern" "$file"; then
    fail "$message ($file)"
  fi
}

check_small_tool_contract() {
  local file=$1

  check_regex "$file" '^func main\(\)' 'missing main entrypoint'
  check_regex "$file" '^func run\(args \[\]string, .*\) error' 'missing run(args, ..., ...) error contract'

  local exit_calls
  exit_calls=$(rg -c 'os.Exit\(' "$file")
  if [[ "$exit_calls" -ne 1 ]]; then
    fail "expected exactly one os.Exit call in small tool main (found $exit_calls) ($file)"
  fi
}

small_tools=(
  "internal/tools/i18ncheck/main.go"
  "internal/tools/i18nstatus/main.go"
  "internal/tools/testruntimereport/main.go"
  "internal/tools/webdoccheck/main.go"
  "internal/tools/websmokeauth/main.go"
)

for file in "${small_tools[@]}"; do
  check_small_tool_contract "$file"
done

# Tool library entrypoints should expose explicit ParseConfig/Run contracts where applicable.
check_regex "internal/tools/maintenance/maintenance.go" '^func ParseConfig\(fs \*flag\.FlagSet, args \[\]string\) \(Config, error\)' 'maintenance ParseConfig contract missing'
check_regex "internal/tools/maintenance/maintenance.go" '^func Run\(ctx context\.Context, cfg Config, out io\.Writer, errOut io\.Writer\) error' 'maintenance Run contract missing'

check_regex "internal/tools/importer/content/daggerheart/v1/main.go" '^func ParseConfig\(fs \*flag\.FlagSet, args \[\]string\) \(Config, error\)' 'catalog importer ParseConfig contract missing'
check_regex "internal/tools/importer/content/daggerheart/v1/main.go" '^func Run\(ctx context\.Context, cfg Config, out io\.Writer\) error' 'catalog importer Run contract missing'

check_regex "internal/tools/hmackey/hmackey.go" '^func ParseConfig\(fs \*flag\.FlagSet, args \[\]string\) \(Config, error\)' 'hmackey ParseConfig contract missing'
check_regex "internal/tools/hmackey/hmackey.go" '^func Run\(cfg Config, out io\.Writer, reader io\.Reader\) error' 'hmackey Run contract missing'

check_regex "internal/tools/joingrant/joingrant.go" '^func Run\(out io\.Writer, reader io\.Reader\) error' 'joingrant Run contract missing'

echo 'Tool CLI contract check passed.'
