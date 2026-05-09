"use client";
import { Suspense } from "react";
import { useSearchParams } from "next/navigation";
import { useQuery } from "@tanstack/react-query";
import Link from "next/link";
import { chainName } from "@/lib/format";
import { ChainIcon } from "@/components/ChainIcon";
import { Card, Skeleton, Empty, PageHeader } from "@/components/ui";

const SUPPORTED = [1, 137, 42161];

type Row = { chain_id: number; tx_hash: string; protocol: string; event_type: string };
type RpcBlock = { number: string; transactions: unknown[]; timestamp: string };
type Result = { chainId: number; rows: Row[]; block: RpcBlock | null };

export default function BlockLookupWrapper() {
  return <Suspense fallback={null}><BlockLookup /></Suspense>;
}

function BlockLookup() {
  const sp = useSearchParams();
  const n = sp.get("n");
  const blockN = n ? parseInt(n, 10) : NaN;

  const { data, isLoading } = useQuery<Result[]>({
    queryKey: ["block-lookup", blockN],
    enabled: Number.isFinite(blockN),
    queryFn: async () => {
      const out = await Promise.all(
        SUPPORTED.map(async (chainId): Promise<Result> => {
          const r = await fetch(`/api/block/${chainId}/${blockN}`);
          if (!r.ok) return { chainId, rows: [], block: null };
          const j = (await r.json()) as { rows: Row[]; block: RpcBlock | null };
          return { chainId, rows: j.rows ?? [], block: j.block ?? null };
        }),
      );
      return out;
    },
  });

  const matches = (data ?? []).filter((d) => d.block !== null);

  return (
    <div>
      <PageHeader
        eyebrow="Block lookup"
        title={`Block #${Number.isFinite(blockN) ? blockN.toLocaleString() : "—"}`}
        description="Searched across Ethereum, Polygon, Arbitrum."
      />

      {!Number.isFinite(blockN) && <Card><Empty title="Provide a block number" hint="Use ?n=18000000" /></Card>}

      {isLoading && <Skeleton className="h-32 w-full rounded-2xl" />}

      {!isLoading && Number.isFinite(blockN) && matches.length === 0 && (
        <Card><Empty title="Block not found on any indexed chain" hint="Block height may not exist yet." /></Card>
      )}

      <div className="space-y-3">
        {matches.map((m) => (
          <Link
            key={m.chainId}
            href={`/block/${m.chainId}/${blockN}`}
            className="card-surface p-5 flex items-center justify-between gap-4 hover:border-[var(--fg-strong)] transition-colors"
          >
            <div className="flex items-center gap-3">
              <ChainIcon id={m.chainId} size={28} />
              <div>
                <div className="font-display text-[16px] font-bold text-[var(--fg-strong)]">{chainName(m.chainId)}</div>
                <div className="text-[11.5px] font-mono text-[var(--fg-muted)] mt-0.5">
                  {m.block?.transactions.length ?? 0} txs · {m.rows.length} decoded DeFi event{m.rows.length === 1 ? "" : "s"}
                </div>
              </div>
            </div>
            <span className="text-[11px] font-mono uppercase tracking-[0.14em] text-[var(--accent)]">View block →</span>
          </Link>
        ))}
      </div>
    </div>
  );
}
