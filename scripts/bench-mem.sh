#!/usr/bin/env bash
# scripts/bench-mem.sh — measure resident memory for every container in
# the chainpulse-* stack. Samples for SAMPLE_SECONDS, prints a per-service
# peak / mean table plus a Go-services-only subtotal so the issue #161
# acceptance criterion ("sum of Go services <= 512MB on 4-chain config")
# is verifiable from the command line.
#
# Default samples the full stack (Kafka JVM, ClickHouse, Redis, plus the
# four ChainPulse Go binaries). Pass LITE=1 to swap the lite overlay
# (Redpanda) before sampling.
#
# Usage:
#   ./scripts/bench-mem.sh                      # full stack, 60s window
#   SAMPLE_SECONDS=120 ./scripts/bench-mem.sh   # custom window
#   LITE=1 ./scripts/bench-mem.sh               # lite overlay (Redpanda)

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

SAMPLE_SECONDS="${SAMPLE_SECONDS:-60}"
SAMPLE_INTERVAL="${SAMPLE_INTERVAL:-2}"

GO_SERVICES=(chainpulse-indexer chainpulse-processor chainpulse-api chainpulse-mcp)
INFRA_SERVICES=(chainpulse-kafka chainpulse-clickhouse chainpulse-redis)

if [[ "${LITE:-0}" == "1" ]]; then
  echo "==> lite overlay (Redpanda)"
  COMPOSE_ARGS=(-f docker-compose.yml -f docker-compose.lite.yml)
else
  echo "==> default overlay"
  COMPOSE_ARGS=(-f docker-compose.yml)
fi

echo "==> bringing stack up (skipping grafana — bench reads docker stats directly, no dashboard needed)"
docker compose "${COMPOSE_ARGS[@]}" up -d --build --scale grafana=0 >/dev/null

echo "==> waiting 20s for warmup"
sleep 20

declare -A PEAK
declare -A SUM
declare -A COUNT

end=$((SECONDS + SAMPLE_SECONDS))
echo "==> sampling for ${SAMPLE_SECONDS}s"

while (( SECONDS < end )); do
  while IFS=$'\t' read -r name mem; do
    # mem is like "123.4MiB / 8GiB" — pick the first token, strip suffix.
    raw=$(echo "$mem" | awk '{print $1}')
    case "$raw" in
      *MiB) bytes=$(awk -v v="${raw%MiB}" 'BEGIN{printf "%d", v*1024*1024}');;
      *GiB) bytes=$(awk -v v="${raw%GiB}" 'BEGIN{printf "%d", v*1024*1024*1024}');;
      *KiB) bytes=$(awk -v v="${raw%KiB}" 'BEGIN{printf "%d", v*1024}');;
      *)    bytes=0;;
    esac
    cur=${PEAK[$name]:-0}
    if (( bytes > cur )); then PEAK[$name]=$bytes; fi
    SUM[$name]=$(( ${SUM[$name]:-0} + bytes ))
    COUNT[$name]=$(( ${COUNT[$name]:-0} + 1 ))
  done < <(docker stats --no-stream --format '{{.Name}}\t{{.MemUsage}}' | grep '^chainpulse-')
  sleep "$SAMPLE_INTERVAL"
done

mib() { awk -v b="$1" 'BEGIN{printf "%.1f", b/1024/1024}'; }

print_row() {
  local svc="$1"
  local peak="${PEAK[$svc]:-0}"
  local sum="${SUM[$svc]:-0}"
  local count="${COUNT[$svc]:-1}"
  local mean=$(( sum / count ))
  printf "  %-26s peak=%7s MiB  mean=%7s MiB\n" "$svc" "$(mib "$peak")" "$(mib "$mean")"
}

echo
echo "==> Go services"
go_peak_sum=0
for s in "${GO_SERVICES[@]}"; do
  print_row "$s"
  go_peak_sum=$(( go_peak_sum + ${PEAK[$s]:-0} ))
done
printf "  %-26s sum-of-peaks=%s MiB  (target: <= 512 MiB)\n" "GO_SUBTOTAL" "$(mib "$go_peak_sum")"

echo
echo "==> Infra"
for s in "${INFRA_SERVICES[@]}"; do
  print_row "$s"
done
