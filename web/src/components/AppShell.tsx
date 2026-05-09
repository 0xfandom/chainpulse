"use client";
import { useState } from "react";
import { usePathname } from "next/navigation";
import Sidebar from "./Sidebar";
import Topbar from "./Topbar";
import Footer from "./Footer";

export default function AppShell({ children }: { children: React.ReactNode }) {
  const [open, setOpen] = useState(false);
  const pathname = usePathname();

  return (
    <div className="min-h-screen flex flex-col">
      <Topbar onToggleSidebar={() => setOpen((o) => !o)} sidebarOpen={open} />

      {/* Horizontal nav drawer drops from top */}
      <div
        className={`overflow-hidden transition-[max-height] duration-300 ease-in-out border-b ${
          open ? "max-h-32 border-[var(--border)]" : "max-h-0 border-transparent"
        }`}
      >
        <Sidebar />
      </div>

      <div className="flex-1 flex flex-col min-w-0">
        <main key={pathname} className="flex-1 px-6 lg:px-12 py-10 max-w-[1320px] w-full mx-auto fade-in-up">{children}</main>
        <Footer />
      </div>
    </div>
  );
}
