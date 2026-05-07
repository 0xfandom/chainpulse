-- token_transfers
--
-- Append-only stream of every ERC-20 Transfer event the indexer recognizes.
-- Idempotency: ReplacingMergeTree dedups by full ORDER BY tuple. Since
-- (chain_id, block_number, log_index) uniquely identifies a log within a
-- chain, re-inserting the same row collapses on merge.
-- Retention: 90 days, partitioned monthly for clean drops.
CREATE TABLE IF NOT EXISTS token_transfers
(
    chain_id     UInt64,
    block_number UInt64,
    tx_hash      String,
    log_index    UInt32,
    token        String,
    from_addr    String,
    to_addr      String,
    amount       String,        -- uint256 stringified
    timestamp    DateTime,

    -- Projections sized for the two read patterns the API exercises:
    -- recent transfers from a wallet, and recent transfers to a wallet.
    -- Backs /v1/token/{addr}/transfers wallet-side queries.
    PROJECTION p_from_time
    (
        SELECT *
        ORDER BY (from_addr, timestamp)
    ),
    PROJECTION p_to_time
    (
        SELECT *
        ORDER BY (to_addr, timestamp)
    )
)
ENGINE = ReplacingMergeTree(block_number)
PARTITION BY toYYYYMM(timestamp)
ORDER BY (chain_id, block_number, log_index)
TTL timestamp + INTERVAL 90 DAY
SETTINGS index_granularity = 8192,
         non_replicated_deduplication_window = 1000;
