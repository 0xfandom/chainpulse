package types

import (
	"fmt"
	"time"
)

// AppConfig is the top-level configuration loaded from config.toml.
type AppConfig struct {
	App        AppSection       `toml:"app"`
	Kafka      KafkaConfig      `toml:"kafka"`
	Chains     []ChainConfig    `toml:"chains"`
	ClickHouse ClickHouseConfig `toml:"clickhouse"`
	Redis      RedisConfig      `toml:"redis"`
	Processor  ProcessorConfig  `toml:"processor"`
	API        APIConfig        `toml:"api"`
	MCP        MCPConfig        `toml:"mcp"`
}

// AppSection holds process-wide settings.
type AppSection struct {
	LogLevel    string `toml:"log_level"`
	MetricsAddr string `toml:"metrics_addr"`
}

// KafkaConfig holds connection and topic settings for the Kafka pipeline.
type KafkaConfig struct {
	Brokers              []string `toml:"brokers"`
	TopicRawEvents       string   `toml:"topic_raw_events"`
	TopicDecodedEvents   string   `toml:"topic_decoded_events"`
	TopicPositionsUpdate string   `toml:"topic_positions_update"`
	ClientID             string   `toml:"client_id"`
}

// ChainConfig holds the per-chain ingestion parameters.
type ChainConfig struct {
	ChainID       uint64   `toml:"chain_id"`
	Name          string   `toml:"name"`
	RPCWSS        string   `toml:"rpc_wss"`
	RPCHTTP       string   `toml:"rpc_http"`
	StartBlock    uint64   `toml:"start_block"`
	Confirmations uint64   `toml:"confirmations"`
	Contracts     []string `toml:"contracts"`
}

// ClickHouseConfig holds the analytical-store connection + batching policy.
type ClickHouseConfig struct {
	DSN           string   `toml:"dsn"`
	Database      string   `toml:"database"`
	BatchSize     int      `toml:"batch_size"`
	BatchInterval Duration `toml:"batch_interval"`
}

// RedisConfig holds the hot-cache connection + TTL policy.
type RedisConfig struct {
	Addr       string   `toml:"addr"`
	Password   string   `toml:"password"`
	DB         int      `toml:"db"`
	DefaultTTL Duration `toml:"default_ttl"`
}

// ProcessorConfig holds the consumer-side knobs for the processor binary.
type ProcessorConfig struct {
	ConsumerGroup string `toml:"consumer_group"`
	MaxInFlight   int    `toml:"max_in_flight"`
}

// APIConfig holds the api binary's runtime settings.
type APIConfig struct {
	Addr           string             `toml:"addr"`
	GRPCAddr       string             `toml:"grpc_addr"`
	RequestTimeout Duration           `toml:"request_timeout"`
	ShutdownDrain  Duration           `toml:"shutdown_drain"`
	CORS           APICORSConfig      `toml:"cors"`
	RateLimit      APIRateLimitConfig `toml:"rate_limit"`
	WebSocket      APIWSConfig        `toml:"websocket"`
}

// APICORSConfig drives the CORS middleware.
type APICORSConfig struct {
	AllowedOrigins   []string `toml:"allowed_origins"`
	AllowCredentials bool     `toml:"allow_credentials"`
}

// APIRateLimitConfig drives the per-IP rate limiter.
type APIRateLimitConfig struct {
	PerIPPerMinute int `toml:"per_ip_per_minute"`
	Burst          int `toml:"burst"`
}

// APIWSConfig drives the WebSocket server settings.
type APIWSConfig struct {
	ReadBufferBytes  int    `toml:"read_buffer_bytes"`
	WriteBufferBytes int    `toml:"write_buffer_bytes"`
	OriginCheck      string `toml:"origin_check"` // "strict" | "permissive"
}

// MCPConfig holds the mcp binary's runtime settings.
type MCPConfig struct {
	Addr      string   `toml:"addr"`
	Transport string   `toml:"transport"` // "stdio" | "sse"
	CacheTTL  Duration `toml:"cache_ttl"`
}

// Duration is a time.Duration wrapper that parses from TOML strings via
// time.ParseDuration. Allows config like batch_interval = "1s".
type Duration time.Duration

// UnmarshalText implements encoding.TextUnmarshaler so go-toml v2 picks up
// the parser.
func (d *Duration) UnmarshalText(text []byte) error {
	if len(text) == 0 {
		*d = 0
		return nil
	}
	parsed, err := time.ParseDuration(string(text))
	if err != nil {
		return fmt.Errorf("parse duration %q: %w", text, err)
	}
	*d = Duration(parsed)
	return nil
}

// MarshalText implements encoding.TextMarshaler for symmetric serialization.
func (d Duration) MarshalText() ([]byte, error) {
	return []byte(time.Duration(d).String()), nil
}

// AsDuration returns the value as a time.Duration.
func (d Duration) AsDuration() time.Duration { return time.Duration(d) }
