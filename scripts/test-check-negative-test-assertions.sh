#!/usr/bin/env bash

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SCRIPT="$ROOT/scripts/check-negative-test-assertions.sh"

tmp_dir="$(mktemp -d)"
cleanup() {
  rm -rf "$tmp_dir"
}
trap cleanup EXIT

write_file() {
  local path="$1"
  shift
  cat >"$path" <<EOF
$*
EOF
}

expect_pass() {
  local file="$1"
  if ! bash "$SCRIPT" "$file" >/dev/null 2>&1; then
    echo "expected check to pass for $file" >&2
    exit 1
  fi
}

expect_fail() {
  local file="$1"
  local output
  local status=0
  output="$(
    set +e
    bash "$SCRIPT" "$file" 2>&1
    status=$?
    set -e
    echo "__STATUS__${status}"
  )"
  local exit_code="${output##*__STATUS__}"
  local stderr_text="${output%__STATUS__*}"

  if [[ "$exit_code" -eq 0 ]]; then
    echo "expected check to fail for $file" >&2
    exit 1
  fi

  if ! grep -q "low-value negative assertion" <<<"$stderr_text"; then
    echo "expected low-value negative assertion error for $file" >&2
    exit 1
  fi
}

test_fails_for_unannotated_negative_assertion() {
  local file="$tmp_dir/unannotated_test.go"
  write_file "$file" 'package sample

import "testing"

func TestBad(t *testing.T) {
  assertNotContains(t, body, "<!doctype html>")
}'

  expect_fail "$file"
}

test_passes_for_invariant_annotated_negative_assertion() {
  local file="$tmp_dir/annotated_test.go"
  write_file "$file" 'package sample

import "testing"

func TestInvariant(t *testing.T) {
  // Invariant: HTMX fragments must not include a full document wrapper.
  assertNotContains(t, body, "<!doctype html>")
}'

  expect_pass "$file"
}

test_passes_when_file_has_no_target_assertions() {
  local file="$tmp_dir/positive_test.go"
  write_file "$file" 'package sample

import "testing"

func TestPositive(t *testing.T) {
  assertContains(t, body, "<title>Dashboard</title>")
}'

  expect_pass "$file"
}

test_passes_for_helper_definition() {
  local file="$tmp_dir/helper_definition_test.go"
  write_file "$file" 'package sample

import "testing"

func assertNotContains(t *testing.T, body string, unexpected string) {
  t.Helper()
}'

  expect_pass "$file"
}

test_fails_for_unannotated_negative_assertion
test_passes_for_invariant_annotated_negative_assertion
test_passes_when_file_has_no_target_assertions
test_passes_for_helper_definition

echo "PASS"
