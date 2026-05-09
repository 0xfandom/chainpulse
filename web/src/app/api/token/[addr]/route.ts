import { NextRequest, NextResponse } from "next/server";
import { tokenTransfersForToken } from "@/lib/ch";

export const dynamic = "force-dynamic";

export async function GET(req: NextRequest, ctx: { params: Promise<{ addr: string }> }) {
  const { addr } = await ctx.params;
  if (!/^0x[a-fA-F0-9]{40}$/.test(addr)) {
    return NextResponse.json({ error: "bad address" }, { status: 400 });
  }
  const sp = req.nextUrl.searchParams;
  const chain = sp.get("chain");
  const limit = Math.min(parseInt(sp.get("limit") ?? "100", 10), 500);
  try {
    const rows = await tokenTransfersForToken(addr, chain ? parseInt(chain, 10) : undefined, limit);
    return NextResponse.json({ rows, token: addr });
  } catch (e) {
    return NextResponse.json({ error: String(e) }, { status: 500 });
  }
}
