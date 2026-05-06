# syntax=docker/dockerfile:1.6
# Multi-stage build for the four ChainPulse binaries. One build stage
# compiles every binary; per-binary final stages copy a single static
# binary into a distroless runtime image.

# ---------- build ----------
FROM golang:1.25-alpine AS build

ARG VERSION=dev

WORKDIR /src

# Cache module downloads.
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOFLAGS=-trimpath

RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    go build -ldflags="-s -w -X main.version=${VERSION}" -o /out/indexer   ./cmd/indexer && \
    go build -ldflags="-s -w -X main.version=${VERSION}" -o /out/processor ./cmd/processor && \
    go build -ldflags="-s -w -X main.version=${VERSION}" -o /out/api       ./cmd/api && \
    go build -ldflags="-s -w -X main.version=${VERSION}" -o /out/mcp       ./cmd/mcp

# ---------- runtime base (shared) ----------
FROM gcr.io/distroless/static-debian12:nonroot AS runtime-base
USER nonroot:nonroot
WORKDIR /app

# ---------- indexer ----------
FROM runtime-base AS indexer
COPY --from=build /out/indexer /app/indexer
EXPOSE 9100
ENTRYPOINT ["/app/indexer"]
CMD ["--config", "/etc/chainpulse/config.toml"]

# ---------- processor ----------
FROM runtime-base AS processor
COPY --from=build /out/processor /app/processor
EXPOSE 9101
ENTRYPOINT ["/app/processor"]
CMD ["--config", "/etc/chainpulse/config.toml"]

# ---------- api ----------
FROM runtime-base AS api
COPY --from=build /out/api /app/api
EXPOSE 8080 8081 9102
ENTRYPOINT ["/app/api"]
CMD ["--config", "/etc/chainpulse/config.toml"]

# ---------- mcp ----------
FROM runtime-base AS mcp
COPY --from=build /out/mcp /app/mcp
EXPOSE 3001 9103
ENTRYPOINT ["/app/mcp"]
CMD ["--config", "/etc/chainpulse/config.toml"]
