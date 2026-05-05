#!/usr/bin/env bash
# Demonstrate the MCP HTTP+SSE transport against a running chainpulse-mcp.
# Usage: BASE=http://127.0.0.1:3001 ./examples/mcp_curl_session.sh
#
# 1. Open the SSE stream in the background and capture the session id
# 2. POST a tools/list request to /messages
# 3. Read responses off the SSE stream

set -euo pipefail

BASE="${BASE:-http://127.0.0.1:3001}"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"; kill ${SSE_PID:-0} 2>/dev/null || true' EXIT

echo "==> opening SSE stream at $BASE/sse"
curl -sN "$BASE/sse" > "$TMP/stream" &
SSE_PID=$!

# Wait for the endpoint event with the session id.
SESSION=""
for _ in $(seq 1 50); do
  if grep -q "session=" "$TMP/stream" 2>/dev/null; then
    SESSION="$(grep -m1 'session=' "$TMP/stream" | sed 's/.*session=//' | tr -d '\r')"
    break
  fi
  sleep 0.1
done

if [[ -z "$SESSION" ]]; then
  echo "ERROR: did not receive session id within 5s" >&2
  exit 1
fi
echo "==> session id: $SESSION"

echo "==> POSTing tools/list"
curl -sS -X POST -H 'Content-Type: application/json' \
  --data '{"jsonrpc":"2.0","id":1,"method":"tools/list"}' \
  "$BASE/messages?session=$SESSION" >/dev/null

echo "==> waiting for response on SSE stream..."
for _ in $(seq 1 50); do
  if grep -q '"tools"' "$TMP/stream"; then
    grep '"tools"' "$TMP/stream" | head -1
    echo "==> success"
    exit 0
  fi
  sleep 0.1
done

echo "ERROR: did not receive tools/list response within 5s" >&2
exit 1
