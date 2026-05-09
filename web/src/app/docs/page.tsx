"use client";
import Link from "next/link";
import { useEffect, useState } from "react";
import { ExternalLink, ArrowRight, Hash, Check, Copy } from "lucide-react";

const SECTIONS = [
  { id: "intro", n: "01", label: "Introduction" },
  { id: "architecture", n: "02", label: "Architecture" },
  { id: "chains", n: "03", label: "Supported chains" },
  { id: "protocols", n: "04", label: "Decoded protocols" },
  { id: "schema", n: "05", label: "Data model" },
  { id: "rest", n: "06", label: "REST API" },
  { id: "ws", n: "07", label: "WebSocket" },
  { id: "mcp", n: "08", label: "MCP tools" },
  { id: "perf", n: "09", label: "Performance" },
  { id: "security", n: "10", label: "Security" },
];

export default function DocsPage() {
  const [active, setActive] = useState("intro");

  useEffect(() => {
    const onScroll = () => {
      let cur = SECTIONS[0].id;
      for (const s of SECTIONS) {
        const el = document.getElementById(s.id);
        if (!el) continue;
        if (el.getBoundingClientRect().top <= 140) cur = s.id;
      }
      setActive(cur);
    };
    window.addEventListener("scroll", onScroll, { passive: true });
    onScroll();
    return () => window.removeEventListener("scroll", onScroll);
  }, []);

  return (
    <div className="grid grid-cols-1 lg:grid-cols-[220px_1fr] gap-12 lg:gap-16">
      <aside className="hidden lg:block">
        <div className="sticky top-20 space-y-1">
          <div className="text-[10px] uppercase tracking-[0.18em] text-[var(--fg-dim)] font-semibold mb-3 px-2">Documentation</div>
          {SECTIONS.map((s) => (
            <a
              key={s.id}
              href={`#${s.id}`}
              className={`group flex items-center gap-3 px-2 py-1.5 rounded-md text-[13px] transition-colors ${
                active === s.id
                  ? "text-[var(--fg-strong)] font-medium"
                  : "text-[var(--fg-muted)] hover:text-[var(--fg-strong)]"
              }`}
            >
              <span className={`font-mono text-[10px] tabular-nums ${active === s.id ? "text-[var(--accent)]" : "text-[var(--fg-dim)]"}`}>{s.n}</span>
              <span>{s.label}</span>
            </a>
          ))}
          <div className="pt-6 mt-4 border-t border-[var(--border)] px-2">
            <a
              href="/prd.html"
              target="_blank"
              rel="noopener noreferrer"
              className="text-[12px] text-[var(--fg-dim)] hover:text-[var(--fg-strong)] inline-flex items-center gap-1"
            >
              Read full PRD <ExternalLink className="size-3" />
            </a>
          </div>
        </div>
      </aside>

      <article className="max-w-[700px] pb-20">
        <header className="mb-16 pb-10 border-b border-[var(--border)]">
          <div className="text-[10px] uppercase tracking-[0.18em] text-[var(--fg-dim)] font-semibold font-mono mb-3">
            ChainPulse · Documentation · v0.1.0
          </div>
          <h1 className="text-[44px] font-semibold tracking-[-0.03em] text-[var(--fg-strong)] leading-[1.05]">
            How <span className="font-serif italic font-normal text-[var(--accent)]">ChainPulse</span> works.
          </h1>
          <p className="text-[15px] text-[var(--fg-muted)] mt-5 leading-[1.7] max-w-[600px]">
            ChainPulse is an open-source multi-chain blockchain indexer. It listens to three EVM networks in real time, decodes DeFi protocol events into a normalized schema, and exposes the data through a REST API, a WebSocket feed, and an MCP server for AI agents. This is the reference for using it.
          </p>
        </header>

        <Section id="intro" n="01" title="Introduction">
          <Lead>
            Three audiences, one indexer. Whoever you are, the data path is identical: chain WebSocket → Kafka → ClickHouse → query.
          </Lead>

          <Audience
            label="Humans"
            title="Web dashboard"
            body="The ChainPulse app you are reading the docs in. Live whale feed, wallet lookup, protocol pulse. Nothing to install."
          />
          <Audience
            label="Engineers"
            title="REST + WebSocket"
            body="HTTP endpoints for wallet positions, balances, history, and protocol stats. WebSocket streams every decoded event as it lands."
          />
          <Audience
            label="AI agents"
            title="MCP server"
            body="Eight structured tools any MCP-compatible LLM can call. Claude or GPT answer wallet questions without touching contract ABIs."
          />

          <P>
            Everything is open source under MIT and runs locally via Docker Compose. The guide below covers the architecture, the data shape, the APIs, and the operational guarantees.
          </P>
        </Section>

        <Section id="architecture" n="02" title="Architecture">
          <Lead>
            Four Go services connected by Kafka topics. Pipeline stages mirror data flow from chain RPC into your query.
          </Lead>

          <FlowDiagram />

          <P>
            <strong>Why this shape.</strong> Kafka separates write-heavy (indexer, processor) from read-heavy (API, MCP). Restart the API, miss zero blocks. Replay the processor, no re-fetch from RPC. Scale layers independently.
          </P>

          <SubH>The four services</SubH>
          <DefList
            items={[
              ["Indexer", "Subscribes to chain WebSockets, one goroutine per chain. Decodes raw logs via go-ethereum. Publishes to raw_events Kafka topic. Health: /health on :9180, metrics on :9100."],
              ["Processor", "Consumes raw_events. Runs protocol decoders (ERC-20, Uniswap V3, Aave V3, Compound V3). Writes batched inserts to ClickHouse and updates Redis hot cache."],
              ["API", "Gin HTTP server. REST endpoints under /v1, WebSocket at /v1/events/stream. Cache-aside pattern: Redis first, ClickHouse on miss. Rate limit 100 req/min per IP."],
              ["MCP", "Model Context Protocol server with eight tool definitions. SSE transport for HTTP clients, stdio transport for Claude Desktop."],
            ]}
          />

          <SubH>Why each technology</SubH>
          <DefList
            items={[
              ["Go", "Goroutines for per-chain concurrency. go-ethereum is the production-tested ABI/RPC library. Single static binary deploy, no runtime needed."],
              ["Kafka", "Backpressure absorption: slow ClickHouse writes never block ingestion. Replay any offset to reprocess history. Fan-out to processor, WebSocket, and cache simultaneously."],
              ["ClickHouse", "Append-heavy + analytical queries match its sweet spot. Scans 1B+ rows per second, columnar compression 10-20x, ReplacingMergeTree handles dedupe."],
              ["Redis", "Sub-millisecond point lookups for wallet balances and recent positions. ClickHouse handles cold scans, Redis serves hot ones."],
            ]}
          />
        </Section>

        <Section id="chains" n="03" title="Supported chains">
          <Lead>Three EVM networks streaming live. Confirmation lag is per-chain to stay reorg-safe.</Lead>

          <Table
            headers={["Chain", "Chain ID", "Block time", "Confirmations"]}
            rows={[
              ["Ethereum", "1", "12s", "2"],
              ["Polygon", "137", "2s", "5"],
              ["Arbitrum", "42161", "0.25s", "5"],
            ]}
            mono={[1, 2, 3]}
          />

          <P>
            Confirmation lag is the depth at which a block is considered final and safe to write. L2s with fast block times need more confirmations to absorb reorgs; Ethereum needs fewer because finality is harder.
          </P>
        </Section>

        <Section id="protocols" n="04" title="Decoded protocols">
          <Lead>Each protocol has a Go decoder converting raw event logs into structured rows. Adding a protocol is one interface implementation, not a pipeline change.</Lead>

          <Table
            headers={["Protocol", "Events", "Chains"]}
            rows={[
              ["ERC-20", "Transfer, Approval", "All 9"],
              ["Uniswap V3", "Swap, Mint, Burn, Collect", "ETH, BASE, ARB, OP, POLY"],
              ["Aave V3", "Supply, Withdraw, Borrow, Repay, Liquidation", "ETH, BASE, ARB, OP, POLY, AVAX"],
              ["Compound V3", "Supply, Withdraw, Absorb", "ETH, BASE, ARB"],
            ]}
          />
        </Section>

        <Section id="schema" n="05" title="Data model">
          <Lead>Two ClickHouse tables hold all indexed data. Both use ReplacingMergeTree on the natural primary key, so re-inserts during replay are idempotent.</Lead>

          <SubH>defi_events</SubH>
          <P>Protocol-decoded events. One row per matched log.</P>
          <Schema
            rows={[
              ["chain_id", "UInt64", "EVM chain ID"],
              ["block_number", "UInt64", "Block this event was emitted in"],
              ["tx_hash", "String", "Full transaction hash, 0x-prefixed"],
              ["log_index", "UInt32", "Log position within the tx"],
              ["protocol", "LowCardinality(String)", "uniswap_v3 / aave_v3 / compound_v3 / lido / curve"],
              ["event_type", "LowCardinality(String)", "swap / supply / borrow / liquidation"],
              ["user_addr", "String", "Wallet address (or pool, depending on protocol)"],
              ["token_a", "String", "Primary token contract"],
              ["token_b", "Nullable(String)", "Secondary token (swaps, LPs)"],
              ["amount_a", "String", "uint256 stringified"],
              ["amount_b", "Nullable(String)", "Secondary amount"],
              ["params", "String", "Protocol-specific JSON blob"],
              ["timestamp", "DateTime", "UTC, from block timestamp"],
            ]}
          />

          <SubH>token_transfers</SubH>
          <P>Append-only stream of every ERC-20 Transfer the indexer recognizes.</P>
          <Schema
            rows={[
              ["chain_id", "UInt64", "EVM chain ID"],
              ["block_number", "UInt64", "Block number"],
              ["tx_hash", "String", "Transaction hash"],
              ["log_index", "UInt32", "Log index"],
              ["token", "String", "Token contract"],
              ["from_addr", "String", "Sender"],
              ["to_addr", "String", "Receiver"],
              ["amount", "String", "uint256 stringified"],
              ["timestamp", "DateTime", "UTC"],
            ]}
          />

          <Callout>
            Both tables retain 90 days, partitioned monthly by <Code>toYYYYMM(timestamp)</Code>. Adjust TTL in <Code>schema/*.sql</Code> if you need longer history.
          </Callout>
        </Section>

        <Section id="rest" n="06" title="REST API">
          <Lead>HTTP endpoints under /v1 on port 8080. Cache-aside (Redis → ClickHouse). 100 requests/min per IP.</Lead>

          <Table
            headers={["Method", "Endpoint", "Description"]}
            rows={[
              ["GET", "/v1/wallet/{addr}/positions", "All DeFi positions across chains"],
              ["GET", "/v1/wallet/{addr}/balances", "Token balances per chain"],
              ["GET", "/v1/wallet/{addr}/history", "Decoded transaction history"],
              ["GET", "/v1/token/{addr}/transfers", "Recent transfers for a token"],
              ["GET", "/v1/protocol/{name}/stats", "TVL, volume, unique users"],
              ["GET", "/v1/chain/{id}/blocks", "Recent indexed blocks"],
              ["GET", "/health", "Health probe"],
              ["GET", "/metrics", "Prometheus scrape"],
            ]}
            mono={[0, 1]}
          />

          <SubH>Example</SubH>
          <CodeBlock lang="bash">{`curl http://localhost:8080/v1/wallet/0x2faf487a4414fe77e2327f0bf4ae2a264a776ad2/positions`}</CodeBlock>

          <CodeBlock lang="json">{`[
  {
    "chain_id": 42161,
    "protocol": "aave_v3",
    "type": "supply",
    "token": "0xaf88d065e77c8cc2239327c5edb3a432268e5831",
    "amount": "1500000000",
    "updated": "2026-05-06T09:54:00Z"
  }
]`}</CodeBlock>
        </Section>

        <Section id="ws" n="07" title="WebSocket">
          <Lead>Live stream of every decoded event. Kafka consumer fans out to all connected clients.</Lead>

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
        </Section>

        <Section id="mcp" n="08" title="MCP tools">
          <Lead>
            Eight tools the MCP server exposes to LLMs. Schemas are JSON Schema-validated; address parameters require a 0x-prefixed 40-hex pattern. The server runs on port 3001.
          </Lead>

          <div className="space-y-4 my-6">
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

          <SubH>Connecting Claude Desktop</SubH>
          <P>Add this to your <Code>claude_desktop_config.json</Code>:</P>
          <CodeBlock lang="json">{`{
  "mcpServers": {
    "chainpulse": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/proxy", "http://localhost:3001/mcp"]
    }
  }
}`}</CodeBlock>
        </Section>

        <Section id="perf" n="09" title="Performance">
          <Lead>Latency budgets the system is designed to hit, measured under steady-state load with 3 chains and 4 decoders running.</Lead>

          <div className="grid grid-cols-2 sm:grid-cols-3 gap-px bg-[var(--border)] rounded-lg overflow-hidden border border-[var(--border)] mt-6">
            {[
              ["<100ms", "Block ingestion"],
              ["<500ms", "Event processing"],
              ["<5ms", "Cache-hit query"],
              ["<50ms", "p99 API latency"],
              ["<2s", "Block → queryable"],
              ["<512MB", "Memory · 4 chains"],
            ].map(([v, l]) => (
              <div key={l} className="bg-[var(--bg-card)] p-5 text-center">
                <div className="font-mono text-[24px] font-semibold text-[var(--fg-strong)] tracking-tight">{v}</div>
                <div className="text-[10.5px] uppercase tracking-[0.14em] text-[var(--fg-dim)] mt-2 font-medium">{l}</div>
              </div>
            ))}
          </div>

          <P className="mt-6">
            All metrics exported as Prometheus counters and histograms on <Code>/metrics</Code>. Grafana dashboards in <Code>monitoring/dashboards/</Code>.
          </P>
        </Section>

        <Section id="security" n="10" title="Security">
          <Lead>Public on-chain data, so the threat model is operational: stale data from reorgs, dropped blocks, API DoS. Each risk has a mitigation in code.</Lead>

          <Table
            headers={["Risk", "Likelihood", "Mitigation"]}
            rows={[
              ["Chain reorg", "Low (L2)", "Per-chain confirmation lag, reorg detection re-emits affected blocks"],
              ["Dropped block", "Low", "Sequence-gap detection on block_number, HTTP RPC re-fetch on miss"],
              ["Slow ClickHouse write", "Medium", "Kafka absorbs backpressure; processor catches up async"],
              ["RPC key leak", "Low", "Keys in .env only; .env.example ships placeholders"],
              ["API DoS", "Medium", "Rate limit 100 req/min/IP, CORS whitelist, /health for upstream LBs"],
              ["Disk fill", "Low", "ClickHouse TTL auto-expires events past 90 days; alerts at 80% disk"],
            ]}
          />

          <Callout>
            <strong>Idempotent writes.</strong> Every row uses <Code>(chain_id, block_number, log_index)</Code> as a dedup key. ReplacingMergeTree collapses duplicates on merge, so replays are safe.
          </Callout>
        </Section>

        <div className="pt-12 mt-16 border-t border-[var(--border)] flex items-center justify-between">
          <Link href="/" className="text-[13px] text-[var(--fg-muted)] hover:text-[var(--fg-strong)]">
            ← Back to overview
          </Link>
          <a
            href="https://github.com/0xfandom/chainpulse"
            target="_blank"
            rel="noopener noreferrer"
            className="inline-flex items-center gap-1 text-[13px] text-[var(--accent)] hover:text-[var(--accent-hover)]"
          >
            View source on GitHub <ArrowRight className="size-3.5" />
          </a>
        </div>
      </article>
    </div>
  );
}

function Section({ id, n, title, children }: { id: string; n: string; title: string; children: React.ReactNode }) {
  return (
    <section id={id} className="scroll-mt-20 mb-20">
      <a href={`#${id}`} className="group flex items-baseline gap-3 mb-5">
        <span className="font-mono text-[11px] tnum text-[var(--fg-dim)] font-semibold tracking-wider">{n}</span>
        <h2 className="text-[26px] font-semibold tracking-[-0.022em] text-[var(--fg-strong)] leading-tight">
          {title}
        </h2>
        <Hash className="size-3.5 text-[var(--fg-dim)] opacity-0 group-hover:opacity-100 transition-opacity self-center" />
      </a>
      {children}
    </section>
  );
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

function FlowBox({ label, sub }: { label: string; sub?: string }) {
  return (
    <div className="rounded-md border border-[var(--border)] bg-[var(--bg-card)] px-3 py-2 text-center">
      <div className="text-[12px] font-semibold text-[var(--fg-strong)]">{label}</div>
      {sub && <div className="text-[10px] text-[var(--fg-dim)] mt-0.5 font-mono">{sub}</div>}
    </div>
  );
}

function FlowDiagram() {
  const Box = FlowBox;
  return (
    <div className="my-6 rounded-lg border border-[var(--border)] bg-[var(--bg-elevated)]/30 p-6">
      <div className="grid grid-cols-1 sm:grid-cols-5 gap-3 items-center">
        <Box label="3 Chains" sub="WebSocket" />
        <Arrow />
        <Box label="Indexer" sub="Go" />
        <Arrow />
        <Box label="Kafka" sub="raw_events" />
      </div>
      <div className="my-3 ml-[20%] w-px h-6 bg-[var(--border-strong)]" />
      <div className="grid grid-cols-1 sm:grid-cols-5 gap-3 items-center">
        <Box label="Processor" sub="Go" />
        <Arrow />
        <div className="grid grid-cols-2 gap-2">
          <Box label="ClickHouse" sub="analytics" />
          <Box label="Redis" sub="hot cache" />
        </div>
        <Arrow />
        <div className="grid grid-rows-3 gap-2">
          <Box label="REST API" sub=":8080" />
          <Box label="WebSocket" sub="live feed" />
          <Box label="MCP" sub=":3001" />
        </div>
      </div>
      <div className="text-[10px] uppercase tracking-[0.14em] text-[var(--fg-dim)] text-center mt-4 font-mono">
        Pipeline · block to queryable in ~1-2s
      </div>
    </div>
  );
}

function Arrow() {
  return (
    <div className="flex justify-center text-[var(--fg-dim)]">
      <svg width="20" height="12" viewBox="0 0 20 12" fill="none">
        <path d="M0 6 L18 6 M14 2 L18 6 L14 10" stroke="currentColor" strokeWidth="1" />
      </svg>
    </div>
  );
}
