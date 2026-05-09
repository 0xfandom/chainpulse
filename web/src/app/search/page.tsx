"use client";
import { useEffect, Suspense } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { useQuery } from "@tanstack/react-query";
import Link from "next/link";
import { parseIntent, intentToHref, describeIntent } from "@/lib/intent";
import { relTime } from "@/lib/format";
import { Card, PageHeader, Skeleton, Empty, ChainPill } from "@/components/ui";

type Block = { chain_id: number; block_number: string; timestamp: string; events_in_block: string };

export default function SearchPageWrapper() {
  return <Suspense fallback={null}><SearchPage /></Suspense>;
}

function SearchPage() {
  const sp = useSearchParams();
  const router = useRouter();
  const q = sp.get("q") ?? "";
  const intent = parseIntent(q);

  // Auto-redirect for resolvable intents (not search/unknown/latest_block)
  useEffect(() => {
    if (intent.kind === "search" || intent.kind === "unknown" || intent.kind === "latest_block") return;
    const href = intentToHref(intent);
    if (href.startsWith("/search")) return;
    router.replace(href);
  }, [q, intent, router]);

  return (
    <div>
      <PageHeader
        eyebrow="Search"
        title={q || "Search ChainPulse"}
        description={`Parsed: ${describeIntent(intent)}`}
      />

      {intent.kind === "latest_block" && <LatestBlockView chainId={intent.chainId} />}

      {(intent.kind === "search" || intent.kind === "unknown") && (
        <Card>
          <div className="text-[14px] text-[var(--fg-muted)] leading-relaxed mb-4">
            Couldn&apos;t parse this query into a known intent. Try one of these patterns:
          </div>
          <div className="space-y-2 text-[13px]">
            <Hint>Address: <code className="font-mono">0x2faf...a776ad2</code> · routes to wallet</Hint>
            <Hint>Wallet positions on Arbitrum: <code className="font-mono">positions of 0x... on arbitrum</code></Hint>
            <Hint>Wallet balances: <code className="font-mono">balances of 0x...</code></Hint>
            <Hint>Token transfers: <code className="font-mono">transfers of token 0x... on arbitrum</code></Hint>
            <Hint>Tx: <code className="font-mono">0x</code> + 64 hex chars</Hint>
            <Hint>Block: <code className="font-mono">block 18000000 on ethereum</code></Hint>
            <Hint>Latest block: <code className="font-mono">latest block on ethereum</code></Hint>
            <Hint>Whales: <code className="font-mono">whales last 6h on polygon</code></Hint>
            <Hint>Protocol: <code className="font-mono">aave v3 stats on arbitrum</code></Hint>
          </div>
        </Card>
      )}
    </div>
  );
}

function Hint({ children }: { children: React.ReactNode }) {
  return <div className="text-[var(--fg)]">{children}</div>;
}

function LatestBlockView({ chainId }: { chainId?: number }) {
  const { data, isLoading } = useQuery<{ rows: Block[] }>({
    queryKey: ["latest-block", chainId],
    queryFn: async () => {
      const qs = new URLSearchParams();
      if (chainId !== undefined) qs.set("chain", String(chainId));
      const r = await fetch(`/api/latest-block?${qs}`);
      if (!r.ok) throw new Error(await r.text());
      return r.json();
    },
    refetchInterval: 5_000,
  });

  return (
    <Card title="Latest indexed blocks" subtitle="Per chain · refreshes every 5s">
      <div className="space-y-2">
        {isLoading && Array.from({ length: 4 }).map((_, i) => <Skeleton key={i} className="h-12 w-full" />)}
        {!isLoading && (data?.rows.length ?? 0) === 0 && <Empty title="No data yet" />}
        {(data?.rows ?? []).map((r) => (
          <Link
            href={`/block/${r.chain_id}/${r.block_number}`}
            key={r.chain_id}
            className="flex items-center justify-between gap-4 py-3 px-4 rounded-xl border border-[var(--border)] hover:border-[var(--fg-strong)] transition-colors"
          >
            <div className="flex items-center gap-3">
              <ChainPill id={r.chain_id} />
              <span className="font-mono text-[14px] font-semibold text-[var(--fg-strong)]">#{parseInt(r.block_number, 10).toLocaleString()}</span>
            </div>
            <div className="flex items-center gap-4 text-[11.5px] font-mono text-[var(--fg-muted)]">
              <span>{r.events_in_block} events</span>
              <span>{relTime(r.timestamp)}</span>
            </div>
          </Link>
        ))}
      </div>
    </Card>
  );
}
