"use client";
import { ReactNode } from "react";
import { chainColor, chainName } from "@/lib/format";
import { ChainIcon } from "@/components/ChainIcon";

export function Card({ title, subtitle, action, children, className = "" }: { title?: string; subtitle?: string; action?: ReactNode; children: ReactNode; className?: string }) {
  return (
    <div className={`card-surface p-6 ${className}`}>
      {(title || action) && (
        <div className="flex items-start justify-between mb-5">
          <div>
            {title && <h3 className="font-display text-[16px] font-bold text-[var(--fg-strong)] tracking-tight">{title}</h3>}
            {subtitle && <p className="text-[12px] text-[var(--fg-muted)] mt-0.5">{subtitle}</p>}
          </div>
          {action}
        </div>
      )}
      {children}
    </div>
  );
}

export function Stat({ label, value, hint }: { label: string; value: ReactNode; hint?: string }) {
  return (
    <div className="card-surface p-6">
      <div className="text-[10.5px] uppercase tracking-[0.18em] text-[var(--fg-muted)] font-mono font-semibold">{label}</div>
      <div className="num-display text-[32px] mt-3 text-[var(--fg-strong)] leading-[1]">{value}</div>
      {hint && <div className="text-[12px] text-[var(--fg-muted)] mt-2.5">{hint}</div>}
    </div>
  );
}

export function ChainPill({ id }: { id: number | string }) {
  const color = chainColor(id);
  return (
    <span className="inline-flex items-center gap-2 px-3 py-1.5 rounded-full border border-[var(--border)] bg-[var(--bg-elevated)] text-[13px] font-medium">
      <ChainIcon id={id} size={18} fallbackColor={color} />
      <span className="text-[var(--fg-muted)]">{chainName(id)}</span>
    </span>
  );
}

export function ProtocolPill({ p }: { p: string }) {
  return (
    <span className="inline-flex px-2.5 py-1 rounded-full border border-[var(--border)] bg-[var(--bg-elevated)] text-[11px] font-mono text-[var(--fg-muted)]">
      {p}
    </span>
  );
}

export function EventPill({ kind }: { kind: string }) {
  const k = kind.toLowerCase();
  const cls =
    k.includes("swap") ? "bg-blue-50 text-blue-700 border-blue-200 dark:bg-blue-500/10 dark:text-blue-300 dark:border-blue-400/30"
    : k.includes("borrow") || k.includes("withdraw") ? "bg-amber-50 text-amber-700 border-amber-200 dark:bg-amber-500/10 dark:text-amber-300 dark:border-amber-400/30"
    : k.includes("supply") || k.includes("deposit") || k.includes("mint") ? "bg-[var(--accent-soft)] text-[var(--accent-fg)] border-[var(--accent-border)]"
    : k.includes("repay") ? "bg-violet-50 text-violet-700 border-violet-200 dark:bg-violet-500/10 dark:text-violet-300 dark:border-violet-400/30"
    : k.includes("liquidat") ? "bg-rose-50 text-rose-700 border-rose-200 dark:bg-rose-500/10 dark:text-rose-300 dark:border-rose-400/30"
    : "bg-stone-100 text-stone-700 border-stone-300 dark:bg-stone-500/10 dark:text-stone-300 dark:border-stone-400/30";
  return <span className={`inline-flex px-2 py-0.5 rounded-full border text-[10.5px] font-semibold uppercase tracking-wider ${cls}`}>{kind}</span>;
}

export function TagPill({ children, className = "" }: { children: ReactNode; className?: string }) {
  return <span className={`tag-pill ${className}`}>{children}</span>;
}

export function Skeleton({ className = "" }: { className?: string }) {
  return <div className={`skeleton ${className}`} />;
}

export function Empty({ title, hint }: { title: string; hint?: string }) {
  return (
    <div className="py-16 flex flex-col items-center justify-center text-center">
      <div className="size-12 rounded-full border border-[var(--border)] bg-[var(--bg-elevated)] flex items-center justify-center mb-3">
        <span className="size-1.5 rounded-full bg-[var(--fg-dim)]" />
      </div>
      <div className="text-sm text-[var(--fg-muted)] font-medium">{title}</div>
      {hint && <div className="text-xs text-[var(--fg-dim)] mt-1">{hint}</div>}
    </div>
  );
}

export function PageHeader({ eyebrow, title, description, action }: { eyebrow?: string; title: ReactNode; description?: string; action?: ReactNode }) {
  return (
    <div className="flex items-end justify-between flex-wrap gap-4 pb-8 border-b border-[var(--border)] mb-8">
      <div>
        {eyebrow && <TagPill className="mb-4">{eyebrow}</TagPill>}
        <h1 className="font-display text-[42px] font-bold tracking-[-0.025em] text-[var(--fg-strong)] leading-[1.05]">{title}</h1>
        {description && <p className="text-[14.5px] text-[var(--fg-muted)] mt-3 max-w-xl leading-relaxed">{description}</p>}
      </div>
      {action}
    </div>
  );
}

export function IconBtn({ children, onClick, className = "", "aria-label": ariaLabel }: { children: ReactNode; onClick?: () => void; className?: string; "aria-label"?: string }) {
  return (
    <button onClick={onClick} aria-label={ariaLabel} className={`btn-icon ${className}`}>
      {children}
    </button>
  );
}
