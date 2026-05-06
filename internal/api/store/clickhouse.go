// Package store implements the api binary's read-side persistence layer:
// ClickHouse for analytical queries and Redis for hot-path lookups.
package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// SQL queries used by ReadStore. Exposed as constants so they can be
// linted, grepped, and asserted on from tests.
const (
	sqlWalletHistory = `
SELECT chain_id, block_number, tx_hash, log_index,
       protocol, event_type, user_addr, token_a, amount_a, params, timestamp
FROM defi_events
WHERE user_addr = ?
ORDER BY timestamp DESC
LIMIT ?`

	sqlWalletDefiPositions = `
SELECT chain_id, protocol, event_type, user_addr, token_a, amount_a, params, timestamp
FROM defi_events
WHERE user_addr = ? AND event_type IN ('supply', 'borrow', 'withdraw', 'repay')
ORDER BY timestamp DESC
LIMIT ?`

	sqlTokenTransfers = `
SELECT chain_id, block_number, tx_hash, log_index,
       token, from_addr, to_addr, amount, timestamp
FROM token_transfers
WHERE token = ?
ORDER BY timestamp DESC
LIMIT ?`

	sqlProtocolStats = `
SELECT chain_id, protocol,
       count() AS event_count,
       uniqExact(user_addr) AS unique_users,
       max(timestamp) AS last_seen
FROM defi_events
WHERE protocol = ? AND timestamp >= now() - INTERVAL 24 HOUR
GROUP BY chain_id, protocol
ORDER BY chain_id`

	sqlTokenTransfersByAddress = `
SELECT chain_id, block_number, tx_hash, log_index,
       token, from_addr, to_addr, amount, timestamp
FROM token_transfers
WHERE token = ? OR from_addr = ? OR to_addr = ?
ORDER BY timestamp DESC
LIMIT ?`

	sqlWhaleTransfers = `
SELECT chain_id, block_number, tx_hash, log_index,
       token, from_addr, to_addr, amount, timestamp
FROM token_transfers
WHERE timestamp >= now() - INTERVAL ? HOUR
  AND toUInt256OrZero(amount) >= toUInt256OrZero(?)
ORDER BY toUInt256OrZero(amount) DESC, timestamp DESC
LIMIT ?`

	sqlWhaleTransfersByChain = `
SELECT chain_id, block_number, tx_hash, log_index,
       token, from_addr, to_addr, amount, timestamp
FROM token_transfers
WHERE chain_id = ?
  AND timestamp >= now() - INTERVAL ? HOUR
  AND toUInt256OrZero(amount) >= toUInt256OrZero(?)
ORDER BY toUInt256OrZero(amount) DESC, timestamp DESC
LIMIT ?`

	sqlChainRecentBlocks = `
SELECT chain_id, block_number, count() AS event_count, max(timestamp) AS latest_ts
FROM (
    SELECT chain_id, block_number, timestamp FROM defi_events WHERE chain_id = ?
    UNION ALL
    SELECT chain_id, block_number, timestamp FROM token_transfers WHERE chain_id = ?
)
GROUP BY chain_id, block_number
ORDER BY block_number DESC
LIMIT ?`
)

// HistoryRow is one row of a wallet's recent decoded events.
type HistoryRow struct {
	ChainID     uint64    `json:"chain_id"`
	BlockNumber uint64    `json:"block_number"`
	TxHash      string    `json:"tx_hash"`
	LogIndex    uint32    `json:"log_index"`
	Protocol    string    `json:"protocol"`
	EventType   string    `json:"event_type"`
	UserAddr    string    `json:"user_addr"`
	TokenA      string    `json:"token_a"`
	AmountA     string    `json:"amount_a"`
	Params      string    `json:"params"`
	Timestamp   time.Time `json:"timestamp"`
}

// PositionRow is one DeFi position event for a wallet.
type PositionRow struct {
	ChainID   uint64    `json:"chain_id"`
	Protocol  string    `json:"protocol"`
	EventType string    `json:"event_type"`
	UserAddr  string    `json:"user_addr"`
	TokenA    string    `json:"token_a"`
	AmountA   string    `json:"amount_a"`
	Params    string    `json:"params"`
	Timestamp time.Time `json:"timestamp"`
}

// TransferRow is one ERC-20 transfer.
type TransferRow struct {
	ChainID     uint64    `json:"chain_id"`
	BlockNumber uint64    `json:"block_number"`
	TxHash      string    `json:"tx_hash"`
	LogIndex    uint32    `json:"log_index"`
	Token       string    `json:"token"`
	FromAddr    string    `json:"from_addr"`
	ToAddr      string    `json:"to_addr"`
	Amount      string    `json:"amount"`
	Timestamp   time.Time `json:"timestamp"`
}

// ProtocolStat is one (chain, protocol) aggregate over the last 24h.
type ProtocolStat struct {
	ChainID     uint64    `json:"chain_id"`
	Protocol    string    `json:"protocol"`
	EventCount  uint64    `json:"event_count"`
	UniqueUsers uint64    `json:"unique_users"`
	LastSeen    time.Time `json:"last_seen"`
}

// BlockSummary is a single indexed block with the count of events seen.
type BlockSummary struct {
	ChainID     uint64    `json:"chain_id"`
	BlockNumber uint64    `json:"block_number"`
	EventCount  uint64    `json:"event_count"`
	LatestTS    time.Time `json:"latest_ts"`
}

// QueryConn is the subset of clickhouse-go's driver.Conn the read store
// uses. Defined as an interface so unit tests can substitute a fake.
type QueryConn interface {
	Query(ctx context.Context, query string, args ...any) (driver.Rows, error)
	Ping(ctx context.Context) error
	Close() error
}

// ReadStore is the api's analytical query layer over ClickHouse.
type ReadStore struct {
	conn QueryConn
}

// DialReadStore parses dsn and opens a clickhouse-go connection. PINGs
// before returning so config errors fail fast.
func DialReadStore(ctx context.Context, dsn string) (*ReadStore, error) {
	if dsn == "" {
		return nil, errors.New("read store: empty dsn")
	}
	opts, err := clickhouse.ParseDSN(dsn)
	if err != nil {
		return nil, fmt.Errorf("read store: parse dsn: %w", err)
	}
	conn, err := clickhouse.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("read store: open: %w", err)
	}
	if err := conn.Ping(ctx); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("read store: ping: %w", err)
	}
	return &ReadStore{conn: conn}, nil
}

// NewReadStore wraps an existing QueryConn. Used by tests.
func NewReadStore(conn QueryConn) *ReadStore { return &ReadStore{conn: conn} }

// Close releases the connection pool.
func (s *ReadStore) Close() error {
	if s == nil || s.conn == nil {
		return nil
	}
	return s.conn.Close()
}

// WalletHistory returns the most recent decoded events for a wallet.
func (s *ReadStore) WalletHistory(ctx context.Context, wallet string, limit int) ([]HistoryRow, error) {
	rows, err := s.conn.Query(ctx, sqlWalletHistory, wallet, limit)
	if err != nil {
		return nil, fmt.Errorf("wallet history: %w", err)
	}
	defer rows.Close()

	var out []HistoryRow
	for rows.Next() {
		var r HistoryRow
		if err := rows.Scan(&r.ChainID, &r.BlockNumber, &r.TxHash, &r.LogIndex,
			&r.Protocol, &r.EventType, &r.UserAddr, &r.TokenA, &r.AmountA, &r.Params, &r.Timestamp); err != nil {
			return nil, fmt.Errorf("wallet history scan: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// WalletDefiPositions returns DeFi position-affecting events for a wallet.
func (s *ReadStore) WalletDefiPositions(ctx context.Context, wallet string, limit int) ([]PositionRow, error) {
	rows, err := s.conn.Query(ctx, sqlWalletDefiPositions, wallet, limit)
	if err != nil {
		return nil, fmt.Errorf("wallet defi positions: %w", err)
	}
	defer rows.Close()

	var out []PositionRow
	for rows.Next() {
		var r PositionRow
		if err := rows.Scan(&r.ChainID, &r.Protocol, &r.EventType, &r.UserAddr,
			&r.TokenA, &r.AmountA, &r.Params, &r.Timestamp); err != nil {
			return nil, fmt.Errorf("wallet defi positions scan: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// TokenTransfers returns the latest transfers for a token contract.
func (s *ReadStore) TokenTransfers(ctx context.Context, token string, limit int) ([]TransferRow, error) {
	rows, err := s.conn.Query(ctx, sqlTokenTransfers, token, limit)
	if err != nil {
		return nil, fmt.Errorf("token transfers: %w", err)
	}
	defer rows.Close()

	var out []TransferRow
	for rows.Next() {
		var r TransferRow
		if err := rows.Scan(&r.ChainID, &r.BlockNumber, &r.TxHash, &r.LogIndex,
			&r.Token, &r.FromAddr, &r.ToAddr, &r.Amount, &r.Timestamp); err != nil {
			return nil, fmt.Errorf("token transfers scan: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// TokenTransfersByAddress returns recent transfers where the address
// appears as the token contract, the sender, or the recipient. Used by
// the MCP get_token_transfers tool which accepts either a wallet or a
// token contract.
func (s *ReadStore) TokenTransfersByAddress(ctx context.Context, address string, limit int) ([]TransferRow, error) {
	rows, err := s.conn.Query(ctx, sqlTokenTransfersByAddress, address, address, address, limit)
	if err != nil {
		return nil, fmt.Errorf("token transfers by address: %w", err)
	}
	defer rows.Close()

	var out []TransferRow
	for rows.Next() {
		var r TransferRow
		if err := rows.Scan(&r.ChainID, &r.BlockNumber, &r.TxHash, &r.LogIndex,
			&r.Token, &r.FromAddr, &r.ToAddr, &r.Amount, &r.Timestamp); err != nil {
			return nil, fmt.Errorf("token transfers by address scan: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// WhaleTransfers returns the largest token transfers in the trailing
// hours window where amount >= minAmount (decimal-string uint256). When
// chainID is non-zero results are scoped to that chain.
func (s *ReadStore) WhaleTransfers(ctx context.Context, hours int, minAmount string, chainID uint64, limit int) ([]TransferRow, error) {
	var rows driver.Rows
	var err error
	if chainID == 0 {
		rows, err = s.conn.Query(ctx, sqlWhaleTransfers, hours, minAmount, limit)
	} else {
		rows, err = s.conn.Query(ctx, sqlWhaleTransfersByChain, chainID, hours, minAmount, limit)
	}
	if err != nil {
		return nil, fmt.Errorf("whale transfers: %w", err)
	}
	defer rows.Close()

	var out []TransferRow
	for rows.Next() {
		var r TransferRow
		if err := rows.Scan(&r.ChainID, &r.BlockNumber, &r.TxHash, &r.LogIndex,
			&r.Token, &r.FromAddr, &r.ToAddr, &r.Amount, &r.Timestamp); err != nil {
			return nil, fmt.Errorf("whale transfers scan: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ProtocolStats returns 24h activity per chain for a protocol.
func (s *ReadStore) ProtocolStats(ctx context.Context, protocol string) ([]ProtocolStat, error) {
	rows, err := s.conn.Query(ctx, sqlProtocolStats, protocol)
	if err != nil {
		return nil, fmt.Errorf("protocol stats: %w", err)
	}
	defer rows.Close()

	var out []ProtocolStat
	for rows.Next() {
		var r ProtocolStat
		if err := rows.Scan(&r.ChainID, &r.Protocol, &r.EventCount, &r.UniqueUsers, &r.LastSeen); err != nil {
			return nil, fmt.Errorf("protocol stats scan: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ChainRecentBlocks returns the most recent indexed blocks for a chain
// with their event counts (across token_transfers + defi_events).
func (s *ReadStore) ChainRecentBlocks(ctx context.Context, chainID uint64, limit int) ([]BlockSummary, error) {
	rows, err := s.conn.Query(ctx, sqlChainRecentBlocks, chainID, chainID, limit)
	if err != nil {
		return nil, fmt.Errorf("chain blocks: %w", err)
	}
	defer rows.Close()

	var out []BlockSummary
	for rows.Next() {
		var r BlockSummary
		if err := rows.Scan(&r.ChainID, &r.BlockNumber, &r.EventCount, &r.LatestTS); err != nil {
			return nil, fmt.Errorf("chain blocks scan: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
