#!/usr/bin/env bash

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SCRIPT="$ROOT/scripts/pre-commit-fmt.sh"

tmp_dir="$(mktemp -d)"
cleanup() {
  rm -rf "$tmp_dir"
}
trap cleanup EXIT

new_repo() {
  local name="$1"
  local repo="$tmp_dir/$name"
  mkdir -p "$repo"

  git -C "$repo" init -q
  git -C "$repo" config user.name "Test User"
  git -C "$repo" config user.email "test@example.com"

  cat > "$repo/go.mod" <<'EOF'
module example.com/testrepo

go 1.24
EOF

  printf '%s\n' "$repo"
}

write_fake_make() {
  local repo="$1"
  cat > "$repo/fake-make.sh" <<'EOF'
#!/usr/bin/env bash

set -euo pipefail

if [[ "${1:-}" != "fmt" ]]; then
  echo "expected fmt target" >&2
  exit 1
fi

if [[ $# -ne 2 || "${2#FILE=}" == "$2" ]]; then
  echo "missing FILE argument" >&2
  exit 1
fi

printf '%s\n' "$2" >> "$MAKE_LOG"

file="${2#FILE=}"
printf '// formatted\n' >> "$file"
EOF
  chmod +x "$repo/fake-make.sh"
}

test_skips_when_no_go_files_are_staged() {
  local repo
  repo="$(new_repo "no-go-staged")"
  local make_log="$repo/make.log"
  : > "$make_log"
  write_fake_make "$repo"

  cat > "$repo/README.md" <<'EOF'
# demo
EOF
  git -C "$repo" add README.md

  (
    cd "$repo"
    MAKE_CMD="$repo/fake-make.sh" MAKE_LOG="$make_log" bash "$SCRIPT"
  )

  if [[ -s "$make_log" ]]; then
    echo "expected no formatter invocation when no Go files are staged" >&2
    exit 1
  fi
}

test_formats_and_restages_staged_go_files() {
  local repo
  repo="$(new_repo "go-staged")"
  local make_log="$repo/make.log"
  : > "$make_log"
  write_fake_make "$repo"

  mkdir -p "$repo/dir"
  cat > "$repo/a.go" <<'EOF'
package demo

func A() {}
EOF
  cat > "$repo/dir/space name.go" <<'EOF'
package demo

func B() {}
EOF
  git -C "$repo" add a.go "dir/space name.go"

  (
    cd "$repo"
    MAKE_CMD="$repo/fake-make.sh" MAKE_LOG="$make_log" bash "$SCRIPT"
  )

  mapfile -t invocations < "$make_log"
  if [[ ${#invocations[@]} -ne 2 ]]; then
    echo "expected one formatter invocation per staged Go file" >&2
    exit 1
  fi

  if ! grep -qx 'FILE=a.go' "$make_log"; then
    echo "expected formatter invocation for a.go" >&2
    exit 1
  fi

  if ! grep -qx 'FILE=dir/space name.go' "$make_log"; then
    echo "expected formatter invocation for dir/space name.go" >&2
    exit 1
  fi

  if ! git -C "$repo" diff --cached --name-only | grep -qx 'a.go'; then
    echo "expected staged file to remain staged after formatting" >&2
    exit 1
  fi

  if ! git -C "$repo" diff --cached --name-only | grep -qx 'dir/space name.go'; then
    echo "expected staged file with spaces to remain staged after formatting" >&2
    exit 1
  fi

  if ! git -C "$repo" show :a.go | grep -q '^// formatted$'; then
    echo "expected formatted content to be staged" >&2
    exit 1
  fi

  if ! git -C "$repo" show ':dir/space name.go' | grep -q '^// formatted$'; then
    echo "expected formatted content to be staged for file with spaces" >&2
    exit 1
  fi
}

test_fails_when_staged_go_file_has_unstaged_changes() {
  local repo
  repo="$(new_repo "partial-stage")"
  local make_log="$repo/make.log"
  : > "$make_log"
  write_fake_make "$repo"

  cat > "$repo/a.go" <<'EOF'
package demo

func A() {}
EOF
  git -C "$repo" add a.go
  git -C "$repo" commit -q -m "initial"

  cat > "$repo/a.go" <<'EOF'
package demo

func A() {}
func B() {}
EOF
  git -C "$repo" add a.go

  cat >> "$repo/a.go" <<'EOF'
func C() {}
EOF

  local output
  local status=0
  output="$(
    cd "$repo"
    set +e
    MAKE_CMD="$repo/fake-make.sh" MAKE_LOG="$make_log" bash "$SCRIPT" 2>&1
    status=$?
    set -e
    echo "__STATUS__${status}"
  )"

  local exit_code="${output##*__STATUS__}"
  local stderr_text="${output%__STATUS__*}"
  if [[ "$exit_code" -eq 0 ]]; then
    echo "expected non-zero exit when staged Go files are partially staged" >&2
    exit 1
  fi

  if ! grep -q 'partially staged' <<<"$stderr_text"; then
    echo "expected partial staging error message" >&2
    exit 1
  fi

  if [[ -s "$make_log" ]]; then
    echo "formatter should not run when partial staging is detected" >&2
    exit 1
  fi
}

test_skips_when_no_go_files_are_staged
test_formats_and_restages_staged_go_files
test_fails_when_staged_go_file_has_unstaged_changes

echo "PASS"
