import { CHAIN_NAMES } from "./format";

export type Intent =
  | { kind: "wallet"; addr: string; subtool?: "positions" | "balances" | "history"; chainId?: number; protocol?: string }
  | { kind: "tx"; hash: string }
  | { kind: "token"; addr: string; chainId?: number }
  | { kind: "block"; chainId?: number; n: number }
  | { kind: "protocol"; name: string; chainId?: number }
  | { kind: "whales"; hours?: number; chainId?: number; protocol?: string }
  | { kind: "latest_block"; chainId?: number }
  | { kind: "search"; q: string }
  | { kind: "unknown"; q: string };

const PROTOCOLS = ["erc20", "uniswap_v3", "aave_v3", "compound_v3", "lido", "curve"];

const CHAIN_ALIASES: Record<string, number> = {
  ethereum: 1,
  eth: 1,
  mainnet: 1,
  polygon: 137,
  matic: 137,
  arbitrum: 42161,
  arb: 42161,
};

const ADDR_RE = /0x[a-fA-F0-9]{40}/;
const TX_RE = /0x[a-fA-F0-9]{64}/;

function findChain(q: string): number | undefined {
  const lower = q.toLowerCase();
  for (const [alias, id] of Object.entries(CHAIN_ALIASES)) {
    const re = new RegExp(`\\b${alias}\\b`);
    if (re.test(lower)) return id;
  }
  return undefined;
}

function findProtocol(q: string): string | undefined {
  const lower = q.toLowerCase().replace(/\s+/g, "_");
  for (const p of PROTOCOLS) {
    if (lower.includes(p)) return p;
  }
  if (/\baave\b/i.test(q)) return "aave_v3";
  if (/\buniswap\b/i.test(q)) return "uniswap_v3";
  if (/\bcompound\b/i.test(q)) return "compound_v3";
  if (/\blido\b/i.test(q)) return "lido";
  if (/\bcurve\b/i.test(q)) return "curve";
  return undefined;
}

function findHours(q: string): number | undefined {
  const m = q.match(/(\d+)\s*(?:h|hr|hrs|hour|hours)\b/i);
  if (m) return parseInt(m[1], 10);
  if (/\b(?:24h|day|today)\b/i.test(q)) return 24;
  if (/\b(?:week|7\s*days?)\b/i.test(q)) return 168;
  return undefined;
}

function findBlockNum(q: string): number | undefined {
  const cleaned = q.replace(/[,_](?=\d)/g, "").replace(/\s+(?=\d)/g, " ");
  const compact = cleaned.replace(/[,_\s]/g, "");
  const m = cleaned.match(/\b(?:block|height)\s*(?:number\s*)?#?\s*(\d{5,})\b/i);
  if (m) return parseInt(m[1], 10);
  if (/^\d{5,}$/.test(compact)) return parseInt(compact, 10);
  const lead = cleaned.trim().match(/^(\d{5,})\b/);
  if (lead) return parseInt(lead[1], 10);
  return undefined;
}

export function parseIntent(rawQuery: string): Intent {
  const q = rawQuery.trim();
  if (!q) return { kind: "unknown", q };

  // Pure 64-hex tx hash
  const txMatch = q.match(TX_RE);
  if (txMatch && q.length < 80 && !ADDR_RE.test(q.replace(txMatch[0], ""))) {
    return { kind: "tx", hash: txMatch[0] };
  }

  // Address present
  const addrMatch = q.match(ADDR_RE);
  const lower = q.toLowerCase();
  const chainId = findChain(q);
  const protocol = findProtocol(q);

  if (addrMatch) {
    const addr = addrMatch[0];
    // Token-specific
    if (/\b(?:token|erc.?20|contract)\b/i.test(q) && /transfer/i.test(q)) {
      return { kind: "token", addr, chainId };
    }
    // Wallet sub-intents
    let subtool: "positions" | "balances" | "history" | undefined;
    if (/\bposition/i.test(q)) subtool = "positions";
    else if (/\bbalance/i.test(q)) subtool = "balances";
    else if (/\b(?:history|activity|events?|tx|txs|transactions?)\b/i.test(q)) subtool = "history";

    return { kind: "wallet", addr, subtool, chainId, protocol };
  }

  // Block lookup
  const bn = findBlockNum(q);
  if (bn !== undefined) {
    return { kind: "block", chainId, n: bn };
  }
  if (/\blatest\s+block/i.test(q)) {
    return { kind: "latest_block", chainId };
  }

  // Whale activity
  if (/\bwhale|large transfer|big swap/i.test(q)) {
    return { kind: "whales", hours: findHours(q), chainId, protocol };
  }

  // Protocol stats
  if (/\b(?:tvl|volume|stats|protocol)\b/i.test(q) && protocol) {
    return { kind: "protocol", name: protocol, chainId };
  }
  if (protocol && !addrMatch) {
    return { kind: "protocol", name: protocol, chainId };
  }

  // Latest block fallback for "block" with chain
  if (/\bblock/i.test(q) && chainId !== undefined) {
    return { kind: "latest_block", chainId };
  }

  return { kind: "search", q };
}

export function intentToHref(i: Intent): string {
  switch (i.kind) {
    case "wallet": {
      const params = new URLSearchParams();
      if (i.subtool) params.set("view", i.subtool);
      if (i.chainId !== undefined) params.set("chain", String(i.chainId));
      if (i.protocol) params.set("protocol", i.protocol);
      const qs = params.toString();
      return `/wallet/${i.addr}${qs ? `?${qs}` : ""}`;
    }
    case "tx":
      return `/tx/${i.hash}`;
    case "token": {
      const params = new URLSearchParams();
      if (i.chainId !== undefined) params.set("chain", String(i.chainId));
      const qs = params.toString();
      return `/token/${i.addr}${qs ? `?${qs}` : ""}`;
    }
    case "block":
      return i.chainId !== undefined ? `/block/${i.chainId}/${i.n}` : `/block/lookup?n=${i.n}`;
    case "protocol": {
      const params = new URLSearchParams();
      if (i.chainId !== undefined) params.set("chain", String(i.chainId));
      const qs = params.toString();
      return `/protocol/${i.name}${qs ? `?${qs}` : ""}`;
    }
    case "whales": {
      const params = new URLSearchParams();
      if (i.hours) params.set("hours", String(i.hours));
      if (i.chainId !== undefined) params.set("chain", String(i.chainId));
      if (i.protocol) params.set("protocol", i.protocol);
      const qs = params.toString();
      return `/whales${qs ? `?${qs}` : ""}`;
    }
    case "latest_block":
      return `/search?q=${encodeURIComponent(`latest block${i.chainId ? ` on ${CHAIN_NAMES[i.chainId] ?? i.chainId}` : ""}`)}`;
    case "search":
      return `/search?q=${encodeURIComponent(i.q)}`;
    case "unknown":
      return `/search?q=${encodeURIComponent(i.q)}`;
  }
}

export function describeIntent(i: Intent): string {
  switch (i.kind) {
    case "wallet": {
      const sub = i.subtool ? ` ${i.subtool}` : "";
      const chain = i.chainId !== undefined ? ` on ${CHAIN_NAMES[i.chainId] ?? `chain ${i.chainId}`}` : "";
      const proto = i.protocol ? ` (${i.protocol})` : "";
      return `Wallet${sub}${chain}${proto}: ${i.addr.slice(0, 10)}…`;
    }
    case "tx":
      return `Transaction: ${i.hash.slice(0, 10)}…`;
    case "token":
      return `Token transfers: ${i.addr.slice(0, 10)}…`;
    case "block":
      return `Block #${i.n}${i.chainId !== undefined ? ` on ${CHAIN_NAMES[i.chainId] ?? `chain ${i.chainId}`}` : " (any chain)"}`;
    case "protocol":
      return `Protocol stats: ${i.name}${i.chainId !== undefined ? ` on ${CHAIN_NAMES[i.chainId]}` : ""}`;
    case "whales":
      return `Whale activity${i.hours ? ` last ${i.hours}h` : ""}${i.chainId !== undefined ? ` on ${CHAIN_NAMES[i.chainId]}` : ""}`;
    case "latest_block":
      return `Latest block${i.chainId !== undefined ? ` on ${CHAIN_NAMES[i.chainId]}` : " (all chains)"}`;
    case "search":
      return `Search: "${i.q}"`;
    case "unknown":
      return "Unknown query";
  }
}
