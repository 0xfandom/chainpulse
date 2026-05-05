# ChainPulse

Multi-chain blockchain event indexer with REST, gRPC, WebSocket, and MCP (Model Context Protocol) query surfaces.

Ingests on-chain events from EVM chains via WebSocket, decodes them through protocol-specific decoders (ERC-20, Uniswap V3, Aave V3, Compound V3), and exposes the data to dashboards (REST + WS), backend services (gRPC), and AI agents (MCP).

## Stack

- **Go 1.24** — core language
- **go-ethereum** — chain interaction
- **Apache Kafka (KRaft)** — pipeline backbone
- **ClickHouse 24.3** — analytical storage
- **Redis 7** — hot cache
- **Gin** — REST API
- **gRPC** — typed RPC
- **MCP (JSON-RPC 2.0)** — AI agent tooling, stdio + SSE transports
- **Prometheus + Grafana** — observability
- **Docker Compose** — local + single-host deploy

## Architecture

Four independent binaries connected by Kafka topics:

```
EVM RPC (eth/poly/bsc/arb)
        |
        v
  cmd/indexer  -- subscribes to newHeads, decodes logs, writes raw_events
        |
        v
  Kafka  raw_events
        |
        v
  cmd/processor  -- decodes via Registry, batches into ClickHouse,
                    updates Redis hot state, republishes decoded_events
        |
        v
  ClickHouse + Redis
        |
        v
  cmd/api   -- REST :8080, gRPC :8081, WS /v1/events
  cmd/mcp   -- stdio (Claude Desktop) or SSE :3001 (HTTP agents)
```

A separate annotated diagram lives at `examples/architecture.html` (open in a browser).

## Quick start (full stack)

```bash
cp .env.example .env
# fill in BASE_WSS_URL with an RPC URL (Alchemy, Infura, public node, etc.)

docker compose up -d --build
```

Wait for healthchecks (~30s on first boot). Then:

- REST: `curl http://localhost:8080/health`
- gRPC: `grpcurl -plaintext localhost:8081 list`
- WebSocket: `ws://localhost:8080/v1/events`
- MCP (SSE): `curl -N http://localhost:3001/sse`
- Grafana: `http://localhost:3000` (admin/admin by default)
- Prometheus: `http://localhost:9090`

## Configuration

Every binary reads the same TOML file. `${VAR}` placeholders are substituted from the environment (or `.env`). Required fields are flagged at boot.

| Section | Key | Default | Description |
|---|---|---|---|
| `[app]` | `log_level` | `info` | `debug` \| `info` \| `warn` \| `error` |
| `[app]` | `metrics_addr` | `:9100` | Prometheus scrape port (per-binary) |
| `[app]` | `health_addr` | `:9180` | `/health` liveness endpoint (set `""` to disable) |
| `[app]` | `shutdown_timeout` | `30s` | Max drain window on SIGINT/SIGTERM |
| `[kafka]` | `brokers` | required | List of `host:port` |
| `[kafka]` | `topic_raw_events` | `raw_events` | indexer → processor channel |
| `[kafka]` | `topic_decoded_events` | `decoded_events` | processor → WS channel |
| `[clickhouse]` | `dsn` | required | `clickhouse://user:pass@host:9000/db` |
| `[clickhouse]` | `batch_size` | `1000` | Rows per INSERT |
| `[clickhouse]` | `batch_interval` | `1s` | Force flush every N |
| `[redis]` | `addr` | required | `host:port` |
| `[redis]` | `default_ttl` | `60s` | Cache-aside TTL |
| `[processor]` | `consumer_group` | `chainpulse-processor` | Kafka consumer group id |
| `[processor]` | `max_in_flight` | `256` | Bounded fetch buffer |
| `[api]` | `addr` | `:8080` | REST listener |
| `[api]` | `grpc_addr` | `:8081` | gRPC listener |
| `[api]` | `request_timeout` | `10s` | Per-request handler timeout |
| `[api.cors]` | `allowed_origins` | `["*"]` | CORS whitelist |
| `[api.rate_limit]` | `per_ip_per_minute` | `100` | Token bucket per IP |
| `[api.websocket]` | `origin_check` | `strict` | `strict` \| `permissive` |
| `[mcp]` | `addr` | `:3001` | SSE listener (stdio ignores this) |
| `[mcp]` | `transport` | `sse` | `stdio` \| `sse` |
| `[mcp]` | `cache_ttl` | `60s` | Tool-result Redis TTL |
| `[mcp]` | `bearer_token` | `""` | If set, `/sse` and `/messages` require `Authorization: Bearer <token>` |
| `[[chains]]` | `chain_id` | required | Numeric chain id (e.g. 8453) |
| `[[chains]]` | `name` | required | Lowercase identifier |
| `[[chains]]` | `rpc_wss` | required | WebSocket RPC URL |
| `[[chains]]` | `confirmations` | `0` | Blocks to wait before processing |

See `config/config.example.toml` for the full template.

## REST API

| Method | Path | Description |
|---|---|---|
| GET | `/health` | Liveness probe |
| GET | `/metrics` | Prometheus metrics |
| GET | `/v1/wallets/:addr/positions` | DeFi positions (supply/borrow/etc.) |
| GET | `/v1/wallets/:addr/balances?chain_id=N` | Per-token balances |
| GET | `/v1/wallets/:addr/history?limit=N` | Recent decoded events |
| GET | `/v1/tokens/:addr/transfers?limit=N` | Recent transfers for a token |
| GET | `/v1/protocols/:name/stats` | 24h aggregates per chain |
| GET | `/v1/chains/:id/blocks?limit=N` | Latest indexed blocks |
| GET | `/v1/events` | WebSocket: live decoded events |

All wallet/token addresses are lowercase-normalized server-side.

## gRPC

Service definitions in `internal/api/grpc/proto/chainpulse.proto`. Reflection is enabled.

```
grpcurl -plaintext localhost:8081 list
grpcurl -plaintext -d '{"wallet":"0xaaaa...","chain_id":1}' localhost:8081 chainpulse.v1.Wallet/GetWalletPositions
```

## WebSocket

```
ws://localhost:8080/v1/events
```

Each connection joins an ephemeral Kafka consumer group at `LastOffset` so it sees only fresh events. Heartbeat: 30s ping / 60s pong / 5s write.

## MCP (Model Context Protocol)

Two transports.

### stdio — Claude Desktop

Add to `~/Library/Application Support/Claude/claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "chainpulse": {
      "command": "/usr/local/bin/chainpulse-mcp",
      "args": ["--config", "/etc/chainpulse/config.toml"]
    }
  }
}
```

Restart Claude Desktop. The 7 tools below appear in the tools panel.

### SSE — HTTP agents

```bash
# Stream
curl -N http://localhost:3001/sse
# (note the session id printed as: event: endpoint / data: /messages?session=<id>)

# POST a request to that session
curl -X POST -H 'Content-Type: application/json' \
     -d '{"jsonrpc":"2.0","id":1,"method":"tools/list"}' \
     'http://localhost:3001/messages?session=<id>'
```

When `bearer_token` is set, attach `Authorization: Bearer <token>` to every request.

### Tools

| Tool | Required args | Returns |
|---|---|---|
| `get_wallet_positions` | `wallet` | DeFi positions across chains |
| `get_wallet_balances` | `wallet` | Token balances (per-chain or aggregated) |
| `get_wallet_history` | `wallet` | Recent decoded events |
| `get_token_transfers` | `address` | Transfers for a token or wallet |
| `get_defi_positions` | `wallet`, `protocol` | Protocol-scoped positions |
| `get_protocol_stats` | `protocol` | 24h aggregates |
| `get_whale_activity` | `hours` | Largest transfers in window |

Each tool is JSON-Schema-validated at boot; bad input returns `-32602 InvalidParams` with `data: [{path, message}]`.

## Metrics

All binaries register against the default Prometheus registry. Notable series:

| Metric | Type | Labels |
|---|---|---|
| `raw_events_produced_total` | counter | `chain`, `event_name` |
| `block_processing_duration_seconds` | histogram | `chain` |
| `chain_listener_connected` | gauge | `chain` |
| `decoded_events_produced_total` | counter | `chain`, `protocol`, `event_type` |
| `processor_consume_errors_total` | counter | `kind` |
| `clickhouse_batch_flush_seconds` | histogram | — |
| `processor_kafka_lag_messages` | gauge | `topic`, `partition` |
| `api_request_duration_seconds` | histogram | `method`, `path`, `status` |
| `api_requests_total` | counter | `method`, `path`, `status` |
| `mcp_tool_calls_total` | counter | `tool`, `status` |
| `mcp_tool_latency_seconds` | histogram | `tool` |
| `mcp_active_sessions` | gauge | — |

A 6-panel Grafana dashboard ships pre-provisioned at `monitoring/dashboards/chainpulse.json`.

## Development

```bash
make build          # build all four binaries to ./bin/
make test           # go test -race ./...
make lint           # vet + gofmt -l
make proto          # regenerate gRPC pb files (requires protoc)
make docker-up      # boot local stack (no ChainPulse services)
make integration-test  # spin up full stack, run e2e, tear down
```

Per-binary local run:

```bash
make run-indexer
make run-processor
make run-api
make run-mcp
```

## Repository layout

```
cmd/                indexer | processor | api | mcp binaries
internal/
  config/           TOML loader + env substitution
  types/            shared event + config structs
  log/              zerolog wrapper
  monitor/          Prometheus metrics + /health + shutdown helpers
  ingestion/        indexer-side: listener, kafka producer, abi decoder
  processor/        consumer, decoders, batch + cache writers, aggregator
    protocols/      ERC-20, Uniswap V3, Aave V3, Compound V3
  api/
    store/          ClickHouse + Redis read clients
    handlers/       REST handlers
    middleware/     CORS, rate-limit, logger
    grpc/           gRPC server + proto + generated stubs
    server.go       Gin engine wiring + WS handler
  mcp/              JSON-RPC 2.0 server, tool registry, stdio + SSE transports
    tools/          7 concrete tool handlers
schema/             ClickHouse DDL (mounted by docker-entrypoint-initdb.d)
docker/             Dockerfile (multi-stage, all 4 binaries)
config/             config.example.toml + config.docker.toml
monitoring/         prometheus.yml + grafana provisioning + dashboard JSON
test/integration/   compose-driven e2e test
examples/           Claude Desktop config sample, SSE curl script
.github/workflows/  CI + nightly integration
```

## License

Apache-2.0.
