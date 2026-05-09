"use client";
import { use } from "react";
import { useQuery } from "@tanstack/react-query";
import Link from "next/link";
import { fmtAmount, relTime, shortAddr, chainName } from "@/lib/format";
import { Card, ChainPill, ProtocolPill, EventPill, Skeleton, Empty, PageHeader } from "@/components/ui";

type RpcResp = {
  chainId?: number;
  tx?: { hash: string; blockNumber: string | null; from: string; to: string | null; value: string; gas: string; gasPrice: string | null; nonce: string; input: string };
  receipt?: { status: string; gasUsed: string; effectiveGasPrice?: string; contractAddress: string | null; logs: unknown[] } | null;
  found?: boolean;
};

function hexToBig(h?: string | null): bigint | null {
  if (!h) return null;
  try { return BigInt(h); } catch { return null; }
}
function fmtEth(weiHex: string | null | undefined): string {
  const v = hexToBig(weiHex);
  if (v === null) return "—";
  const denom = 10n ** 18n;
  const whole = v / denom;
  const frac = (v % denom).toString().padStart(18, "0").slice(0, 6).replace(/0+$/, "");
  return frac ? `${whole}.${frac}` : `${whole}`;
}
function fmtGwei(weiHex: string | null | undefined): string {
  const v = hexToBig(weiHex);
  if (v === null) return "—";
  const gwei = Number(v) / 1e9;
  return gwei.toFixed(2);
}

type Row = {
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

type Transfer = {
  chain_id: number;
  block_number: number;
  tx_hash: string;
  log_index: number;
  token: string;
  from_addr: string;
  to_addr: string;
  amount: string;
  timestamp: string;
};

export default function TxPage({ params }: { params: Promise<{ hash: string }> }) {
  const { hash } = use(params);
  const { data, isLoading, error } = useQuery<{ rows: Row[]; transfers: Transfer[]; hash: string }>({
    queryKey: ["tx", hash],
    queryFn: async () => {
      const r = await fetch(`/api/tx/${hash}`);
      if (!r.ok) throw new Error(await r.text());
      return r.json();
    },
  });

  const { data: rpc, isLoading: rpcLoading } = useQuery<RpcResp>({
    queryKey: ["tx-rpc", hash],
    queryFn: async () => {
      const r = await fetch(`/api/tx-rpc/${hash}`);
      if (!r.ok) return { found: false };
      return r.json();
    },
  });

  return (
    <div>
      <PageHeader
        eyebrow="Transaction"
        title={<span className="font-mono text-[18px] break-all">{hash}</span>}
        description={`Decoded events emitted in this transaction.`}
      />

      {error && <div className="rounded-2xl border border-rose-200 bg-rose-50 p-4 text-sm text-rose-700 dark:bg-rose-500/10 dark:text-rose-300 dark:border-rose-400/30">{String(error)}</div>}

      {rpcLoading && <Skeleton className="h-32 w-full rounded-2xl mb-4" />}

      {!rpcLoading && rpc?.tx && rpc.chainId !== undefined && (
        <Card className="mb-4">
          <div className="flex items-center justify-between flex-wrap gap-3 mb-4">
            <div className="flex items-center gap-2">
              <ChainPill id={rpc.chainId} />
              <span className={`inline-flex px-2.5 py-1 rounded-full border text-[11px] font-medium ${
                rpc.receipt?.status === "0x1"
                  ? "border-emerald-300/40 bg-emerald-500/10 text-emerald-600 dark:text-emerald-400"
                  : rpc.receipt?.status === "0x0"
                  ? "border-rose-300/40 bg-rose-500/10 text-rose-600 dark:text-rose-400"
                  : "border-[var(--border)] bg-[var(--bg-elevated)] text-[var(--fg-muted)]"
              }`}>
                {rpc.receipt?.status === "0x1" ? "Success" : rpc.receipt?.status === "0x0" ? "Failed" : "Pending"}
              </span>
            </div>
            <div className="text-[11px] font-mono uppercase tracking-[0.14em] text-[var(--fg-muted)]">
              On-chain · {chainName(rpc.chainId)}
            </div>
          </div>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-3 text-[13px]">
            <Field label="Block">
              <Link href={`/block/${rpc.chainId}/${rpc.tx.blockNumber ? Number(BigInt(rpc.tx.blockNumber)) : 0}`} className="font-mono text-[12px] hover:text-[var(--accent)]">
                {rpc.tx.blockNumber ? Number(BigInt(rpc.tx.blockNumber)).toString() : "pending"}
              </Link>
            </Field>
            <Field label="Nonce">
              <span className="font-mono text-[12px]">{Number(BigInt(rpc.tx.nonce))}</span>
            </Field>
            <Field label="From">
              <Link href={`/wallet/${rpc.tx.from}`} className="font-mono text-[12px] hover:text-[var(--accent)]">{shortAddr(rpc.tx.from)}</Link>
            </Field>
            <Field label="To">
              {rpc.tx.to ? (
                <Link href={`/wallet/${rpc.tx.to}`} className="font-mono text-[12px] hover:text-[var(--accent)]">{shortAddr(rpc.tx.to)}</Link>
              ) : (
                <span className="font-mono text-[12px] text-[var(--fg-muted)]">contract creation{rpc.receipt?.contractAddress ? ` → ${shortAddr(rpc.receipt.contractAddress)}` : ""}</span>
              )}
            </Field>
            <Field label="Value">
              <span className="font-mono tnum">{fmtEth(rpc.tx.value)} {rpc.chainId === 137 ? "MATIC" : "ETH"}</span>
            </Field>
            <Field label="Gas used / limit">
              <span className="font-mono tnum">{rpc.receipt?.gasUsed ? Number(BigInt(rpc.receipt.gasUsed)).toLocaleString() : "—"} / {Number(BigInt(rpc.tx.gas)).toLocaleString()}</span>
            </Field>
            <Field label="Gas price">
              <span className="font-mono tnum">{fmtGwei(rpc.receipt?.effectiveGasPrice ?? rpc.tx.gasPrice)} gwei</span>
            </Field>
            <Field label="Logs emitted">
              <span className="font-mono tnum">{rpc.receipt?.logs?.length ?? 0}</span>
            </Field>
          </div>
          {rpc.tx.input && rpc.tx.input !== "0x" && (
            <details className="mt-3">
              <summary className="text-[11px] uppercase tracking-[0.14em] text-[var(--fg-muted)] cursor-pointer font-mono">Input data</summary>
              <pre className="mt-2 text-[11px] bg-[var(--bg-elevated)] rounded-xl p-3 overflow-x-auto font-mono text-[var(--fg-strong)] break-all whitespace-pre-wrap">{rpc.tx.input}</pre>
            </details>
          )}
        </Card>
      )}

      {isLoading && <Skeleton className="h-32 w-full rounded-2xl" />}

      {!isLoading && !rpcLoading && (data?.rows.length ?? 0) === 0 && (data?.transfers.length ?? 0) === 0 && !rpc?.tx && (
        <Card><Empty title="No data found for this tx" hint="Tx not on Ethereum / Polygon / Arbitrum, or hash invalid." /></Card>
      )}

      {!isLoading && (data?.rows.length ?? 0) === 0 && (data?.transfers.length ?? 0) > 0 && (
        <div className="rounded-2xl border border-[var(--border)] bg-[var(--bg-elevated)] p-4 text-[12.5px] text-[var(--fg-muted)] mb-4">
          No DeFi protocol decoded this tx. Showing raw token transfers below.
        </div>
      )}

      <div className="space-y-3">
        {(data?.transfers ?? []).map((t) => (
          <Card key={`t-${t.log_index}`}>
            <div className="flex items-center justify-between flex-wrap gap-3 mb-4">
              <div className="flex items-center gap-2">
                <ChainPill id={t.chain_id} />
                <ProtocolPill p="erc20" />
                <EventPill kind="transfer" />
              </div>
              <div className="text-[11px] font-mono uppercase tracking-[0.14em] text-[var(--fg-muted)]">
                Block {t.block_number} · log {t.log_index} · {relTime(t.timestamp)}
              </div>
            </div>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-3 text-[13px]">
              <Field label="From">
                <Link href={`/wallet/${t.from_addr}`} className="font-mono text-[12px] hover:text-[var(--accent)]">{shortAddr(t.from_addr)}</Link>
              </Field>
              <Field label="To">
                <Link href={`/wallet/${t.to_addr}`} className="font-mono text-[12px] hover:text-[var(--accent)]">{shortAddr(t.to_addr)}</Link>
              </Field>
              <Field label="Token">
                <Link href={`/token/${t.token}?chain=${t.chain_id}`} className="font-mono text-[12px] hover:text-[var(--accent)]">{shortAddr(t.token)}</Link>
              </Field>
              <Field label="Amount">
                <span className="font-mono tnum">{fmtAmount(t.amount, t.token, t.chain_id)}</span>
              </Field>
            </div>
          </Card>
        ))}
        {(data?.rows ?? []).map((r) => (
          <Card key={r.log_index}>
            <div className="flex items-center justify-between flex-wrap gap-3 mb-4">
              <div className="flex items-center gap-2">
                <ChainPill id={r.chain_id} />
                <ProtocolPill p={r.protocol} />
                <EventPill kind={r.event_type} />
              </div>
              <div className="text-[11px] font-mono uppercase tracking-[0.14em] text-[var(--fg-muted)]">
                Block {r.block_number} · log {r.log_index} · {relTime(r.timestamp)}
              </div>
            </div>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-3 text-[13px]">
              <Field label="User">
                <Link href={`/wallet/${r.user_addr}`} className="font-mono text-[12px] hover:text-[var(--accent)]">{shortAddr(r.user_addr)}</Link>
              </Field>
              <Field label="Token A">
                <span className="font-mono text-[12px]">{shortAddr(r.token_a)}</span>
              </Field>
              <Field label="Amount A">
                <span className="font-mono tnum">{fmtAmount(r.amount_a, r.token_a, r.chain_id)}</span>
              </Field>
              {r.token_b && (
                <Field label="Token B / Amount B">
                  <span className="font-mono text-[12px]">{shortAddr(r.token_b)} · <span className="tnum">{fmtAmount(r.amount_b ?? "0", r.token_b, r.chain_id)}</span></span>
                </Field>
              )}
            </div>
            {r.params && (
              <details className="mt-3">
                <summary className="text-[11px] uppercase tracking-[0.14em] text-[var(--fg-muted)] cursor-pointer font-mono">Raw params</summary>
                <pre className="mt-2 text-[11px] bg-[var(--bg-elevated)] rounded-xl p-3 overflow-x-auto font-mono text-[var(--fg-strong)]">{JSON.stringify(JSON.parse(r.params || "{}"), null, 2)}</pre>
              </details>
            )}
          </Card>
        ))}
      </div>
    </div>
  );
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div>
      <div className="text-[10px] uppercase tracking-[0.16em] text-[var(--fg-muted)] font-mono font-semibold mb-1">{label}</div>
      <div className="text-[var(--fg-strong)]">{children}</div>
    </div>
  );
}
