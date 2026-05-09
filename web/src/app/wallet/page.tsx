"use client";
import { useRouter } from "next/navigation";
import { useEffect, useRef, useState } from "react";
import { Search, ArrowRight, Wallet } from "lucide-react";
import { TagPill } from "@/components/ui";

type Example = { label: string; addr: string; hint: string };

const EXAMPLES: Example[] = [
  { label: "Top indexed wallet", addr: "0x278d858f05b94576c1e6f73285886876ff6ef8d2", hint: "5k+ events, 3 chains last 24h" },
  { label: "Uniswap V3 NFT manager", addr: "0xc36442b4a4522e871399cd717abdd847ab11fe88", hint: "LP positions contract" },
  { label: "Uniswap V2 router", addr: "0xe592427a0aece92de3edee1f18e0157c05861564", hint: "Frequent swap routes" },
  { label: "Active multi-chain", addr: "0x6747bcaf9bd5a5f0758cbe08903490e45ddfacb5", hint: "Cross-chain DeFi flows" },
];

export default function WalletLanding() {
  const [addr, setAddr] = useState("");
  const [open, setOpen] = useState(false);
  const router = useRouter();
  const wrapRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);
  const valid = /^0x[a-fA-F0-9]{40}$/.test(addr.trim());

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

  const showExamples = open && addr.trim().length === 0;

  const pickExample = (a: string) => {
    setOpen(false);
    router.push(`/wallet/${a}`);
  };

  return (
    <div className="max-w-2xl mx-auto pt-12 lg:pt-16">
      <div className="text-center">
        <TagPill className="mb-6">Wallet Lookup</TagPill>
        <h1 className="font-display text-[44px] lg:text-[56px] font-extrabold tracking-[-0.03em] leading-[1.05] text-[var(--fg-strong)]">
          Track any wallet,<br />every chain.
        </h1>
        <p className="text-[15px] text-[var(--fg-muted)] mt-5 max-w-md mx-auto leading-relaxed">
          Paste an EVM address. See its DeFi activity, token flows, and protocol footprint across every chain ChainPulse indexes.
        </p>
      </div>

      <div ref={wrapRef} className="relative mt-10">
        <form
          onSubmit={(e) => {
            e.preventDefault();
            if (valid) router.push(`/wallet/${addr.trim()}`);
          }}
        >
          <div className={`flex items-center gap-2 rounded-full border bg-[var(--bg-card)] pl-5 pr-2 py-2 transition-all ${valid ? "border-[var(--accent)] ring-2 ring-[var(--accent-soft)]" : "border-[var(--border-strong)] focus-within:border-[var(--fg-strong)]"}`}>
            <Search className="size-4 text-[var(--fg-muted)] shrink-0" />
            <input
              ref={inputRef}
              value={addr}
              onChange={(e) => setAddr(e.target.value)}
              onFocus={() => setOpen(true)}
              placeholder="0x..."
              className="flex-1 bg-transparent font-mono text-[14px] focus:outline-none placeholder:text-[var(--fg-dim)] py-2"
              spellCheck={false}
              autoFocus
            />
            <button type="submit" disabled={!valid} className="btn-primary">
              Look Up <ArrowRight className="size-3.5" />
            </button>
          </div>
          {addr && !valid && <p className="text-[12px] text-rose-600 dark:text-rose-300 mt-3 px-2">Not a valid 0x-address (40 hex chars).</p>}
        </form>

        {showExamples && (
          <div
            role="listbox"
            className="absolute top-full left-0 right-0 mt-2 rounded-2xl border border-[var(--border)] bg-[var(--bg-card)] shadow-[var(--shadow)] overflow-hidden z-30 pop-in"
          >
            <div className="px-4 pt-3 pb-2 text-[10px] uppercase tracking-[0.18em] font-mono font-semibold text-[var(--fg-muted)]">
              Try a known wallet
            </div>
            <div className="max-h-[60vh] overflow-y-auto pb-2">
              {EXAMPLES.map((ex) => (
                <button
                  key={ex.addr}
                  type="button"
                  onClick={() => pickExample(ex.addr)}
                  className="w-full flex items-center gap-3 px-4 py-2.5 hover:bg-[var(--bg-elevated)] transition-colors text-left"
                >
                  <span className="size-7 rounded-full border border-[var(--border)] bg-[var(--bg-elevated)] flex items-center justify-center shrink-0">
                    <Wallet className="size-3.5 text-[var(--fg-muted)]" />
                  </span>
                  <span className="flex flex-col min-w-0 flex-1">
                    <span className="text-[13px] font-semibold text-[var(--fg-strong)] truncate">{ex.label}</span>
                    <span className="text-[11px] text-[var(--fg-muted)] truncate">{ex.hint}</span>
                  </span>
                  <span className="text-[10.5px] font-mono text-[var(--fg-dim)] truncate hidden md:block">
                    {ex.addr.slice(0, 6)}…{ex.addr.slice(-4)}
                  </span>
                </button>
              ))}
            </div>
            <div className="px-4 py-2 border-t border-[var(--border)] flex items-center justify-between text-[10px] font-mono uppercase tracking-[0.14em] text-[var(--fg-dim)]">
              <span>Esc to close</span>
              <span>Click jumps straight to wallet</span>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
