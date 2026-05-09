import { NextRequest, NextResponse } from "next/server";
import { blockEvents } from "@/lib/ch";
import { fetchBlockFromRpc } from "@/lib/rpc";

export const dynamic = "force-dynamic";

export async function GET(_req: NextRequest, ctx: { params: Promise<{ chain: string; n: string }> }) {
  const { chain, n } = await ctx.params;
  const chainId = parseInt(chain, 10);
  const blockNumber = parseInt(n, 10);
  if (!chainId || !blockNumber) return NextResponse.json({ error: "bad params" }, { status: 400 });
  try {
    const [rows, block] = await Promise.all([
      blockEvents(chainId, blockNumber),
      fetchBlockFromRpc(chainId, blockNumber),
    ]);
    return NextResponse.json({ rows, block, chainId, blockNumber });
  } catch (e) {
    return NextResponse.json({ error: String(e) }, { status: 500 });
  }
}
