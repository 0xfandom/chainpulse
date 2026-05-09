"use client";
import { use, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import Link from "next/link";
import { fmtAmount, relTime, shortAddr, chainName } from "@/lib/format";
import { Card, ChainPill, ProtocolPill, EventPill, Skeleton, Empty, PageHeader, Stat } from "@/components/ui";

type Row = {
  chain_id: number;
  block_number: number;
  tx_hash: string;
  log_index: number;
  protocol: string;
  event_type: string;
  user_addr: string;
  token_a: string;
  amount_a: string;
  timestamp: string;
};

type RpcBlock = {
  number: string;
  hash: string;
  parentHash: string;
  timestamp: string;
  miner: string;
  gasLimit: string;
  gasUsed: string;
  size: string;
  baseFeePerGas?: string;
  transactions: { hash: string; from: string; to: string | null; value: string; gas: string }[];
};

const PAGE_SIZE = 25;

function hex(s?: string | null): bigint {
  if (!s) return 0n;
  try { return BigInt(s); } catch { return 0n; }
}
function fmtNative(weiHex: string, chainId: number): string {
  const v = hex(weiHex);
  const denom = 10n ** 18n;
  const whole = v / denom;
  const frac = (v % denom).toString().padStart(18, "0").slice(0, 6).replace(/0+$/, "");
  const sym = chainId === 137 ? "MATIC" : "ETH";
  return frac ? `${whole}.${frac} ${sym}` : `${whole} ${sym}`;
}

export default function BlockPage({ params }: { params: Promise<{ chain: string; n: string }> }) {
  const { chain, n } = use(params);
  const chainId = parseInt(chain, 10);
  const blockNumber = parseInt(n, 10);
  const [page, setPage] = useState(0);

  const { data, isLoading } = useQuery<{ rows: Row[]; block: RpcBlock | null }>({
    queryKey: ["block", chainId, blockNumber],
    queryFn: async () => {
      const r = await fetch(`/api/block/${chainId}/${blockNumber}`);
      if (!r.ok) throw new Error(await r.text());
      return r.json();
    },
  });

  const block = data?.block ?? null;
  const txs = block?.transactions ?? [];
  const totalPages = Math.max(1, Math.ceil(txs.length / PAGE_SIZE));
  const safePage = Math.min(page, totalPages - 1);
  const pageTxs = txs.slice(safePage * PAGE_SIZE, (safePage + 1) * PAGE_SIZE);

  const decodedByTx: Record<string, Row[]> = {};
  for (const r of data?.rows ?? []) (decodedByTx[r.tx_hash.toLowerCase()] ??= []).push(r);

  return (
    <div>
      <PageHeader
        eyebrow={`Block · ${chainName(chainId)}`}
        title={`#${blockNumber.toLocaleString()}`}
        description={block ? `Mined ${relTime(new Date(Number(hex(block.timestamp)) * 1000).toISOString())} · ${txs.length} transactions` : "On-chain block + decoded DeFi events."}
      />

      {isLoading && <Skeleton className="h-32 w-full rounded-2xl mb-4" />}

      {!isLoading && block && (
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-6">
          <Stat label="Transactions" value={txs.length.toString()} />
          <Stat label="Gas used" value={Number(hex(block.gasUsed)).toLocaleString()} hint={`/ ${Number(hex(block.gasLimit)).toLocaleString()}`} />
          <Stat label="Decoded events" value={(data?.rows.length ?? 0).toString()} hint="DeFi protocols" />
          <Stat label="Base fee" value={block.baseFeePerGas ? `${(Number(hex(block.baseFeePerGas)) / 1e9).toFixed(2)} gwei` : "—"} />
        </div>
      )}

      {!isLoading && !block && (data?.rows.length ?? 0) === 0 && (
        <Card><Empty title="Block not found" hint="Check the block number and chain." /></Card>
      )}

      {block && (
        <Card title={`All transactions · page ${safePage + 1} / ${totalPages}`}>
          <div className="space-y-1">
            {pageTxs.map((t) => {
              const decoded = decodedByTx[t.hash.toLowerCase()] ?? [];
              return (
                <Link
                  key={t.hash}
                  href={`/tx/${t.hash}`}
                  className="flex items-center justify-between gap-3 py-2 px-2 rounded-xl hover:bg-[var(--bg-elevated)]/60 transition-colors"
                >
                  <div className="flex items-center gap-2 min-w-0 flex-1">
                    <span className="font-mono text-[11px] text-[var(--fg-dim)] truncate max-w-[180px]">{t.hash.slice(0, 14)}…</span>
                    <span className="text-[10px] font-mono text-[var(--fg-muted)] hidden md:inline">{shortAddr(t.from)} →</span>
                    <span className="text-[10px] font-mono text-[var(--fg-muted)] hidden md:inline">{t.to ? shortAddr(t.to) : "create"}</span>
                    {decoded.length > 0 && decoded.slice(0, 1).map((r, i) => (
                      <span key={i} className="hidden md:inline-flex gap-1">
                        <ProtocolPill p={r.protocol} />
                        <EventPill kind={r.event_type} />
                      </span>
                    ))}
                  </div>
                  <span className="text-[11px] font-mono tnum text-[var(--fg-strong)] shrink-0">{fmtNative(t.value, chainId)}</span>
                </Link>
              );
            })}
          </div>
          {totalPages > 1 && (
            <div className="flex items-center justify-between mt-4 pt-3 border-t border-[var(--border)]">
              <button
                onClick={() => setPage((p) => Math.max(0, p - 1))}
                disabled={safePage === 0}
                className="text-[11px] font-mono uppercase tracking-[0.14em] text-[var(--fg-muted)] hover:text-[var(--fg-strong)] disabled:opacity-30"
              >
                ← Prev
              </button>
              <span className="text-[11px] font-mono text-[var(--fg-dim)]">{safePage * PAGE_SIZE + 1}–{Math.min((safePage + 1) * PAGE_SIZE, txs.length)} of {txs.length}</span>
              <button
                onClick={() => setPage((p) => Math.min(totalPages - 1, p + 1))}
                disabled={safePage >= totalPages - 1}
                className="text-[11px] font-mono uppercase tracking-[0.14em] text-[var(--fg-muted)] hover:text-[var(--fg-strong)] disabled:opacity-30"
              >
                Next →
              </button>
            </div>
          )}
        </Card>
      )}

      {(data?.rows.length ?? 0) > 0 && (
        <div className="mt-6">
          <h3 className="font-display text-[18px] font-bold mb-3">Decoded DeFi events</h3>
          <div className="space-y-2">
            {(data?.rows ?? []).map((r) => (
              <Link href={`/tx/${r.tx_hash}`} key={`${r.tx_hash}-${r.log_index}`} className="card-surface p-4 flex items-center justify-between gap-4 hover:border-[var(--fg-strong)] transition-colors">
                <div className="flex items-center gap-2 min-w-0">
                  <ChainPill id={r.chain_id} />
                  <ProtocolPill p={r.protocol} />
                  <EventPill kind={r.event_type} />
                  <span className="font-mono text-[11.5px] text-[var(--fg-muted)] truncate">{shortAddr(r.user_addr)}</span>
                </div>
                <div className="text-right tnum text-[12px] font-mono font-semibold text-[var(--fg-strong)] shrink-0">
                  {fmtAmount(r.amount_a, r.token_a, r.chain_id)}
                </div>
              </Link>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
