package types

// AppConfig is the top-level configuration loaded from config.toml.
type AppConfig struct {
	App    AppSection    `toml:"app"`
	Kafka  KafkaConfig   `toml:"kafka"`
	Chains []ChainConfig `toml:"chains"`
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
