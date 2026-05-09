"use client";
import { useQuery } from "@tanstack/react-query";
import { Area, AreaChart, XAxis, YAxis, Tooltip, ResponsiveContainer, BarChart, Bar, CartesianGrid, Cell } from "recharts";
import Link from "next/link";
import { ArrowUpRight, ArrowDown } from "lucide-react";
import { chainColor, chainName, compactNum } from "@/lib/format";
import { Card, Stat, Skeleton, Empty, ProtocolPill, TagPill } from "@/components/ui";
import { ChainIcon } from "@/components/ChainIcon";

type Stats = {
  chains: { chain_id: number; events: string }[];
  protocols: { protocol: string; event_type: string; events: string; users: string }[];
  series: { bucket: string; events: string }[];
};

export default function Home() {
  const WINDOW_MIN = 1440;
  const { data, isLoading, error } = useQuery<Stats>({
    queryKey: ["stats", WINDOW_MIN],
    queryFn: async () => {
      const r = await fetch(`/api/stats?minutes=${WINDOW_MIN}`);
      if (!r.ok) throw new Error(await r.text());
      return r.json();
    },
    refetchInterval: 10_000,
  });

  const totalEvents = data?.series.reduce((a, b) => a + parseInt(b.events, 10), 0) ?? 0;
  const series = (data?.series ?? []).map((p) => {
    const iso = p.bucket.includes("T") ? p.bucket : p.bucket.replace(" ", "T") + "Z";
    const d = new Date(iso);
    return {
      t: d.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit", hour12: false, timeZone: "UTC" }),
      ist: d.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit", hour12: false, timeZone: "Asia/Kolkata" }),
      events: parseInt(p.events, 10),
    };
  });
  const chains = (data?.chains ?? []).map((c) => ({
    id: c.chain_id,
    name: chainName(c.chain_id),
    events: parseInt(c.events, 10),
    fill: chainColor(c.chain_id),
  }));
  const totalUsers = (data?.protocols ?? []).reduce((a, b) => a + parseInt(b.users, 10), 0);
  const protocolCount = new Set(data?.protocols.map((p) => p.protocol)).size;

  return (
    <div className="space-y-12">
      {/* HERO */}
      <section className="text-center pt-8 pb-4">
        <TagPill className="mb-6">DeFi · Real-time</TagPill>
        <h1 className="font-display text-[64px] lg:text-[88px] font-extrabold leading-[0.95] tracking-[-0.035em] text-[var(--fg-strong)]">
          Watch DeFi
          <br />
          <span className="inline-flex items-center gap-4">
            <span>flow,</span>
            <span className="inline-flex items-center justify-center h-[0.6em] px-[0.32em] rounded-full bg-gradient-to-br from-[var(--accent)] to-[var(--accent-hover)] border border-[var(--accent-hover)] live-glow">
              <span className="text-white font-display font-extrabold text-[0.3em] tracking-[0.12em] live-text">LIVE</span>
            </span>
            <span>.</span>
          </span>
        </h1>
        <p className="text-[16px] text-[var(--fg-muted)] mt-6 max-w-xl mx-auto leading-relaxed">
          Multi-chain blockchain indexer with sub-block latency. Watch whales move, look up wallets, monitor protocol flow across three networks.
        </p>
        <div className="mt-8 flex items-center justify-center gap-3 flex-wrap">
          <Link href="/whales" className="btn-primary">
            Open Live Feed <ArrowUpRight className="size-4" />
          </Link>
          <Link href="/wallet" className="btn-ghost">
            Wallet Lookup
          </Link>
        </div>
      </section>

      {/* PARTNERS / CHAIN STRIP */}
      <section className="card-surface px-8 py-6 flex items-center justify-between flex-wrap gap-4">
        <div className="flex items-center gap-3">
          <span className="size-9 rounded-full border border-[var(--border-strong)] flex items-center justify-center">
            <span className="size-1.5 rounded-full bg-[var(--accent)] pulse-dot" />
          </span>
          <span className="text-[10px] font-mono uppercase tracking-[0.18em] font-semibold text-[var(--fg-strong)]">Streaming · 3 of 3 Chains</span>
        </div>
        <div className="flex items-center gap-6 flex-wrap">
          {["Ethereum", "Arbitrum", "Polygon"].map((c) => (
            <span key={c} className="inline-flex items-center gap-2.5 text-[15px] font-mono uppercase tracking-[0.14em] text-[var(--fg-muted)]">
              <ChainIcon name={c} size={24} />
              {c}
            </span>
          ))}
        </div>
        <a
          href="#live-pulse"
          onClick={(e) => {
            e.preventDefault();
            const el = document.getElementById("live-pulse");
            if (!el) return;
            const top = el.getBoundingClientRect().top + window.scrollY - 96;
            window.scrollTo({ top, behavior: "smooth" });
          }}
          className="text-[10px] uppercase tracking-[0.18em] font-mono font-semibold text-[var(--fg-strong)] hover:text-[var(--accent)] inline-flex items-center gap-1.5"
        >
          Scroll Down <ArrowDown className="size-3.5" />
        </a>
      </section>

      {error && <div className="rounded-2xl border border-rose-200 bg-rose-50 p-4 text-sm text-rose-700 dark:bg-rose-500/10 dark:text-rose-300 dark:border-rose-400/30">{String(error)}</div>}

      {/* WHAT WE SHOW */}
      <section>
        <div className="text-center mb-10">
          <TagPill className="mb-4">What We Show</TagPill>
          <h2 className="font-display text-[36px] lg:text-[48px] font-bold leading-[1.05] tracking-[-0.025em]">
            Real-Time Insights<br />Across Every Chain
          </h2>
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-3 gap-5">
          <FeatureCard
            num="01"
            title="Whale Activity"
            body="Track large swaps, borrows, and liquidations as they happen on Uniswap, Aave, and Compound."
            href="/whales"
          />
          <FeatureCard
            num="02"
            title="Wallet Lookup"
            body="Paste any EVM address. See DeFi positions, token transfers, and protocol footprint instantly."
            href="/wallet"
          />
          <FeatureCard
            num="03"
            title="Protocol Pulse"
            body="Volume, unique users, and event flow per protocol across three indexed chains."
            href="/whales"
          />
        </div>
      </section>

      {/* IMPACT NUMBER */}
      <section id="live-pulse" className="card-surface px-8 py-12 text-center scroll-mt-24">
        <TagPill className="mb-4">Live Impact</TagPill>
        <h2 className="font-display text-[28px] lg:text-[36px] font-bold leading-[1.1]">
          Together, we&apos;re indexing<br />a global DeFi heartbeat.
        </h2>
        <div className="num-display text-[80px] lg:text-[112px] text-[var(--accent)] mt-8 leading-none">
          {isLoading ? "—" : compactNum(totalEvents)}
        </div>
        <div className="text-[14px] font-semibold text-[var(--fg-strong)] mt-4">Events Indexed (Last 2h)</div>
        <div className="grid grid-cols-3 gap-4 mt-10 max-w-2xl mx-auto">
          <MiniStat label="Active chains" value={isLoading ? "—" : `${data?.chains.length ?? 0} / 3`} />
          <MiniStat label="Protocols" value={isLoading ? "—" : protocolCount.toString()} />
          <MiniStat label="Wallets" value={isLoading ? "—" : compactNum(totalUsers)} />
        </div>
      </section>

      {/* THROUGHPUT CHART */}
      <Card
        title="Event Throughput"
        subtitle="Events per 5 minutes · last 2 hours"
        action={
          <span className="text-[10px] font-mono uppercase tracking-[0.16em] text-[var(--fg-muted)] font-semibold">Refreshes 10s</span>
        }
      >
        <div className="h-72">
          {isLoading ? (
            <Skeleton className="h-full w-full" />
          ) : series.length === 0 ? (
            <Empty title="No events yet" hint="Indexer warming up." />
          ) : (
            <ResponsiveContainer width="100%" height="100%">
              <AreaChart data={series} margin={{ left: 0, right: 8, top: 10, bottom: 24 }}>
                <defs>
                  <linearGradient id="evgrad" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="0%" stopColor="#22c55e" stopOpacity={0.25} />
                    <stop offset="100%" stopColor="#22c55e" stopOpacity={0} />
                  </linearGradient>
                </defs>
                <CartesianGrid stroke="var(--chart-grid)" strokeDasharray="2 4" vertical={false} />
                <XAxis
                  dataKey="t"
                  stroke="var(--chart-axis)"
                  fontSize={11}
                  tickLine={false}
                  axisLine={false}
                  dy={4}
                  tickFormatter={(v, i) => {
                    const ist = series[i]?.ist;
                    return ist ? `${v} · ${ist}` : v;
                  }}
                  interval="preserveStartEnd"
                  minTickGap={60}
                  label={{ value: "Time · UTC · IST", position: "insideBottom", offset: -16, fill: "var(--fg-muted)", fontSize: 10, letterSpacing: 1, style: { textTransform: "uppercase" } }}
                />
                <YAxis stroke="var(--chart-axis)" fontSize={11} tickLine={false} axisLine={false} tickFormatter={(v) => compactNum(v)} dx={-4} />
                <Tooltip
                  contentStyle={{ background: "var(--chart-tooltip-bg)", border: "1px solid var(--chart-tooltip-border)", borderRadius: 12, fontSize: 12, padding: "8px 12px", boxShadow: "var(--shadow)" }}
                  labelStyle={{ color: "var(--chart-tooltip-fg)", fontSize: 10, marginBottom: 4, textTransform: "uppercase", letterSpacing: 1 }}
                  itemStyle={{ color: "var(--accent)", fontWeight: 700 }}
                  cursor={{ stroke: "var(--accent)", strokeWidth: 1, strokeDasharray: "3 3" }}
                  labelFormatter={(label, payload) => {
                    const p = payload?.[0]?.payload as { t: string; ist: string } | undefined;
                    return p ? `${p.t} UTC · ${p.ist} IST` : String(label);
                  }}
                />
                <Area type="monotone" dataKey="events" stroke="var(--accent)" strokeWidth={2.5} fill="url(#evgrad)" />
              </AreaChart>
            </ResponsiveContainer>
          )}
        </div>
      </Card>

      {/* DUAL */}
      <div className="grid grid-cols-1 lg:grid-cols-5 gap-5">
        <Card title="By Chain" subtitle="Event distribution" className="lg:col-span-2">
          <div className="h-64">
            {isLoading ? (
              <Skeleton className="h-full w-full" />
            ) : chains.length === 0 ? (
              <Empty title="Awaiting events" />
            ) : (
              <ResponsiveContainer width="100%" height="100%">
                <BarChart data={chains} margin={{ left: -10, right: 0, top: 10, bottom: 0 }}>
                  <CartesianGrid stroke="var(--chart-grid)" strokeDasharray="2 4" vertical={false} />
                  <XAxis dataKey="name" stroke="var(--chart-axis)" fontSize={10} tickLine={false} axisLine={false} dy={4} />
                  <YAxis stroke="var(--chart-axis)" fontSize={11} tickLine={false} axisLine={false} tickFormatter={(v) => compactNum(v)} dx={-4} />
                  <Tooltip
                    contentStyle={{ background: "var(--chart-tooltip-bg)", border: "1px solid var(--chart-tooltip-border)", borderRadius: 12, fontSize: 12, padding: "8px 12px" }}
                    cursor={{ fill: "var(--accent-soft)" }}
                  />
                  <Bar dataKey="events" radius={[8, 8, 0, 0]}>
                    {chains.map((c, i) => <Cell key={i} fill={c.fill} />)}
                  </Bar>
                </BarChart>
              </ResponsiveContainer>
            )}
          </div>
        </Card>

        <Card title="Top Protocols" subtitle="Sorted by event volume" className="lg:col-span-3">
          <div className="space-y-1">
            {isLoading && Array.from({ length: 6 }).map((_, i) => <Skeleton key={i} className="h-9 w-full" />)}
            {!isLoading && (data?.protocols.length ?? 0) === 0 && <Empty title="No protocol activity yet" />}
            {(data?.protocols ?? []).slice(0, 8).map((p, i) => {
              const events = parseInt(p.events, 10);
              const max = parseInt(data?.protocols[0]?.events ?? "1", 10);
              const pct = (events / max) * 100;
              return (
                <div key={i} className="relative">
                  <div className="absolute inset-0 rounded-full bg-[var(--accent-soft)]" style={{ width: `${pct}%` }} />
                  <div className="relative flex items-center justify-between text-sm py-2 px-3 rounded-full">
                    <div className="flex items-center gap-2 min-w-0">
                      <span className="text-[10px] tabular-nums text-[var(--fg-dim)] w-5 font-mono font-semibold">{(i + 1).toString().padStart(2, "0")}</span>
                      <ProtocolPill p={p.protocol} />
                      <span className="text-[var(--fg-muted)] text-[11.5px]">{p.event_type}</span>
                    </div>
                    <div className="flex gap-5 text-[12px] tnum shrink-0">
                      <span className="text-[var(--fg-strong)] font-semibold">{compactNum(events)}</span>
                      <span className="text-[var(--fg-muted)]">{compactNum(parseInt(p.users, 10))} users</span>
                    </div>
                  </div>
                </div>
              );
            })}
          </div>
        </Card>
      </div>
    </div>
  );
}

function FeatureCard({ num, title, body, href }: { num: string; title: string; body: string; href: string }) {
  return (
    <Link href={href} className="card-surface p-6 group hover:border-[var(--fg-strong)] transition-all flex flex-col gap-4">
      <div className="flex items-center justify-between">
        <span className="size-9 rounded-full border border-[var(--border)] bg-[var(--bg-elevated)] flex items-center justify-center font-mono text-[11px] font-bold text-[var(--fg-strong)]">{num}</span>
        <span className="size-9 rounded-full border border-[var(--border)] bg-[var(--bg-elevated)] flex items-center justify-center group-hover:bg-[var(--fg-strong)] group-hover:border-[var(--fg-strong)] transition-all">
          <ArrowUpRight className="size-4 text-[var(--fg-strong)] group-hover:text-[var(--bg)] transition-colors" />
        </span>
      </div>
      <div>
        <h3 className="font-display text-[20px] font-bold text-[var(--fg-strong)] tracking-tight">{title}</h3>
        <p className="text-[13.5px] text-[var(--fg-muted)] mt-2 leading-relaxed">{body}</p>
      </div>
    </Link>
  );
}

function MiniStat({ label, value }: { label: string; value: string }) {
  return (
    <div className="text-center">
      <div className="num-display text-[28px] text-[var(--fg-strong)]">{value}</div>
      <div className="text-[11px] uppercase tracking-[0.14em] text-[var(--fg-muted)] font-mono font-semibold mt-1">{label}</div>
    </div>
  );
}
