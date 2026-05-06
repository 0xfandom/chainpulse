#!/usr/bin/env bash
# scripts/demo.sh — bring up the full ChainPulse stack for a local
# end-to-end demo. Idempotent: re-running just rebuilds the changed
# binaries and waits for health.

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

if [[ ! -f .env ]]; then
  echo "ERROR: .env missing. Copy .env.example to .env and fill RPC keys."
  exit 1
fi

check_port_free() {
  local port="$1"
  local label="$2"
  if lsof -iTCP:"$port" -sTCP:LISTEN -n -P >/dev/null 2>&1; then
    local pid
    pid=$(lsof -tiTCP:"$port" -sTCP:LISTEN -n -P | head -1)
    local cmd
    cmd=$(ps -p "$pid" -o comm= 2>/dev/null || echo "?")
    echo "ERROR: port $port ($label) held by pid $pid ($cmd). Stop it first."
    return 1
  fi
}

echo "==> checking host ports"
check_port_free 9092 "kafka"
check_port_free 9000 "clickhouse-native"
check_port_free 8123 "clickhouse-http"
check_port_free 6379 "redis"
check_port_free 8080 "api-rest"
check_port_free 8081 "api-grpc"
check_port_free 3001 "mcp"
check_port_free 9090 "prometheus"
check_port_free 3000 "grafana"

echo "==> docker compose up -d --build"
docker compose up -d --build

wait_healthy() {
  local svc="$1"
  local timeout="${2:-90}"
  local elapsed=0
  while (( elapsed < timeout )); do
    local status
    status=$(docker inspect -f '{{.State.Health.Status}}' "$svc" 2>/dev/null || echo "absent")
    if [[ "$status" == "healthy" ]]; then
      return 0
    fi
    sleep 2
    elapsed=$((elapsed+2))
  done
  echo "ERROR: $svc did not become healthy in ${timeout}s"
  docker logs --tail 40 "$svc" || true
  return 1
}

echo "==> waiting for infra"
wait_healthy chainpulse-kafka 120
wait_healthy chainpulse-clickhouse 120
wait_healthy chainpulse-redis 60

echo "==> waiting for binaries to come up (no healthcheck; sleep 5)"
sleep 5

cat <<'EOF'

==================================================================
ChainPulse demo stack ready.

Services
  REST API           http://localhost:8080
  gRPC               localhost:8081
  MCP (SSE)          http://localhost:3001
  Prometheus         http://localhost:9090
  Grafana            http://localhost:3000  (admin / admin)

Live REST tail
  curl -s 'http://localhost:8080/v1/protocol/uniswap_v3/stats' | jq
  curl -s 'http://localhost:8080/v1/wallet/0xab5801a7d398351b8be11c439e05c5b3259aec9b/balances' | jq
  curl -s 'http://localhost:8080/v1/wallet/0xab5801a7d398351b8be11c439e05c5b3259aec9b/history?limit=5' | jq

Live WebSocket tail
  wscat -c ws://localhost:8080/v1/events/stream

Add MCP server to Claude Code
  claude mcp add chainpulse-local --transport http http://localhost:3001/sse

Sample MCP prompts (paste in Claude Code chat)
  Use chainpulse get_latest_block
  Use chainpulse get_protocol_stats for uniswap_v3
  Use chainpulse get_whale_activity hours=1 min_amount=1000000000
  Use chainpulse get_token_transfers token=0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48 limit=5

Tear down
  make demo-down
==================================================================
EOF
