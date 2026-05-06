-- defi_events
--
-- Protocol-decoded events from Aave, Uniswap, Compound, etc. Same dedup
-- guarantee as token_transfers via ReplacingMergeTree on the unique
-- (chain_id, block_number, log_index) tuple. protocol is included in the
-- ORDER BY ahead of block_number to make per-protocol scans cheap.
CREATE TABLE IF NOT EXISTS defi_events
(
    chain_id     UInt64,
    block_number UInt64,
    tx_hash      String,
    log_index    UInt32,
    protocol     LowCardinality(String),
    event_type   LowCardinality(String),
    user_addr    String,
    token_a      String,
    token_b      Nullable(String),
    amount_a     String,
    amount_b     Nullable(String),
    params       String,         -- JSON blob with protocol-specific fields
    timestamp    DateTime
)
ENGINE = ReplacingMergeTree(block_number)
PARTITION BY toYYYYMM(timestamp)
ORDER BY (chain_id, protocol, block_number, log_index)
TTL timestamp + INTERVAL 90 DAY
SETTINGS index_granularity = 8192,
         non_replicated_deduplication_window = 1000;
