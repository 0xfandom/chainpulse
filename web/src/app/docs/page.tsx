"use client";
import Link from "next/link";
import { createContext, useContext, useEffect, useState } from "react";
import { ArrowRight, Check, Copy, Users, Code2, Sparkles, Zap, Network, Activity, Database, ArrowUpRight, Server, Layers, Globe2, Bot, Eye, Container, Package, Cpu, Wifi } from "lucide-react";

const SECTIONS = [
  { id: "intro", n: "01", label: "Introduction" },
  { id: "state", n: "02", label: "Current state" },
  { id: "architecture", n: "03", label: "Architecture" },
  { id: "chains", n: "04", label: "Supported chains" },
  { id: "protocols", n: "05", label: "Decoded protocols" },
  { id: "schema", n: "06", label: "Data model" },
  { id: "rest", n: "07", label: "REST + gRPC" },
  { id: "ws", n: "08", label: "WebSocket" },
  { id: "mcp", n: "09", label: "MCP tools" },
  { id: "ops", n: "10", label: "Operations" },
  { id: "perf", n: "11", label: "Performance" },
  { id: "security", n: "12", label: "Security" },
];

const ActiveCtx = createContext<string>("intro");

export default function DocsPage() {
  const [active, setActive] = useState("intro");

  useEffect(() => {
    if (typeof window === "undefined") return;
    const fromHash = window.location.hash.slice(1);
    if (fromHash && SECTIONS.some((s) => s.id === fromHash)) {
      setActive(fromHash);
    }
    const onHash = () => {
      const h = window.location.hash.slice(1);
      if (h && SECTIONS.some((s) => s.id === h)) setActive(h);
    };
    window.addEventListener("hashchange", onHash);
    return () => window.removeEventListener("hashchange", onHash);
  }, []);

  const go = (id: string) => {
    setActive(id);
    if (typeof window !== "undefined") {
      history.replaceState(null, "", `#${id}`);
      window.scrollTo({ top: 0, behavior: "instant" as ScrollBehavior });
    }
  };

  const activeIdx = SECTIONS.findIndex((s) => s.id === active);
  const prev = activeIdx > 0 ? SECTIONS[activeIdx - 1] : null;
  const next = activeIdx < SECTIONS.length - 1 ? SECTIONS[activeIdx + 1] : null;

  return (
    <ActiveCtx.Provider value={active}>
      <div className="flex min-h-[calc(100vh-80px)]">
        <article className="flex-1 min-w-0 px-6 sm:px-10 lg:px-16 py-12 pb-24 order-1">
          <div className="max-w-[1100px] mx-auto">
            <header className="mb-12 pb-8 border-b border-[var(--border)] flex items-center justify-between flex-wrap gap-3">
              <div>
                <div className="text-[10px] uppercase tracking-[0.18em] text-[var(--fg-dim)] font-semibold font-mono mb-2">
                  Section {SECTIONS[activeIdx]?.n} of {SECTIONS.length.toString().padStart(2, "0")}
                </div>
                <h1 className="text-[36px] sm:text-[44px] font-semibold tracking-[-0.03em] text-[var(--fg-strong)] leading-[1.05]">
                  {SECTIONS[activeIdx]?.label}
                </h1>
              </div>
              <div className="lg:hidden">
                <select
                  value={active}
                  onChange={(e) => go(e.target.value)}
                  className="text-[12px] font-mono bg-[var(--bg-card)] border border-[var(--border)] rounded-md px-3 py-2 text-[var(--fg-strong)]"
                >
                  {SECTIONS.map((s) => (
                    <option key={s.id} value={s.id}>{s.n} · {s.label}</option>
                  ))}
                </select>
              </div>
            </header>

        <Section id="intro" n="01" title="Introduction">
          {/* Hero block */}
          <div className="relative overflow-hidden rounded-2xl border border-[var(--border)] bg-gradient-to-br from-[var(--accent-soft)] via-[var(--bg-card)] to-[var(--bg-card)] p-8 sm:p-10 mb-10">
            <div className="absolute -right-12 -top-12 size-56 rounded-full bg-[var(--accent-soft)] blur-3xl opacity-60 pointer-events-none" />
            <div className="relative z-10 max-w-[820px]">
              <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full border border-[var(--accent-border)] bg-[var(--accent-soft)] text-[10.5px] font-mono uppercase tracking-[0.18em] text-[var(--accent-fg)] font-semibold">
                <span className="size-1.5 rounded-full bg-[var(--accent)] pulse-dot" />
                Open source · Apache-2.0 · Self-hosted
              </div>
              <h2 className="font-display text-[40px] sm:text-[56px] font-extrabold tracking-[-0.035em] text-[var(--fg-strong)] leading-[1.02] mt-5">
                The on-chain data layer
                <br />
                <span className="text-[var(--accent)] font-serif italic font-medium">your stack already wanted.</span>
              </h2>
              <p className="text-[16px] sm:text-[17px] text-[var(--fg-muted)] mt-6 leading-[1.65] max-w-[680px]">
                ChainPulse listens to 9 EVM networks in real time, decodes protocol events into a normalised schema, and exposes the data through REST, gRPC, WebSocket, and an MCP server for AI agents. Block on chain to queryable row in <span className="text-[var(--fg-strong)] font-semibold">1–2 seconds</span>. No third-party API, no telemetry, no data leaves your laptop.
              </p>

              <div className="flex flex-wrap items-center gap-2 mt-7">
                <button
                  onClick={() => go("architecture")}
                  className="inline-flex items-center gap-1.5 px-4 py-2 rounded-full bg-[var(--accent)] hover:bg-[var(--accent-hover)] text-white text-[13px] font-semibold transition-colors"
                >
                  See the architecture <ArrowRight className="size-3.5" />
                </button>
                <button
                  onClick={() => go("mcp")}
                  className="inline-flex items-center gap-1.5 px-4 py-2 rounded-full border border-[var(--border-strong)] hover:border-[var(--fg-strong)] text-[var(--fg-strong)] text-[13px] font-semibold transition-colors"
                >
                  Plug Claude into it <ArrowRight className="size-3.5" />
                </button>
                <a
                  href="https://github.com/0xfandom/chainpulse"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="inline-flex items-center gap-1.5 px-4 py-2 rounded-full text-[var(--fg-muted)] hover:text-[var(--fg-strong)] text-[13px] font-semibold transition-colors"
                >
                  GitHub <ArrowUpRight className="size-3.5" />
                </a>
              </div>
            </div>
          </div>

          {/* Stat strip */}
          <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-6 gap-3 mb-12">
            {[
              { v: "9", l: "EVM chains", s: "3 default · 6 optional" },
              { v: "6", l: "Protocols", s: "ERC-20 + 5 DeFi" },
              { v: "8", l: "MCP tools", s: "Claude Desktop ready" },
              { v: "4", l: "Surfaces", s: "REST · gRPC · WS · MCP" },
              { v: "<2s", l: "Block to row", s: "RPC to ClickHouse" },
              { v: "100%", l: "Self-hosted", s: "Docker Compose" },
            ].map((m) => (
              <div
                key={m.l}
                className="rounded-xl border border-[var(--border)] bg-[var(--bg-card)] px-4 py-5 flex flex-col items-start justify-between min-h-[124px] hover:border-[var(--border-strong)] transition-colors"
              >
                <div className="font-mono text-[28px] font-bold text-[var(--fg-strong)] tabular-nums leading-none tracking-[-0.02em] h-[28px] flex items-center">
                  {m.v}
                </div>
                <div className="w-full">
                  <div className="text-[10.5px] uppercase tracking-[0.16em] text-[var(--fg-strong)] font-semibold leading-tight">
                    {m.l}
                  </div>
                  <div className="text-[10.5px] text-[var(--fg-muted)] leading-snug mt-1 line-clamp-2">
                    {m.s}
                  </div>
                </div>
              </div>
            ))}
          </div>

          {/* Three audiences as bold cards */}
          <div className="mb-2">
            <div className="text-[10.5px] uppercase tracking-[0.18em] text-[var(--fg-dim)] font-mono font-semibold mb-2">Built for</div>
            <h3 className="text-[22px] sm:text-[26px] font-semibold tracking-[-0.022em] text-[var(--fg-strong)] leading-tight">Three audiences, one indexer.</h3>
            <p className="text-[14px] text-[var(--fg-muted)] mt-2 leading-[1.65] max-w-[680px]">
              Whoever you are, the data path is identical: chain WebSocket → Kafka → ClickHouse → query. Pick the surface that fits the caller.
            </p>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-3 gap-4 my-8">
            {[
              {
                icon: Users,
                tone: "accent",
                label: "Humans",
                title: "Web dashboard",
                body: "Live whale feed across chains, wallet/token/block lookup, protocol pulse. Bundled Next.js explorer on :3000. Nothing to install — already running.",
                chips: ["Whale feed", "Wallet lookup", "Search bar"],
                action: { label: "Open the dashboard", href: "/" as const, external: false },
              },
              {
                icon: Code2,
                tone: "info",
                label: "Engineers",
                title: "REST · gRPC · WebSocket",
                body: "HTTP endpoints for wallet positions, balances, history, and protocol stats. WebSocket streams every decoded event as it lands. gRPC for typed backend-to-backend calls.",
                chips: ["/v1 endpoints", ":8081 gRPC", "/v1/events/stream"],
                action: { label: "Jump to REST + gRPC", section: "rest" },
              },
              {
                icon: Sparkles,
                tone: "violet",
                label: "AI agents",
                title: "MCP server",
                body: "8 structured tools any MCP-compatible LLM can call. Claude or GPT answer wallet questions without touching contract ABIs. stdio for Claude Desktop, SSE for HTTP agents.",
                chips: ["8 tools", "stdio", "SSE :3001"],
                action: { label: "Wire up Claude", section: "mcp" },
              },
            ].map((card) => {
              const Icon = card.icon;
              const toneStyles: Record<string, { ring: string; bg: string; fg: string }> = {
                accent: { ring: "border-[var(--accent-border)] hover:border-[var(--accent)]", bg: "bg-[var(--accent-soft)]", fg: "text-[var(--accent-fg)]" },
                info:   { ring: "border-[#3b82f655] hover:border-[var(--info)]", bg: "bg-[#3b82f61a]", fg: "text-[#1d4ed8] dark:text-[#93c5fd]" },
                violet: { ring: "border-[#a78bfa55] hover:border-[#a78bfa]", bg: "bg-[#a78bfa1a]", fg: "text-[#6d28d9] dark:text-[#c4b5fd]" },
              };
              const s = toneStyles[card.tone];
              return (
                <div key={card.label} className={`group rounded-xl border ${s.ring} bg-[var(--bg-card)] p-6 flex flex-col transition-colors`}>
                  <div className={`size-10 rounded-lg ${s.bg} ${s.fg} flex items-center justify-center mb-4`}>
                    <Icon className="size-5" />
                  </div>
                  <div className="text-[10px] uppercase tracking-[0.18em] font-mono text-[var(--fg-dim)] font-semibold">{card.label}</div>
                  <div className="text-[16px] font-semibold text-[var(--fg-strong)] tracking-tight mt-1">{card.title}</div>
                  <p className="text-[13px] text-[var(--fg-muted)] mt-3 leading-[1.65] flex-1">{card.body}</p>
                  <div className="flex flex-wrap gap-1.5 mt-4">
                    {card.chips.map((c) => (
                      <span key={c} className="inline-flex items-center px-2 py-0.5 rounded border border-[var(--border)] bg-[var(--bg-elevated)] text-[10.5px] font-mono text-[var(--fg-muted)]">{c}</span>
                    ))}
                  </div>
                  {card.action && (
                    "href" in card.action ? (
                      <Link href={card.action.href} className={`mt-5 inline-flex items-center gap-1 text-[12.5px] font-semibold ${s.fg} hover:opacity-80`}>
                        {card.action.label} <ArrowRight className="size-3.5" />
                      </Link>
                    ) : (
                      <button onClick={() => go(card.action.section)} className={`mt-5 inline-flex items-center gap-1 text-[12.5px] font-semibold ${s.fg} hover:opacity-80 self-start`}>
                        {card.action.label} <ArrowRight className="size-3.5" />
                      </button>
                    )
                  )}
                </div>
              );
            })}
          </div>

          {/* Suggested reading paths */}
          <div className="mt-12 mb-4">
            <div className="text-[10.5px] uppercase tracking-[0.18em] text-[var(--fg-dim)] font-mono font-semibold mb-2">Where to next</div>
            <h3 className="text-[22px] sm:text-[26px] font-semibold tracking-[-0.022em] text-[var(--fg-strong)] leading-tight">Read this if you want to…</h3>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-3 my-6">
            {[
              { icon: Network,  ask: "Understand how a block becomes a row",   target: "architecture", n: "03" },
              { icon: Database, ask: "See the ClickHouse schema you'll query", target: "schema",       n: "06" },
              { icon: Zap,      ask: "Wire Claude / Cursor / agents in",       target: "mcp",          n: "09" },
              { icon: Activity, ask: "Check what's green vs what's red",       target: "ops",          n: "10" },
            ].map((p) => {
              const Icon = p.icon;
              return (
                <button
                  key={p.target}
                  onClick={() => go(p.target)}
                  className="group flex items-center gap-4 px-5 py-4 rounded-lg border border-[var(--border)] bg-[var(--bg-card)] hover:border-[var(--fg-strong)] hover:bg-[var(--bg-hover)] transition-colors text-left"
                >
                  <div className="size-9 rounded-md bg-[var(--bg-elevated)] border border-[var(--border)] flex items-center justify-center shrink-0 text-[var(--fg-muted)] group-hover:text-[var(--accent)] transition-colors">
                    <Icon className="size-4" />
                  </div>
                  <div className="flex-1 min-w-0">
                    <div className="text-[13.5px] text-[var(--fg-strong)] font-medium leading-tight">{p.ask}</div>
                    <div className="text-[11px] text-[var(--fg-dim)] font-mono mt-0.5">Section {p.n}</div>
                  </div>
                  <ArrowRight className="size-4 text-[var(--fg-dim)] group-hover:text-[var(--fg-strong)] group-hover:translate-x-0.5 transition-all shrink-0" />
                </button>
              );
            })}
          </div>

          <Callout>
            <strong>Everything runs locally.</strong> ChainPulse has no telemetry, no phone-home, no usage tracking. The chain data you see depends entirely on the RPC URLs you point it at — your provider keys never leave the host.
          </Callout>
        </Section>

        <Section id="state" n="02" title="Current state">
          <Lead>
            Snapshot of what is live in the repo and the demo stack today. Numbers match the published container images and the default <Code>config/config.example.toml</Code>.
          </Lead>

          {/* Status banner */}
          <div className="rounded-xl border border-[var(--accent-border)] bg-[var(--accent-soft)] px-5 py-4 mb-8 flex items-center gap-4">
            <span className="size-2.5 rounded-full bg-[var(--accent)] pulse-dot shrink-0" />
            <div className="flex-1 min-w-0">
              <div className="text-[13px] font-semibold text-[var(--fg-strong)]">Demo stack is live</div>
              <div className="text-[12px] text-[var(--fg-muted)] mt-0.5">Indexing Ethereum, Polygon, Arbitrum · 6 protocols decoded · 8 MCP tools online · single <Code>docker compose up</Code> away on your laptop.</div>
            </div>
            <span className="hidden sm:inline-flex items-center gap-1.5 px-3 py-1 rounded-full border border-[var(--accent-border)] bg-[var(--bg-card)] text-[10px] font-mono uppercase tracking-[0.16em] text-[var(--accent-fg)] font-semibold whitespace-nowrap">
              v0.1.0 · main
            </span>
          </div>

          {/* Big stat cards */}
          <div className="text-[10.5px] uppercase tracking-[0.18em] text-[var(--fg-dim)] font-mono font-semibold mb-3">Coverage</div>
          <div className="grid grid-cols-2 sm:grid-cols-4 gap-3 mb-10">
            {[
              { icon: Network,  v: "9",  l: "EVM chains", s: "3 default · 6 optional" },
              { icon: Layers,   v: "6",  l: "Protocols",  s: "ERC-20 + 5 DeFi" },
              { icon: Bot,      v: "8",  l: "MCP tools",  s: "Claude Desktop ready" },
              { icon: Server,   v: "4",  l: "Binaries",   s: "indexer · processor · api · mcp" },
            ].map((m) => {
              const Icon = m.icon;
              return (
                <div key={m.l} className="rounded-xl border border-[var(--border)] bg-[var(--bg-card)] p-5 flex flex-col gap-3 hover:border-[var(--border-strong)] transition-colors">
                  <div className="flex items-center justify-between">
                    <div className="size-9 rounded-lg bg-[var(--bg-elevated)] border border-[var(--border)] flex items-center justify-center text-[var(--fg-muted)]">
                      <Icon className="size-4" />
                    </div>
                    <div className="font-mono text-[28px] font-bold text-[var(--fg-strong)] tabular-nums leading-none tracking-[-0.02em]">
                      {m.v}
                    </div>
                  </div>
                  <div>
                    <div className="text-[11px] uppercase tracking-[0.16em] text-[var(--fg-strong)] font-semibold leading-tight">{m.l}</div>
                    <div className="text-[11px] text-[var(--fg-muted)] leading-snug mt-1 line-clamp-2">{m.s}</div>
                  </div>
                </div>
              );
            })}
          </div>

          {/* Query surfaces */}
          <div className="text-[10.5px] uppercase tracking-[0.18em] text-[var(--fg-dim)] font-mono font-semibold mb-3">Query surfaces</div>
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3 mb-10">
            {[
              { icon: Globe2,   port: ":8080", name: "REST",      path: "/v1/...",            tone: "info" },
              { icon: Code2,    port: ":8081", name: "gRPC",      path: "chainpulse.v1 + reflection", tone: "info" },
              { icon: Wifi,     port: ":8080", name: "WebSocket", path: "/v1/events/stream",  tone: "violet" },
              { icon: Sparkles, port: ":3001", name: "MCP",       path: "stdio · SSE",         tone: "accent" },
            ].map((s) => {
              const Icon = s.icon;
              const toneStyles: Record<string, string> = {
                accent: "border-[var(--accent-border)] bg-[var(--accent-soft)] text-[var(--accent-fg)]",
                info:   "border-[#3b82f655] bg-[#3b82f61a] text-[#1d4ed8] dark:text-[#93c5fd]",
                violet: "border-[#a78bfa55] bg-[#a78bfa1a] text-[#6d28d9] dark:text-[#c4b5fd]",
              };
              return (
                <div key={s.name} className="rounded-xl border border-[var(--border)] bg-[var(--bg-card)] p-5">
                  <div className="flex items-center gap-3 mb-3">
                    <div className={`size-9 rounded-lg border flex items-center justify-center ${toneStyles[s.tone]}`}>
                      <Icon className="size-4" />
                    </div>
                    <div>
                      <div className="text-[14px] font-semibold text-[var(--fg-strong)] tracking-tight leading-tight">{s.name}</div>
                      <div className="font-mono text-[10.5px] text-[var(--fg-dim)] tracking-tight">{s.port}</div>
                    </div>
                  </div>
                  <div className="text-[11px] font-mono text-[var(--fg-muted)] leading-snug break-words">{s.path}</div>
                </div>
              );
            })}
          </div>

          {/* Run + watch */}
          <div className="text-[10.5px] uppercase tracking-[0.18em] text-[var(--fg-dim)] font-mono font-semibold mb-3">Run · watch · ship</div>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-3 mb-2">
            {[
              {
                icon: Container,
                label: "Self-host",
                value: "docker compose up",
                body: "Single command boots the full stack on a laptop. Multi-stage Dockerfile, lite-memory profile for dev machines.",
              },
              {
                icon: Cpu,
                label: "Web UI",
                value: ":3000",
                body: "Next.js explorer · live whale feed · wallet/token/block lookup · light + dark themes.",
              },
              {
                icon: Eye,
                label: "Observability",
                value: "Prom + Grafana",
                body: "Prometheus on :9090, auto-provisioned 6-panel Grafana on :3030 — events, requests, MCP, latency, chains, lag.",
              },
              {
                icon: Package,
                label: "Distribution",
                value: "GHCR",
                body: "Container images for all four binaries published to GitHub Container Registry on every tag.",
              },
            ].map((c) => {
              const Icon = c.icon;
              return (
                <div key={c.label} className="rounded-xl border border-[var(--border)] bg-[var(--bg-card)] p-5 flex flex-col gap-3 hover:border-[var(--border-strong)] transition-colors">
                  <div className="flex items-center gap-3">
                    <div className="size-9 rounded-lg bg-[var(--bg-elevated)] border border-[var(--border)] flex items-center justify-center text-[var(--fg-muted)]">
                      <Icon className="size-4" />
                    </div>
                    <div>
                      <div className="text-[10px] uppercase tracking-[0.16em] text-[var(--fg-dim)] font-mono font-semibold">{c.label}</div>
                      <div className="text-[14px] font-semibold text-[var(--fg-strong)] tracking-tight leading-tight font-mono">{c.value}</div>
                    </div>
                  </div>
                  <p className="text-[12px] text-[var(--fg-muted)] leading-[1.6]">{c.body}</p>
                </div>
              );
            })}
          </div>
        </Section>

        <Section id="architecture" n="03" title="Architecture">
          <Lead>
            Four Go binaries connected by two Kafka topics. The pipeline is one straight line: a block on chain enters the indexer, leaves the API as JSON. Every stage exists because the previous stage&apos;s output forces it — no aesthetic choices, the data dependency is the shape.
          </Lead>

          <P>
            Read the diagram below top to bottom. Each card is one stage in the pipeline, with the components that live inside it (chips) and a sentence on what it does. Colour codes the role: green = ingest, purple = Kafka bus, amber = process, red = storage, blue = serve, teal = client. The whole journey from "a new block was just produced" to "a wallet&apos;s positions are queryable" runs in ~1–2 seconds.
          </P>

          <FlowDiagram />

          <div className="text-[10.5px] uppercase tracking-[0.18em] text-[var(--fg-dim)] font-mono font-semibold mt-10 mb-3">Why this shape</div>
          <P>
            Kafka in the middle is the load-bearing decision. It separates write-heavy work (the indexer, which must keep up with chain headers in real time) from read-heavy work (the API and MCP, which serve many concurrent reads). Three things fall out of that split.
          </P>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-3 mb-8">
            {[
              { icon: Activity, name: "Backpressure", body: "If ClickHouse is slow or recovering, the processor lag grows but the indexer keeps writing to Kafka. No event is lost. The chain does not wait for the database." },
              { icon: Database, name: "Replay",       body: "Add a new decoder next week and reprocess the last 7 days of raw_events from Kafka — no need to re-fetch logs from RPC. Free reindex of history." },
              { icon: Network,  name: "Fan-out",      body: "decoded_events is a second topic written after storage. Every WebSocket client joins an ephemeral group on it at LastOffset — one decode feeds storage + every live subscriber." },
            ].map((c) => {
              const Icon = c.icon;
              return (
                <div key={c.name} className="rounded-xl border border-[var(--border)] bg-[var(--bg-card)] p-5">
                  <div className="size-9 rounded-lg bg-[var(--bg-elevated)] border border-[var(--border)] flex items-center justify-center text-[var(--fg-muted)] mb-3">
                    <Icon className="size-4" />
                  </div>
                  <div className="text-[13.5px] font-semibold text-[var(--fg-strong)] tracking-tight">{c.name}</div>
                  <p className="text-[12px] text-[var(--fg-muted)] leading-[1.6] mt-2">{c.body}</p>
                </div>
              );
            })}
          </div>

          <div className="text-[10.5px] uppercase tracking-[0.18em] text-[var(--fg-dim)] font-mono font-semibold mt-10 mb-3">The four binaries</div>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-3 mb-8">
            {[
              { name: "cmd/indexer",   tag: "ingest",  tone: "accent", body: "One goroutine per chain. Holds a WSS subscription, decodes log topics, wraps the event in a ChainEvent, publishes to raw_events. /health + /ready on :9180, /metrics on :9100. Backoff 1s → 30s × 10 on disconnect." },
              { name: "cmd/processor", tag: "process", tone: "warn",   body: "Consumes raw_events with manual offset commit — bookmark advances only after the write succeeds. ProtocolDecoder registry (first match wins). Batches ClickHouse, updates Redis, republishes to decoded_events. ClickHouse dedupes replays." },
              { name: "cmd/api",       tag: "serve",   tone: "info",   body: "Gin REST on :8080, gRPC on :8081 (reflection on), WebSocket on /v1/events/stream. Cache-aside (Redis → ClickHouse), 100 req/min/IP, CORS whitelist, structured access logs, Prometheus middleware." },
              { name: "cmd/mcp",       tag: "serve",   tone: "violet", body: "JSON-RPC 2.0 with 8 tool definitions. stdio (Claude Desktop child process) or SSE :3001 with optional bearer token. Reuses ReadStore + ReadCache — no separate query path. Bad args fail with -32602 InvalidParams + data[]." },
            ].map((b) => {
              const toneStyles: Record<string, string> = {
                accent: "border-l-[var(--accent)]",
                warn:   "border-l-[var(--warn)]",
                info:   "border-l-[var(--info)]",
                violet: "border-l-[#a78bfa]",
              };
              return (
                <div key={b.name} className={`rounded-xl border border-[var(--border)] border-l-[3px] ${toneStyles[b.tone]} bg-[var(--bg-card)] p-5`}>
                  <div className="flex items-center justify-between gap-2 mb-2">
                    <code className="font-mono text-[14px] text-[var(--fg-strong)] font-semibold tracking-tight">{b.name}</code>
                    <span className="font-mono text-[9.5px] uppercase tracking-[0.16em] text-[var(--fg-dim)] font-semibold">{b.tag}</span>
                  </div>
                  <p className="text-[12px] text-[var(--fg-muted)] leading-[1.6]">{b.body}</p>
                </div>
              );
            })}
          </div>

          <div className="text-[10.5px] uppercase tracking-[0.18em] text-[var(--fg-dim)] font-mono font-semibold mt-10 mb-3">Why each technology</div>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-3 mb-8">
            {[
              { name: "Go",                tag: "language",  body: "Goroutines are perfect for per-chain concurrency — each chain gets its own listener without OS-thread overhead. go-ethereum is the production-tested ABI/RPC library (Geth's own code). Single static binary, no runtime, small Docker images, fast cold start." },
              { name: "Kafka (KRaft mode)", tag: "bus",      body: "Backpressure, replay, fan-out — see above. KRaft means Kafka runs its own consensus instead of needing a separate ZooKeeper cluster, so the compose file is one broker container, not two. 7-day retention on both topics. chain_id partition key preserves per-chain ordering." },
              { name: "ClickHouse 24.3",    tag: "storage",  body: "Append-heavy + analytical is exactly its sweet spot. Columnar compression 10–20x, scans 1B+ rows/sec on commodity hardware. ReplacingMergeTree dedupes on (chain_id, block_number, log_index) on background merges, so at-least-once delivery from Kafka is safe. Two tables only." },
              { name: "Redis 7",            tag: "cache",    body: "Sub-millisecond point lookups for the queries an MCP agent fires every turn — current balances, latest positions. HINCRBY is atomic so multiple processor replicas race-safe. MCP tool results cache under mcp:<tool>:sha256(args)[:8] with 60s TTL." },
            ].map((t) => (
              <div key={t.name} className="rounded-xl border border-[var(--border)] bg-[var(--bg-card)] p-5">
                <div className="flex items-center justify-between gap-2 mb-2">
                  <div className="text-[14px] font-semibold text-[var(--fg-strong)] tracking-tight">{t.name}</div>
                  <span className="font-mono text-[9.5px] uppercase tracking-[0.16em] text-[var(--fg-dim)] font-semibold">{t.tag}</span>
                </div>
                <p className="text-[12px] text-[var(--fg-muted)] leading-[1.6]">{t.body}</p>
              </div>
            ))}
          </div>

          <div className="text-[10.5px] uppercase tracking-[0.18em] text-[var(--fg-dim)] font-mono font-semibold mt-10 mb-3">Resilience</div>
          <P>
            Three independent mechanisms keep the indexer alive across transient failures, instead of one big retry loop.
          </P>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-3 mb-8">
            {[
              { icon: Eye,    name: "Readiness probe",     body: "/ready on :9180 returns 503 if any chain head is older than readiness_head_timeout. Docker flips the container unhealthy and restarts it — stuck chain self-heals." },
              { icon: Wifi,   name: "WSS head watchdog",   body: "Even if the WS subscription has not surfaced an error, no headers within head_timeout (90s default) forces a reconnect. Catches the silent-stall failure mode." },
              { icon: Zap,    name: "Publish fail-fast",   body: "If Kafka publishes fail for kafka_publish_fail_threshold consecutive blocks, the indexer exits non-zero. Docker restarts instead of silently dropping events." },
            ].map((r) => {
              const Icon = r.icon;
              return (
                <div key={r.name} className="rounded-xl border border-[var(--border)] bg-[var(--bg-card)] p-5">
                  <div className="size-9 rounded-lg bg-[var(--bg-elevated)] border border-[var(--border)] flex items-center justify-center text-[var(--fg-muted)] mb-3">
                    <Icon className="size-4" />
                  </div>
                  <div className="text-[13.5px] font-semibold text-[var(--fg-strong)] tracking-tight">{r.name}</div>
                  <p className="text-[12px] text-[var(--fg-muted)] leading-[1.6] mt-2">{r.body}</p>
                </div>
              );
            })}
          </div>

          <div className="text-[10.5px] uppercase tracking-[0.18em] text-[var(--fg-dim)] font-mono font-semibold mt-10 mb-3">What flows on which topic</div>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-3 mb-2">
            {[
              { name: "raw_events",          dir: "indexer → processor",          body: "One record per recognised log. Partition key chain_id. 7-day retention. Replayable from any offset." },
              { name: "decoded_events",      dir: "processor → WebSocket",        body: "Typed events published after the storage write succeeds. Per-socket ephemeral group chainpulse-ws-<uuid> at LastOffset — fresh clients see live traffic only." },
              { name: "Cache-aside (Aside[T])", dir: "REST + gRPC + MCP",         body: "Shared read path imported by every surface. Single source of truth for query logic — fix a SQL bug once, every surface fixed." },
            ].map((t) => (
              <div key={t.name} className="rounded-xl border border-[var(--border)] bg-[var(--bg-card)] p-5">
                <code className="font-mono text-[13px] text-[var(--fg-strong)] font-semibold tracking-tight block">{t.name}</code>
                <div className="font-mono text-[10.5px] uppercase tracking-[0.14em] text-[var(--fg-dim)] mt-1">{t.dir}</div>
                <p className="text-[12px] text-[var(--fg-muted)] leading-[1.6] mt-3">{t.body}</p>
              </div>
            ))}
          </div>
        </Section>

        <Section id="chains" n="04" title="Supported chains">
          <Lead>Nine EVM networks are wired up in <Code>config/config.example.toml</Code>. Three run by default in the demo stack; the other six are commented out behind their own RPC keys. Comment lines back in to enable them.</Lead>

          <div className="text-[10.5px] uppercase tracking-[0.18em] text-[var(--fg-dim)] font-mono font-semibold mt-8 mb-3">Default chains · indexing now</div>
          <div className="grid grid-cols-1 sm:grid-cols-3 gap-3 mb-8">
            {[
              { name: "Ethereum", id: "1",     bt: "12s",   slug: "ethereum" },
              { name: "Polygon",  id: "137",   bt: "2s",    slug: "polygon" },
              { name: "Arbitrum", id: "42161", bt: "0.25s", slug: "arbitrum" },
            ].map((c) => (
              <div key={c.name} className="rounded-xl border border-[var(--accent-border)] bg-[var(--accent-soft)] p-5 flex items-center gap-4">
                <ChainLogo slug={c.slug} className="size-12 shrink-0 rounded-full" />
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2">
                    <div className="text-[14.5px] font-semibold text-[var(--fg-strong)] tracking-tight">{c.name}</div>
                    <span className="size-1.5 rounded-full bg-[var(--accent)] pulse-dot" />
                  </div>
                  <div className="flex items-center gap-3 mt-1 text-[11px] font-mono text-[var(--fg-muted)]">
                    <span>id {c.id}</span>
                    <span>·</span>
                    <span>{c.bt} block</span>
                  </div>
                </div>
              </div>
            ))}
          </div>

          <div className="text-[10.5px] uppercase tracking-[0.18em] text-[var(--fg-dim)] font-mono font-semibold mb-3">Optional chains · uncomment + add RPC keys</div>
          <div className="grid grid-cols-2 sm:grid-cols-3 gap-3 mb-8">
            {[
              { name: "Base",       id: "8453",   bt: "2s",  slug: "base" },
              { name: "Optimism",   id: "10",     bt: "2s",  slug: "optimism" },
              { name: "BSC",        id: "56",     bt: "3s",  slug: "bsc" },
              { name: "Avalanche",  id: "43114",  bt: "2s",  slug: "avalanche" },
              { name: "Scroll",     id: "534352", bt: "3s",  slug: "scroll" },
              { name: "zkSync Era", id: "324",    bt: "1s",  slug: "zksync-era" },
            ].map((c) => (
              <div key={c.name} className="rounded-xl border border-[var(--border)] bg-[var(--bg-card)] p-4 flex items-center gap-3 hover:border-[var(--border-strong)] transition-colors">
                <ChainLogo slug={c.slug} className="size-10 shrink-0 rounded-full" />
                <div className="flex-1 min-w-0">
                  <div className="text-[13px] font-semibold text-[var(--fg-strong)] tracking-tight">{c.name}</div>
                  <div className="text-[10.5px] font-mono text-[var(--fg-muted)] mt-0.5">id {c.id} · {c.bt}</div>
                </div>
              </div>
            ))}
          </div>

          <SubH>Subscribe modes</SubH>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-3 mt-4">
            <div className="rounded-xl border border-[var(--accent-border)] bg-[var(--bg-card)] p-5">
              <div className="flex items-center gap-2 mb-2">
                <span className="inline-flex items-center px-2 py-0.5 rounded border border-[var(--accent-border)] bg-[var(--accent-soft)] text-[10px] font-mono uppercase tracking-[0.14em] text-[var(--accent-fg)] font-semibold">default</span>
                <code className="font-mono text-[12.5px] text-[var(--fg-strong)] font-semibold">subscribe_mode = &quot;logs&quot;</code>
              </div>
              <p className="text-[12.5px] text-[var(--fg-muted)] leading-[1.6]">
                Topic-filtered <Code>eth_subscribe(&apos;logs&apos;, ...)</Code> so the provider only streams events the registered decoders can handle. Free-tier WSS friendly. Ignores <Code>confirmations</Code>.
              </p>
            </div>
            <div className="rounded-xl border border-[var(--border)] bg-[var(--bg-card)] p-5">
              <div className="flex items-center gap-2 mb-2">
                <span className="inline-flex items-center px-2 py-0.5 rounded border border-[var(--border)] bg-[var(--bg-elevated)] text-[10px] font-mono uppercase tracking-[0.14em] text-[var(--fg-muted)] font-semibold">legacy</span>
                <code className="font-mono text-[12.5px] text-[var(--fg-strong)] font-semibold">subscribe_mode = &quot;blocks&quot;</code>
              </div>
              <p className="text-[12.5px] text-[var(--fg-muted)] leading-[1.6]">
                <Code>newHeads</Code> + per-block <Code>eth_getLogs</Code>. Honours <Code>confirmations</Code> for reorg safety. Heavy on RPC — needs a paid tier on busy chains.
              </p>
            </div>
          </div>
        </Section>

        <Section id="protocols" n="05" title="Decoded protocols">
          <Lead>Six decoders ship today. Each lives in <Code>internal/processor/protocols/</Code> behind a single <Code>Register()</Code> in <Code>init()</Code> — adding a new one is one file, no central switch, no pipeline change.</Lead>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-3 my-8">
            {[
              {
                name: "ERC-20",
                slug: "erc20",
                color: "#22c55e",
                events: ["Transfer", "Approval"],
                note: "Baseline for every token across chains. Powers get_token_transfers and get_whale_activity.",
              },
              {
                name: "Uniswap V3",
                slug: "uniswap_v3",
                color: "#ff007a",
                events: ["Swap", "Mint", "Burn", "Collect"],
                note: "Pool factory pattern. Decoder routes by topic hash.",
              },
              {
                name: "Aave V3",
                slug: "aave_v3",
                color: "#b6509e",
                events: ["Supply", "Withdraw", "Borrow", "Repay", "LiquidationCall"],
                note: "Drives DeFi positions for the wallet endpoints.",
              },
              {
                name: "Compound V3",
                slug: "compound_v3",
                color: "#00d395",
                events: ["Supply", "Withdraw", "AbsorbCollateral"],
                note: "Comet markets. One row per protocol action.",
              },
              {
                name: "Curve",
                slug: "curve",
                color: "#fbbf24",
                events: ["TokenExchange", "AddLiquidity", "RemoveLiquidity"],
                note: "Indexes pools whose ABIs match the registered set.",
              },
              {
                name: "Lido",
                slug: "lido",
                color: "#00a3ff",
                events: ["Submitted", "Transfer (stETH)", "TokensRebased"],
                note: "Captures staking inflows + rebases on Ethereum mainnet.",
              },
            ].map((p) => (
              <div key={p.name} className="rounded-xl border border-[var(--border)] bg-[var(--bg-card)] p-5 hover:border-[var(--border-strong)] transition-colors">
                <div className="flex items-center gap-3 mb-3">
                  <ProtocolLogo slug={p.slug} className="size-10 shrink-0 rounded-full" />
                  <div className="flex-1 min-w-0">
                    <div className="text-[14.5px] font-semibold text-[var(--fg-strong)] tracking-tight leading-tight">{p.name}</div>
                    <code className="font-mono text-[10.5px] text-[var(--fg-dim)]">{p.slug}</code>
                  </div>
                </div>
                <div className="flex flex-wrap gap-1.5 mb-3">
                  {p.events.map((e) => (
                    <span key={e} className="inline-flex items-center px-2 py-0.5 rounded border border-[var(--border)] bg-[var(--bg-elevated)] text-[10.5px] font-mono text-[var(--fg-muted)]">{e}</span>
                  ))}
                </div>
                <p className="text-[12.5px] text-[var(--fg-muted)] leading-[1.6]">{p.note}</p>
              </div>
            ))}
          </div>

          <Callout>
            <strong>Unknown events drop here.</strong> ~80–95% of logs on a busy chain are stuff the indexer does not track (other DEXes, NFT transfers, governance, spam). Unrecognised events are committed back to Kafka and discarded — replay catches them for free if a decoder ships later.
          </Callout>
        </Section>

        <Section id="schema" n="06" title="Data model">
          <Lead>Two ClickHouse tables hold all indexed data. Both use <Code>ReplacingMergeTree</Code> on the natural primary key, so re-inserts during replay are idempotent.</Lead>

          <div className="grid grid-cols-2 sm:grid-cols-4 gap-3 my-6">
            {[
              { v: "2",          l: "Tables" },
              { v: "ReplacingMergeTree", l: "Engine" },
              { v: "90d",        l: "TTL" },
              { v: "Monthly",    l: "Partition" },
            ].map((m) => (
              <div key={m.l} className="rounded-lg border border-[var(--border)] bg-[var(--bg-card)] px-4 py-3">
                <div className="font-mono text-[15px] font-bold text-[var(--fg-strong)] tracking-tight leading-tight truncate">{m.v}</div>
                <div className="text-[10.5px] uppercase tracking-[0.14em] text-[var(--fg-muted)] font-semibold mt-1">{m.l}</div>
              </div>
            ))}
          </div>

          <SchemaCard
            name="defi_events"
            tagline="Protocol-decoded events. One row per matched log."
            tone="amber"
            engine="ReplacingMergeTree"
            partition="toYYYYMM(timestamp)"
            orderBy="(chain_id, block_number, log_index)"
            rows={[
              ["chain_id",     "UInt64",                  "EVM chain ID"],
              ["block_number", "UInt64",                  "Block this event was emitted in"],
              ["tx_hash",      "String",                  "Full transaction hash, 0x-prefixed"],
              ["log_index",    "UInt32",                  "Log position within the tx"],
              ["protocol",     "LowCardinality(String)",  "uniswap_v3 / aave_v3 / compound_v3 / lido / curve"],
              ["event_type",   "LowCardinality(String)",  "swap / supply / borrow / liquidation"],
              ["user_addr",    "String",                  "Wallet address (or pool, depending on protocol)"],
              ["token_a",      "String",                  "Primary token contract"],
              ["token_b",      "Nullable(String)",        "Secondary token (swaps, LPs)"],
              ["amount_a",     "String",                  "uint256 stringified"],
              ["amount_b",     "Nullable(String)",        "Secondary amount"],
              ["params",       "String",                  "Protocol-specific JSON blob"],
              ["timestamp",    "DateTime",                "UTC, from block timestamp"],
            ]}
          />

          <SchemaCard
            name="token_transfers"
            tagline="Append-only stream of every ERC-20 Transfer the indexer recognises."
            tone="green"
            engine="ReplacingMergeTree"
            partition="toYYYYMM(timestamp)"
            orderBy="(chain_id, block_number, log_index)"
            rows={[
              ["chain_id",     "UInt64",   "EVM chain ID"],
              ["block_number", "UInt64",   "Block number"],
              ["tx_hash",      "String",   "Transaction hash"],
              ["log_index",    "UInt32",   "Log index"],
              ["token",        "String",   "Token contract"],
              ["from_addr",    "String",   "Sender"],
              ["to_addr",      "String",   "Receiver"],
              ["amount",       "String",   "uint256 stringified"],
              ["timestamp",    "DateTime", "UTC"],
            ]}
          />

          <Callout>
            Both tables retain <strong>90 days</strong>, partitioned monthly by <Code>toYYYYMM(timestamp)</Code>. Adjust TTL in <Code>schema/*.sql</Code> for longer history. Dedup key <Code>(chain_id, block_number, log_index)</Code> guarantees idempotent replay on any background merge.
          </Callout>
        </Section>

        <Section id="rest" n="07" title="REST + gRPC">
          <Lead>HTTP endpoints under /v1 on port 8080. gRPC mirror on :8081 (reflection enabled). Cache-aside (Redis → ClickHouse). 100 requests/min per IP.</Lead>

          <div className="grid grid-cols-2 sm:grid-cols-4 gap-3 my-6">
            {[
              { v: ":8080",   l: "REST" },
              { v: ":8081",   l: "gRPC" },
              { v: "100",     l: "req / min / IP" },
              { v: "Cache-aside", l: "Redis → CH" },
            ].map((m) => (
              <div key={m.l} className="rounded-lg border border-[var(--border)] bg-[var(--bg-card)] px-4 py-3">
                <div className="font-mono text-[16px] font-bold text-[var(--fg-strong)] tabular-nums tracking-tight leading-tight">{m.v}</div>
                <div className="text-[10.5px] uppercase tracking-[0.14em] text-[var(--fg-muted)] font-semibold mt-1">{m.l}</div>
              </div>
            ))}
          </div>

          <div className="text-[10.5px] uppercase tracking-[0.18em] text-[var(--fg-dim)] font-mono font-semibold mt-8 mb-3">Endpoints</div>
          <div className="rounded-xl border border-[var(--border)] bg-[var(--bg-card)] overflow-hidden mb-6">
            {[
              { p: "/v1/wallet/{addr}/positions",   d: "All DeFi positions across chains" },
              { p: "/v1/wallet/{addr}/balances",    d: "Token balances per chain" },
              { p: "/v1/wallet/{addr}/history",     d: "Decoded transaction history" },
              { p: "/v1/token/{addr}/transfers",    d: "Recent transfers for a token" },
              { p: "/v1/protocol/{name}/stats",     d: "TVL, volume, unique users" },
              { p: "/v1/chain/{id}/blocks",         d: "Recent indexed blocks" },
              { p: "/health",                       d: "Health probe", muted: true },
              { p: "/metrics",                      d: "Prometheus scrape", muted: true },
            ].map((e) => (
              <div key={e.p} className="flex items-center gap-3 px-4 py-3 border-t border-[var(--border)] first:border-t-0 hover:bg-[var(--bg-hover)] transition-colors">
                <span className="inline-flex items-center justify-center px-2 py-0.5 rounded font-mono text-[10px] uppercase tracking-[0.14em] font-bold bg-[#22c55e1f] text-[#15803d] dark:text-[#86efac] border border-[#22c55e55] w-12">GET</span>
                <code className={`font-mono text-[12.5px] flex-1 min-w-0 truncate ${e.muted ? "text-[var(--fg-muted)]" : "text-[var(--fg-strong)] font-medium"}`}>{e.p}</code>
                <span className="text-[12px] text-[var(--fg-muted)] hidden sm:inline">{e.d}</span>
              </div>
            ))}
          </div>

          <div className="text-[10.5px] uppercase tracking-[0.18em] text-[var(--fg-dim)] font-mono font-semibold mt-10 mb-3">Example · request → response</div>
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-4 mb-8">
            <div className="rounded-xl border border-[var(--border)] bg-[var(--bg-card)] p-5">
              <div className="flex items-center gap-2 mb-3">
                <span className="inline-flex items-center justify-center px-2 py-0.5 rounded font-mono text-[10px] uppercase tracking-[0.14em] font-bold bg-[#22c55e1f] text-[#15803d] dark:text-[#86efac] border border-[#22c55e55]">GET</span>
                <span className="text-[12px] font-semibold text-[var(--fg-strong)]">Request</span>
              </div>
              <CodeBlock lang="bash">{`curl http://localhost:8080/v1/wallet/0x2faf487a4414fe77e2327f0bf4ae2a264a776ad2/positions`}</CodeBlock>
            </div>
            <div className="rounded-xl border border-[var(--border)] bg-[var(--bg-card)] p-5">
              <div className="flex items-center gap-2 mb-3">
                <span className="inline-flex items-center justify-center px-2 py-0.5 rounded font-mono text-[10px] uppercase tracking-[0.14em] font-bold bg-[#3b82f61a] text-[#1d4ed8] dark:text-[#93c5fd] border border-[#3b82f655]">200</span>
                <span className="text-[12px] font-semibold text-[var(--fg-strong)]">Response</span>
              </div>
              <CodeBlock lang="json">{`[
  {
    "chain_id": 42161,
    "protocol": "aave_v3",
    "type": "supply",
    "token": "0xaf88d...8e5831",
    "amount": "1500000000",
    "updated": "2026-05-06T09:54:00Z"
  }
]`}</CodeBlock>
            </div>
          </div>

          <div className="text-[10.5px] uppercase tracking-[0.18em] text-[var(--fg-dim)] font-mono font-semibold mt-10 mb-3">gRPC mirror · :8081</div>
          <div className="rounded-xl border border-[var(--border)] bg-[var(--bg-card)] p-5">
            <div className="flex items-center gap-3 mb-3">
              <div className="size-9 rounded-lg bg-[#3b82f61a] border border-[#3b82f655] flex items-center justify-center text-[#1d4ed8] dark:text-[#93c5fd]">
                <Code2 className="size-4" />
              </div>
              <div className="flex-1 min-w-0">
                <div className="text-[13.5px] font-semibold text-[var(--fg-strong)] tracking-tight">chainpulse.v1</div>
                <div className="text-[11px] text-[var(--fg-muted)] mt-0.5">Typed protobuf, HTTP/2, reflection enabled · same data, same handlers, same cache as REST.</div>
              </div>
            </div>
            <CodeBlock lang="bash">{`grpcurl -plaintext localhost:8081 list
grpcurl -plaintext -d '{"wallet":"0x...","chain_id":1}' \\
        localhost:8081 chainpulse.v1.Wallet/GetWalletPositions`}</CodeBlock>
            <div className="flex flex-wrap gap-1.5 mt-3">
              {["Wallet", "Token", "Protocol", "Chain"].map((svc) => (
                <span key={svc} className="inline-flex items-center px-2 py-0.5 rounded border border-[var(--border)] bg-[var(--bg-elevated)] text-[10.5px] font-mono text-[var(--fg-muted)]">{svc}</span>
              ))}
            </div>
          </div>
        </Section>

        <Section id="ws" n="08" title="WebSocket">
          <Lead>Live stream of every decoded event. Kafka consumer fans out to all connected clients. Each socket gets its own ephemeral consumer group at LastOffset — fresh clients see live traffic only, never the 7-day backlog.</Lead>

          {/* Live endpoint card */}
          <div className="relative overflow-hidden rounded-xl border border-[var(--border)] bg-gradient-to-br from-[#a78bfa1a] via-[var(--bg-card)] to-[var(--bg-card)] p-6 my-6">
            <div className="absolute -right-8 -top-8 size-40 rounded-full bg-[#a78bfa1a] blur-3xl pointer-events-none" />
            <div className="relative z-10 flex items-center gap-4 flex-wrap">
              <div className="size-12 rounded-full bg-[#a78bfa33] border border-[#a78bfa55] flex items-center justify-center text-[#7c3aed] dark:text-[#c4b5fd]">
                <Wifi className="size-5" />
              </div>
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2">
                  <span className="size-2 rounded-full bg-[var(--accent)] pulse-dot" />
                  <span className="font-mono text-[10.5px] uppercase tracking-[0.16em] text-[var(--accent-fg)] font-semibold">Live</span>
                </div>
                <code className="font-mono text-[15px] text-[var(--fg-strong)] font-semibold tracking-tight block mt-1 break-all">ws://localhost:8080/v1/events/stream</code>
              </div>
              <div className="flex gap-4 text-center font-mono">
                <div>
                  <div className="text-[18px] font-bold text-[var(--fg-strong)] tabular-nums">30s</div>
                  <div className="text-[9.5px] uppercase tracking-[0.14em] text-[var(--fg-dim)]">Ping</div>
                </div>
                <div>
                  <div className="text-[18px] font-bold text-[var(--fg-strong)] tabular-nums">60s</div>
                  <div className="text-[9.5px] uppercase tracking-[0.14em] text-[var(--fg-dim)]">Pong</div>
                </div>
                <div>
                  <div className="text-[18px] font-bold text-[var(--fg-strong)] tabular-nums">5s</div>
                  <div className="text-[9.5px] uppercase tracking-[0.14em] text-[var(--fg-dim)]">Write TO</div>
                </div>
              </div>
            </div>
          </div>

          <CodeBlock lang="javascript">{`const ws = new WebSocket("ws://localhost:8080/v1/events/stream");

ws.onmessage = (msg) => {
  const event = JSON.parse(msg.data);
  console.log(event.protocol, event.event_type, event.amount_a);
};`}</CodeBlock>

          <P>
            Each frame is a single decoded event with the same shape as <Code>defi_events</Code> rows. Filter client-side by chain, protocol, or event type.
          </P>

          <Callout>
            For replay or historical queries, use the REST API. The WebSocket only delivers events as they happen.
          </Callout>

          <div className="text-[10.5px] uppercase tracking-[0.18em] text-[var(--fg-dim)] font-mono font-semibold mt-10 mb-3">Who consumes the stream</div>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3 mb-2">
            {[
              { icon: Activity,  name: "Live whale feed",     body: "Dashboards prepend a row every time a >$1M transfer lands. No polling, no refresh churn." },
              { icon: Zap,       name: "Trading bots",        body: "Filter for protocol=aave_v3 event=LiquidationCall and react before the next block." },
              { icon: Bot,       name: "Telegram / Discord",  body: "Pipe the stream into a channel: 'Whale moved 5M USDC on Polygon.' Pure consumer, no DB." },
              { icon: Sparkles,  name: "MEV / liquidation",   body: "Watch Borrow / Repay live, compute health factors, fire a liquidation tx before block close." },
              { icon: Eye,       name: "DEX activity ticker", body: "Rolling 'X just swapped 2 ETH → 6,300 USDC on Uniswap V3' on a page or office TV." },
              { icon: Code2,     name: "Custom backends",     body: "Pipe decoded_events into your own service — alerting, analytics, replication, anything." },
            ].map((u) => {
              const Icon = u.icon;
              return (
                <div key={u.name} className="rounded-xl border border-[var(--border)] bg-[var(--bg-card)] p-5 hover:border-[var(--border-strong)] transition-colors">
                  <div className="size-9 rounded-lg bg-[var(--bg-elevated)] border border-[var(--border)] flex items-center justify-center text-[var(--fg-muted)] mb-3">
                    <Icon className="size-4" />
                  </div>
                  <div className="text-[13.5px] font-semibold text-[var(--fg-strong)] tracking-tight">{u.name}</div>
                  <p className="text-[12px] text-[var(--fg-muted)] leading-[1.6] mt-2">{u.body}</p>
                </div>
              );
            })}
          </div>
        </Section>

        <Section id="mcp" n="09" title="MCP tools">
          <Lead>
            Eight tools the MCP server exposes to LLMs. Schemas are JSON Schema-validated; address parameters require a 0x-prefixed 40-hex pattern. The server runs on port 3001.
          </Lead>

          {/* MCP hero */}
          <div className="relative overflow-hidden rounded-2xl border border-[var(--border)] bg-gradient-to-br from-[var(--accent-soft)] via-[var(--bg-card)] to-[var(--bg-card)] p-7 my-8">
            <div className="absolute -right-12 -top-12 size-56 rounded-full bg-[var(--accent-soft)] blur-3xl opacity-50 pointer-events-none" />
            <div className="relative z-10 flex items-start gap-5 flex-wrap">
              <div className="size-14 rounded-xl bg-[var(--accent-soft)] border border-[var(--accent-border)] flex items-center justify-center text-[var(--accent-fg)]">
                <Sparkles className="size-6" />
              </div>
              <div className="flex-1 min-w-[260px]">
                <div className="text-[10.5px] uppercase tracking-[0.18em] font-mono text-[var(--accent-fg)] font-semibold">Model Context Protocol</div>
                <div className="text-[20px] font-semibold text-[var(--fg-strong)] tracking-tight leading-tight mt-1">USB for LLMs · 8 tools, two transports.</div>
                <p className="text-[13px] text-[var(--fg-muted)] mt-2 leading-[1.65]">
                  Claude, Cursor, or any MCP-aware agent calls <Code>get_wallet_positions(&quot;0x...&quot;)</Code> like a regular function and gets clean JSON back — no ABIs, no contract addresses to memorise.
                </p>
              </div>
              <div className="grid grid-cols-3 gap-4 text-center font-mono">
                <div>
                  <div className="text-[20px] font-bold text-[var(--fg-strong)] tabular-nums">8</div>
                  <div className="text-[9.5px] uppercase tracking-[0.14em] text-[var(--fg-dim)]">Tools</div>
                </div>
                <div>
                  <div className="text-[20px] font-bold text-[var(--fg-strong)] tabular-nums">2</div>
                  <div className="text-[9.5px] uppercase tracking-[0.14em] text-[var(--fg-dim)]">Transports</div>
                </div>
                <div>
                  <div className="text-[20px] font-bold text-[var(--fg-strong)] tabular-nums">60s</div>
                  <div className="text-[9.5px] uppercase tracking-[0.14em] text-[var(--fg-dim)]">Cache TTL</div>
                </div>
              </div>
            </div>
          </div>

          <div className="text-[10.5px] uppercase tracking-[0.18em] text-[var(--fg-dim)] font-mono font-semibold mt-2 mb-3">All eight tools</div>
          <div className="space-y-3 mb-2">
            <McpTool
              name="get_wallet_positions"
              desc="Active DeFi positions (supply, borrow, withdraw, repay) for a wallet across indexed chains."
              params={[
                { name: "wallet", type: "string", required: true, hint: "0x-prefixed 40-hex" },
                { name: "chain_id", type: "integer", required: false, hint: "Optional chain filter" },
                { name: "protocol", type: "enum", required: false, hint: "erc20 / uniswap_v3 / aave_v3 / compound_v3 / lido / curve" },
              ]}
            />
            <McpTool
              name="get_wallet_balances"
              desc="Per-token balances for a wallet. Scopes to one chain with chain_id, otherwise returns balances across every cached chain."
              params={[
                { name: "wallet", type: "string", required: true, hint: "0x-prefixed 40-hex" },
                { name: "chain_id", type: "integer", required: false, hint: "Optional chain filter" },
              ]}
            />
            <McpTool
              name="get_wallet_history"
              desc="Most recent decoded events involving a wallet across all indexed chains."
              params={[
                { name: "wallet", type: "string", required: true, hint: "0x-prefixed 40-hex" },
                { name: "limit", type: "integer", required: false, hint: "1-500, default 100" },
              ]}
            />
            <McpTool
              name="get_token_transfers"
              desc="Recent ERC-20 transfers where the address is the token contract or one of the parties (from / to)."
              params={[
                { name: "address", type: "string", required: true, hint: "Token or wallet, 0x-prefixed" },
                { name: "limit", type: "integer", required: false, hint: "1-500" },
                { name: "chain_id", type: "integer", required: false, hint: "Optional chain filter" },
              ]}
            />
            <McpTool
              name="get_whale_activity"
              desc="Largest token transfers in the last N hours where amount >= min_amount (decimal-string uint256). USD pricing not in scope."
              params={[
                { name: "hours", type: "integer", required: true, hint: "1-168" },
                { name: "min_amount", type: "string", required: false, hint: "uint256 decimal" },
                { name: "chain_id", type: "integer", required: false, hint: "Optional chain filter" },
                { name: "limit", type: "integer", required: false, hint: "1-500" },
              ]}
            />
            <McpTool
              name="get_defi_positions"
              desc="Positions for a wallet inside a single protocol. Optional chain_id scope."
              params={[
                { name: "wallet", type: "string", required: true, hint: "0x-prefixed 40-hex" },
                { name: "protocol", type: "enum", required: true, hint: "erc20 / uniswap_v3 / aave_v3 / compound_v3 / lido / curve" },
                { name: "chain_id", type: "integer", required: false, hint: "Optional chain filter" },
              ]}
            />
            <McpTool
              name="get_protocol_stats"
              desc="24h volume and unique-user counts per chain for a protocol."
              params={[
                { name: "protocol", type: "enum", required: true, hint: "erc20 / uniswap_v3 / aave_v3 / compound_v3 / lido / curve" },
                { name: "chain_id", type: "integer", required: false, hint: "Optional chain filter" },
              ]}
            />
            <McpTool
              name="get_latest_block"
              desc="Most recently indexed block per chain. Without chain_id, returns every chain in storage. Use it to verify indexer freshness."
              params={[
                { name: "chain_id", type: "integer", required: false, hint: "Optional chain filter" },
              ]}
            />
          </div>

          <div className="text-[10.5px] uppercase tracking-[0.18em] text-[var(--fg-dim)] font-mono font-semibold mt-10 mb-3">Wire it up · two transports</div>
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-4 mb-8">
            <div className="rounded-xl border border-[var(--accent-border)] bg-[var(--bg-card)] p-5 flex flex-col">
              <div className="flex items-center gap-2 mb-3">
                <span className="inline-flex items-center px-2 py-0.5 rounded border border-[var(--accent-border)] bg-[var(--accent-soft)] text-[10px] font-mono uppercase tracking-[0.14em] text-[var(--accent-fg)] font-semibold">recommended</span>
                <div className="text-[14px] font-semibold text-[var(--fg-strong)] tracking-tight">Option A — stdio</div>
              </div>
              <p className="text-[12.5px] text-[var(--fg-muted)] leading-[1.6] mb-3">
                Claude Desktop launches the MCP binary as a child process and pipes JSON-RPC over stdin/stdout. No network, no auth — trust comes from the parent process. Run <Code>make build</Code> once, install, then drop this in your config.
              </p>
              <CodeBlock lang="json">{`{
  "mcpServers": {
    "chainpulse": {
      "command": "/usr/local/bin/chainpulse-mcp",
      "args": ["--config", "/etc/chainpulse/config.toml"],
      "env": {
        "CLICKHOUSE_DSN": "clickhouse://default:@localhost:9000/chainpulse",
        "REDIS_ADDR":     "localhost:6379"
      }
    }
  }
}`}</CodeBlock>
              <p className="text-[11.5px] text-[var(--fg-muted)] leading-[1.6] mt-3">
                macOS path: <Code>~/Library/Application Support/Claude/claude_desktop_config.json</Code>. Quit Claude (Cmd+Q), reopen — 8 tools appear in the picker.
              </p>
            </div>

            <div className="rounded-xl border border-[var(--border)] bg-[var(--bg-card)] p-5 flex flex-col">
              <div className="flex items-center gap-2 mb-3">
                <span className="inline-flex items-center px-2 py-0.5 rounded border border-[#a78bfa55] bg-[#a78bfa1a] text-[10px] font-mono uppercase tracking-[0.14em] text-[#6d28d9] dark:text-[#c4b5fd] font-semibold">remote</span>
                <div className="text-[14px] font-semibold text-[var(--fg-strong)] tracking-tight">Option B — HTTP / SSE</div>
              </div>
              <p className="text-[12.5px] text-[var(--fg-muted)] leading-[1.6] mb-3">
                SSE on :3001 inside the Docker stack. Any MCP-aware client points at it. Set <Code>bearer_token</Code> in <Code>[mcp]</Code> for auth — comparisons use <Code>crypto/subtle.ConstantTimeCompare</Code> against timing attacks.
              </p>
              <CodeBlock lang="bash">{`# open SSE, capture session id from first event
curl -N http://localhost:3001/sse \\
  -H "Authorization: Bearer $MCP_BEARER_TOKEN"

# POST a JSON-RPC tools/call to that session
curl -X POST "http://localhost:3001/messages?session=$SID" \\
  -H "Authorization: Bearer $MCP_BEARER_TOKEN" \\
  -H "Content-Type: application/json" \\
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/call",
       "params":{"name":"get_wallet_balances",
                 "arguments":{"wallet":"0xd8da...96045"}}}'`}</CodeBlock>
            </div>
          </div>

          <div className="text-[10.5px] uppercase tracking-[0.18em] text-[var(--fg-dim)] font-mono font-semibold mt-8 mb-3">Talk to it in English</div>
          <p className="text-[13px] text-[var(--fg-muted)] mb-4 leading-[1.65] max-w-[700px]">
            Each tool ships a description + JSON Schema. Claude reads both at session start and picks the right tool with no system prompt — you stop talking in JSON-RPC and start talking in English.
          </p>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-3 mb-2">
            {[
              {
                icon: Users,
                label: "Wallet",
                prompts: [
                  "What does vitalik.eth hold across all chains right now?",
                  "Show me the last 50 events for 0xd8da…96045.",
                  "Does 0xABC have any open Aave positions?",
                ],
              },
              {
                icon: Layers,
                label: "Token",
                prompts: [
                  "List the most recent USDC transfers above $10k.",
                  "Show all transfers of token 0x… from yesterday.",
                ],
              },
              {
                icon: Activity,
                label: "Protocol",
                prompts: [
                  "What is the 24h borrow volume on Aave V3 across chains?",
                  "How many Uniswap V3 swaps happened on Arbitrum in the last day?",
                ],
              },
              {
                icon: Zap,
                label: "Whale",
                prompts: [
                  "Show me the 20 largest transfers in the last 6 hours.",
                  "Any whale wallets just supplied to Aave?",
                ],
              },
            ].map((c) => {
              const Icon = c.icon;
              return (
                <div key={c.label} className="rounded-xl border border-[var(--border)] bg-[var(--bg-card)] p-5">
                  <div className="flex items-center gap-2 mb-3">
                    <div className="size-7 rounded-md bg-[var(--bg-elevated)] border border-[var(--border)] flex items-center justify-center text-[var(--fg-muted)]">
                      <Icon className="size-3.5" />
                    </div>
                    <div className="text-[10.5px] uppercase tracking-[0.16em] text-[var(--fg-strong)] font-mono font-semibold">{c.label}</div>
                  </div>
                  <ul className="space-y-2">
                    {c.prompts.map((p) => (
                      <li key={p} className="text-[12.5px] text-[var(--fg-muted)] leading-[1.55] before:content-['❝'] before:text-[var(--fg-dim)] before:mr-1 italic">{p}</li>
                    ))}
                  </ul>
                </div>
              );
            })}
          </div>

          <Callout>
            <strong>Speed expectations.</strong> Cached calls (same query within 60s) return in &lt;5ms. Cold ClickHouse scans for 24h windows land in &lt;200ms. Whale queries across 4 chains and millions of rows finish under one second.
          </Callout>
        </Section>

        <Section id="ops" n="10" title="Operations">
          <Lead>
            Prometheus scrapes every binary&apos;s <Code>/metrics</Code> on a 15s cadence and stores 15 days of series. Grafana ships pre-provisioned at <Code>http://localhost:3030</Code> (admin / admin) with one dashboard called <strong>ChainPulse Overview</strong>.
          </Lead>

          <div className="grid grid-cols-2 sm:grid-cols-4 gap-3 my-6">
            {[
              { v: ":3030",  l: "Grafana" },
              { v: ":9090",  l: "Prometheus" },
              { v: "15s",    l: "Scrape interval" },
              { v: "6",      l: "Dashboard panels" },
            ].map((m) => (
              <div key={m.l} className="rounded-lg border border-[var(--border)] bg-[var(--bg-card)] px-4 py-3">
                <div className="font-mono text-[16px] font-bold text-[var(--fg-strong)] tabular-nums leading-tight">{m.v}</div>
                <div className="text-[10.5px] uppercase tracking-[0.14em] text-[var(--fg-muted)] font-semibold mt-1">{m.l}</div>
              </div>
            ))}
          </div>

          <div className="text-[10.5px] uppercase tracking-[0.18em] text-[var(--fg-dim)] font-mono font-semibold mt-8 mb-3">ChainPulse Overview · 6 panels</div>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-3 mb-6">
            {[
              { icon: Activity, name: "Events Indexed (24h)",     healthy: "Steady upward stair, activity on every enabled chain", worry: "Flat line on a chain = indexer/processor stopped seeing it. Crash = RPC paused or rate-limited." },
              { icon: Globe2,   name: "API Requests (24h)",        healthy: "Mostly 2xx, tiny 4xx, near-zero 5xx",                  worry: "5xx climbing = ClickHouse/Redis unhappy. 4xx spike = a client misuses an endpoint." },
              { icon: Sparkles, name: "MCP Tool Calls (24h)",      healthy: "Distribution across multiple tools, calls every minute when an agent is active", worry: "One tool dominates with errors = its schema or query path broke. Flat zero = no agent connected." },
              { icon: Zap,      name: "API p99 Latency",           healthy: "<50ms for cached lookups, 100–250ms for analytical scans", worry: "p99 climbs without traffic growing = cache miss storm or ClickHouse degrading." },
              { icon: Network,  name: "Chains Indexed",            healthy: "Every enabled chain green, last-seen within seconds of real chain time", worry: "Chain goes red or drifts >30s = WSS disconnected, RPC paused, or provider throttling." },
              { icon: Database, name: "Kafka Consumer Lag",        healthy: "Near zero, small spikes drain inside a minute",        worry: "Lag growing without recovery = processor cannot keep up. Restart and check logs." },
            ].map((p) => {
              const Icon = p.icon;
              return (
                <div key={p.name} className="rounded-xl border border-[var(--border)] bg-[var(--bg-card)] p-5">
                  <div className="flex items-center gap-3 mb-3">
                    <div className="size-9 rounded-lg bg-[var(--bg-elevated)] border border-[var(--border)] flex items-center justify-center text-[var(--fg-muted)]">
                      <Icon className="size-4" />
                    </div>
                    <div className="text-[13.5px] font-semibold text-[var(--fg-strong)] tracking-tight">{p.name}</div>
                  </div>
                  <div className="space-y-2">
                    <div className="flex items-start gap-2">
                      <span className="inline-flex shrink-0 items-center px-1.5 py-0.5 rounded text-[9.5px] font-mono uppercase tracking-[0.14em] font-semibold bg-[var(--accent-soft)] text-[var(--accent-fg)] border border-[var(--accent-border)] mt-0.5">healthy</span>
                      <span className="text-[12px] text-[var(--fg-muted)] leading-[1.55]">{p.healthy}</span>
                    </div>
                    <div className="flex items-start gap-2">
                      <span className="inline-flex shrink-0 items-center px-1.5 py-0.5 rounded text-[9.5px] font-mono uppercase tracking-[0.14em] font-semibold bg-[#ef44441a] text-[#991b1b] dark:text-[#fca5a5] border border-[#ef444455] mt-0.5">worry</span>
                      <span className="text-[12px] text-[var(--fg-muted)] leading-[1.55]">{p.worry}</span>
                    </div>
                  </div>
                </div>
              );
            })}
          </div>

          <Callout>
            Read the dashboard top-to-bottom: top three panels = is the product doing work; middle = is it doing it fast enough; bottom two = are the moving parts healthy. When triaging, start at the bottom — if the moving parts are red the top goes red within minutes.
          </Callout>

          <div className="text-[10.5px] uppercase tracking-[0.18em] text-[var(--fg-dim)] font-mono font-semibold mt-10 mb-3">Health probes</div>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-3 mb-2">
            {[
              { code: "/health on :9180",        body: "Liveness on every binary. Docker uses this to know whether to restart." },
              { code: "/ready on :9180",         body: "Indexer-only. Returns 503 if any chain head is older than readiness_head_timeout. Stalled chain triggers auto-restart." },
              { code: "WSS head watchdog",        body: "Forces a reconnect if no header arrives within head_timeout (default 90s), even when the WS subscription has not surfaced an error." },
              { code: "Kafka publish fail-fast",  body: "Indexer exits non-zero after kafka_publish_fail_threshold consecutive blocks with publish failures. Docker restarts instead of silent drop." },
            ].map((h) => (
              <div key={h.code} className="rounded-xl border border-[var(--border)] bg-[var(--bg-card)] p-5">
                <code className="font-mono text-[12.5px] text-[var(--fg-strong)] font-semibold tracking-tight">{h.code}</code>
                <p className="text-[12px] text-[var(--fg-muted)] leading-[1.6] mt-2">{h.body}</p>
              </div>
            ))}
          </div>
        </Section>

        <Section id="perf" n="11" title="Performance">
          <Lead>Latency budgets the system is designed to hit, measured under steady-state load with 3 chains and 4 decoders running.</Lead>

          <div className="text-[10.5px] uppercase tracking-[0.18em] text-[var(--fg-dim)] font-mono font-semibold mt-8 mb-3">Targets</div>
          <div className="grid grid-cols-2 sm:grid-cols-3 gap-3 mb-8">
            {[
              { v: "<100ms", l: "Block ingestion",   s: "RPC header to indexer" },
              { v: "<500ms", l: "Event processing",  s: "Decode + storage write" },
              { v: "<5ms",   l: "Cache-hit query",   s: "Redis point lookup" },
              { v: "<50ms",  l: "p99 API latency",   s: "REST / gRPC handler" },
              { v: "<2s",    l: "Block → queryable", s: "End-to-end pipeline" },
              { v: "<512MB", l: "Memory · 4 chains", s: "Go binaries only" },
            ].map((m) => (
              <div key={m.l} className="rounded-xl border border-[var(--border)] bg-[var(--bg-card)] p-5 hover:border-[var(--border-strong)] transition-colors">
                <div className="font-mono text-[24px] font-bold text-[var(--fg-strong)] tabular-nums tracking-[-0.02em] leading-none">{m.v}</div>
                <div className="text-[11px] uppercase tracking-[0.14em] text-[var(--fg-strong)] font-semibold mt-3">{m.l}</div>
                <div className="text-[10.5px] text-[var(--fg-muted)] mt-1 leading-snug">{m.s}</div>
              </div>
            ))}
          </div>

          <P>
            All metrics exported as Prometheus counters and histograms on <Code>/metrics</Code>. Grafana dashboards in <Code>monitoring/dashboards/</Code>.
          </P>

          <div className="text-[10.5px] uppercase tracking-[0.18em] text-[var(--fg-dim)] font-mono font-semibold mt-10 mb-3">PRD targets vs reality</div>
          <div className="space-y-3 mb-6">
            {[
              { tgt: "Block ingestion <100ms",          st: "Mostly met",    tone: "warn",  where: "Paid RPC: yes. Free tier (BlockPI) drops subs and fails.", todo: "Pay for a production RPC plan." },
              { tgt: "Event processing <500ms",         st: "Likely",        tone: "info",  where: "Flush cut to 250ms + Redis pipelining shipped. Not measured.", todo: "Run the stack an hour, read processing time off Grafana." },
              { tgt: "Cache-hit query <5ms",            st: "Met",           tone: "ok",    where: "Redis ~0.5ms + in-process cache on top.", todo: "Show the number on the public dashboard." },
              { tgt: "API latency <50ms",               st: "Partial",       tone: "warn",  where: "Cached lookups: yes. Cold analytical scans: 80–250ms.", todo: "Split the target — fast lookups vs heavier scans — or pre-warm." },
              { tgt: "Block → queryable <2s",           st: "Likely",        tone: "info",  where: "Architecture says 1–1.5s. Instrumented but never measured 24h.", todo: "Let it run a day, grab the Grafana panel screenshot." },
              { tgt: "Total memory <512MB (4 chains)",  st: "Scope unclear", tone: "muted", where: "Go services alone: yes. Full stack: 700MB lite, 1.5GB default.", todo: "Decide what counts and update the PRD." },
            ].map((row) => {
              const toneStyles: Record<string, string> = {
                ok:    "bg-[var(--accent-soft)] text-[var(--accent-fg)] border-[var(--accent-border)]",
                info:  "bg-[#3b82f61a] text-[#1d4ed8] dark:text-[#93c5fd] border-[#3b82f655]",
                warn:  "bg-[#f59e0b1a] text-[#92400e] dark:text-[#fcd34d] border-[#f59e0b55]",
                muted: "bg-[var(--bg-elevated)] text-[var(--fg-muted)] border-[var(--border)]",
              };
              return (
                <div key={row.tgt} className="rounded-xl border border-[var(--border)] bg-[var(--bg-card)] p-5">
                  <div className="flex items-center justify-between gap-3 flex-wrap mb-3">
                    <code className="font-mono text-[13px] text-[var(--fg-strong)] font-semibold tracking-tight">{row.tgt}</code>
                    <span className={`inline-flex items-center px-2 py-0.5 rounded border text-[10px] font-mono uppercase tracking-[0.14em] font-semibold ${toneStyles[row.tone]}`}>{row.st}</span>
                  </div>
                  <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                    <div>
                      <div className="text-[9.5px] uppercase tracking-[0.16em] text-[var(--fg-dim)] font-mono font-semibold mb-1">Where we are</div>
                      <div className="text-[12px] text-[var(--fg-muted)] leading-[1.55]">{row.where}</div>
                    </div>
                    <div>
                      <div className="text-[9.5px] uppercase tracking-[0.16em] text-[var(--fg-dim)] font-mono font-semibold mb-1">What we still need</div>
                      <div className="text-[12px] text-[var(--fg-muted)] leading-[1.55]">{row.todo}</div>
                    </div>
                  </div>
                </div>
              );
            })}
          </div>

          <Callout>
            <strong>Get the real numbers.</strong> Bring the stack up, let it run ~10 minutes, then check the Grafana dashboard. Memory is one command (<Code>make bench-mem</Code>); latency lives on the auto-provisioned panel at <Code>localhost:3030</Code>.
          </Callout>
        </Section>

        <Section id="security" n="12" title="Security">
          <Lead>Public on-chain data, so the threat model is operational: stale data from reorgs, dropped blocks, API DoS. Each risk has a mitigation in code.</Lead>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-3 my-6">
            {[
              { risk: "Chain reorg",            level: "Low (L2)", tone: "ok",   mit: "Per-chain confirmation lag. Reorg detection re-emits affected blocks." },
              { risk: "Dropped block",          level: "Low",      tone: "ok",   mit: "Sequence-gap detection on block_number. HTTP RPC re-fetch on miss." },
              { risk: "Slow ClickHouse write",  level: "Medium",   tone: "warn", mit: "Kafka absorbs backpressure. Processor catches up async, no data lost." },
              { risk: "RPC key leak",           level: "Low",      tone: "ok",   mit: "Keys in .env only. .env.example ships placeholders. Never committed." },
              { risk: "API DoS",                level: "Medium",   tone: "warn", mit: "Rate limit 100 req/min/IP. CORS whitelist. /health for upstream LBs." },
              { risk: "Disk fill",              level: "Low",      tone: "ok",   mit: "ClickHouse TTL auto-expires events past 90 days. Alerts at 80% disk." },
            ].map((r) => {
              const toneStyles: Record<string, string> = {
                ok:   "bg-[var(--accent-soft)] text-[var(--accent-fg)] border-[var(--accent-border)]",
                warn: "bg-[#f59e0b1a] text-[#92400e] dark:text-[#fcd34d] border-[#f59e0b55]",
              };
              return (
                <div key={r.risk} className="rounded-xl border border-[var(--border)] bg-[var(--bg-card)] p-5">
                  <div className="flex items-center justify-between gap-3 mb-3">
                    <div className="text-[14px] font-semibold text-[var(--fg-strong)] tracking-tight">{r.risk}</div>
                    <span className={`inline-flex items-center px-2 py-0.5 rounded border text-[10px] font-mono uppercase tracking-[0.14em] font-semibold ${toneStyles[r.tone]}`}>{r.level}</span>
                  </div>
                  <div className="flex items-start gap-2">
                    <Check className="size-3.5 mt-0.5 text-[var(--accent)] shrink-0" />
                    <p className="text-[12.5px] text-[var(--fg-muted)] leading-[1.6]">{r.mit}</p>
                  </div>
                </div>
              );
            })}
          </div>

          <Callout>
            <strong>Idempotent writes.</strong> Every row uses <Code>(chain_id, block_number, log_index)</Code> as a dedup key. ReplacingMergeTree collapses duplicates on merge, so replays are safe.
          </Callout>
        </Section>

            <div className="pt-12 mt-16 border-t border-[var(--border)] flex items-center justify-between gap-4 flex-wrap">
              {prev ? (
                <button
                  onClick={() => go(prev.id)}
                  className="group text-left max-w-[45%] hover:text-[var(--fg-strong)] transition-colors"
                >
                  <div className="text-[10px] uppercase tracking-[0.18em] text-[var(--fg-dim)] font-mono">← Previous</div>
                  <div className="text-[14px] text-[var(--fg-muted)] group-hover:text-[var(--fg-strong)] mt-1 font-medium">
                    <span className="font-mono text-[11px] text-[var(--fg-dim)] mr-1.5">{prev.n}</span>
                    {prev.label}
                  </div>
                </button>
              ) : (
                <Link href="/" className="text-[13px] text-[var(--fg-muted)] hover:text-[var(--fg-strong)]">
                  ← Back to overview
                </Link>
              )}
              {next ? (
                <button
                  onClick={() => go(next.id)}
                  className="group text-right max-w-[45%] hover:text-[var(--fg-strong)] transition-colors"
                >
                  <div className="text-[10px] uppercase tracking-[0.18em] text-[var(--fg-dim)] font-mono">Next →</div>
                  <div className="text-[14px] text-[var(--fg-muted)] group-hover:text-[var(--fg-strong)] mt-1 font-medium">
                    {next.label}
                    <span className="font-mono text-[11px] text-[var(--fg-dim)] ml-1.5">{next.n}</span>
                  </div>
                </button>
              ) : (
                <a
                  href="https://github.com/0xfandom/chainpulse"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="inline-flex items-center gap-1 text-[13px] text-[var(--accent)] hover:text-[var(--accent-hover)]"
                >
                  View source on GitHub <ArrowRight className="size-3.5" />
                </a>
              )}
            </div>

            <div className="mt-12 flex items-center justify-center gap-2 text-[10px] uppercase tracking-[0.18em] font-mono">
              <span className="text-[var(--fg-strong)] font-semibold">ChainPulse</span>
              <span className="text-[var(--fg-dim)]">·</span>
              <span className="text-[var(--fg-muted)]">Documentation</span>
              <span className="text-[var(--fg-dim)]">·</span>
              <span className="text-[var(--fg-dim)]">v0.1.0</span>
            </div>
          </div>
        </article>

        <aside className="hidden lg:flex flex-col w-[260px] shrink-0 border-l border-[var(--border)] bg-[var(--bg-soft)] sticky top-20 self-start h-[calc(100vh-80px)] overflow-y-auto order-2">
          <nav className="flex-1 px-3 py-6 space-y-0.5">
            {SECTIONS.map((s) => {
              const isActive = active === s.id;
              return (
                <button
                  key={s.id}
                  onClick={() => go(s.id)}
                  className={`group w-full flex items-center gap-3 px-3 py-2 rounded-md text-[13px] transition-colors text-left ${
                    isActive
                      ? "bg-[var(--accent-soft)] text-[var(--fg-strong)] font-medium"
                      : "text-[var(--fg-muted)] hover:bg-[var(--bg-hover)] hover:text-[var(--fg-strong)]"
                  }`}
                >
                  <span className={`font-mono text-[10px] tabular-nums ${isActive ? "text-[var(--accent)]" : "text-[var(--fg-dim)]"}`}>{s.n}</span>
                  <span className="truncate">{s.label}</span>
                </button>
              );
            })}
          </nav>
        </aside>
      </div>
    </ActiveCtx.Provider>
  );
}

function Section({ id, children }: { id: string; n: string; title: string; children: React.ReactNode }) {
  const active = useContext(ActiveCtx);
  if (active !== id) return null;
  return <section id={id}>{children}</section>;
}

function Lead({ children }: { children: React.ReactNode }) {
  return <p className="text-[16px] leading-[1.7] text-[var(--fg)] mb-6 font-normal">{children}</p>;
}

function P({ children, className = "" }: { children: React.ReactNode; className?: string }) {
  return <p className={`text-[14px] text-[var(--fg)] leading-[1.75] mb-4 ${className}`}>{children}</p>;
}

function SubH({ children }: { children: React.ReactNode }) {
  return <h3 className="text-[15px] font-semibold text-[var(--fg-strong)] mt-10 mb-3 tracking-tight">{children}</h3>;
}

function Code({ children }: { children: React.ReactNode }) {
  return <code className="font-mono text-[12.5px] bg-[var(--bg-elevated)] border border-[var(--border)] px-1.5 py-0.5 rounded text-[var(--fg-strong)]">{children}</code>;
}

function Audience({ label, title, body }: { label: string; title: string; body: string }) {
  return (
    <div className="grid grid-cols-[100px_1fr] gap-6 py-4 border-b border-[var(--border)] last:border-b-0 first:border-t border-[var(--border)] first:pt-5">
      <div className="text-[10px] uppercase tracking-[0.16em] text-[var(--fg-dim)] font-mono pt-0.5">{label}</div>
      <div>
        <div className="text-[14px] font-semibold text-[var(--fg-strong)] tracking-tight">{title}</div>
        <div className="text-[13.5px] text-[var(--fg-muted)] mt-1 leading-[1.65]">{body}</div>
      </div>
    </div>
  );
}

function DefList({ items }: { items: [string, string][] }) {
  return (
    <dl className="space-y-3 my-4">
      {items.map(([term, def]) => (
        <div key={term} className="grid grid-cols-[120px_1fr] gap-5 items-baseline">
          <dt className="text-[12.5px] font-semibold text-[var(--fg-strong)] font-mono">{term}</dt>
          <dd className="text-[13.5px] text-[var(--fg-muted)] leading-[1.7]">{def}</dd>
        </div>
      ))}
    </dl>
  );
}

function Callout({ children }: { children: React.ReactNode }) {
  return (
    <div className="my-5 rounded-md border border-[var(--accent-border)] bg-[var(--accent-soft)] px-4 py-3 text-[13px] text-[var(--fg)] leading-[1.7]">
      {children}
    </div>
  );
}

function CodeBlock({ children, lang }: { children: string; lang?: string }) {
  const [copied, setCopied] = useState(false);
  return (
    <div className="rounded-md border border-[var(--border)] bg-[#1a1614] my-4 overflow-hidden group">
      <div className="px-4 py-2 flex items-center justify-between border-b border-white/5">
        {lang && <div className="text-[10px] uppercase tracking-widest text-[#a89481] font-mono">{lang}</div>}
        <button
          onClick={() => {
            navigator.clipboard.writeText(children);
            setCopied(true);
            setTimeout(() => setCopied(false), 1200);
          }}
          className="text-[10px] uppercase tracking-widest text-[#a89481] hover:text-[#e8dfcf] flex items-center gap-1 ml-auto transition-colors"
        >
          {copied ? <><Check className="size-3" /> Copied</> : <><Copy className="size-3" /> Copy</>}
        </button>
      </div>
      <pre className="px-4 py-3 overflow-x-auto text-[12.5px] text-[#e8dfcf] font-mono leading-[1.65]">{children}</pre>
    </div>
  );
}

function Table({ headers, rows, mono = [] }: { headers: string[]; rows: string[][]; mono?: number[] }) {
  return (
    <div className="rounded-md border border-[var(--border)] overflow-hidden my-5 bg-[var(--bg-card)]">
      <table className="w-full text-[13px]">
        <thead>
          <tr className="border-b border-[var(--border)]">
            {headers.map((h) => (
              <th key={h} className="text-left px-4 py-2.5 text-[10px] uppercase tracking-[0.12em] text-[var(--fg-dim)] font-semibold bg-[var(--bg-elevated)]/40">{h}</th>
            ))}
          </tr>
        </thead>
        <tbody>
          {rows.map((r, i) => (
            <tr key={i} className="border-t border-[var(--border)]/60 first:border-t-0 hover:bg-[var(--bg-elevated)]/30 transition-colors">
              {r.map((cell, j) => (
                <td key={j} className={`px-4 py-2.5 ${j === 0 ? "font-medium text-[var(--fg-strong)]" : "text-[var(--fg-muted)]"} ${mono.includes(j) ? "font-mono text-[12px]" : ""}`}>{cell}</td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function SchemaCard({
  name,
  tagline,
  tone,
  engine,
  partition,
  orderBy,
  rows,
}: {
  name: string;
  tagline: string;
  tone: "amber" | "green";
  engine: string;
  partition: string;
  orderBy: string;
  rows: [string, string, string][];
}) {
  const toneStyles: Record<string, { ring: string; chip: string; bar: string }> = {
    amber: {
      ring: "border-[#f59e0b55]",
      chip: "bg-[#f59e0b1a] text-[#92400e] dark:text-[#fcd34d] border-[#f59e0b55]",
      bar:  "from-[#f59e0b] to-[#fbbf24]",
    },
    green: {
      ring: "border-[var(--accent-border)]",
      chip: "bg-[var(--accent-soft)] text-[var(--accent-fg)] border-[var(--accent-border)]",
      bar:  "from-[var(--accent)] to-[#4ade80]",
    },
  };
  const s = toneStyles[tone];
  return (
    <div className={`rounded-xl border ${s.ring} bg-[var(--bg-card)] overflow-hidden mb-6`}>
      <div className={`h-[3px] bg-gradient-to-r ${s.bar}`} />
      <div className="px-5 py-4 border-b border-[var(--border)] flex items-center justify-between gap-3 flex-wrap">
        <div className="flex items-center gap-3 min-w-0">
          <Database className="size-4 text-[var(--fg-muted)] shrink-0" />
          <code className="font-mono text-[15px] font-semibold text-[var(--fg-strong)] tracking-tight">{name}</code>
          <span className={`inline-flex items-center px-2 py-0.5 rounded border text-[10px] font-mono uppercase tracking-[0.14em] font-semibold ${s.chip}`}>{rows.length} cols</span>
        </div>
        <div className="text-[11px] text-[var(--fg-muted)] leading-snug">{tagline}</div>
      </div>
      <div className="px-5 py-3 border-b border-[var(--border)] bg-[var(--bg-elevated)]/40 flex flex-wrap gap-x-5 gap-y-1 text-[11px] font-mono">
        <div>
          <span className="text-[var(--fg-dim)] uppercase tracking-[0.14em] mr-2">engine</span>
          <span className="text-[var(--fg-strong)] font-semibold">{engine}</span>
        </div>
        <div>
          <span className="text-[var(--fg-dim)] uppercase tracking-[0.14em] mr-2">partition</span>
          <span className="text-[var(--fg-strong)] font-semibold">{partition}</span>
        </div>
        <div>
          <span className="text-[var(--fg-dim)] uppercase tracking-[0.14em] mr-2">order by</span>
          <span className="text-[var(--fg-strong)] font-semibold">{orderBy}</span>
        </div>
      </div>
      <div>
        {rows.map(([col, type, desc], i) => (
          <div
            key={col}
            className={`grid grid-cols-1 sm:grid-cols-[180px_180px_1fr] gap-x-4 gap-y-1 px-5 py-2.5 ${
              i !== 0 ? "border-t border-[var(--border)]/60" : ""
            } hover:bg-[var(--bg-hover)]/40 transition-colors`}
          >
            <code className="font-mono text-[12.5px] text-[var(--fg-strong)] font-semibold tracking-tight">{col}</code>
            <code className="font-mono text-[11.5px] text-[var(--accent)] truncate">{type}</code>
            <div className="text-[12px] text-[var(--fg-muted)] leading-snug">{desc}</div>
          </div>
        ))}
      </div>
    </div>
  );
}

function Schema({ rows }: { rows: [string, string, string][] }) {
  return (
    <div className="rounded-md border border-[var(--border)] overflow-hidden my-4 bg-[var(--bg-card)]">
      <table className="w-full text-[13px]">
        <tbody>
          {rows.map(([col, type, desc], i) => (
            <tr key={i} className="border-t border-[var(--border)]/60 first:border-t-0">
              <td className="px-4 py-2 font-mono text-[12px] text-[var(--fg-strong)] font-medium align-top whitespace-nowrap">{col}</td>
              <td className="px-4 py-2 font-mono text-[11.5px] text-[var(--accent)] align-top whitespace-nowrap">{type}</td>
              <td className="px-4 py-2 text-[12.5px] text-[var(--fg-muted)] leading-[1.6]">{desc}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function McpTool({ name, desc, params }: { name: string; desc: string; params: { name: string; type: string; required: boolean; hint: string }[] }) {
  return (
    <div className="rounded-lg border border-[var(--border)] bg-[var(--bg-card)] overflow-hidden">
      <div className="px-5 py-3 border-b border-[var(--border)] flex items-center justify-between bg-[var(--bg-elevated)]/30">
        <code className="font-mono text-[13.5px] text-[var(--fg-strong)] font-semibold tracking-tight">{name}</code>
        <span className="text-[9.5px] uppercase tracking-[0.16em] text-[var(--fg-dim)] font-mono">tool</span>
      </div>
      <div className="px-5 py-4">
        <div className="text-[13px] text-[var(--fg)] mb-4 leading-[1.65]">{desc}</div>
        <div className="space-y-1.5">
          {params.map((p) => (
            <div key={p.name} className="grid grid-cols-[140px_60px_1fr] gap-3 items-baseline text-[12px]">
              <code className="font-mono text-[12px] text-[var(--fg-strong)] font-medium">{p.name}</code>
              <span className="font-mono text-[11px] text-[var(--accent)]">{p.type}</span>
              <div className="flex items-center gap-2 text-[var(--fg-muted)]">
                {p.required && <span className="text-[9.5px] uppercase tracking-[0.14em] text-[#9f1239] font-semibold">required</span>}
                <span>{p.hint}</span>
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

function ChainLogo({ slug, className = "" }: { slug: string; className?: string }) {
  const id = `cl-${slug}`;
  const props = { viewBox: "0 0 40 40", className, "aria-hidden": true } as const;
  switch (slug) {
    case "ethereum":
      return (
        <svg {...props}>
          <defs>
            <linearGradient id={`${id}-bg`} x1="0" y1="0" x2="1" y2="1">
              <stop offset="0%" stopColor="#8aa1ff" />
              <stop offset="55%" stopColor="#627eea" />
              <stop offset="100%" stopColor="#3f56c4" />
            </linearGradient>
            <linearGradient id={`${id}-shine`} x1="0" y1="0" x2="0" y2="1">
              <stop offset="0%" stopColor="#ffffff" stopOpacity=".35" />
              <stop offset="60%" stopColor="#ffffff" stopOpacity="0" />
            </linearGradient>
          </defs>
          <circle cx="20" cy="20" r="20" fill={`url(#${id}-bg)`} />
          <circle cx="20" cy="14" r="14" fill={`url(#${id}-shine)`} />
          <g fill="#fff">
            <path d="M20 6.5v10.2l8.5 3.8z" opacity=".55" />
            <path d="M20 6.5 11.5 20.5 20 16.7z" />
            <path d="M20 25v8.5l8.6-12z" opacity=".55" />
            <path d="M20 33.5v-8.5L11.4 21.5z" />
            <path d="M20 23.7l8.5-4.95L20 16.7z" opacity=".25" />
            <path d="M11.5 18.75 20 23.7v-7z" opacity=".7" />
          </g>
        </svg>
      );
    case "polygon":
      return (
        <svg {...props}>
          <defs>
            <linearGradient id={`${id}-bg`} x1="0" y1="0" x2="1" y2="1">
              <stop offset="0%" stopColor="#a166ff" />
              <stop offset="55%" stopColor="#8247e5" />
              <stop offset="100%" stopColor="#5e29bf" />
            </linearGradient>
          </defs>
          <circle cx="20" cy="20" r="20" fill={`url(#${id}-bg)`} />
          <circle cx="20" cy="14" r="14" fill="#fff" opacity=".18" />
          <path
            fill="#fff"
            d="M25.6 16.2c-.4-.2-1-.2-1.4 0l-3.2 1.9-2.2 1.2-3.2 1.9c-.4.2-1 .2-1.4 0l-2.5-1.5c-.4-.2-.7-.7-.7-1.2v-3c0-.5.3-1 .7-1.2l2.5-1.4c.4-.2 1-.2 1.4 0l2.5 1.5c.4.2.7.7.7 1.2v1.9l2.2-1.3v-1.9c0-.5-.3-1-.7-1.2l-4.7-2.8c-.4-.2-1-.2-1.4 0l-4.8 2.9c-.4.2-.7.7-.7 1.2v5.6c0 .5.3 1 .7 1.2l4.8 2.8c.4.2 1 .2 1.4 0l3.2-1.9 2.2-1.2 3.2-1.9c.4-.2 1-.2 1.4 0l2.5 1.5c.4.2.7.7.7 1.2v3c0 .5-.3 1-.7 1.2l-2.5 1.4c-.4.2-1 .2-1.4 0l-2.5-1.5c-.4-.2-.7-.7-.7-1.2v-1.9l-2.2 1.3v1.9c0 .5.3 1 .7 1.2l4.8 2.8c.4.2 1 .2 1.4 0l4.7-2.8c.4-.2.7-.7.7-1.2V19c0-.5-.3-1-.7-1.2l-4.7-2.8z"
          />
        </svg>
      );
    case "arbitrum":
      return (
        <svg {...props}>
          <defs>
            <linearGradient id={`${id}-bg`} x1="0" y1="0" x2="1" y2="1">
              <stop offset="0%" stopColor="#2d3a59" />
              <stop offset="100%" stopColor="#11192e" />
            </linearGradient>
            <linearGradient id={`${id}-mark`} x1="0" y1="0" x2="1" y2="1">
              <stop offset="0%" stopColor="#28a0f0" />
              <stop offset="100%" stopColor="#1675c3" />
            </linearGradient>
          </defs>
          <circle cx="20" cy="20" r="20" fill={`url(#${id}-bg)`} />
          <circle cx="20" cy="14" r="14" fill="#fff" opacity=".06" />
          <path fill={`url(#${id}-mark)`} d="M18.7 11 10 28h4.1l1.5-3h7l1.5 3H28L19.3 11h-.6zm.3 4.9 2.7 5.6h-5.4l2.7-5.6z" />
          <path fill="#9dcced" d="m24.3 16.5 1.7 3.3L29 28h-4l-2.8-7z" />
        </svg>
      );
    case "base":
      return (
        <svg {...props}>
          <defs>
            <linearGradient id={`${id}-bg`} x1="0" y1="0" x2="1" y2="1">
              <stop offset="0%" stopColor="#3b7bff" />
              <stop offset="100%" stopColor="#0040d4" />
            </linearGradient>
          </defs>
          <circle cx="20" cy="20" r="20" fill={`url(#${id}-bg)`} />
          <circle cx="20" cy="14" r="14" fill="#fff" opacity=".18" />
          <path
            fill="#fff"
            d="M20 32a12 12 0 1 1 0-24 12 12 0 0 1 11.93 11H8a12 12 0 0 0 12 13z"
          />
        </svg>
      );
    case "optimism":
      return (
        <svg {...props}>
          <defs>
            <linearGradient id={`${id}-bg`} x1="0" y1="0" x2="1" y2="1">
              <stop offset="0%" stopColor="#ff2d3f" />
              <stop offset="100%" stopColor="#c1001b" />
            </linearGradient>
          </defs>
          <circle cx="20" cy="20" r="20" fill={`url(#${id}-bg)`} />
          <circle cx="20" cy="14" r="14" fill="#fff" opacity=".18" />
          <path
            fill="#fff"
            d="M14 26.3c-1.4 0-2.5-.4-3.3-1.1-.8-.7-1.2-1.7-1.2-3 0-.3 0-.6.1-1 .2-1.2.4-2.2.8-3 1-2.5 3.1-3.7 6.1-3.7.9 0 1.6.2 2.3.4.6.3 1.1.7 1.5 1.2.4.6.6 1.2.6 2 0 .3 0 .6-.1 1-.2 1.2-.5 2.2-.8 3-.5 1.2-1.3 2.1-2.3 2.8-1 .6-2.3 1-3.7 1zm.4-2.7c.6 0 1.2-.2 1.6-.6.5-.4.8-1 1-1.7.2-.9.4-1.6.5-2.1.1-.3.1-.6.1-.8 0-.9-.5-1.4-1.4-1.4-.7 0-1.3.2-1.7.6-.5.4-.8 1-1 1.6-.3.7-.4 1.4-.5 2.2-.1.3-.1.6-.1.8 0 .9.5 1.4 1.5 1.4zM21.4 26.1c-.1 0-.2 0-.3-.1-.1-.1-.1-.2 0-.3l2.2-10.4c0-.1.1-.2.2-.2.1-.1.2-.1.3-.1h4.2c1.2 0 2.1.2 2.8.7.7.5 1 1.2 1 2.1 0 .3 0 .5-.1.8-.2 1-.7 1.7-1.4 2.3-.7.5-1.7.8-2.9.8h-2.1l-.7 3.5c0 .1-.1.2-.2.2-.1.1-.2.1-.3.1h-2.7zm5.5-6.6c.4 0 .7-.1 1-.3.2-.2.4-.5.5-.9 0-.1 0-.2 0-.3 0-.2-.1-.4-.3-.5-.2-.1-.4-.2-.7-.2h-1.9l-.4 2.2h1.8z"
          />
        </svg>
      );
    case "bsc":
      return (
        <svg {...props}>
          <defs>
            <linearGradient id={`${id}-bg`} x1="0" y1="0" x2="1" y2="1">
              <stop offset="0%" stopColor="#fbcf4d" />
              <stop offset="100%" stopColor="#e5a200" />
            </linearGradient>
          </defs>
          <circle cx="20" cy="20" r="20" fill={`url(#${id}-bg)`} />
          <circle cx="20" cy="14" r="14" fill="#fff" opacity=".22" />
          <g fill="#fff">
            <path d="m20 9 3.7 3.7L20 16.4l-3.7-3.7z" />
            <path d="m12.7 16.3 3.7 3.7-3.7 3.7L9 20z" />
            <path d="m20 23.6 3.7 3.7L20 31l-3.7-3.7z" />
            <path d="m27.3 16.3 3.7 3.7-3.7 3.7L23.6 20z" />
            <path d="m20 16.4 3.6 3.6L20 23.6 16.4 20z" opacity=".55" />
          </g>
        </svg>
      );
    case "avalanche":
      return (
        <svg {...props}>
          <defs>
            <linearGradient id={`${id}-bg`} x1="0" y1="0" x2="1" y2="1">
              <stop offset="0%" stopColor="#ff5b5b" />
              <stop offset="100%" stopColor="#c00f10" />
            </linearGradient>
          </defs>
          <circle cx="20" cy="20" r="20" fill={`url(#${id}-bg)`} />
          <circle cx="20" cy="14" r="14" fill="#fff" opacity=".18" />
          <path fill="#fff" d="M23.2 11.2 31 28h-6l-1.8-3.5-1.8-3.5-2.4-4.4 4.2-5.4z" />
          <path fill="#fff" d="M14.6 20.5 18 28H9.5l5.1-7.5z" />
        </svg>
      );
    case "scroll":
      return (
        <svg {...props}>
          <defs>
            <linearGradient id={`${id}-bg`} x1="0" y1="0" x2="1" y2="1">
              <stop offset="0%" stopColor="#fff6e2" />
              <stop offset="100%" stopColor="#f3d8a7" />
            </linearGradient>
            <linearGradient id={`${id}-fg`} x1="0" y1="0" x2="0" y2="1">
              <stop offset="0%" stopColor="#fab26a" />
              <stop offset="100%" stopColor="#d98a3a" />
            </linearGradient>
          </defs>
          <circle cx="20" cy="20" r="20" fill={`url(#${id}-bg)`} />
          <circle cx="20" cy="14" r="14" fill="#fff" opacity=".25" />
          <path
            fill={`url(#${id}-fg)`}
            d="M10 14.5c0-1.4 1.1-2.5 2.5-2.5h13a5 5 0 0 1 0 10H22v7c0 1.4-1.1 2.5-2.5 2.5h-7c-1.4 0-2.5-1.1-2.5-2.5v-14.5zm2.5-.5a.5.5 0 0 0-.5.5v14.5c0 .3.2.5.5.5h6.6a2.5 2.5 0 0 1-.6-1.6v-13.9h-6zm12.5 0h-4.5v4.4c0 .4-.1.7-.2.9h4.7a3 3 0 0 0 0-6 .8.8 0 0 1-.7-.7v-1.1c0-.4.3-.5.7-.5z"
          />
          <circle cx="25" cy="14.6" r="1.3" fill={`url(#${id}-bg)`} />
        </svg>
      );
    case "zksync-era":
      return (
        <svg {...props}>
          <defs>
            <linearGradient id={`${id}-bg`} x1="0" y1="0" x2="1" y2="1">
              <stop offset="0%" stopColor="#3a3a3a" />
              <stop offset="100%" stopColor="#0a0a0a" />
            </linearGradient>
            <linearGradient id={`${id}-fg`} x1="0" y1="0" x2="1" y2="0">
              <stop offset="0%" stopColor="#ffffff" />
              <stop offset="100%" stopColor="#c2c2c2" />
            </linearGradient>
          </defs>
          <circle cx="20" cy="20" r="20" fill={`url(#${id}-bg)`} />
          <circle cx="20" cy="14" r="14" fill="#fff" opacity=".08" />
          <path
            fill={`url(#${id}-fg)`}
            d="M31 20l-8-7.5v5.4l-5.1.1L13 14.6v4.4l-5 .8 5 .8v4.4l5-3.2 5 .1V25.5z"
          />
        </svg>
      );
    default:
      return (
        <svg {...props}>
          <circle cx="20" cy="20" r="20" fill="#a1a1aa" />
        </svg>
      );
  }
}

function ProtocolLogo({ slug, className = "" }: { slug: string; className?: string }) {
  const id = `pl-${slug}`;
  const props = { viewBox: "0 0 40 40", className, "aria-hidden": true } as const;
  switch (slug) {
    case "erc20":
      return (
        <svg {...props}>
          <defs>
            <linearGradient id={`${id}-bg`} x1="0" y1="0" x2="1" y2="1">
              <stop offset="0%" stopColor="#ffd166" />
              <stop offset="100%" stopColor="#e09f1e" />
            </linearGradient>
            <linearGradient id={`${id}-rim`} x1="0" y1="0" x2="0" y2="1">
              <stop offset="0%" stopColor="#fff5d0" />
              <stop offset="100%" stopColor="#c4881a" />
            </linearGradient>
          </defs>
          <circle cx="20" cy="20" r="20" fill={`url(#${id}-bg)`} />
          <circle cx="20" cy="20" r="15" fill="none" stroke={`url(#${id}-rim)`} strokeWidth="1.5" />
          <circle cx="20" cy="14" r="14" fill="#fff" opacity=".22" />
          <text
            x="20"
            y="27"
            textAnchor="middle"
            fill="#fff"
            fontFamily="ui-sans-serif, system-ui"
            fontWeight={900}
            fontSize="18"
          >
            $
          </text>
        </svg>
      );
    case "uniswap_v3":
      return (
        <svg {...props}>
          <defs>
            <linearGradient id={`${id}-bg`} x1="0" y1="0" x2="1" y2="1">
              <stop offset="0%" stopColor="#ff4fa3" />
              <stop offset="100%" stopColor="#d40068" />
            </linearGradient>
          </defs>
          <circle cx="20" cy="20" r="20" fill={`url(#${id}-bg)`} />
          <circle cx="20" cy="14" r="14" fill="#fff" opacity=".22" />
          <path
            fill="#fff"
            d="M14.5 9.5c-.9-.2-1.8.3-2 1.2-.1.4 0 .9.2 1.3l4.6 6.4c.9 1.2 1.3 2.2 1.1 3.4-.1 1.3-.9 2.2-1.8 2.2-.7 0-1.2-.3-1.6-.8l-2.2-2.9a1 1 0 0 0-.7-.4c-.5 0-.9.5-.9 1 0 .3.1.6.3.8l2.1 2.8c1.4 1.8 3.4 2.7 5.3 2 2.2-.8 3.4-3 2.9-5.3-.1-.9-.5-1.8-1.1-2.6l-4.5-6.4 8 6c.5.4.6 1.1.3 1.6-.3.4-.9.5-1.3.2l-1.3-1a.9.9 0 0 0-1.2 0 .9.9 0 0 0 0 1.3l1.3 1c1.3.9 3.1.8 4-.5.9-1.3.7-3.1-.5-4l-10.9-8.2a2 2 0 0 0-1.1-.4z"
          />
          <circle cx="16.5" cy="13.4" r="1.4" fill="#fff" />
        </svg>
      );
    case "aave_v3":
      return (
        <svg {...props}>
          <defs>
            <linearGradient id={`${id}-bg`} x1="0" y1="0" x2="1" y2="1">
              <stop offset="0%" stopColor="#c469af" />
              <stop offset="100%" stopColor="#7a2d6b" />
            </linearGradient>
          </defs>
          <circle cx="20" cy="20" r="20" fill={`url(#${id}-bg)`} />
          <circle cx="20" cy="14" r="14" fill="#fff" opacity=".22" />
          <path
            fill="#fff"
            d="M20 8c-4.4 0-8 3.6-8 8v11.6c0 .7.7 1.1 1.3.8L15 27.5l2.3 1c.4.2.9.2 1.3 0l2.3-1 2.3 1c.4.2.9.2 1.3 0l1.5-.8c.6.3 1.3-.1 1.3-.8V16c0-4.4-3.6-8-8-8z"
          />
          <circle cx="16.8" cy="17" r="1.6" fill="#7a2d6b" />
          <circle cx="23.2" cy="17" r="1.6" fill="#7a2d6b" />
          <path d="M16 21c1.5 1.5 6.5 1.5 8 0" stroke="#7a2d6b" strokeWidth="1.4" strokeLinecap="round" fill="none" />
        </svg>
      );
    case "compound_v3":
      return (
        <svg {...props}>
          <defs>
            <linearGradient id={`${id}-bg`} x1="0" y1="0" x2="1" y2="1">
              <stop offset="0%" stopColor="#3df0b3" />
              <stop offset="100%" stopColor="#009d6e" />
            </linearGradient>
          </defs>
          <circle cx="20" cy="20" r="20" fill={`url(#${id}-bg)`} />
          <circle cx="20" cy="14" r="14" fill="#fff" opacity=".22" />
          <path
            fill="none"
            stroke="#fff"
            strokeWidth="3"
            strokeLinecap="round"
            d="M27.5 15.5a9 9 0 1 0 0 9"
          />
          <path
            fill="none"
            stroke="#fff"
            strokeWidth="3"
            strokeLinecap="round"
            opacity=".5"
            d="M24.5 13.2a6 6 0 1 0 0 13.6"
          />
        </svg>
      );
    case "curve":
      return (
        <svg {...props}>
          <defs>
            <linearGradient id={`${id}-bg`} x1="0" y1="0" x2="1" y2="1">
              <stop offset="0%" stopColor="#ffd84f" />
              <stop offset="100%" stopColor="#e09c00" />
            </linearGradient>
          </defs>
          <circle cx="20" cy="20" r="20" fill={`url(#${id}-bg)`} />
          <circle cx="20" cy="14" r="14" fill="#fff" opacity=".22" />
          <path
            fill="none"
            stroke="#1b1b1b"
            strokeWidth="2.6"
            strokeLinecap="round"
            d="M7 14c3.5 0 3.5 10 7 10s3.5-10 7-10 3.5 10 7 10"
          />
          <path
            fill="none"
            stroke="#1b1b1b"
            strokeWidth="2.6"
            strokeLinecap="round"
            opacity=".45"
            d="M7 21c3.5 0 3.5 10 7 10s3.5-10 7-10"
          />
        </svg>
      );
    case "lido":
      return (
        <svg {...props}>
          <defs>
            <linearGradient id={`${id}-bg`} x1="0" y1="0" x2="1" y2="1">
              <stop offset="0%" stopColor="#5cc4ff" />
              <stop offset="100%" stopColor="#0080cc" />
            </linearGradient>
            <linearGradient id={`${id}-drop`} x1="0" y1="0" x2="0" y2="1">
              <stop offset="0%" stopColor="#ffffff" />
              <stop offset="100%" stopColor="#dfeefd" />
            </linearGradient>
          </defs>
          <circle cx="20" cy="20" r="20" fill={`url(#${id}-bg)`} />
          <circle cx="20" cy="14" r="14" fill="#fff" opacity=".22" />
          <path
            fill={`url(#${id}-drop)`}
            d="M20 7c-2.1 4-7 8.4-7 14a7 7 0 0 0 14 0c0-5.6-4.9-10-7-14z"
          />
          <path
            fill="#0080cc"
            opacity=".25"
            d="M20 28.5c2.6 0 5-1.7 6-4.2-2.2 1-3.9 1.4-6 1.4s-3.8-.4-6-1.4c1 2.5 3.4 4.2 6 4.2z"
          />
          <path fill="#fff" opacity=".55" d="M18 12c-1 1.4-2.6 3.3-3.2 5.4.6-.4 1.4-.7 2.4-.8.6-1.5 1.4-2.9 2.1-4z" />
        </svg>
      );
    default:
      return (
        <svg {...props}>
          <circle cx="20" cy="20" r="20" fill="#a1a1aa" />
        </svg>
      );
  }
}

type Tone = "ingest" | "bus" | "process" | "store" | "serve" | "client";

const TONE_STYLES: Record<Tone, { ring: string; chip: string; label: string }> = {
  ingest:  { ring: "border-l-[var(--accent)]", chip: "bg-[var(--accent-soft)] text-[var(--accent-fg)] border-[var(--accent-border)]", label: "Ingest" },
  bus:     { ring: "border-l-[#a78bfa]", chip: "bg-[#a78bfa1f] text-[#7c3aed] border-[#a78bfa55] dark:text-[#c4b5fd]", label: "Bus" },
  process: { ring: "border-l-[var(--warn)]", chip: "bg-[#f59e0b1a] text-[#92400e] border-[#f59e0b55] dark:text-[#fcd34d]", label: "Process" },
  store:   { ring: "border-l-[var(--danger)]", chip: "bg-[#ef44441a] text-[#991b1b] border-[#ef444455] dark:text-[#fca5a5]", label: "Store" },
  serve:   { ring: "border-l-[var(--info)]", chip: "bg-[#3b82f61a] text-[#1d4ed8] border-[#3b82f655] dark:text-[#93c5fd]", label: "Serve" },
  client:  { ring: "border-l-[#14b8a6]", chip: "bg-[#14b8a61a] text-[#0f766e] border-[#14b8a655] dark:text-[#5eead4]", label: "Client" },
};

function Stage({
  n, tone, title, sub, chips, body,
}: {
  n: string;
  tone: Tone;
  title: string;
  sub?: string;
  chips: string[];
  body: string;
}) {
  const s = TONE_STYLES[tone];
  return (
    <div className={`relative rounded-md border border-[var(--border)] bg-[var(--bg-card)] border-l-[3px] ${s.ring} p-5`}>
      <div className="grid grid-cols-1 sm:grid-cols-[80px_1fr] gap-x-5 gap-y-3">
        <div className="flex sm:flex-col items-start gap-2 sm:gap-1">
          <div className="font-mono text-[10.5px] tracking-[0.16em] text-[var(--fg-dim)] uppercase">Stage {n}</div>
          <div className="font-mono text-[9.5px] tracking-[0.18em] uppercase text-[var(--fg-dim)]">{s.label}</div>
        </div>
        <div>
          <div className="text-[14.5px] font-semibold text-[var(--fg-strong)] tracking-tight leading-tight">{title}</div>
          {sub && <div className="text-[12px] text-[var(--fg-muted)] mt-0.5 font-mono">{sub}</div>}
          <div className="flex flex-wrap gap-1.5 mt-3">
            {chips.map((c) => (
              <span key={c} className={`inline-flex items-center px-2 py-0.5 rounded border text-[10.5px] font-mono tracking-tight ${s.chip}`}>{c}</span>
            ))}
          </div>
          <p className="text-[13px] text-[var(--fg)] leading-[1.7] mt-3">{body}</p>
        </div>
      </div>
    </div>
  );
}

function StageGap() {
  return (
    <div className="flex justify-center py-1.5">
      <svg width="14" height="22" viewBox="0 0 14 22" fill="none" className="text-[var(--border-strong)]">
        <path d="M7 0 L7 16 M2 12 L7 17 L12 12" stroke="currentColor" strokeWidth="1.2" strokeLinecap="round" strokeLinejoin="round" />
      </svg>
    </div>
  );
}

function FlowDiagram() {
  return (
    <div className="my-8 rounded-xl border border-[var(--border)] bg-[var(--bg-elevated)]/30 p-6 sm:p-8">
      <div className="flex items-center justify-between flex-wrap gap-3 mb-6 pb-4 border-b border-[var(--border)]">
        <div>
          <div className="text-[10px] uppercase tracking-[0.18em] text-[var(--fg-dim)] font-mono">Pipeline · top to bottom</div>
          <div className="text-[15px] font-semibold text-[var(--fg-strong)] tracking-tight mt-1">A new block leaves the chain at Stage 1, a JSON response leaves at Stage 8</div>
        </div>
        <div className="flex flex-wrap gap-1.5">
          {(["ingest","bus","process","store","serve","client"] as Tone[]).map((t) => (
            <span key={t} className={`inline-flex items-center gap-1.5 px-2 py-0.5 rounded border text-[10px] font-mono uppercase tracking-wider ${TONE_STYLES[t].chip}`}>
              <span className="w-1.5 h-1.5 rounded-full bg-current" />
              {TONE_STYLES[t].label}
            </span>
          ))}
        </div>
      </div>

      <div>
        <Stage
          n="1"
          tone="ingest"
          title="Chain listener"
          sub="cmd/indexer · one goroutine per chain"
          chips={["WSS subscribe", "eth_subscribe('logs', ...)", "1s→30s backoff", "head_timeout watchdog"]}
          body="Holds a persistent WebSocket to each chain's RPC. Topic-filtered subscription so the provider only streams events the registered decoders can handle — what makes free-tier WSS endpoints viable. If no header arrives within head_timeout (default 90s) the listener forces a reconnect even when the WS subscription has not surfaced an error."
        />
        <StageGap />
        <Stage
          n="2"
          tone="ingest"
          title="Decode + signature router"
          sub="indexer-side · in-process"
          chips={["go-ethereum ABI", "topics[0] keccak256", "ERC-20 fast path", "tag + forward"]}
          body="Every event log starts with topics[0], a 32-byte hash of the event's signature. An in-memory map of known hashes tags the event as recognised, then forwards it to the Kafka producer. Heavy decoding happens later in the processor — this stage just labels."
        />
        <StageGap />
        <Stage
          n="3"
          tone="bus"
          title="Kafka raw_events"
          sub="topic · KRaft mode · 7d retention"
          chips={["partition = chain_id", "segmentio/kafka-go", "manual offset commit", "replayable"]}
          body="Durable append-only log between indexer and processor. Per-chain ordering is preserved by partitioning on chain_id (Aave repay must not arrive before its supply). Kafka absorbs backpressure when ClickHouse slows down, enables replay if a new decoder ships, and fans out the same stream to multiple consumer groups."
        />
        <StageGap />
        <Stage
          n="4"
          tone="process"
          title="Processor"
          sub="cmd/processor · consumer + decoder registry + aggregator"
          chips={["first-match registry", "erc20 · uniswap_v3 · aave_v3 · compound_v3", "curve · lido", "drop unknown · commit offset"]}
          body="Consumes raw_events, asks each registered ProtocolDecoder 'can you handle this?' in insertion order, runs Decode() on the first match, then routes the typed DecodedEvent to the storage writers. Unknown events (the majority on any busy chain) are committed back to Kafka and dropped — replay catches them for free if a decoder ships later."
        />
        <StageGap />
        <Stage
          n="5"
          tone="store"
          title="Storage fan-out"
          sub="ClickHouse 24.3 + Redis 7 + decoded_events topic"
          chips={["BatchWriter 1000 rows / 1s", "HINCRBY balances", "SET positions", "ReplacingMergeTree dedupe"]}
          body="The processor writes three places at once. BatchWriter buffers up to 1000 rows or 1s into bulk INSERTs against token_transfers + defi_events. CacheWriter updates Redis hot state atomically (HINCRBY balance:wallet:chain, SET position:wallet:protocol). A second Kafka producer republishes the decoded event to decoded_events for live WS subscribers. Idempotent: every row dedupes on (chain_id, block_number, log_index) so replays are safe."
        />
        <StageGap />
        <Stage
          n="6"
          tone="serve"
          title="Read layer"
          sub="internal/api/store · shared by REST, gRPC, MCP"
          chips={["ReadStore (CH)", "ReadCache (Redis)", "Aside[T] cache-aside", "60s TTL"]}
          body="A thin set of typed query helpers around clickhouse-go/v2 and go-redis/v9 — WalletHistory, TokenTransfers, ProtocolStats, WhaleTransfers, and friends. The same instances are imported by every surface above, so no SQL is duplicated. Aside[T] centralises the try-Redis-then-ClickHouse-then-write-back pattern."
        />
        <StageGap />
        <Stage
          n="7"
          tone="serve"
          title="Query surfaces"
          sub="cmd/api + cmd/mcp"
          chips={["REST :8080 (Gin)", "gRPC :8081 (proto + reflection)", "WS /v1/events/stream", "MCP stdio + SSE :3001"]}
          body="Four ways out, same data underneath. REST for browsers and scripts, gRPC for typed backend-to-backend calls, WebSocket for live event streams (ephemeral consumer group per socket at LastOffset), MCP for AI agents over stdio (Claude Desktop) or SSE (HTTP)."
        />
        <StageGap />
        <Stage
          n="8"
          tone="client"
          title="Clients"
          sub="three audiences, one indexer"
          chips={["Web UI :3000", "Backend services", "Claude / Cursor / agents", "Telegram / Discord bots"]}
          body="Dashboards and the bundled Next.js explorer hit REST + WebSocket. Backend services use gRPC for low-latency typed calls. AI agents call MCP tools like regular functions and get clean JSON back without touching contract ABIs."
        />
      </div>

      <div className="mt-6 pt-4 border-t border-[var(--border)] text-[11px] text-[var(--fg-dim)] font-mono uppercase tracking-[0.14em] text-center">
        Block on chain → queryable row in ~1–2s
      </div>
    </div>
  );
}
