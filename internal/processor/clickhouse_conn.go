package processor

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// clickhouseConn implements BatchConn using clickhouse-go/v2.
type clickhouseConn struct {
	conn driver.Conn
}

// DialClickHouse opens a native-protocol connection from a DSN. Returns
// a BatchConn implementation suitable for BatchWriter.
func DialClickHouse(ctx context.Context, dsn string) (BatchConn, error) {
	if dsn == "" {
		return nil, errors.New("clickhouse: empty dsn")
	}
	opts, err := clickhouse.ParseDSN(dsn)
	if err != nil {
		return nil, fmt.Errorf("clickhouse: parse dsn: %w", err)
	}
	conn, err := clickhouse.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("clickhouse: open: %w", err)
	}
	if err := conn.Ping(ctx); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("clickhouse: ping: %w", err)
	}
	return &clickhouseConn{conn: conn}, nil
}

// InsertTransfers runs a single batched INSERT into token_transfers.
//
// Idempotency: a deterministic insert_deduplication_token is derived
// from the batch's (chain_id, block_number, log_index) tuples so that
// retrying the exact same batch is dropped server-side. The dedup
// window (`non_replicated_deduplication_window`) is set on the table.
// `deduplicate_blocks_in_dependent_materialized_views=1` cascades the
// dedup decision to the wallet_balances MV so duplicate transfers
// don't double-count balances.
func (c *clickhouseConn) InsertTransfers(ctx context.Context, rows []TransferRow) error {
	if len(rows) == 0 {
		return nil
	}
	keys := make([]string, 0, len(rows))
	for _, r := range rows {
		keys = append(keys, strconv.FormatUint(r.ChainID, 10)+":"+strconv.FormatUint(r.BlockNumber, 10)+":"+strconv.FormatUint(uint64(r.LogIndex), 10))
	}
	ctx = withDedupSettings(ctx, "tt:"+hashKeys(keys))

	batch, err := c.conn.PrepareBatch(ctx, "INSERT INTO token_transfers")
	if err != nil {
		return fmt.Errorf("clickhouse: prepare token_transfers: %w", err)
	}
	for _, r := range rows {
		if err := batch.Append(
			r.ChainID,
			r.BlockNumber,
			r.TxHash,
			r.LogIndex,
			r.Token,
			r.FromAddr,
			r.ToAddr,
			r.Amount,
			r.Timestamp,
		); err != nil {
			return fmt.Errorf("clickhouse: append token_transfers: %w", err)
		}
	}
	if err := batch.Send(); err != nil {
		return fmt.Errorf("clickhouse: send token_transfers: %w", err)
	}
	return nil
}

// InsertDefiEvents runs a single batched INSERT into defi_events. Same
// dedup contract as InsertTransfers; the namespace prefix in the
// dedup token (de: vs tt:) prevents cross-table collisions.
func (c *clickhouseConn) InsertDefiEvents(ctx context.Context, rows []DefiEventRow) error {
	if len(rows) == 0 {
		return nil
	}
	keys := make([]string, 0, len(rows))
	for _, r := range rows {
		keys = append(keys, strconv.FormatUint(r.ChainID, 10)+":"+strconv.FormatUint(r.BlockNumber, 10)+":"+strconv.FormatUint(uint64(r.LogIndex), 10))
	}
	ctx = withDedupSettings(ctx, "de:"+hashKeys(keys))

	batch, err := c.conn.PrepareBatch(ctx, "INSERT INTO defi_events")
	if err != nil {
		return fmt.Errorf("clickhouse: prepare defi_events: %w", err)
	}
	for _, r := range rows {
		var tokenB any
		if r.TokenB.Valid {
			tokenB = r.TokenB.String
		}
		var amountB any
		if r.AmountB.Valid {
			amountB = r.AmountB.String
		}
		if err := batch.Append(
			r.ChainID,
			r.BlockNumber,
			r.TxHash,
			r.LogIndex,
			r.Protocol,
			r.EventType,
			r.UserAddr,
			r.TokenA,
			tokenB,
			r.AmountA,
			amountB,
			r.Params,
			r.Timestamp,
		); err != nil {
			return fmt.Errorf("clickhouse: append defi_events: %w", err)
		}
	}
	if err := batch.Send(); err != nil {
		return fmt.Errorf("clickhouse: send defi_events: %w", err)
	}
	return nil
}

// Close releases the connection pool.
func (c *clickhouseConn) Close() error {
	if c == nil || c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

// withDedupSettings returns ctx augmented with the per-INSERT settings
// that turn ClickHouse's block-deduplication on for this batch and
// cascade the decision to dependent materialized views.
func withDedupSettings(ctx context.Context, token string) context.Context {
	return clickhouse.Context(ctx, clickhouse.WithSettings(clickhouse.Settings{
		"insert_deduplication_token":                         token,
		"deduplicate_blocks_in_dependent_materialized_views": uint8(1),
	}))
}

// hashKeys returns the SHA-256 hex digest of the joined keys. Used as
// the deterministic insert_deduplication_token for a batch.
func hashKeys(keys []string) string {
	h := sha256.New()
	for _, k := range keys {
		h.Write([]byte(k))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}
