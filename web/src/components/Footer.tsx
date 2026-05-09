import Link from "next/link";
import { ArrowUp } from "lucide-react";
import { Logo } from "./Logo";
import { GithubIcon } from "./GithubIcon";

export default function Footer() {
  return (
    <footer className="border-t border-[var(--border)] mt-20 pt-16 pb-8 bg-[var(--bg)]">
      <div className="max-w-[1320px] mx-auto px-6 lg:px-12">
        <div className="text-center mb-12">
          <Link href="https://github.com/0xfandom/chainpulse" className="btn-primary" target="_blank" rel="noopener noreferrer">
            View on GitHub
          </Link>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-3 gap-12 pb-12">
          <div>
            <Link href="/" className="flex items-center gap-2.5">
              <Logo size={56} />
              <span className="font-display text-[22px] font-bold tracking-[-0.01em]">ChainPulse</span>
            </Link>
            <p className="text-[13px] text-[var(--fg-muted)] mt-4 max-w-xs leading-relaxed">
              Open-source multi-chain DeFi indexer with sub-block latency.
            </p>
          </div>

          <div>
            <div className="text-[11px] uppercase tracking-[0.16em] text-[var(--fg-muted)] font-mono font-semibold mb-4">Quick Links</div>
            <ul className="space-y-2.5 text-[13.5px] font-medium">
              <li><Link href="/" className="hover:text-[var(--accent)]">Home</Link></li>
              <li><Link href="/whales" className="hover:text-[var(--accent)]">Live Feed</Link></li>
              <li><Link href="/wallet" className="hover:text-[var(--accent)]">Wallet Lookup</Link></li>
              <li><Link href="/docs" className="hover:text-[var(--accent)]">Docs</Link></li>
            </ul>
          </div>

          <div>
            <div className="text-[11px] uppercase tracking-[0.16em] text-[var(--fg-muted)] font-mono font-semibold mb-4">Contact</div>
            <a
              href="https://github.com/0xfandom/chainpulse"
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex items-center gap-2 text-[13.5px] font-bold text-[var(--fg-strong)] hover:text-[var(--accent)] transition-colors"
            >
              <GithubIcon size={20} />
              GitHub
            </a>
          </div>
        </div>

        <div className="border-t border-[var(--border)] pt-6 flex items-center justify-between flex-wrap gap-4 text-[11.5px] text-[var(--fg-muted)] font-mono">
          <div className="uppercase tracking-[0.12em]">© 2026 ChainPulse</div>
          <div className="flex items-center gap-3 uppercase tracking-[0.12em]">
            <span>v0.1.0</span>
            <a href="#top" className="inline-flex items-center gap-1.5 hover:text-[var(--fg-strong)]">
              Back to top <ArrowUp className="size-3" />
            </a>
          </div>
        </div>

      </div>
    </footer>
  );
}
