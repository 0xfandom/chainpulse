"use client";
import { useQuery } from "@tanstack/react-query";
import { use, useState } from "react";
import Link from "next/link";
import { ArrowDownLeft, ArrowUpRight, Copy, Check, ChevronLeft } from "lucide-react";
import { fmtAmount, relTime, shortAddr } from "@/lib/format";
import { Card, ChainPill, ProtocolPill, EventPill, Stat, Skeleton, Empty, TagPill } from "@/components/ui";

type Event = {
  chain_id: number;
  block_number: number;
  tx_hash: string;
  log_index: number;
  protocol: string;
  event_type: string;
  token_a: string;
  token_b: string | null;
  amount_a: string;
  amount_b: string | null;
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

export default function WalletPage({ params }: { params: Promise<{ addr: string }> }) {
  const { addr } = use(params);
  const [copied, setCopied] = useState(false);
  const { data, isLoading, error } = useQuery<{ events: Event[]; transfers: Transfer[] }>({
    queryKey: ["wallet", addr],
    queryFn: async () => {
      const r = await fetch(`/api/wallet/${addr}`);
      if (!r.ok) throw new Error(await r.text());
      return r.json();
    },
    refetchInterval: 10_000,
  });

  const protoCounts: Record<string, number> = {};
  for (const e of data?.events ?? []) protoCounts[e.protocol] = (protoCounts[e.protocol] ?? 0) + 1;
  const chainsTouched = new Set([
    ...(data?.events ?? []).map((e) => e.chain_id),
    ...(data?.transfers ?? []).map((t) => t.chain_id),
  ]);

  const copy = async () => {
    await navigator.clipboard.writeText(addr);
    setCopied(true);
    setTimeout(() => setCopied(false), 1200);
  };

  return (
    <div className="space-y-6">
      <Link href="/wallet" className="inline-flex items-center gap-1 text-[12px] text-[var(--fg-muted)] hover:text-[var(--fg-strong)] transition-colors uppercase tracking-[0.14em] font-mono font-semibold">
        <ChevronLeft className="size-3.5" /> New Lookup
      </Link>

      <div className="card-surface p-6">
        <TagPill className="mb-3">Wallet</TagPill>
        <div className="flex items-center gap-2 mt-1">
          <h1 className="text-[18px] lg:text-[22px] font-mono break-all text-[var(--fg-strong)] tracking-tight">{addr}</h1>
          <button onClick={copy} className="btn-icon shrink-0 size-8">
            {copied ? <Check className="size-3.5 text-[var(--accent)]" /> : <Copy className="size-3.5" />}
          </button>
        </div>
      </div>

      {error && <div className="rounded-2xl border border-rose-200 bg-rose-50 p-4 text-sm text-rose-700 dark:bg-rose-500/10 dark:text-rose-300 dark:border-rose-400/30">{String(error)}</div>}

      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
        <Stat label="DeFi events" value={isLoading ? <Skeleton className="h-8 w-16" /> : (data?.events.length ?? 0).toString()} hint="last 90 days" />
        <Stat label="Token transfers" value={isLoading ? <Skeleton className="h-8 w-16" /> : (data?.transfers.length ?? 0).toString()} hint="ERC-20 events" />
        <Stat label="Protocols used" value={isLoading ? <Skeleton className="h-8 w-12" /> : Object.keys(protoCounts).length.toString()} hint="distinct apps" />
        <Stat label="Chains touched" value={isLoading ? <Skeleton className="h-8 w-12" /> : chainsTouched.size.toString()} hint="active networks" />
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-5">
        <Card title="DeFi Activity" subtitle={`${data?.events.length ?? 0} events`}>
          <div className="space-y-1 max-h-[28rem] overflow-y-auto -mx-2 px-2">
            {isLoading && Array.from({ length: 5 }).map((_, i) => <Skeleton key={i} className="h-10 w-full" />)}
            {!isLoading && (data?.events.length ?? 0) === 0 && <Empty title="No DeFi events" hint="Last 90 days." />}
            {(data?.events ?? []).map((e) => (
              <div key={`${e.chain_id}-${e.block_number}-${e.log_index}`} className="flex items-center justify-between gap-3 py-2 px-2 rounded-xl hover:bg-[var(--bg-elevated)]/60 transition-colors">
                <div className="flex items-center gap-2 min-w-0">
                  <span className="text-[10px] tnum text-[var(--fg-dim)] w-12 shrink-0 font-mono">{relTime(e.timestamp)}</span>
                  <ProtocolPill p={e.protocol} />
                  <EventPill kind={e.event_type} />
                  <ChainPill id={e.chain_id} />
                </div>
                <div className="text-right tnum text-[12px] text-[var(--fg-strong)] whitespace-nowrap shrink-0 font-mono font-semibold">
                  {fmtAmount(e.amount_a, e.token_a, e.chain_id)}
                </div>
              </div>
            ))}
          </div>
        </Card>

        <Card title="Token Transfers" subtitle={`${data?.transfers.length ?? 0} events`}>
          <div className="space-y-1 max-h-[28rem] overflow-y-auto -mx-2 px-2">
            {isLoading && Array.from({ length: 5 }).map((_, i) => <Skeleton key={i} className="h-10 w-full" />)}
            {!isLoading && (data?.transfers.length ?? 0) === 0 && <Empty title="No token transfers" />}
            {(data?.transfers ?? []).map((t) => {
              const out = t.from_addr.toLowerCase() === addr.toLowerCase();
              return (
                <div key={`${t.chain_id}-${t.block_number}-${t.log_index}`} className="flex items-center justify-between gap-3 py-2 px-2 rounded-xl hover:bg-[var(--bg-elevated)]/60 transition-colors">
                  <div className="flex items-center gap-2 min-w-0">
                    <span className="text-[10px] tnum text-[var(--fg-dim)] w-12 shrink-0 font-mono">{relTime(t.timestamp)}</span>
                    <span className={`inline-flex items-center justify-center size-6 rounded-full border ${out ? "bg-rose-50 text-rose-600 border-rose-200 dark:bg-rose-500/10 dark:text-rose-300 dark:border-rose-400/30" : "bg-[var(--accent-soft)] text-[var(--accent-fg)] border-[var(--accent-border)]"}`}>
                      {out ? <ArrowUpRight className="size-3" /> : <ArrowDownLeft className="size-3" />}
                    </span>
                    <span className="text-[var(--fg-muted)] text-[11px] font-mono">{shortAddr(t.token, 3)}</span>
                    <span className="text-[var(--fg-dim)] text-[10px] uppercase tracking-wider">{out ? "to" : "from"}</span>
                    <span className="text-[var(--fg-muted)] text-[11px] font-mono">{shortAddr(out ? t.to_addr : t.from_addr, 3)}</span>
                  </div>
                  <div className="text-right tnum text-[12px] text-[var(--fg-strong)] whitespace-nowrap shrink-0 font-mono font-semibold">
                    {fmtAmount(t.amount, t.token, t.chain_id)}
                  </div>
                </div>
              );
            })}
          </div>
        </Card>
      </div>

      {Object.keys(protoCounts).length > 0 && (
        <Card title="Protocol Breakdown" subtitle="Events per protocol">
          <div className="flex flex-wrap gap-2">
            {Object.entries(protoCounts).sort((a, b) => b[1] - a[1]).map(([p, n]) => (
              <div key={p} className="flex items-center gap-2 px-4 py-2 rounded-full border border-[var(--border)] bg-[var(--bg-elevated)]">
                <ProtocolPill p={p} />
                <span className="text-[11.5px] tnum text-[var(--fg-strong)] font-semibold">{n}</span>
              </div>
            ))}
          </div>
        </Card>
      )}
    </div>
  );
}
