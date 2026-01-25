#!/usr/bin/env bash

# Script to start both the gRPC server and MCP HTTP server for CI/testing.
# This script is separate from scripts/mcp.sh because:
#   - scripts/mcp.sh runs MCP server in stdio mode for local development/Cursor integration
#   - This script starts both gRPC server and MCP HTTP server for CI/testing scenarios
set -e

# Resolve repo root relative to this script
ROOT="$(cd "$(dirname "$0")/.." && pwd)"

cd "$ROOT"

# Cleanup function to kill background jobs
cleanup() {
  if [ -n "$GRPC_PID" ]; then
    kill "$GRPC_PID" 2>/dev/null || true
  fi
  if [ -n "$MCP_PID" ]; then
    kill "$MCP_PID" 2>/dev/null || true
  fi
  # Kill any remaining background jobs
  jobs -p | xargs -r kill 2>/dev/null || true
}

# Set up trap to cleanup on exit
trap cleanup EXIT INT TERM

# Start gRPC server in background
echo "Starting gRPC server on port 8080..."
go run ./cmd/server -port=8080 &
GRPC_PID=$!

# Wait a moment for gRPC server to start
sleep 2

# Start MCP HTTP server in background
echo "Starting MCP HTTP server on port 3001..."
go run ./cmd/mcp -transport=http -http-addr=localhost:3001 -addr=localhost:8080 &
MCP_PID=$!

# Wait for both processes
wait
