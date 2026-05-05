-- wallet_balances_mv
--
-- Materialized view over token_transfers that maintains a running balance
-- delta per (chain_id, wallet, token). UNION ALL emits two rows per
-- Transfer (debit on from_addr, credit on to_addr); SummingMergeTree
-- aggregates on background merges. Self-transfers net to zero correctly
-- because both sides are emitted.
--
-- Amount is parsed via toInt256 from the String column. Token amounts that
-- exceed Int256 range will overflow; in practice all real ERC-20 amounts
-- fit comfortably (Int256 max ~ 5.7e76, uint256 max ~ 1.15e77).
CREATE MATERIALIZED VIEW IF NOT EXISTS wallet_balances_mv
ENGINE = SummingMergeTree()
ORDER BY (chain_id, wallet, token)
AS
SELECT
    chain_id,
    from_addr           AS wallet,
    token,
    -toInt256(amount)   AS balance_delta,
    timestamp           AS last_updated
FROM token_transfers
UNION ALL
SELECT
    chain_id,
    to_addr             AS wallet,
    token,
    toInt256(amount)    AS balance_delta,
    timestamp           AS last_updated
FROM token_transfers;
