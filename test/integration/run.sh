#!/usr/bin/env bash
# Boot the full stack via docker compose, run the integration test,
# and tear everything back down. Used locally and from CI.
#
# Env override:
#   KEEP_STACK=1   leave compose running on success (handy for debugging)
#   COMPOSE_FILE   override the compose file path

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"

COMPOSE_FILE="${COMPOSE_FILE:-docker-compose.yml}"

cleanup() {
  if [[ "${KEEP_STACK:-0}" != "1" ]]; then
    echo "==> tearing down compose stack"
    docker compose -f "$COMPOSE_FILE" logs --tail=200 > /tmp/chainpulse-compose.log 2>&1 || true
    docker compose -f "$COMPOSE_FILE" down -v --remove-orphans >/dev/null 2>&1 || true
  fi
}
trap cleanup EXIT

echo "==> building + starting stack"
docker compose -f "$COMPOSE_FILE" up -d --build kafka clickhouse redis processor api

echo "==> waiting for /health on api"
deadline=$((SECONDS + 120))
until curl -sf http://localhost:8080/health >/dev/null 2>&1; do
  if (( SECONDS >= deadline )); then
    echo "ERROR: api /health never came up" >&2
    docker compose -f "$COMPOSE_FILE" ps
    exit 1
  fi
  sleep 2
done

echo "==> running integration test"
KAFKA_BROKERS=localhost:9092 \
KAFKA_TOPIC_RAW_EVENTS=raw_events \
API_URL=http://localhost:8080 \
  go test -v -tags integration -timeout=180s ./test/integration/...

echo "==> integration test passed"
