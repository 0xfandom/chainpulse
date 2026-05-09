"use client";
import { useQuery } from "@tanstack/react-query";
import Link from "next/link";
import { Suspense, useEffect, useState } from "react";
import { useSearchParams } from "next/navigation";
import { Pause, Play } from "lucide-react";
import { fmtAmount, fmtSwap, relTime, shortAddr } from "@/lib/format";
import { ChainPill, ProtocolPill, EventPill, Skeleton, Empty, PageHeader } from "@/components/ui";
import { ChainIcon } from "@/components/ChainIcon";

type Whale = {
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

const CHAINS = [
  { id: undefined, label: "All chains" },
  { id: 1, label: "Ethereum" },
  { id: 42161, label: "Arbitrum" },
  { id: 137, label: "Polygon" },
];
const PROTOS = [
  { id: "all", label: "All protocols" },
  { id: "uniswap_v3", label: "Uniswap V3" },
  { id: "aave_v3", label: "Aave V3" },
  { id: "compound_v3", label: "Compound V3" },
  { id: "lido", label: "Lido" },
  { id: "curve", label: "Curve" },
];

export default function WhalesPageWrapper() {
  return <Suspense fallback={null}><WhalesPage /></Suspense>;
}

function WhalesPage() {
  const searchParams = useSearchParams();
  const initialChain = (() => {
    const c = searchParams.get("chain");
    if (!c) return undefined;
    const n = parseInt(c, 10);
    return Number.isFinite(n) ? n : undefined;
  })();
  const initialProto = searchParams.get("protocol") ?? "all";

  const [chain, setChain] = useState<number | undefined>(initialChain);
  const [proto, setProto] = useState<string>(initialProto);
  const [paused, setPaused] = useState(false);

  useEffect(() => {
    const c = searchParams.get("chain");
    const p = searchParams.get("protocol");
    setChain(c ? (Number.isFinite(parseInt(c, 10)) ? parseInt(c, 10) : undefined) : undefined);
    setProto(p ?? "all");
  }, [searchParams]);

  const { data, isLoading } = useQuery<{ rows: Whale[] }>({
    queryKey: ["whales", chain, proto],
    queryFn: async () => {
      const sp = new URLSearchParams({ minutes: "1440", limit: "150" });
      if (chain !== undefined) sp.set("chain", String(chain));
      if (proto !== "all") sp.set("protocol", proto);
      const r = await fetch(`/api/whales?${sp}`);
      if (!r.ok) throw new Error(await r.text());
      return r.json();
    },
    refetchInterval: paused ? false : 4_000,
  });

  return (
    <div>
      <PageHeader
        eyebrow="Real-time"
        title="Live Whale Feed"
        description="DeFi events as the indexer sees them. Sub-block latency, auto-refresh every 4 seconds."
        action={
          <button onClick={() => setPaused((p) => !p)} className="btn-ghost">
            {paused ? <><Play className="size-3.5" /> Resume</> : <><Pause className="size-3.5" /> Pause</>}
          </button>
        }
      />

      <div className="space-y-4 mb-6">
        <FilterRow label="Chain">
          {CHAINS.map((c) => (
            <FilterChip key={c.label} active={chain === c.id} onClick={() => setChain(c.id)}>
              {c.id !== undefined && <ChainIcon id={c.id} size={18} className="mr-2" />}
              {c.label}
            </FilterChip>
          ))}
        </FilterRow>
        <FilterRow label="Protocol">
          {PROTOS.map((p) => (
            <FilterChip key={p.id} active={proto === p.id} onClick={() => setProto(p.id)}>{p.label}</FilterChip>
          ))}
        </FilterRow>
      </div>

      <div className="card-surface overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead className="bg-[var(--bg-elevated)]">
              <tr className="text-[10px] uppercase tracking-[0.16em] text-[var(--fg-muted)] font-mono font-semibold border-b border-[var(--border)]">
                <th className="text-left px-6 py-3.5">Time</th>
                <th className="text-left px-3 py-3.5">Chain</th>
                <th className="text-left px-3 py-3.5">Protocol</th>
                <th className="text-left px-3 py-3.5">Event</th>
                <th className="text-left px-3 py-3.5">User</th>
                <th className="text-left px-3 py-3.5">Tokens</th>
                <th className="text-right px-6 py-3.5">Amount</th>
              </tr>
            </thead>
            <tbody>
              {isLoading && Array.from({ length: 8 }).map((_, i) => (
                <tr key={i} className="border-b border-[var(--border)] last:border-0">
                  <td colSpan={7} className="px-6 py-3"><Skeleton className="h-5 w-full" /></td>
                </tr>
              ))}
              {!isLoading && (data?.rows.length ?? 0) === 0 && (
                <tr><td colSpan={7}><Empty title="No events match these filters" hint="Try a different chain or protocol." /></td></tr>
              )}
              {(data?.rows ?? []).map((r) => {
                const swap = r.protocol === "uniswap_v3" && r.event_type === "swap"
                  ? fmtSwap(r.token_a, r.amount_a, r.amount_b, r.chain_id)
                  : null;
                return (
                  <tr key={`${r.chain_id}-${r.block_number}-${r.log_index}`} className="border-b border-[var(--border)]/60 last:border-0 hover:bg-[var(--bg-elevated)]/40 transition-colors">
                    <td className="px-6 py-3 text-[var(--fg-muted)] whitespace-nowrap text-[11.5px] tnum font-mono">{relTime(r.timestamp)}</td>
                    <td className="px-3 py-3"><ChainPill id={r.chain_id} /></td>
                    <td className="px-3 py-3"><ProtocolPill p={r.protocol} /></td>
                    <td className="px-3 py-3"><EventPill kind={r.event_type} /></td>
                    <td className="px-3 py-3 font-mono text-[11.5px]">
                      <Link href={`/wallet/${r.user_addr}`} className="text-[var(--fg-strong)] hover:text-[var(--accent)] transition-colors underline-offset-2 hover:underline">{shortAddr(r.user_addr)}</Link>
                    </td>
                    <td className="px-3 py-3 font-mono text-[11px] text-[var(--fg-muted)] whitespace-nowrap">
                      {shortAddr(r.token_a, 3)}{r.token_b ? <span className="text-[var(--fg-dim)]"> · {shortAddr(r.token_b, 3)}</span> : null}
                    </td>
                    <td className="px-6 py-3 text-right tnum whitespace-nowrap">
                      {swap ? (
                        <>
                          <div className="text-[var(--fg-strong)] font-semibold text-[13px]">{swap.primary}</div>
                          <div className="text-[10.5px] text-[var(--fg-muted)] mt-0.5">↓ {swap.secondary}</div>
                        </>
                      ) : (
                        <>
                          <div className="text-[var(--fg-strong)] font-semibold text-[13px]">{fmtAmount(r.amount_a, r.token_a, r.chain_id)}</div>
                          {r.amount_b && <div className="text-[10.5px] text-[var(--fg-dim)] mt-0.5">{fmtAmount(r.amount_b, r.token_b, r.chain_id)}</div>}
                        </>
                      )}
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      </div>

      {!isLoading && (data?.rows.length ?? 0) > 0 && (
        <div className="text-[11px] text-[var(--fg-muted)] text-center mt-5 font-mono uppercase tracking-[0.14em]">
          {data?.rows.length} most recent · last 120 minutes
        </div>
      )}
    </div>
  );
}

function FilterRow({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="flex items-start gap-3">
      <div className="text-[10px] uppercase tracking-[0.16em] text-[var(--fg-muted)] font-mono font-semibold pt-2 w-16 shrink-0">{label}</div>
      <div className="flex flex-wrap gap-1.5 flex-1">{children}</div>
    </div>
  );
}

function FilterChip({ active, onClick, children }: { active: boolean; onClick: () => void; children: React.ReactNode }) {
  return (
    <button
      onClick={onClick}
      className={`inline-flex items-center px-4 py-2 text-[13px] rounded-full border font-medium transition-all ${
        active
          ? "bg-[var(--fg-strong)] border-[var(--fg-strong)] text-[var(--bg)]"
          : "border-[var(--border-strong)] bg-[var(--bg-card)] text-[var(--fg-muted)] hover:border-[var(--fg-strong)] hover:text-[var(--fg-strong)]"
      }`}
    >
      {children}
    </button>
  );
}
