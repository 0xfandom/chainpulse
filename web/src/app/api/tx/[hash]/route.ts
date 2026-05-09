import { NextRequest, NextResponse } from "next/server";
import { txByHash, transfersByTxHash } from "@/lib/ch";

export const dynamic = "force-dynamic";

export async function GET(_req: NextRequest, ctx: { params: Promise<{ hash: string }> }) {
  const { hash } = await ctx.params;
  if (!/^0x[a-fA-F0-9]{64}$/.test(hash)) {
    return NextResponse.json({ error: "bad tx hash" }, { status: 400 });
  }
  try {
    const [rows, transfers] = await Promise.all([txByHash(hash), transfersByTxHash(hash)]);
    return NextResponse.json({ rows, transfers, hash });
  } catch (e) {
    return NextResponse.json({ error: String(e) }, { status: 500 });
  }
}
