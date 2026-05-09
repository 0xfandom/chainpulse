import { createClient } from "@clickhouse/client";

let _client: ReturnType<typeof createClient> | null = null;

export function ch() {
  if (_client) return _client;
  _client = createClient({
    url: process.env.CLICKHOUSE_URL ?? "http://localhost:8123",
    database: process.env.CLICKHOUSE_DB ?? "chainpulse",
    username: process.env.CLICKHOUSE_USER ?? "default",
    password: process.env.CLICKHOUSE_PASSWORD ?? "",
  });
  return _client;
}

export const SUPPORTED_CHAIN_IDS = [1, 137, 42161] as const;
const SUPPORTED_CHAINS_SQL = `chain_id IN (${SUPPORTED_CHAIN_IDS.join(",")})`;

const LIDO_STETH: Record<number, string> = {
  1: "0xae7ab96520DE3A18E5e111B5EaAb095312D7fE84",
};

type HydratableRow = {
  chain_id: number | string;
  protocol?: string;
  event_type?: string;
  user_addr?: string;
  token_a?: string;
  amount_a?: string;
  params?: string;
};

function hydrate<T extends HydratableRow>(rows: T[]): T[] {
  return rows.map((r) => {
    let p: Record<string, unknown> = {};
    try { p = JSON.parse(r.params ?? "{}"); } catch {}
    const out: T = { ...r };
    const cid = typeof r.chain_id === "string" ? parseInt(r.chain_id, 10) : r.chain_id;

    if (r.protocol === "lido") {
      if (!r.token_a) out.token_a = LIDO_STETH[cid] ?? "";
      if (!r.user_addr && typeof p.sender === "string") out.user_addr = p.sender;
    }
    if (r.protocol === "curve") {
      if (!r.user_addr && typeof p.buyer === "string") out.user_addr = p.buyer;
      if (!r.amount_a) {
        if (typeof p.tokens_sold === "string") out.amount_a = p.tokens_sold;
        else if (typeof p.tokens_bought === "string") out.amount_a = p.tokens_bought;
      }
    }
    return out;
  });
}

export async function query<T = Record<string, unknown>>(sql: string, params?: Record<string, unknown>): Promise<T[]> {
  const r = await ch().query({
    query: sql,
    query_params: params,
    format: "JSONEachRow",
  });
  return (await r.json()) as T[];
}

export type WhaleRow = {
  chain_id: number;
  block_number: number;
  tx_hash: string;
  log_index: number;
  protocol: string;
  event_type: string;
  user_addr: string;
  token_a: string;
  token_b: string | null;
  amount_a: string;
  amount_b: string | null;
  params: string;
  timestamp: string;
};

export async function recentWhales(opts: { chainId?: number; minutes?: number; limit?: number; protocol?: string } = {}) {
  const { chainId, minutes = 1440, limit = 100, protocol } = opts;
  const where: string[] = [SUPPORTED_CHAINS_SQL, `timestamp >= now() - INTERVAL {minutes:UInt32} MINUTE`];
  const params: Record<string, unknown> = { minutes, limit };
  if (chainId !== undefined) {
    where.push(`chain_id = {chainId:UInt64}`);
    params.chainId = chainId;
  }
  if (protocol) {
    where.push(`protocol = {protocol:String}`);
    params.protocol = protocol;
  }
  const sql = `
    SELECT chain_id, block_number, tx_hash, log_index, protocol, event_type,
           user_addr, token_a, token_b, amount_a, amount_b, params, timestamp
    FROM defi_events
    WHERE ${where.join(" AND ")}
    ORDER BY timestamp DESC
    LIMIT {limit:UInt32}
  `;
  return hydrate(await query<WhaleRow>(sql, params));
}

export async function walletActivity(addr: string, limit = 200) {
  const sql = `
    SELECT chain_id, block_number, tx_hash, log_index, protocol, event_type,
           token_a, token_b, amount_a, amount_b, params, timestamp
    FROM defi_events
    WHERE ${SUPPORTED_CHAINS_SQL} AND lower(user_addr) = lower({addr:String})
    ORDER BY timestamp DESC
    LIMIT {limit:UInt32}
  `;
  return hydrate(await query<WhaleRow>(sql, { addr, limit }));
}

export async function walletTransfers(addr: string, limit = 200) {
  const sql = `
    SELECT chain_id, block_number, tx_hash, log_index, token, from_addr, to_addr, amount, timestamp
    FROM token_transfers
    WHERE ${SUPPORTED_CHAINS_SQL} AND (lower(from_addr) = lower({addr:String}) OR lower(to_addr) = lower({addr:String}))
    ORDER BY timestamp DESC
    LIMIT {limit:UInt32}
  `;
  return query(sql, { addr, limit });
}

export async function protocolStats(minutes = 1440) {
  const sql = `
    SELECT protocol, event_type, count() AS events, uniq(user_addr) AS users
    FROM defi_events
    WHERE ${SUPPORTED_CHAINS_SQL} AND timestamp >= now() - INTERVAL {minutes:UInt32} MINUTE
    GROUP BY protocol, event_type
    ORDER BY events DESC
  `;
  return query<{ protocol: string; event_type: string; events: string; users: string }>(sql, { minutes });
}

export async function chainSplit(minutes = 1440) {
  const sql = `
    SELECT chain_id, count() AS events
    FROM defi_events
    WHERE ${SUPPORTED_CHAINS_SQL} AND timestamp >= now() - INTERVAL {minutes:UInt32} MINUTE
    GROUP BY chain_id
    ORDER BY events DESC
  `;
  return query<{ chain_id: number; events: string }>(sql, { minutes });
}

export async function protocolStatsByChain(protocol: string, chainId?: number, hours = 24) {
  const where: string[] = [SUPPORTED_CHAINS_SQL, `protocol = {protocol:String}`, `timestamp >= now() - INTERVAL {hours:UInt32} HOUR`];
  const params: Record<string, unknown> = { protocol, hours };
  if (chainId !== undefined) {
    where.push(`chain_id = {chainId:UInt64}`);
    params.chainId = chainId;
  }
  const sql = `
    SELECT chain_id, count() AS events, uniq(user_addr) AS users, uniq(tx_hash) AS txs, event_type, count() AS event_count
    FROM defi_events
    WHERE ${where.join(" AND ")}
    GROUP BY chain_id, event_type
    ORDER BY events DESC
  `;
  return query<{ chain_id: number; events: string; users: string; txs: string; event_type: string; event_count: string }>(sql, params);
}

export async function txByHash(hash: string) {
  const sql = `
    SELECT chain_id, block_number, tx_hash, log_index, protocol, event_type,
           user_addr, token_a, token_b, amount_a, amount_b, params, timestamp
    FROM defi_events
    WHERE ${SUPPORTED_CHAINS_SQL} AND tx_hash = {hash:String}
    ORDER BY log_index
  `;
  return hydrate(await query<WhaleRow>(sql, { hash }));
}

export async function transfersByTxHash(hash: string) {
  const sql = `
    SELECT chain_id, block_number, tx_hash, log_index, token, from_addr, to_addr, amount, timestamp
    FROM token_transfers
    WHERE ${SUPPORTED_CHAINS_SQL} AND tx_hash = {hash:String}
    ORDER BY log_index
  `;
  return query<{ chain_id: number; block_number: number; tx_hash: string; log_index: number; token: string; from_addr: string; to_addr: string; amount: string; timestamp: string }>(sql, { hash });
}

export async function blockEvents(chainId: number, blockNumber: number) {
  const sql = `
    SELECT chain_id, block_number, tx_hash, log_index, protocol, event_type,
           user_addr, token_a, amount_a, params, timestamp
    FROM defi_events
    WHERE ${SUPPORTED_CHAINS_SQL} AND chain_id = {chainId:UInt64} AND block_number = {blockNumber:UInt64}
    ORDER BY log_index
  `;
  return hydrate(await query<WhaleRow>(sql, { chainId, blockNumber }));
}

export async function tokenTransfersForToken(addr: string, chainId?: number, limit = 100) {
  const where: string[] = [SUPPORTED_CHAINS_SQL, `lower(token) = lower({addr:String})`];
  const params: Record<string, unknown> = { addr, limit };
  if (chainId !== undefined) {
    where.push(`chain_id = {chainId:UInt64}`);
    params.chainId = chainId;
  }
  const sql = `
    SELECT chain_id, block_number, tx_hash, log_index, token, from_addr, to_addr, amount, timestamp
    FROM token_transfers
    WHERE ${where.join(" AND ")}
    ORDER BY timestamp DESC
    LIMIT {limit:UInt32}
  `;
  return query(sql, params);
}

export async function latestBlocks(chainId?: number) {
  const where = chainId !== undefined
    ? `WHERE ${SUPPORTED_CHAINS_SQL} AND chain_id = {chainId:UInt64}`
    : `WHERE ${SUPPORTED_CHAINS_SQL}`;
  const params = chainId !== undefined ? { chainId } : {};
  const sql = `
    SELECT chain_id, max(block_number) AS block_number, max(timestamp) AS timestamp, count() AS events_in_block
    FROM defi_events
    ${where}
    GROUP BY chain_id
    ORDER BY chain_id
  `;
  return query<{ chain_id: number; block_number: string; timestamp: string; events_in_block: string }>(sql, params);
}

export async function eventsPerMinute(minutes = 1440) {
  const bucketFn = minutes > 240 ? "toStartOfHour" : minutes > 60 ? "toStartOfFiveMinutes" : "toStartOfMinute";
  const sql = `
    SELECT ${bucketFn}(timestamp) AS bucket, count() AS events
    FROM defi_events
    WHERE ${SUPPORTED_CHAINS_SQL} AND timestamp >= now() - INTERVAL {minutes:UInt32} MINUTE
    GROUP BY bucket
    ORDER BY bucket
  `;
  return query<{ bucket: string; events: string }>(sql, { minutes });
}
