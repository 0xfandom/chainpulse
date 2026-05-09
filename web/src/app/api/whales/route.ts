import { NextRequest, NextResponse } from "next/server";
import { recentWhales } from "@/lib/ch";

export const dynamic = "force-dynamic";

export async function GET(req: NextRequest) {
  const sp = req.nextUrl.searchParams;
  const chain = sp.get("chain");
  const protocol = sp.get("protocol");
  const minutes = parseInt(sp.get("minutes") ?? "60", 10);
  const limit = Math.min(parseInt(sp.get("limit") ?? "100", 10), 500);
  try {
    const rows = await recentWhales({
      chainId: chain ? parseInt(chain, 10) : undefined,
      protocol: protocol ?? undefined,
      minutes,
      limit,
    });
    return NextResponse.json({ rows });
  } catch (e) {
    return NextResponse.json({ error: String(e) }, { status: 500 });
  }
}
