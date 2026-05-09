"use client";
import { use, Suspense } from "react";
import { useQuery } from "@tanstack/react-query";
import { useSearchParams } from "next/navigation";
import Link from "next/link";
import { fmtAmount, relTime, shortAddr } from "@/lib/format";
import { Card, ChainPill, Skeleton, Empty, PageHeader } from "@/components/ui";

type Row = {
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

export default function TokenPageWrapper({ params }: { params: Promise<{ addr: string }> }) {
  return <Suspense fallback={null}><TokenPage params={params} /></Suspense>;
}

function TokenPage({ params }: { params: Promise<{ addr: string }> }) {
  const { addr } = use(params);
  const sp = useSearchParams();
  const chain = sp.get("chain");
  const { data, isLoading } = useQuery<{ rows: Row[]; token: string }>({
    queryKey: ["token", addr, chain],
    queryFn: async () => {
      const qs = new URLSearchParams({ limit: "200" });
      if (chain) qs.set("chain", chain);
      const r = await fetch(`/api/token/${addr}?${qs}`);
      if (!r.ok) throw new Error(await r.text());
      return r.json();
    },
    refetchInterval: 10_000,
  });

  return (
    <div>
      <PageHeader
        eyebrow="Token"
        title={<span className="font-mono text-[20px] break-all">{addr}</span>}
        description="Recent ERC-20 transfers involving this token contract."
      />

      <Card title={`${data?.rows.length ?? 0} transfers`} subtitle="latest first">
        <div className="space-y-1.5 max-h-[36rem] overflow-y-auto">
          {isLoading && Array.from({ length: 8 }).map((_, i) => <Skeleton key={i} className="h-10 w-full" />)}
          {!isLoading && (data?.rows.length ?? 0) === 0 && <Empty title="No transfers found" />}
          {(data?.rows ?? []).map((r) => (
            <div key={`${r.chain_id}-${r.block_number}-${r.log_index}`} className="flex items-center justify-between gap-3 py-2 px-2 rounded-xl hover:bg-[var(--bg-elevated)]/50">
              <div className="flex items-center gap-2 min-w-0">
                <span className="text-[10px] tnum text-[var(--fg-dim)] w-12 shrink-0 font-mono">{relTime(r.timestamp)}</span>
                <ChainPill id={r.chain_id} />
                <Link href={`/wallet/${r.from_addr}`} className="font-mono text-[11px] text-[var(--fg-muted)] hover:text-[var(--accent)]">{shortAddr(r.from_addr)}</Link>
                <span className="text-[var(--fg-dim)] text-[10px] uppercase tracking-wider">→</span>
                <Link href={`/wallet/${r.to_addr}`} className="font-mono text-[11px] text-[var(--fg-muted)] hover:text-[var(--accent)]">{shortAddr(r.to_addr)}</Link>
              </div>
              <div className="text-right tnum text-[12px] font-mono font-semibold text-[var(--fg-strong)] shrink-0">
                {fmtAmount(r.amount, r.token, r.chain_id)}
              </div>
            </div>
          ))}
        </div>
      </Card>
    </div>
  );
}
