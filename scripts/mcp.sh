#!/usr/bin/env bash

# script to run the mcp server from the repo root regardless of working directory
# useful for running the MCP server from the Cursor extension
set -e

# Resolve repo root relative to this script
ROOT="$(cd "$(dirname "$0")/.." && pwd)"

cd "$ROOT"

exec go run ./cmd/mcp -- "$@"
