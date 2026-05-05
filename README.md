# ChainPulse

Multi-chain blockchain event indexer with AI agent query layer.

Ingests on-chain events from EVM chains via WebSocket, decodes them through protocol-specific decoders (ERC-20, Uniswap V3, Aave V3, Compound V3), and exposes the data through REST, gRPC, WebSocket, and an MCP (Model Context Protocol) server that lets AI agents query live on-chain data conversationally.

## Stack

- **Go 1.22+** — core language
- **go-ethereum** — chain interaction
- **Apache Kafka** — pipeline backbone
- **ClickHouse** — analytical storage
- **Redis** — hot cache
- **Gin** — REST API
- **gRPC** — internal RPC
- **MCP** — AI agent tooling
- **Prometheus + Grafana** — observability
- **Docker Compose** — deployment

## Architecture

Four independent binaries connected by Kafka topics:

```
RPC → indexer → Kafka(raw_events) → processor → ClickHouse + Redis
                                                       ↓
                                                  api / mcp
```

## Status

Day 1 — scaffolding + multi-chain ingestion (in progress).

Initial chain: **Base mainnet**. More chains added once single-chain pipeline is stable.

## Quick Start

```bash
cp .env.example .env
# fill in BASE_WSS_URL with your Alchemy key

docker compose up -d kafka
make run-indexer
```

## License

MIT
