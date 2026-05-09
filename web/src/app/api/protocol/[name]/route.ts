import { NextRequest, NextResponse } from "next/server";
import { protocolStatsByChain } from "@/lib/ch";

export const dynamic = "force-dynamic";

export async function GET(req: NextRequest, ctx: { params: Promise<{ name: string }> }) {
  const { name } = await ctx.params;
  const sp = req.nextUrl.searchParams;
  const chain = sp.get("chain");
  const hours = parseInt(sp.get("hours") ?? "24", 10);
  try {
    const rows = await protocolStatsByChain(name, chain ? parseInt(chain, 10) : undefined, hours);
    return NextResponse.json({ rows, protocol: name, hours });
  } catch (e) {
    return NextResponse.json({ error: String(e) }, { status: 500 });
  }
}
