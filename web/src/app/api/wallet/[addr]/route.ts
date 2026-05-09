import { NextRequest, NextResponse } from "next/server";
import { walletActivity, walletTransfers } from "@/lib/ch";

export const dynamic = "force-dynamic";

export async function GET(_req: NextRequest, ctx: { params: Promise<{ addr: string }> }) {
  const { addr } = await ctx.params;
  if (!/^0x[a-fA-F0-9]{40}$/.test(addr)) {
    return NextResponse.json({ error: "bad address" }, { status: 400 });
  }
  try {
    const [events, transfers] = await Promise.all([walletActivity(addr), walletTransfers(addr)]);
    return NextResponse.json({ events, transfers });
  } catch (e) {
    return NextResponse.json({ error: String(e) }, { status: 500 });
  }
}
