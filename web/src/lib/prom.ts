const PROM_URL = process.env.PROMETHEUS_URL ?? "http://localhost:9090";

export type PromVector = { metric: Record<string, string>; value: [number, string] };
export type PromMatrix = { metric: Record<string, string>; values: [number, string][] };

export async function promQuery(expr: string): Promise<PromVector[]> {
  const url = `${PROM_URL}/api/v1/query?query=${encodeURIComponent(expr)}`;
  const r = await fetch(url, { cache: "no-store" });
  if (!r.ok) throw new Error(`prom ${r.status}: ${await r.text()}`);
  const j = await r.json();
  if (j.status !== "success") throw new Error(`prom: ${j.error ?? "unknown"}`);
  return j.data.result as PromVector[];
}

export async function promRange(expr: string, startSec: number, endSec: number, stepSec: number): Promise<PromMatrix[]> {
  const url = `${PROM_URL}/api/v1/query_range?query=${encodeURIComponent(expr)}&start=${startSec}&end=${endSec}&step=${stepSec}`;
  const r = await fetch(url, { cache: "no-store" });
  if (!r.ok) throw new Error(`prom ${r.status}: ${await r.text()}`);
  const j = await r.json();
  if (j.status !== "success") throw new Error(`prom: ${j.error ?? "unknown"}`);
  return j.data.result as PromMatrix[];
}

export function scalarFromVector(v: PromVector[]): number {
  if (!v.length) return 0;
  const n = parseFloat(v[0].value[1]);
  return Number.isFinite(n) ? n : 0;
}
