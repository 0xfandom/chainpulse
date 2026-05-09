import { NextResponse } from "next/server";
import { promQuery, promRange } from "@/lib/prom";

export const dynamic = "force-dynamic";

export async function GET(req: Request) {
  const url = new URL(req.url);
  const kind = url.searchParams.get("kind") ?? "instant";
  const expr = url.searchParams.get("q");
  if (!expr) return NextResponse.json({ error: "missing q" }, { status: 400 });

  try {
    if (kind === "range") {
      const end = parseInt(url.searchParams.get("end") ?? String(Math.floor(Date.now() / 1000)), 10);
      const start = parseInt(url.searchParams.get("start") ?? String(end - 3600), 10);
      const step = parseInt(url.searchParams.get("step") ?? "30", 10);
      const data = await promRange(expr, start, end, step);
      return NextResponse.json({ data });
    }
    const data = await promQuery(expr);
    return NextResponse.json({ data });
  } catch (e) {
    return NextResponse.json({ error: String(e) }, { status: 500 });
  }
}
