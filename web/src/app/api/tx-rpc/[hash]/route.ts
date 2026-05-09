import { NextRequest, NextResponse } from "next/server";
import { fetchTxFromRpc } from "@/lib/rpc";

export const dynamic = "force-dynamic";

export async function GET(_req: NextRequest, ctx: { params: Promise<{ hash: string }> }) {
  const { hash } = await ctx.params;
  if (!/^0x[a-fA-F0-9]{64}$/.test(hash)) {
    return NextResponse.json({ error: "bad tx hash" }, { status: 400 });
  }
  try {
    const data = await fetchTxFromRpc(hash);
    return NextResponse.json(data ?? { found: false });
  } catch (e) {
    return NextResponse.json({ error: String(e) }, { status: 500 });
  }
}
