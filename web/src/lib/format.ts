export const CHAIN_NAMES: Record<number, string> = {
  1: "Ethereum",
  137: "Polygon",
  42161: "Arbitrum",
};

export function chainName(id: number | string) {
  const n = typeof id === "string" ? parseInt(id, 10) : id;
  return CHAIN_NAMES[n] ?? `chain ${n}`;
}

export function shortAddr(a: string, n = 4) {
  if (!a) return "";
  if (a.length <= 2 + n * 2) return a;
  return `${a.slice(0, 2 + n)}…${a.slice(-n)}`;
}

type TokenInfo = { sym: string; dec: number };

// chainId:address(lowercase) -> {sym, dec}
const TOKEN_REG: Record<string, TokenInfo> = {
  // Ethereum (1)
  "1:0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48": { sym: "USDC", dec: 6 },
  "1:0xdac17f958d2ee523a2206206994597c13d831ec7": { sym: "USDT", dec: 6 },
  "1:0x6b175474e89094c44da98b954eedeac495271d0f": { sym: "DAI", dec: 18 },
  "1:0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2": { sym: "WETH", dec: 18 },
  "1:0x2260fac5e5542a773aa44fbcfedf7c193bc2c599": { sym: "WBTC", dec: 8 },
  "1:0x514910771af9ca656af840dff83e8264ecf986ca": { sym: "LINK", dec: 18 },
  "1:0x7d1afa7b718fb893db30a3abc0cfc608aacfebb0": { sym: "MATIC", dec: 18 },
  // Base (8453)
  "8453:0x833589fcd6edb6e08f4c7c32d4f71b54bda02913": { sym: "USDC", dec: 6 },
  "8453:0x4200000000000000000000000000000000000006": { sym: "WETH", dec: 18 },
  "8453:0x2ae3f1ec7f1f5012cfeab0185bfc7aa3cf0dec22": { sym: "cbETH", dec: 18 },
  "8453:0x50c5725949a6f0c72e6c4a641f24049a917db0cb": { sym: "DAI", dec: 18 },
  "8453:0x0555e30da8f98308edb960aa94c0db47230d2b9c": { sym: "WBTC", dec: 8 },
  // Arbitrum (42161)
  "42161:0xaf88d065e77c8cc2239327c5edb3a432268e5831": { sym: "USDC", dec: 6 },
  "42161:0xff970a61a04b1ca14834a43f5de4533ebddb5cc8": { sym: "USDC.e", dec: 6 },
  "42161:0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9": { sym: "USDT", dec: 6 },
  "42161:0x82af49447d8a07e3bd95bd0d56f35241523fbab1": { sym: "WETH", dec: 18 },
  "42161:0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f": { sym: "WBTC", dec: 8 },
  "42161:0xda10009cbd5d07dd0cecc66161fc93d7c9000da1": { sym: "DAI", dec: 18 },
  "42161:0x912ce59144191c1204e64559fe8253a0e49e6548": { sym: "ARB", dec: 18 },
  // Optimism (10)
  "10:0x0b2c639c533813f4aa9d7837caf62653d097ff85": { sym: "USDC", dec: 6 },
  "10:0x7f5c764cbc14f9669b88837ca1490cca17c31607": { sym: "USDC.e", dec: 6 },
  "10:0x94b008aa00579c1307b0ef2c499ad98a8ce58e58": { sym: "USDT", dec: 6 },
  "10:0x4200000000000000000000000000000000000006": { sym: "WETH", dec: 18 },
  "10:0x68f180fcce6836688e9084f035309e29bf0a2095": { sym: "WBTC", dec: 8 },
  "10:0xda10009cbd5d07dd0cecc66161fc93d7c9000da1": { sym: "DAI", dec: 18 },
  "10:0x4200000000000000000000000000000000000042": { sym: "OP", dec: 18 },
  // Polygon (137)
  "137:0x3c499c542cef5e3811e1192ce70d8cc03d5c3359": { sym: "USDC", dec: 6 },
  "137:0x2791bca1f2de4661ed88a30c99a7a9449aa84174": { sym: "USDC.e", dec: 6 },
  "137:0xc2132d05d31c914a87c6611c10748aeb04b58e8f": { sym: "USDT", dec: 6 },
  "137:0x0d500b1d8e8ef31e21c99d1db9a6444d3adf1270": { sym: "WMATIC", dec: 18 },
  "137:0x7ceb23fd6bc0add59e62ac25578270cff1b9f619": { sym: "WETH", dec: 18 },
  "137:0x1bfd67037b42cf73acf2047067bd4f2c47d9bfd6": { sym: "WBTC", dec: 8 },
  "137:0x8f3cf7ad23cd3cadbd9735aff958023239c6a063": { sym: "DAI", dec: 18 },
  // BSC (56) — note USDC/USDT on BSC are 18 decimals!
  "56:0x55d398326f99059ff775485246999027b3197955": { sym: "USDT", dec: 18 },
  "56:0x8ac76a51cc950d9822d68b83fe1ad97b32cd580d": { sym: "USDC", dec: 18 },
  "56:0xe9e7cea3dedca5984780bafc599bd69add087d56": { sym: "BUSD", dec: 18 },
  "56:0xbb4cdb9cbd36b01bd1cbaebf2de08d9173bc095c": { sym: "WBNB", dec: 18 },
  "56:0x2170ed0880ac9a755fd29b2688956bd959f933f8": { sym: "WETH", dec: 18 },
  // Avalanche (43114)
  "43114:0xb97ef9ef8734c71904d8002f8b6bc66dd9c48a6e": { sym: "USDC", dec: 6 },
  "43114:0x9702230a8ea53601f5cd2dc00fdbc13d4df4a8c7": { sym: "USDT", dec: 6 },
  "43114:0xb31f66aa3c1e785363f0875a1b74e27b85fd66c7": { sym: "WAVAX", dec: 18 },
  "43114:0x49d5c2bdffac6ce2bfdb6640f4f80f226bc10bab": { sym: "WETH.e", dec: 18 },
  // Scroll (534352)
  "534352:0x06efdbff2a14a7c8e15944d1f4a48f9f95f663a4": { sym: "USDC", dec: 6 },
  "534352:0x5300000000000000000000000000000000000004": { sym: "WETH", dec: 18 },
  // zkSync (324)
  "324:0x1d17cbcf0d6d143135ae902365d2e5e2a16538d4": { sym: "USDC", dec: 6 },
  "324:0x5aea5775959fbc2557cc8789bc1bf90a239d9a91": { sym: "WETH", dec: 18 },
};

export function lookupToken(chainId: number | string | undefined | null, token: string | null | undefined): TokenInfo | null {
  if (!token || chainId === undefined || chainId === null) return null;
  const cid = typeof chainId === "string" ? parseInt(chainId, 10) : chainId;
  const key = `${cid}:${token.toLowerCase()}`;
  return TOKEN_REG[key] ?? null;
}

const SUFFIX = [
  { v: BigInt(10) ** BigInt(18), s: "Qi" },
  { v: BigInt(10) ** BigInt(15), s: "Q" },
  { v: BigInt(10) ** BigInt(12), s: "T" },
  { v: BigInt(10) ** BigInt(9), s: "B" },
  { v: BigInt(10) ** BigInt(6), s: "M" },
  { v: BigInt(10) ** BigInt(3), s: "K" },
];

function compactRaw(big: bigint, neg: boolean): string {
  if (big === BigInt(0)) return "0";
  const match = SUFFIX.find((x) => big >= x.v);
  let formatted: string;
  if (match) {
    const scaled = Number((big * BigInt(100)) / match.v) / 100;
    formatted = `${scaled.toFixed(scaled >= 100 ? 1 : 2)}${match.s}`;
  } else {
    formatted = big.toString();
  }
  return neg ? `-${formatted}` : formatted;
}

function applyDecimals(big: bigint, decimals: number): { whole: bigint; frac: bigint } {
  const base = BigInt(10) ** BigInt(decimals);
  return { whole: big / base, frac: big % base };
}

function fmtScaled(big: bigint, decimals: number, neg: boolean): string {
  if (big === BigInt(0)) return "0";
  const { whole, frac } = applyDecimals(big, decimals);
  const wholeNum = Number(whole);

  let body: string;
  if (whole >= BigInt(1_000_000_000)) {
    body = `${(Number(big * BigInt(100) / (BigInt(10) ** BigInt(decimals + 9))) / 100).toFixed(2)}B`;
  } else if (whole >= BigInt(1_000_000)) {
    body = `${(wholeNum / 1_000_000).toFixed(2)}M`;
  } else if (whole >= BigInt(1_000)) {
    body = `${(wholeNum / 1_000).toFixed(2)}K`;
  } else if (whole > BigInt(0)) {
    const fracStr = frac.toString().padStart(decimals, "0").slice(0, 2).replace(/0+$/, "");
    body = fracStr ? `${whole}.${fracStr}` : whole.toString();
  } else {
    const fracStr = frac.toString().padStart(decimals, "0").slice(0, 6).replace(/0+$/, "");
    if (!fracStr) body = "0";
    else if (fracStr.length <= 4) body = `0.${fracStr}`;
    else body = `0.${fracStr.slice(0, 4)}`;
    if (body === "0" || body === "0.0" || body === "0.00") body = "<0.001";
  }
  return neg ? `-${body}` : body;
}

export function fmtAmount(raw: string | null | undefined, token?: string | null, chainId?: number | string | null): string {
  if (!raw) return "—";
  try {
    const s = raw.trim().replace(/^-/, "");
    const big = BigInt(s);

    const info = lookupToken(chainId, token);
    if (info) {
      return `${fmtScaled(big, info.dec, false)} ${info.sym}`;
    }
    return "—";
  } catch {
    return "—";
  }
}

/**
 * Resolve Uniswap V3 swap amounts via pool registry.
 * Returns formatted "X TOKEN0 → Y TOKEN1" or null if pool unknown.
 */
import { lookupPool } from "./pools";

export function fmtSwap(
  pool: string | null | undefined,
  amount0: string | null | undefined,
  amount1: string | null | undefined,
  chainId: number | string | null | undefined,
): { primary: string; secondary: string } | null {
  const pair = lookupPool(chainId, pool);
  if (!pair) return null;
  const t0 = lookupToken(chainId, pair.t0);
  const t1 = lookupToken(chainId, pair.t1);
  if (!t0 || !t1) return null;

  // Sign convention: in V3 swap, the side leaving the pool (user receives) is negative.
  // Show "in → out" by ordering positive (user gave) → negative (user got).
  const a0 = (amount0 ?? "0").trim();
  const a1 = (amount1 ?? "0").trim();
  const a0Neg = a0.startsWith("-");
  const a1Neg = a1.startsWith("-");
  const a0Abs = a0.replace(/^-/, "");
  const a1Abs = a1.replace(/^-/, "");

  const s0 = `${fmtScaled(BigInt(a0Abs), t0.dec, false)} ${t0.sym}`;
  const s1 = `${fmtScaled(BigInt(a1Abs), t1.dec, false)} ${t1.sym}`;

  // Default order: token0 → token1
  let inSide = s0;
  let outSide = s1;
  if (a0Neg && !a1Neg) { inSide = s1; outSide = s0; }
  return { primary: inSide, secondary: outSide };
}

export function parseChTime(ts: string): Date {
  if (!ts) return new Date(NaN);
  const iso = ts.includes("T") ? (ts.endsWith("Z") || /[+-]\d\d:?\d\d$/.test(ts) ? ts : ts + "Z") : ts.replace(" ", "T") + "Z";
  return new Date(iso);
}

export function relTime(ts: string): string {
  const t = parseChTime(ts).getTime();
  const diff = Date.now() - t;
  const s = Math.max(0, Math.floor(diff / 1000));
  if (s < 60) return `${s}s ago`;
  const m = Math.floor(s / 60);
  if (m < 60) return `${m}m ago`;
  const h = Math.floor(m / 60);
  if (h < 24) return `${h}h ago`;
  return `${Math.floor(h / 24)}d ago`;
}

export function compactNum(n: number): string {
  if (!isFinite(n)) return "0";
  const abs = Math.abs(n);
  if (abs < 1000) return n.toLocaleString();
  if (abs < 1_000_000) return (n / 1000).toFixed(n < 10_000 ? 1 : 0) + "K";
  if (abs < 1_000_000_000) return (n / 1_000_000).toFixed(1) + "M";
  return (n / 1_000_000_000).toFixed(1) + "B";
}

export const CHAIN_COLORS: Record<number, string> = {
  1: "#627eea",
  137: "#8247e5",
  42161: "#28a0f0",
};

export function chainColor(id: number | string) {
  const n = typeof id === "string" ? parseInt(id, 10) : id;
  return CHAIN_COLORS[n] ?? "#71717a";
}

export const PROTOCOL_COLORS: Record<string, string> = {
  uniswap_v3: "#ff007a",
  aave_v3: "#b6509e",
  compound_v3: "#00d395",
  lido: "#00a3ff",
  curve: "#fbbf24",
};
