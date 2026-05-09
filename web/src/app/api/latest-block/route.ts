import { NextRequest, NextResponse } from "next/server";
import { latestBlocks } from "@/lib/ch";

export const dynamic = "force-dynamic";

export async function GET(req: NextRequest) {
  const chain = req.nextUrl.searchParams.get("chain");
  try {
    const rows = await latestBlocks(chain ? parseInt(chain, 10) : undefined);
    return NextResponse.json({ rows });
  } catch (e) {
    return NextResponse.json({ error: String(e) }, { status: 500 });
  }
}
