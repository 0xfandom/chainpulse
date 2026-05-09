# ChainPulse

Self-hosted, multi-chain blockchain event indexer with REST, gRPC, WebSocket, and MCP query surfaces.

ChainPulse subscribes to live blocks on EVM chains (Ethereum, Arbitrum, Polygon by default — `config/config.toml` to add more), decodes the events you care about (ERC-20 transfers, Uniswap V3 swaps, Aave V3 supply/borrow, Compound V3 actions, Lido stETH, Curve), stores them in a fast analytical database, and exposes that data to dashboards, backend services, and AI agents.

Everything runs on your own machine via Docker Compose. No data leaves your laptop. The chain data you see depends entirely on the RPC URLs you point ChainPulse at.

---

## Table of contents

1. [What is ChainPulse, plain English](#what-is-chainpulse-plain-english)
2. [What is MCP, and why is this useful](#what-is-mcp-and-why-is-this-useful)
3. [Why this stack](#why-this-stack)
4. [What you need before you start](#what-you-need-before-you-start)
5. [Step-by-step setup](#step-by-step-setup)
6. [Where the data comes from](#where-the-data-comes-from)
7. [Using ChainPulse with Claude Desktop (MCP)](#using-chainpulse-with-claude-desktop-mcp)
8. [Using ChainPulse with HTTP agents (MCP over SSE)](#using-chainpulse-with-http-agents-mcp-over-sse)
9. [Available MCP tools](#available-mcp-tools)
10. [REST and gRPC API](#rest-and-grpc-api)
11. [WebSocket stream](#websocket-stream)
12. [Web UI](#web-ui)
13. [Architecture](#architecture)
14. [Configuration reference](#configuration-reference)
15. [Metrics and dashboards](#metrics-and-dashboards)
16. [Troubleshooting](#troubleshooting)
17. [Development](#development)
18. [Repository layout](#repository-layout)
19. [License](#license)

---

## What is ChainPulse, plain English

A blockchain produces blocks. Each block contains transactions. Each transaction emits "logs" (events) that smart contracts use to announce things like "this wallet just transferred 100 USDC to that wallet" or "this user just borrowed 1 ETH on Aave."

Reading these events directly off a chain is slow and expensive. You have to know the contract addresses, the event signatures, the ABI to decode raw log data, and you have to scan every block forever.

ChainPulse does that work once and stores the result. You ask "what does wallet 0xabc... own across DeFi?" and get an answer in milliseconds because the data is already decoded, indexed, and sitting in ClickHouse on your machine.

Use cases:

- A wallet dashboard that shows positions across many chains.
- A trading bot that needs sub-second access to recent swap volume.
- A research tool that aggregates protocol stats over a 24h window.
- An AI agent (Claude, custom GPT, etc.) that can answer on-chain questions in natural language.

---

## What is MCP, and why is this useful

MCP stands for Model Context Protocol. It is an open standard, originally published by Anthropic, that lets AI assistants like Claude call external tools through a well-defined JSON-RPC interface.

If you have ever wished Claude could "look up the current Aave borrow position for this wallet" without you copy-pasting a block explorer screenshot, MCP is how you wire that up.

ChainPulse exposes 7 MCP tools (wallet positions, balances, history, token transfers, DeFi positions, protocol stats, whale activity). Once configured, Claude Desktop can call them directly, and the answers come from your local indexed copy of the chain, not a third-party API.

You do not need to be an MCP expert to use this. The setup section below walks through it as plain configuration steps.

---

## Why this stack

Each piece is here for a specific reason, not because it was trendy.

| Component | Role | Why this one |
|---|---|---|
| Go 1.24 | Core language for all four binaries | Single static binary per service, easy cross-compile, strong concurrency primitives for the WebSocket fan-out and Kafka consumers. |
| go-ethereum | EVM RPC client | Battle-tested, used by every serious Go-based chain tool. Handles WSS subscriptions, ABI decoding, and reorg-aware header logic. |
| Apache Kafka (KRaft mode) | Pipeline backbone between indexer and processor | Decouples ingest from analytics so a slow database does not stall block processing. KRaft means no separate ZooKeeper container. |
| ClickHouse 24.3 | Analytical storage | Columnar database optimised for "scan billions of rows fast." Wallet history queries that would take seconds in Postgres return in milliseconds. Idempotent inserts protect against duplicate emission during recovery. |
| Redis 7 | Hot cache for current balances and tool-call results | Sub-millisecond reads for the tools an MCP agent calls every turn. |
| Gin | REST framework | Tiny, fast, predictable middleware model. |
| gRPC | Typed RPC for backend consumers | Strict schemas via protobuf, bidirectional streaming, generated clients in any language. |
| MCP (JSON-RPC 2.0) | AI agent surface | Open standard. Works with Claude Desktop, custom HTTP agents, anything that speaks JSON-RPC over stdio or SSE. |
| Prometheus + Grafana | Observability | Standard. Pre-provisioned dashboard ships in `monitoring/`. |
| Docker Compose | Single-host orchestration | One command boots the entire stack on a developer laptop or a small VPS. |

---

## What you need before you start

Before cloning, install:

1. **Docker Desktop** (Mac/Windows) or Docker Engine + Compose v2 (Linux). Verify with `docker --version` and `docker compose version`. Compose v2 is required.
2. **Git**. Verify with `git --version`.
3. **An RPC URL for at least one chain.** ChainPulse needs a WebSocket endpoint that supports `eth_subscribe('logs', ...)`. Free options that work today:
   - Alchemy (sign up, free tier covers Ethereum / Arbitrum / Polygon and more).
   - BlockPi (per-chain free key; works for Ethereum / Arbitrum / Polygon).
   - drpc.org (public, no key required; works for most chains).
   - publicnode.com (public, no key required; reliable for Arbitrum / Polygon, sparse for Ethereum).
   - Infura, QuickNode, Chainstack on a paid tier if you need higher rate limits.
   - A self-hosted node such as Geth, Reth, or Erigon if you have the disk space.

   You can start with just one chain (Polygon is the highest-volume; Arbitrum is the lightest). Add more later.
4. **About 4 GB of free RAM** for the full stack on first boot. ClickHouse and Kafka are the heavy services.
5. **A few GB of free disk** for the ClickHouse data volume as you index more blocks.

Optional, only if you plan to use Claude Desktop:

6. **Claude Desktop** installed (free download from claude.ai).

---

## Step-by-step setup

### 1. Clone the repository

```bash
git clone https://github.com/0xfandom/chainpulse.git
cd chainpulse
```

### 2. Create your environment file

```bash
cp .env.example .env
```

Open `.env` in your editor. You will see a block per chain, like:

```
ETH_WSS_URL=wss://ethereum-rpc.publicnode.com
ETH_HTTP_URL=https://ethereum-rpc.publicnode.com
```

Replace each URL with the RPC endpoint you obtained in the previous section. **You only need to fill in the chains you actually want to index.** Every other chain will start, fail to connect, and back off with no data; harmless but noisy in logs.

If you only have one provider key (say, only for Polygon), you can:

- Leave the other chains' env vars unset, and **comment out the matching `[[chains]]` blocks in `config/config.toml`** so the indexer does not even try to start them.

### 3. Choose which chains to index

Open `config/config.example.toml` and copy it:

```bash
cp config/config.example.toml config/config.toml
```

Find the `[[chains]]` blocks near the bottom. Each one looks like:

```toml
[[chains]]
chain_id       = 1
name           = "ethereum"
rpc_wss        = "${ETH_WSS_URL}"
rpc_http       = "${ETH_HTTP_URL}"
start_block    = 0
confirmations  = 4
contracts      = []
subscribe_mode = "logs"   # "logs" (default) or "blocks"
```

Comment out (prefix lines with `#`) any chain you do not have RPC URLs for. `start_block = 0` means "start from the latest head" so you do not backfill history. `confirmations` is a reorg-safety lag; increase it for chains with known reorg patterns (BSC = 15 is a sane default; only honored when `subscribe_mode = "blocks"`).

`subscribe_mode` selects how the indexer listens:

- `"logs"` *(default)*: a single topic-filtered `eth_subscribe('logs', ...)` per chain. The provider only streams events the decoder can actually handle, which is what makes free-tier WSS endpoints viable. `confirmations` is ignored (effectively 1).
- `"blocks"`: legacy `newHeads` + `eth_getLogs` per block. Honors `confirmations` for reorg safety. Requires a paid RPC tier on busy chains.

### 4. Boot the stack

```bash
docker compose up -d --build
```

The first run downloads images, compiles the four Go binaries inside Docker, and starts everything in the background. Expect 2 to 5 minutes on first boot, ~30 seconds on subsequent boots.

Watch the boot:

```bash
docker compose ps
```

Wait until every service shows `Up` or `Up (healthy)`. The indexer healthcheck has a 90-second `start_period` because it needs to dial WSS and receive the first block from each chain.

### 5. Verify it is working

```bash
# REST liveness
curl http://localhost:8080/health

# Indexer readiness (returns 503 until at least one block has been indexed per chain)
docker compose exec -T indexer /app/indexer -healthcheck -probe-url http://localhost:9180/ready
echo $?    # should print 0 once chains have caught up

# Latest indexed block per chain (replace 8453 with whichever chain id you enabled)
curl http://localhost:8080/v1/chain/8453/blocks?limit=1
```

If those work, the pipeline is alive: blocks are flowing from RPC to indexer to Kafka to processor to ClickHouse, and the API can read them back.

### 6. Open the dashboards (optional)

- Grafana: http://localhost:3030 (login `admin` / `admin`, then change the password). The "ChainPulse" dashboard is preloaded.
- Prometheus: http://localhost:9090 (raw metrics, useful for ad-hoc queries).

### 7. Stop and clean up

```bash
docker compose down            # stops everything, keeps data
docker compose down -v         # also wipes the ClickHouse / Kafka / Redis volumes
```

---

## Where the data comes from

This is the most important thing for a new user to understand.

**ChainPulse does not provide chain data.** It indexes whatever an EVM RPC endpoint sends it. The endpoints are entirely under your control:

- The RPC URL in your `.env` decides which chain you see and which provider you depend on.
- If your URL points at Ethereum mainnet, you get Ethereum mainnet data. If it points at Sepolia, you get Sepolia data.
- If your provider rate-limits, the indexer logs `Request timeout on the free tier` or similar and backs off. The data simply stops flowing for that chain until you upgrade or switch providers.
- If you point at a self-hosted Geth, all data is served from your own infrastructure with no third-party visibility.

The downstream stack (Kafka, ClickHouse, Redis, API, MCP) all run inside Docker on your machine. **No indexed data ever leaves your host.** ChainPulse has no telemetry, no phone-home, no usage tracking.

Switching providers later: edit `.env`, then `docker compose restart indexer`. The indexer will reconnect to the new URL and resume from the current safe head; it does not re-index history.

---

## Using ChainPulse with Claude Desktop (MCP)

This walks through the stdio transport, which is the supported path for Claude Desktop.

### 1. Build a standalone MCP binary on your host

The MCP binary inside the Docker container is fine for SSE, but Claude Desktop calls a local executable directly, so you need it on your host machine:

```bash
make build              # produces ./bin/{indexer,processor,api,mcp}
sudo install -m 0755 ./bin/mcp /usr/local/bin/chainpulse-mcp
```

If you do not want to run `make` (Go not installed), you can extract the binary out of the Docker image instead:

```bash
docker compose cp mcp:/app/mcp /tmp/chainpulse-mcp
sudo install -m 0755 /tmp/chainpulse-mcp /usr/local/bin/chainpulse-mcp
```

### 2. Make sure ClickHouse and Redis are reachable from your host

The default `config.toml` points at `localhost:9000` (ClickHouse) and `localhost:6379` (Redis). Docker Compose already maps those ports, so this works out of the box. If you changed the compose file, set `CLICKHOUSE_DSN` and `REDIS_ADDR` env vars in the Claude config below.

### 3. Tell Claude Desktop about ChainPulse

Edit Claude's config file:

- Mac: `~/Library/Application Support/Claude/claude_desktop_config.json`
- Windows: `%APPDATA%\Claude\claude_desktop_config.json`
- Linux: `~/.config/Claude/claude_desktop_config.json`

If the file does not exist, create it. Add (or merge) this section:

```json
{
  "mcpServers": {
    "chainpulse": {
      "command": "/usr/local/bin/chainpulse-mcp",
      "args": ["--config", "/etc/chainpulse/config.toml"],
      "env": {
        "CLICKHOUSE_DSN": "clickhouse://default:@localhost:9000/chainpulse",
        "REDIS_ADDR": "localhost:6379"
      }
    }
  }
}
```

If your `config.toml` lives somewhere other than `/etc/chainpulse/config.toml`, change the `--config` path. A copy at `./config/config.toml` (inside the cloned repo) works too if you give the absolute path.

A ready-made sample lives at `examples/claude_desktop_config.json`.

### 4. Restart Claude Desktop

Quit fully (Cmd+Q on Mac) and reopen. The 7 ChainPulse tools should appear in the tools panel (the small slider icon under the chat input). If they do not, click the icon, look for an error message, and check the troubleshooting section.

### 5. Try it

Ask Claude something like:

> "Using ChainPulse, what are the recent USDC transfers on Polygon for `0x...`?"

Claude will pick the `get_token_transfers` or `get_wallet_history` tool, call it, and answer with live data from your indexer.

---

## Using ChainPulse with HTTP agents (MCP over SSE)

If you are building a custom agent or using something other than Claude Desktop, the MCP server also speaks SSE on `:3001`.

```bash
# 1. Open an SSE stream and grab the session id printed in the first event.
curl -N http://localhost:3001/sse
# event: endpoint
# data: /messages?session=01HXXXXXXXXXXXXXXXXXXXX

# 2. POST JSON-RPC requests to that session.
curl -X POST -H 'Content-Type: application/json' \
     -d '{"jsonrpc":"2.0","id":1,"method":"tools/list"}' \
     'http://localhost:3001/messages?session=01HXXXXXXXXXXXXXXXXXXXX'
```

To require authentication, set `bearer_token` in `[mcp]` (or via `MCP_BEARER_TOKEN` in `.env`) and send `Authorization: Bearer <token>` on every request.

A full curl session script lives at `examples/mcp_curl_session.sh`.

---

## Available MCP tools

| Tool | Required arguments | Returns |
|---|---|---|
| `get_wallet_positions` | `wallet` | DeFi positions for a wallet across protocols and chains. |
| `get_wallet_balances` | `wallet` | Token balances, optionally scoped to a chain. |
| `get_wallet_history` | `wallet` | Recent decoded events for a wallet. |
| `get_token_transfers` | `address` | Recent transfers for a token contract or a wallet. |
| `get_defi_positions` | `wallet`, `protocol` | Positions scoped to a single protocol. |
| `get_protocol_stats` | `protocol` | 24-hour aggregates per protocol per chain. |
| `get_whale_activity` | `hours` | Largest transfers in the requested window. |

Inputs are JSON-Schema-validated at server boot. Bad inputs return `-32602 InvalidParams` with a `data: [{path, message}]` array so an agent can self-correct.

---

## REST and gRPC API

REST endpoints (Gin server on `:8080`):

| Method | Path | Description |
|---|---|---|
| GET | `/health` | Liveness probe |
| GET | `/metrics` | Prometheus metrics |
| GET | `/v1/wallet/:address/positions` | DeFi positions |
| GET | `/v1/wallet/:address/balances?chain_id=N` | Per-token balances |
| GET | `/v1/wallet/:address/history?limit=N` | Recent decoded events |
| GET | `/v1/token/:address/transfers?limit=N` | Recent transfers for a token |
| GET | `/v1/protocol/:name/stats` | 24h aggregates per chain |
| GET | `/v1/chain/:id/blocks?limit=N` | Latest indexed blocks |
| GET | `/v1/events/stream` | WebSocket: live decoded events |

All wallet and token addresses are lowercase-normalised server-side.

gRPC (`:8081`) exposes the same surface with typed responses. Reflection is enabled:

```bash
grpcurl -plaintext localhost:8081 list
grpcurl -plaintext -d '{"wallet":"0xaaaa...","chain_id":1}' \
        localhost:8081 chainpulse.v1.Wallet/GetWalletPositions
```

---

## WebSocket stream

```
ws://localhost:8080/v1/events
```

Each connection joins an ephemeral Kafka consumer group at `LastOffset`, so it only receives events that arrive after the connection was opened. Heartbeat: 30s ping, 60s pong, 5s write timeout.

---

## Web UI

A Next.js explorer that sits on top of the same ClickHouse + Prometheus stack. Lives in `web/` and is shipped as the `ui` service in `docker-compose.yml`.

What it shows:

- Live whale feed across the indexed chains
- Wallet, transaction, and block lookup (raw RPC fallback for any tx/block, even ones the decoders skip)
- Per-protocol pages with chain-by-chain breakdown
- Search bar that parses `whales last 6h on ethereum`, `block 18000000`, raw addresses, raw tx hashes
- Light + dark themes

### Run via Docker (recommended)

```bash
cp .env.example .env   # fill in HTTP RPC keys (ETH_HTTP_URL, POLY_HTTP_URL, ARB_HTTP_URL)
docker compose up -d ui
open http://localhost:3000
```

The container talks to `clickhouse` and `prometheus` over the compose network; HTTP RPC URLs are read from the same `.env` the indexer uses.

### Run for local development

```bash
cd web
cp .env.example .env.local   # point at your local ClickHouse + Prometheus + RPC URLs
npm install
npm run dev
```

`npm run dev` boots Next.js on `http://localhost:3000`. The API routes inside `src/app/api/**` are server-side and read from `CLICKHOUSE_URL`, `PROMETHEUS_URL`, and the `*_RPC_URL` env vars.

---

## Architecture

Four independent binaries connected by Kafka topics:

```
EVM RPC (eth/arb/poly by default; add chains in config.toml)
        |
        v
  cmd/indexer  -- topic-filtered eth_subscribe('logs', ...), decodes, writes raw_events
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

An annotated diagram lives at `examples/architecture.html` (open in a browser).

The indexer survives transient failures via three mechanisms:

- **Readiness probe.** `/ready` on `:9180` flips to 503 if any chain head is older than `readiness_head_timeout`. Docker auto-restarts a stalled indexer.
- **WSS head watchdog.** If no header arrives within `head_timeout` (default 90s), the listener forces a reconnect even if the WS subscription has not surfaced an error.
- **Kafka publish fail-fast.** If publishing fails for `kafka_publish_fail_threshold` consecutive blocks, the indexer exits non-zero so docker restarts the container instead of silently dropping events.

---

## Configuration reference

Every binary reads the same TOML file. `${VAR}` placeholders are substituted from the environment (or `.env`). Required fields are flagged at boot.

| Section | Key | Default | Description |
|---|---|---|---|
| `[app]` | `log_level` | `info` | `debug` \| `info` \| `warn` \| `error` |
| `[app]` | `metrics_addr` | `:9100` | Prometheus scrape port (per-binary) |
| `[app]` | `health_addr` | `:9180` | `/health` (liveness) and `/ready` (readiness) endpoints |
| `[app]` | `shutdown_timeout` | `30s` | Max drain window on SIGINT/SIGTERM |
| `[app]` | `readiness_head_timeout` | `60s` | Indexer-only: `/ready` returns 503 if any chain head is older than this |
| `[app]` | `kafka_publish_fail_threshold` | `5` | Indexer-only: exit non-zero after N consecutive blocks with kafka publish failures |
| `[kafka]` | `brokers` | required | List of `host:port` |
| `[kafka]` | `topic_raw_events` | `raw_events` | indexer to processor channel |
| `[kafka]` | `topic_decoded_events` | `decoded_events` | processor to WS channel |
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
| `[[chains]]` | `chain_id` | required | Numeric chain id (e.g. 1, 137, 42161) |
| `[[chains]]` | `name` | required | Lowercase identifier |
| `[[chains]]` | `rpc_wss` | required | WebSocket RPC URL |
| `[[chains]]` | `rpc_http` | required | HTTP RPC URL (fallback / log filter) |
| `[[chains]]` | `confirmations` | `0` | Blocks to wait before processing (reorg safety, only honored when `subscribe_mode = "blocks"`) |
| `[[chains]]` | `head_timeout` | `90s` | Force WSS reconnect if no log activity arrives in this window |
| `[[chains]]` | `subscribe_mode` | `"logs"` | `"logs"` = topic-filtered `eth_subscribe('logs', ...)` (free-tier friendly, ignores `confirmations`); `"blocks"` = legacy `newHeads` + `eth_getLogs` per block (heavy on RPC, honors `confirmations`) |

Full template at `config/config.example.toml`.

---

## Metrics and dashboards

All binaries register against the default Prometheus registry. Notable series:

| Metric | Type | Labels |
|---|---|---|
| `raw_events_produced_total` | counter | `chain`, `event_name` |
| `block_processing_duration_seconds` | histogram | `chain` |
| `chain_listener_connected` | gauge | `chain` |
| `chain_listener_head_last_seen_timestamp_seconds` | gauge | `chain` |
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

---

## Troubleshooting

**`docker compose up` fails with port already in use.**
Some other process is using one of `8080`, `8081`, `8123`, `9000`, `9092`, `29092`, `3030`, `3011`, `6379`, or `9090`. Stop the other process, or remap the port in `docker-compose.yml`.

**Indexer logs `connection refused` against `kafka:9092` for the first 30 seconds.**
Expected on cold start. The kafka healthcheck has a `start_period` to absorb this; if errors persist past 1 minute, run `docker compose logs kafka` to see why the broker did not come up.

**`/ready` returns 503 forever.**
At least one chain has not produced a head. Check `docker compose logs indexer | grep -i error` for RPC issues (rate limit, bad URL, expired key). Verify the URL works: `wscat -c "$ETH_WSS_URL"`.

**MCP tools do not appear in Claude Desktop.**
Quit Claude fully (Cmd+Q on Mac) and reopen. Check `~/Library/Logs/Claude/mcp-server-chainpulse.log` for stderr from the binary. Common causes: wrong path in `command`, missing `config.toml`, ClickHouse not reachable from the host.

**Free-tier RPC keeps timing out.**
Free tiers throttle aggressively. Either upgrade your provider plan, switch to a self-hosted node, or remove the affected chain from `config.toml` until you have better RPC.

**`docker compose down -v` does not actually free disk.**
Docker Desktop manages a separate VM disk on Mac/Windows. Use Docker Desktop > Settings > Resources > Disk to reclaim space, or `docker system prune -a --volumes`.

**ClickHouse `Memory limit exceeded`.**
Default memory caps in `docker/clickhouse/lite-memory.xml` are tuned for laptops. For production volumes, edit that file or remove the volume mount in `docker-compose.yml`.

---

## Development

```bash
make build               # build all four binaries to ./bin/
make test                # go test -race ./...
make lint                # vet + gofmt -l
make proto               # regenerate gRPC pb files (requires protoc)
make docker-up           # boot local stack (no ChainPulse services)
make integration-test    # spin up full stack, run e2e, tear down
```

Per-binary local run:

```bash
make run-indexer
make run-processor
make run-api
make run-mcp
```

CI runs gofmt, vet, build, and `go test -race -count=1` on every push and PR. Integration tests run on a separate workflow against a real compose stack.

---

## Repository layout

```
cmd/                indexer | processor | api | mcp binaries
internal/
  config/           TOML loader + env substitution
  types/            shared event + config structs
  log/              zerolog wrapper
  monitor/          Prometheus metrics + /health + /ready + shutdown helpers
  ingestion/        indexer-side: listener (with WSS watchdog), kafka producer, abi decoder
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
docker/             Dockerfile (multi-stage, all 4 binaries) + ClickHouse memory overlay
config/             config.example.toml + config.docker.toml
monitoring/         prometheus.yml + grafana provisioning + dashboard JSON
web/                Next.js 16 explorer UI (API routes hit ClickHouse + Prometheus + RPC)
test/integration/   compose-driven e2e test
examples/           Claude Desktop config sample, SSE curl script, architecture diagram
.github/workflows/  CI (Go) + Web (Next.js) + nightly integration
```

---

## License

Apache-2.0.
