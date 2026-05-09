"use client";
import { use, Suspense } from "react";
import { useQuery } from "@tanstack/react-query";
import { useSearchParams } from "next/navigation";
import { compactNum, chainName, chainColor } from "@/lib/format";
import { Card, Stat, Skeleton, Empty, PageHeader, EventPill } from "@/components/ui";

type Row = { chain_id: number; events: string; users: string; txs: string; event_type: string };

export default function ProtocolPageWrapper({ params }: { params: Promise<{ name: string }> }) {
  return <Suspense fallback={null}><ProtocolPage params={params} /></Suspense>;
}

function ProtocolPage({ params }: { params: Promise<{ name: string }> }) {
  const { name } = use(params);
  const sp = useSearchParams();
  const chain = sp.get("chain");
  const { data, isLoading } = useQuery<{ rows: Row[]; protocol: string; hours: number }>({
    queryKey: ["protocol", name, chain],
    queryFn: async () => {
      const qs = new URLSearchParams({ hours: "24" });
      if (chain) qs.set("chain", chain);
      const r = await fetch(`/api/protocol/${name}?${qs}`);
      if (!r.ok) throw new Error(await r.text());
      return r.json();
    },
    refetchInterval: 15_000,
  });

  const totalEvents = (data?.rows ?? []).reduce((a, r) => a + parseInt(r.events, 10), 0);
  const totalUsers = (data?.rows ?? []).reduce((a, r) => a + parseInt(r.users, 10), 0);
  const totalTxs = (data?.rows ?? []).reduce((a, r) => a + parseInt(r.txs, 10), 0);

  // Group by chain
  const byChain: Record<number, Row[]> = {};
  for (const r of data?.rows ?? []) {
    (byChain[r.chain_id] ??= []).push(r);
  }

  return (
    <div>
      <PageHeader
        eyebrow={`Protocol · last 24h`}
        title={name.replace(/_/g, " ")}
        description={`Volume, users, and event breakdown across indexed chains.`}
      />

      <div className="grid grid-cols-3 gap-4 mb-8">
        <Stat label="Events" value={isLoading ? <Skeleton className="h-8 w-20" /> : compactNum(totalEvents)} hint="last 24h" />
        <Stat label="Unique users" value={isLoading ? <Skeleton className="h-8 w-20" /> : compactNum(totalUsers)} hint="distinct wallets" />
        <Stat label="Transactions" value={isLoading ? <Skeleton className="h-8 w-20" /> : compactNum(totalTxs)} hint="unique tx hashes" />
      </div>

      <div className="space-y-4">
        {isLoading && Array.from({ length: 3 }).map((_, i) => <Skeleton key={i} className="h-32 w-full rounded-2xl" />)}
        {!isLoading && Object.keys(byChain).length === 0 && (
          <Card><Empty title={`No ${name} activity in last 24h`} /></Card>
        )}
        {Object.entries(byChain).map(([cid, rows]) => {
          const id = parseInt(cid, 10);
          const sumEv = rows.reduce((a, r) => a + parseInt(r.events, 10), 0);
          const sumUs = rows.reduce((a, r) => a + parseInt(r.users, 10), 0);
          return (
            <Card key={cid}>
              <div className="flex items-center justify-between mb-4">
                <div className="flex items-center gap-3">
                  <span className="size-3 rounded-full" style={{ background: chainColor(id) }} />
                  <span className="font-display text-[18px] font-bold text-[var(--fg-strong)]">{chainName(id)}</span>
                </div>
                <div className="flex gap-4 text-[12px] tnum text-[var(--fg-muted)]">
                  <span><span className="text-[var(--fg-strong)] font-semibold">{compactNum(sumEv)}</span> events</span>
                  <span><span className="text-[var(--fg-strong)] font-semibold">{compactNum(sumUs)}</span> users</span>
                </div>
              </div>
              <div className="space-y-1.5">
                {rows.map((r, i) => {
                  const ev = parseInt(r.events, 10);
                  const max = Math.max(...rows.map((x) => parseInt(x.events, 10)));
                  const pct = (ev / max) * 100;
                  return (
                    <div key={i} className="relative">
                      <div className="absolute inset-0 rounded-full bg-[var(--accent-soft)]" style={{ width: `${pct}%` }} />
                      <div className="relative flex items-center justify-between text-sm py-2 px-3 rounded-full">
                        <EventPill kind={r.event_type} />
                        <div className="flex gap-4 tnum text-[12px]">
                          <span className="text-[var(--fg-strong)] font-semibold">{compactNum(ev)}</span>
                          <span className="text-[var(--fg-muted)]">{compactNum(parseInt(r.users, 10))} users</span>
                        </div>
                      </div>
                    </div>
                  );
                })}
              </div>
            </Card>
          );
        })}
      </div>
    </div>
  );
}
