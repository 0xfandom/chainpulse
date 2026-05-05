.PHONY: help build test lint fmt vet tidy run-indexer run-processor run-api run-mcp docker-up docker-down clean

GO ?= go
BIN_DIR := bin

help:
	@echo "Targets:"
	@echo "  build           build all binaries to ./bin/"
	@echo "  test            go test ./..."
	@echo "  lint            go vet + go fmt -l check"
	@echo "  fmt             gofmt -w on whole tree"
	@echo "  vet             go vet ./..."
	@echo "  tidy            go mod tidy"
	@echo "  run-indexer     run indexer locally"
	@echo "  run-processor   run processor locally"
	@echo "  run-api         run api locally"
	@echo "  run-mcp         run mcp locally"
	@echo "  docker-up       docker compose up -d"
	@echo "  docker-down     docker compose down"
	@echo "  clean           remove ./bin"

build:
	mkdir -p $(BIN_DIR)
	$(GO) build -o $(BIN_DIR)/indexer   ./cmd/indexer
	$(GO) build -o $(BIN_DIR)/processor ./cmd/processor
	$(GO) build -o $(BIN_DIR)/api       ./cmd/api
	$(GO) build -o $(BIN_DIR)/mcp       ./cmd/mcp

test:
	$(GO) test ./... -race -count=1

lint: vet
	@diff -u <(echo -n) <(gofmt -l .)

fmt:
	gofmt -w .

vet:
	$(GO) vet ./...

tidy:
	$(GO) mod tidy

run-indexer:
	$(GO) run ./cmd/indexer --config config/config.toml

run-processor:
	$(GO) run ./cmd/processor --config config/config.toml

run-api:
	$(GO) run ./cmd/api --config config/config.toml

run-mcp:
	$(GO) run ./cmd/mcp --config config/config.toml

docker-up:
	docker compose up -d

docker-down:
	docker compose down

clean:
	rm -rf $(BIN_DIR)
