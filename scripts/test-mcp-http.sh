#!/bin/bash
# Test script for MCP HTTP transport initialization sequence
# This script simulates the proper MCP protocol flow:
# 1. initialize request -> get session cookie
# 2. initialized notification -> complete initialization (using cookie)
# 3. tools/list request -> verify server is ready (using cookie)
#
# MCP spec uses cookies (not custom headers) for session management

set -e

MCP_URL="${MCP_URL:-http://localhost:3001/mcp}"
COOKIE_JAR="/tmp/mcp-cookies.txt"

# Clean up cookie jar
rm -f "$COOKIE_JAR"

echo "=== MCP HTTP Transport Test ==="
echo "Testing endpoint: $MCP_URL"
echo ""

# Step 1: Send initialize request
echo "Step 1: Sending initialize request..."
INIT_RESPONSE=$(curl -sS -c "$COOKIE_JAR" -X POST "$MCP_URL" \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "initialize",
    "params": {
      "protocolVersion": "2024-11-05",
      "clientInfo": {
        "name": "test-client",
        "version": "0.1.0"
      }
    }
  }')

echo "Initialize response: $INIT_RESPONSE"
echo ""

# Check if cookie was set
if [ ! -f "$COOKIE_JAR" ] || ! grep -q "mcp_session" "$COOKIE_JAR"; then
  echo "ERROR: No session cookie found in response"
  echo "Cookie jar contents:"
  cat "$COOKIE_JAR" 2>/dev/null || echo "(empty)"
  exit 1
fi

echo "Session cookie set (stored in $COOKIE_JAR)"
echo ""

# Step 2: Send initialized notification (cookie will be sent automatically)
echo "Step 2: Sending initialized notification..."
INITIALIZED_RESPONSE=$(curl -sS -w "\nHTTP Status: %{http_code}\n" -b "$COOKIE_JAR" -X POST "$MCP_URL" \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "initialized",
    "params": {}
  }')

echo "Initialized response: $INITIALIZED_RESPONSE"
echo ""

# Step 3: Send tools/list request to verify server is ready (cookie will be sent automatically)
echo "Step 3: Sending tools/list request..."
TOOLS_LIST_RESPONSE=$(curl -sS -b "$COOKIE_JAR" -X POST "$MCP_URL" \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 2,
    "method": "tools/list",
    "params": {}
  }')

echo "Tools list response: $TOOLS_LIST_RESPONSE"
echo ""

# Verify the response
if echo "$TOOLS_LIST_RESPONSE" | grep -q '"result"'; then
  echo "✓ Success: Server is initialized and responding to requests"
  exit 0
else
  echo "✗ Error: Server did not respond correctly"
  echo "Response: $TOOLS_LIST_RESPONSE"
  exit 1
fi
