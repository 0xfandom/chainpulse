"use client";
import { useEffect, useRef, useState } from "react";
import { Search, ArrowRight, Wallet, Hash, Box, Waves, Layers } from "lucide-react";
import { MenuIcon } from "./MenuIcon";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { GithubIcon } from "./GithubIcon";
import { Logo } from "./Logo";
import { ThemeToggle } from "./ThemeToggle";
import { parseIntent, intentToHref } from "@/lib/intent";

type Example = { label: string; query: string; hint: string; icon: React.ComponentType<{ className?: string }> };

const EXAMPLES: { group: string; items: Example[] }[] = [
  {
    group: "Active wallets",
    items: [
      { label: "Top wallet (3 chains)", query: "0x278d858f05b94576c1e6f73285886876ff6ef8d2", hint: "Most events last 24h", icon: Wallet },
      { label: "Uniswap v3 NFT mgr", query: "0xc36442b4a4522e871399cd717abdd847ab11fe88", hint: "Liquidity position contract", icon: Wallet },
    ],
  },
  {
    group: "Whales",
    items: [
      { label: "Whales · last 6h · Ethereum", query: "whales last 6h on ethereum", hint: "Cross-protocol large moves", icon: Waves },
      { label: "Whales · Uniswap · Arbitrum", query: "whales on uniswap on arbitrum", hint: "Protocol + chain filter", icon: Waves },
      { label: "Whales · Polygon", query: "whales on polygon", hint: "Single chain feed", icon: Waves },
    ],
  },
  {
    group: "Block & Tx",
    items: [
      { label: "Latest block · Ethereum", query: "latest block on ethereum", hint: "Newest indexed height", icon: Box },
      { label: "Recent Polygon tx", query: "0x339ced21bf557302d56accd0c80b4ceed57732dd56c24c6e933aaa449fa9639d", hint: "Uniswap swap · decoded + raw", icon: Hash },
      { label: "Block 86571320", query: "86571320 on polygon", hint: "All txs + decoded events", icon: Box },
    ],
  },
  {
    group: "Protocol",
    items: [
      { label: "Uniswap v3 · Arbitrum", query: "uniswap on arbitrum", hint: "Protocol stats", icon: Layers },
      { label: "Aave v3 · Ethereum", query: "aave on ethereum", hint: "Protocol stats", icon: Layers },
      { label: "Curve", query: "curve", hint: "All chains", icon: Layers },
    ],
  },
];

export default function Topbar({ onToggleSidebar, sidebarOpen }: { onToggleSidebar: () => void; sidebarOpen: boolean }) {
  const [utc, setUtc] = useState<string>("");
  const [ist, setIst] = useState<string>("");
  const [addr, setAddr] = useState("");
  const [open, setOpen] = useState(false);
  const router = useRouter();
  const wrapRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    const tick = () => {
      const d = new Date();
      setUtc(d.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit", second: "2-digit", hour12: false, timeZone: "UTC" }));
      setIst(d.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit", second: "2-digit", hour12: false, timeZone: "Asia/Kolkata" }));
    };
    tick();
    const i = setInterval(tick, 1000);
    return () => clearInterval(i);
  }, []);

  useEffect(() => {
    if (!open) return;
    const onDown = (e: MouseEvent) => {
      if (!wrapRef.current?.contains(e.target as Node)) setOpen(false);
    };
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") setOpen(false);
    };
    document.addEventListener("mousedown", onDown);
    document.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("mousedown", onDown);
      document.removeEventListener("keydown", onKey);
    };
  }, [open]);

  const trimmed = addr.trim();
  const intent = trimmed ? parseIntent(trimmed) : null;
  const valid = !!intent && intent.kind !== "unknown";
  const showExamples = open && trimmed.length === 0;

  const onSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!intent) return;
    router.push(intentToHref(intent));
    setAddr("");
    setOpen(false);
  };

  const pickExample = (q: string) => {
    setAddr(q);
    setOpen(false);
    inputRef.current?.focus();
  };

  return (
    <div className="h-20 border-b border-[var(--border)] bg-[var(--bg)] sticky top-0 z-20 flex items-center pl-3 pr-6 lg:pr-12 gap-4 shrink-0">
      <div className="flex items-center gap-4 shrink-0">
        <button
          onClick={onToggleSidebar}
          aria-label={sidebarOpen ? "Close sidebar" : "Open sidebar"}
          className="size-12 rounded-lg flex items-center justify-center text-[var(--fg-strong)] hover:bg-[var(--bg-elevated)] transition-colors shrink-0"
        >
          <MenuIcon size={26} open={sidebarOpen} />
        </button>

        <Link href="/" className="flex items-center gap-2 shrink-0">
          <Logo size={36} />
          <span className="font-display text-[18px] font-extrabold tracking-[-0.02em] text-[var(--fg-strong)] hidden sm:inline">ChainPulse</span>
        </Link>
      </div>

      <div ref={wrapRef} className="relative flex-1 min-w-0 flex justify-center">
        <form onSubmit={onSubmit} className="w-full max-w-[720px]">
          <div className={`relative flex items-center w-full gap-2 rounded-full border bg-[var(--bg-card)] pl-4 pr-1 py-1 transition-all ${
            valid ? "border-[var(--accent)] ring-2 ring-[var(--accent-soft)]" : addr ? "border-rose-300 dark:border-rose-400/50" : "border-[var(--border-strong)] focus-within:border-[var(--fg-strong)]"
          }`}>
            <Search className="size-3.5 text-[var(--fg-muted)] shrink-0" />
            <input
              ref={inputRef}
              value={addr}
              onChange={(e) => setAddr(e.target.value)}
              onFocus={() => setOpen(true)}
              placeholder="Address, tx, block, or 'whales last 6h on base'…"
              className="flex-1 bg-transparent text-[12.5px] focus:outline-none placeholder:text-[var(--fg-dim)] py-1.5 min-w-0"
              spellCheck={false}
            />
            <button
              type="submit"
              disabled={!valid}
              className="inline-flex items-center justify-center size-8 rounded-full bg-[var(--accent)] hover:bg-[var(--accent-hover)] text-white disabled:bg-[var(--bg-elevated)] disabled:text-[var(--fg-dim)] transition-all shrink-0"
              aria-label="Search"
            >
              <ArrowRight className="size-3.5" />
            </button>
          </div>
        </form>

        {showExamples && (
          <div
            role="listbox"
            className="absolute top-full left-0 right-0 mt-2 rounded-2xl border border-[var(--border)] bg-[var(--bg-card)] shadow-[var(--shadow)] overflow-hidden z-30 pop-in"
          >
            <div className="px-4 pt-3 pb-2 text-[10px] uppercase tracking-[0.18em] font-mono font-semibold text-[var(--fg-muted)]">
              Try one of these
            </div>
            <div className="max-h-[60vh] overflow-y-auto pb-2">
              {EXAMPLES.map((g) => (
                <div key={g.group}>
                  <div className="px-4 pt-2 pb-1 text-[10.5px] uppercase tracking-[0.16em] font-mono font-semibold text-[var(--fg-dim)]">
                    {g.group}
                  </div>
                  {g.items.map((ex) => {
                    const Icon = ex.icon;
                    return (
                      <button
                        key={ex.query}
                        type="button"
                        onClick={() => pickExample(ex.query)}
                        className="w-full flex items-center gap-3 px-4 py-2.5 hover:bg-[var(--bg-elevated)] transition-colors text-left"
                      >
                        <span className="size-7 rounded-full border border-[var(--border)] bg-[var(--bg-elevated)] flex items-center justify-center shrink-0">
                          <Icon className="size-3.5 text-[var(--fg-muted)]" />
                        </span>
                        <span className="flex flex-col min-w-0 flex-1">
                          <span className="text-[12.5px] font-semibold text-[var(--fg-strong)] truncate">{ex.label}</span>
                          <span className="text-[11px] text-[var(--fg-muted)] truncate">{ex.hint}</span>
                        </span>
                        <span className="text-[10.5px] font-mono text-[var(--fg-dim)] truncate max-w-[40%] hidden md:block">{ex.query}</span>
                      </button>
                    );
                  })}
                </div>
              ))}
            </div>
            <div className="px-4 py-2 border-t border-[var(--border)] flex items-center justify-between text-[10px] font-mono uppercase tracking-[0.14em] text-[var(--fg-dim)]">
              <span>Esc to close</span>
              <span>Tip: paste any 0x address or tx hash</span>
            </div>
          </div>
        )}
      </div>

      <div className="flex items-center justify-end gap-5 ml-auto shrink-0 whitespace-nowrap">
        <div className="hidden lg:flex items-center gap-3 whitespace-nowrap shrink-0">
          <span className="size-2 rounded-full bg-[var(--accent)] pulse-dot" />
          <span className="font-mono uppercase tracking-[0.14em] text-[12px] font-semibold text-[var(--fg-strong)]">Live</span>
          <span className="font-mono text-[12px] uppercase tracking-[0.12em] text-[var(--fg-muted)]">{utc} UTC · {ist} IST</span>
        </div>
        <a
          href="https://github.com/0xfandom/chainpulse"
          target="_blank"
          rel="noopener noreferrer"
          className="inline-flex items-center gap-2 text-[13.5px] font-bold text-[var(--fg-strong)] hover:text-[var(--accent)] transition-colors px-2"
        >
          <GithubIcon size={18} />
          GitHub
        </a>
        <ThemeToggle />
      </div>
    </div>
  );
}
