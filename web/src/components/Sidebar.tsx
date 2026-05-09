"use client";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { Activity, Wallet, LayoutDashboard, FileText, LineChart } from "lucide-react";

const items = [
  { href: "/", label: "Overview", icon: LayoutDashboard },
  { href: "/wallet", label: "Wallet", icon: Wallet },
  { href: "/whales", label: "Live Feed", icon: Activity },
  { href: "/metrics", label: "Metrics", icon: LineChart },
  { href: "/docs", label: "Docs", icon: FileText },
];

export default function Sidebar() {
  const path = usePathname();
  return (
    <nav className="bg-[var(--bg)] px-6 lg:px-12 py-5">
      <div className="max-w-[1320px] mx-auto flex items-center justify-center gap-2 flex-wrap">
        {items.map((it) => {
          const active = it.href === "/" ? path === "/" : path.startsWith(it.href);
          const Icon = it.icon;
          return (
            <Link
              key={it.href}
              href={it.href}
              className={`flex items-center gap-2.5 px-5 py-2.5 rounded-full text-[13px] font-medium transition-all ${
                active
                  ? "bg-[var(--fg-strong)] text-[var(--bg)]"
                  : "text-[var(--fg-muted)] hover:text-[var(--fg-strong)] hover:bg-[var(--bg-elevated)]"
              }`}
            >
              <Icon className="size-[15px]" strokeWidth={2} />
              {it.label}
            </Link>
          );
        })}
      </div>
    </nav>
  );
}
