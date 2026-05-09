import { NextRequest, NextResponse } from "next/server";
import { chainSplit, eventsPerMinute, protocolStats } from "@/lib/ch";

export const dynamic = "force-dynamic";

export async function GET(req: NextRequest) {
  const minutes = parseInt(req.nextUrl.searchParams.get("minutes") ?? "60", 10);
  try {
    const [chains, protocols, series] = await Promise.all([
      chainSplit(minutes),
      protocolStats(minutes),
      eventsPerMinute(minutes),
    ]);
    return NextResponse.json({ chains, protocols, series });
  } catch (e) {
    return NextResponse.json({ error: String(e) }, { status: 500 });
  }
}
