#!/usr/bin/env bash

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

tmp_dir="$(mktemp -d)"
cleanup() {
  rm -rf "$tmp_dir"
}
trap cleanup EXIT

spec_file="$tmp_dir/spec.md"
cat > "$spec_file" <<'EOF'
# Spec runner test

```playwright-cli
step "First"
cli noop
cli noop
```
EOF

fake_cli="$tmp_dir/fake-playwright-cli.sh"
cat > "$fake_cli" <<'EOF'
#!/usr/bin/env bash
echo "fake-playwright-cli $*"
EOF
chmod +x "$fake_cli"

export PLAYWRIGHT_CLI_CMD="$fake_cli"
export ARTIFACT_ROOT="$tmp_dir/artifacts"
export FLOW_NAME="spec-test"

scripts/playwright-run-spec.sh "$spec_file"

report_files=("$ARTIFACT_ROOT"/spec-test__*/report.txt)
if [[ ${#report_files[@]} -ne 1 ]]; then
  echo "Expected one report file, found ${#report_files[@]}" >&2
  exit 1
fi

mapfile -t lines < "${report_files[0]}"
if [[ ${lines[0]:-} != "PASS|First" || ${lines[1]:-} != "PASS|playwright-cli" ]]; then
  printf 'Unexpected report lines:\n%s\n' "${lines[*]}" >&2
  exit 1
fi

echo "PASS"
