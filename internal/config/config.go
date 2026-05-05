// Package config loads the ChainPulse TOML configuration with ${VAR}
// environment variable substitution.
package config

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/pelletier/go-toml/v2"

	"github.com/0xfandom/chainpulse/internal/types"
)

const (
	defaultClickHouseDatabase = "chainpulse"
	defaultClickHouseBatch    = 1000
	defaultClickHouseInterval = time.Second
	defaultRedisTTL           = 60 * time.Second
	defaultConsumerGroup      = "chainpulse-processor"
	defaultMaxInFlight        = 256
	defaultAPIAddr            = ":8080"
	defaultAPIGRPCAddr        = ":8081"
	defaultAPIRequestTimeout  = 10 * time.Second
	defaultAPIShutdownDrain   = 15 * time.Second
	defaultAPIRateLimit       = 100
	defaultAPIBurst           = 20
	defaultAPIWSReadBytes     = 4096
	defaultAPIWSWriteBytes    = 4096
	defaultAPIWSOriginCheck   = "strict"
	defaultMCPAddr            = ":3001"
	defaultMCPTransport       = "sse"
	defaultMCPCacheTTL        = 60 * time.Second
)

// envVarPattern matches ${VAR_NAME} placeholders. Names accept letters,
// digits, and underscores; they must not start with a digit.
var envVarPattern = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)\}`)

// ErrMissingEnvVar is returned by Substitute when a referenced env var has
// no value in the environment.
var ErrMissingEnvVar = errors.New("missing environment variable")

// Load reads the TOML file at path, loads .env via godotenv (best-effort,
// no error if .env is missing), substitutes ${VAR} references from the
// process environment, parses the result into AppConfig, and validates
// required fields.
func Load(path string) (*types.AppConfig, error) {
	// godotenv populates os.Environ() so the env-var lookups below find
	// values from .env. Failure (no .env file) is fine in production.
	_ = godotenv.Load()

	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}

	substituted, err := Substitute(string(raw))
	if err != nil {
		return nil, fmt.Errorf("substitute env vars in %s: %w", path, err)
	}

	var cfg types.AppConfig
	if err := toml.Unmarshal([]byte(substituted), &cfg); err != nil {
		return nil, fmt.Errorf("parse toml %s: %w", path, err)
	}

	applyDefaults(&cfg)

	if err := validate(&cfg); err != nil {
		return nil, fmt.Errorf("validate config %s: %w", path, err)
	}

	return &cfg, nil
}

// applyDefaults fills in zero-valued fields with their documented defaults.
// Required fields (DSN, addresses) are left empty so validate can complain.
func applyDefaults(cfg *types.AppConfig) {
	if cfg.ClickHouse.Database == "" {
		cfg.ClickHouse.Database = defaultClickHouseDatabase
	}
	if cfg.ClickHouse.BatchSize == 0 {
		cfg.ClickHouse.BatchSize = defaultClickHouseBatch
	}
	if cfg.ClickHouse.BatchInterval == 0 {
		cfg.ClickHouse.BatchInterval = types.Duration(defaultClickHouseInterval)
	}
	if cfg.Redis.DefaultTTL == 0 {
		cfg.Redis.DefaultTTL = types.Duration(defaultRedisTTL)
	}
	if cfg.Processor.ConsumerGroup == "" {
		cfg.Processor.ConsumerGroup = defaultConsumerGroup
	}
	if cfg.Processor.MaxInFlight == 0 {
		cfg.Processor.MaxInFlight = defaultMaxInFlight
	}
	if cfg.API.Addr == "" {
		cfg.API.Addr = defaultAPIAddr
	}
	if cfg.API.GRPCAddr == "" {
		cfg.API.GRPCAddr = defaultAPIGRPCAddr
	}
	if cfg.API.RequestTimeout == 0 {
		cfg.API.RequestTimeout = types.Duration(defaultAPIRequestTimeout)
	}
	if cfg.API.ShutdownDrain == 0 {
		cfg.API.ShutdownDrain = types.Duration(defaultAPIShutdownDrain)
	}
	if cfg.API.RateLimit.PerIPPerMinute == 0 {
		cfg.API.RateLimit.PerIPPerMinute = defaultAPIRateLimit
	}
	if cfg.API.RateLimit.Burst == 0 {
		cfg.API.RateLimit.Burst = defaultAPIBurst
	}
	if cfg.API.WebSocket.ReadBufferBytes == 0 {
		cfg.API.WebSocket.ReadBufferBytes = defaultAPIWSReadBytes
	}
	if cfg.API.WebSocket.WriteBufferBytes == 0 {
		cfg.API.WebSocket.WriteBufferBytes = defaultAPIWSWriteBytes
	}
	if cfg.API.WebSocket.OriginCheck == "" {
		cfg.API.WebSocket.OriginCheck = defaultAPIWSOriginCheck
	}
	if cfg.MCP.Addr == "" {
		cfg.MCP.Addr = defaultMCPAddr
	}
	if cfg.MCP.Transport == "" {
		cfg.MCP.Transport = defaultMCPTransport
	}
	if cfg.MCP.CacheTTL == 0 {
		cfg.MCP.CacheTTL = types.Duration(defaultMCPCacheTTL)
	}
}

// Substitute replaces every ${VAR} reference in src with the value of the
// matching environment variable. Returns ErrMissingEnvVar if any reference
// is unset.
func Substitute(src string) (string, error) {
	var missing []string
	out := envVarPattern.ReplaceAllStringFunc(src, func(match string) string {
		name := match[2 : len(match)-1]
		val, ok := os.LookupEnv(name)
		if !ok {
			missing = append(missing, name)
			return match
		}
		return val
	})
	if len(missing) > 0 {
		return "", fmt.Errorf("%w: %s", ErrMissingEnvVar, strings.Join(missing, ", "))
	}
	return out, nil
}

func validate(cfg *types.AppConfig) error {
	if len(cfg.Kafka.Brokers) == 0 {
		return errors.New("kafka.brokers must not be empty")
	}
	if cfg.Kafka.TopicRawEvents == "" {
		return errors.New("kafka.topic_raw_events must be set")
	}
	if len(cfg.Chains) == 0 {
		return errors.New("at least one [[chains]] block is required")
	}
	for i, c := range cfg.Chains {
		if c.ChainID == 0 {
			return fmt.Errorf("chains[%d].chain_id must be non-zero", i)
		}
		if c.Name == "" {
			return fmt.Errorf("chains[%d].name must be set", i)
		}
		if c.RPCWSS == "" {
			return fmt.Errorf("chains[%d].rpc_wss must be set (chain=%s)", i, c.Name)
		}
	}
	if cfg.ClickHouse.DSN == "" {
		return errors.New("clickhouse.dsn must be set")
	}
	if cfg.ClickHouse.BatchSize <= 0 {
		return errors.New("clickhouse.batch_size must be > 0")
	}
	if cfg.ClickHouse.BatchInterval.AsDuration() <= 0 {
		return errors.New("clickhouse.batch_interval must be > 0")
	}
	if cfg.Redis.Addr == "" {
		return errors.New("redis.addr must be set")
	}
	if cfg.API.Addr == "" {
		return errors.New("api.addr must be set")
	}
	if cfg.API.RateLimit.PerIPPerMinute < 0 {
		return errors.New("api.rate_limit.per_ip_per_minute must be >= 0")
	}
	if cfg.API.RateLimit.Burst < 0 {
		return errors.New("api.rate_limit.burst must be >= 0")
	}
	switch cfg.API.WebSocket.OriginCheck {
	case "strict", "permissive":
	default:
		return fmt.Errorf("api.websocket.origin_check must be 'strict' or 'permissive', got %q", cfg.API.WebSocket.OriginCheck)
	}
	switch cfg.MCP.Transport {
	case "stdio", "sse":
	default:
		return fmt.Errorf("mcp.transport must be 'stdio' or 'sse', got %q", cfg.MCP.Transport)
	}
	if cfg.MCP.CacheTTL.AsDuration() <= 0 {
		return errors.New("mcp.cache_ttl must be > 0")
	}
	return nil
}
