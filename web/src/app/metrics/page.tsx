"use client";
import { useQuery } from "@tanstack/react-query";
import { Area, AreaChart, XAxis, YAxis, Tooltip, ResponsiveContainer, CartesianGrid, Line, LineChart, Legend } from "recharts";
import { Card, Stat, Skeleton, Empty, PageHeader } from "@/components/ui";
import { compactNum } from "@/lib/format";

type PromVector = { metric: Record<string, string>; value: [number, string] };
type PromMatrix = { metric: Record<string, string>; values: [number, string][] };

async function fetchInstant(q: string): Promise<PromVector[]> {
  const r = await fetch(`/api/prom?kind=instant&q=${encodeURIComponent(q)}`);
  if (!r.ok) throw new Error(await r.text());
  const j = await r.json();
  return j.data as PromVector[];
}

async function fetchRange(q: string, minutes: number, step: number): Promise<PromMatrix[]> {
  const end = Math.floor(Date.now() / 1000);
  const start = end - minutes * 60;
  const r = await fetch(`/api/prom?kind=range&q=${encodeURIComponent(q)}&start=${start}&end=${end}&step=${step}`);
  if (!r.ok) throw new Error(await r.text());
  const j = await r.json();
  return j.data as PromMatrix[];
}

function scalar(v: PromVector[] | undefined): number {
  if (!v || !v.length) return 0;
  const n = parseFloat(v[0].value[1]);
  return Number.isFinite(n) ? n : 0;
}

function useInstant(name: string, q: string, refetch = 15_000) {
  return useQuery({
    queryKey: ["prom-instant", name],
    queryFn: () => fetchInstant(q),
    refetchInterval: refetch,
  });
}

function useRange(name: string, q: string, minutes = 60, step = 30, refetch = 15_000) {
  return useQuery({
    queryKey: ["prom-range", name, minutes, step],
    queryFn: () => fetchRange(q, minutes, step),
    refetchInterval: refetch,
  });
}

const SERIES_COLORS = ["#22c55e", "#3b82f6", "#f59e0b", "#a855f7", "#ec4899", "#06b6d4", "#f43f5e", "#84cc16"];

function buildChartData(matrices: PromMatrix[] | undefined, labelKey: (m: Record<string, string>) => string) {
  if (!matrices || !matrices.length) return { rows: [], keys: [] as string[] };
  const tsMap = new Map<number, Record<string, number | string>>();
  const keys: string[] = [];
  for (const m of matrices) {
    const key = labelKey(m.metric) || "value";
    keys.push(key);
    for (const [t, v] of m.values) {
      const row = tsMap.get(t) ?? { t };
      const n = parseFloat(v);
      row[key] = Number.isFinite(n) ? n : 0;
      tsMap.set(t, row);
    }
  }
  const rows = Array.from(tsMap.values()).sort((a, b) => (a.t as number) - (b.t as number)).map((r) => ({
    ...r,
    label: new Date((r.t as number) * 1000).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit", hour12: false }),
  }));
  return { rows, keys: Array.from(new Set(keys)) };
}

export default function MetricsPage() {
  const eventsQ = useInstant("events_24h", "sum(increase(raw_events_produced_total[24h]))");
  const apiQ = useInstant("api_24h", "sum(increase(api_requests_total[24h]))");
  const mcpQ = useInstant("mcp_24h", "sum(increase(mcp_tool_calls_total[24h]))");
  const chainsQ = useInstant("chains", "count(count by (chain) (chain_listener_connected == 1))");

  const latencyQ = useRange("api_p99", `histogram_quantile(0.99, sum by (le, path) (rate(api_request_duration_seconds_bucket[5m])))`, 60, 30);
  const lagQ = useRange("kafka_lag", `sum by (topic, partition) (processor_kafka_lag_messages)`, 60, 30);

  const latency = buildChartData(latencyQ.data, (m) => m.path ?? "all");
  const lag = buildChartData(lagQ.data, (m) => `${m.topic ?? "?"}/${m.partition ?? "?"}`);

  const anyLoading = eventsQ.isLoading || apiQ.isLoading || mcpQ.isLoading || chainsQ.isLoading;
  const anyError = eventsQ.error || apiQ.error || mcpQ.error || chainsQ.error || latencyQ.error || lagQ.error;

  return (
    <div className="space-y-8">
      <PageHeader
        eyebrow="Metrics"
        title={<>Live system pulse.</>}
        description="Indexer, API, and Kafka telemetry. Identical signals to the Grafana dashboard. Refreshes every 15s."
      />

      {anyError && (
        <div className="rounded-2xl border border-rose-200 bg-rose-50 p-4 text-sm text-rose-700 dark:bg-rose-500/10 dark:text-rose-300 dark:border-rose-400/30">
          Prometheus unreachable. Confirm <code className="font-mono">localhost:9090</code> is up.
        </div>
      )}

      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
        <Stat label="Events Indexed · 24h" value={anyLoading ? <Skeleton className="h-8 w-20" /> : compactNum(Math.round(scalar(eventsQ.data)))} hint="raw_events_produced_total" />
        <Stat label="API Requests · 24h" value={anyLoading ? <Skeleton className="h-8 w-20" /> : compactNum(Math.round(scalar(apiQ.data)))} hint="api_requests_total" />
        <Stat label="MCP Tool Calls · 24h" value={anyLoading ? <Skeleton className="h-8 w-20" /> : compactNum(Math.round(scalar(mcpQ.data)))} hint="mcp_tool_calls_total" />
        <Stat label="Chains Indexed" value={anyLoading ? <Skeleton className="h-8 w-12" /> : Math.round(scalar(chainsQ.data)).toString()} hint="chain_listener_connected == 1" />
      </div>

      <Card
        title="API p99 Query Latency"
        subtitle="histogram_quantile(0.99, rate[5m]) · seconds · last 60m"
        action={<span className="text-[10px] font-mono uppercase tracking-[0.16em] text-[var(--fg-muted)] font-semibold">Refreshes 15s</span>}
      >
        <div className="h-72">
          {latencyQ.isLoading ? (
            <Skeleton className="h-full w-full" />
          ) : latency.rows.length === 0 ? (
            <Empty title="No latency samples" hint="API may not have served traffic yet." />
          ) : (
            <ResponsiveContainer width="100%" height="100%">
              <LineChart data={latency.rows} margin={{ left: 0, right: 8, top: 10, bottom: 24 }}>
                <CartesianGrid stroke="var(--chart-grid)" strokeDasharray="2 4" vertical={false} />
                <XAxis dataKey="label" stroke="var(--chart-axis)" fontSize={11} tickLine={false} axisLine={false} dy={4} interval="preserveStartEnd" minTickGap={50} />
                <YAxis stroke="var(--chart-axis)" fontSize={11} tickLine={false} axisLine={false} tickFormatter={(v) => `${(v as number).toFixed(2)}s`} dx={-4} />
                <Tooltip
                  contentStyle={{ background: "var(--chart-tooltip-bg)", border: "1px solid var(--chart-tooltip-border)", borderRadius: 12, fontSize: 12, padding: "8px 12px", boxShadow: "var(--shadow)" }}
                  labelStyle={{ color: "var(--chart-tooltip-fg)", fontSize: 10, marginBottom: 4, textTransform: "uppercase", letterSpacing: 1 }}
                  formatter={(v) => `${(v as number).toFixed(3)}s`}
                />
                <Legend wrapperStyle={{ fontSize: 11, color: "var(--fg-muted)" }} />
                {latency.keys.map((k, i) => (
                  <Line key={k} type="monotone" dataKey={k} stroke={SERIES_COLORS[i % SERIES_COLORS.length]} strokeWidth={2} dot={false} />
                ))}
              </LineChart>
            </ResponsiveContainer>
          )}
        </div>
      </Card>

      <Card
        title="Kafka Consumer Lag"
        subtitle="processor_kafka_lag_messages · per topic/partition · last 60m"
        action={<span className="text-[10px] font-mono uppercase tracking-[0.16em] text-[var(--fg-muted)] font-semibold">Refreshes 15s</span>}
      >
        <div className="h-72">
          {lagQ.isLoading ? (
            <Skeleton className="h-full w-full" />
          ) : lag.rows.length === 0 ? (
            <Empty title="No lag data" hint="Kafka may be idle." />
          ) : (
            <ResponsiveContainer width="100%" height="100%">
              <AreaChart data={lag.rows} margin={{ left: 0, right: 8, top: 10, bottom: 24 }}>
                <defs>
                  {lag.keys.map((k, i) => (
                    <linearGradient key={k} id={`lag-${i}`} x1="0" y1="0" x2="0" y2="1">
                      <stop offset="0%" stopColor={SERIES_COLORS[i % SERIES_COLORS.length]} stopOpacity={0.3} />
                      <stop offset="100%" stopColor={SERIES_COLORS[i % SERIES_COLORS.length]} stopOpacity={0} />
                    </linearGradient>
                  ))}
                </defs>
                <CartesianGrid stroke="var(--chart-grid)" strokeDasharray="2 4" vertical={false} />
                <XAxis dataKey="label" stroke="var(--chart-axis)" fontSize={11} tickLine={false} axisLine={false} dy={4} interval="preserveStartEnd" minTickGap={50} />
                <YAxis stroke="var(--chart-axis)" fontSize={11} tickLine={false} axisLine={false} tickFormatter={(v) => compactNum(v as number)} dx={-4} />
                <Tooltip
                  contentStyle={{ background: "var(--chart-tooltip-bg)", border: "1px solid var(--chart-tooltip-border)", borderRadius: 12, fontSize: 12, padding: "8px 12px", boxShadow: "var(--shadow)" }}
                  labelStyle={{ color: "var(--chart-tooltip-fg)", fontSize: 10, marginBottom: 4, textTransform: "uppercase", letterSpacing: 1 }}
                />
                <Legend wrapperStyle={{ fontSize: 11, color: "var(--fg-muted)" }} />
                {lag.keys.map((k, i) => (
                  <Area key={k} type="monotone" dataKey={k} stroke={SERIES_COLORS[i % SERIES_COLORS.length]} strokeWidth={2} fill={`url(#lag-${i})`} />
                ))}
              </AreaChart>
            </ResponsiveContainer>
          )}
        </div>
      </Card>
    </div>
  );
}
