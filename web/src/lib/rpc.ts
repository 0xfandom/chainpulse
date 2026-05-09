type Hex = string;

export type RawTx = {
  hash: Hex;
  blockNumber: Hex | null;
  blockHash: Hex | null;
  from: Hex;
  to: Hex | null;
  value: Hex;
  gas: Hex;
  gasPrice: Hex | null;
  maxFeePerGas?: Hex;
  maxPriorityFeePerGas?: Hex;
  nonce: Hex;
  input: Hex;
  type: Hex;
  chainId?: Hex;
};

export type RawReceipt = {
  status: Hex;
  cumulativeGasUsed: Hex;
  gasUsed: Hex;
  effectiveGasPrice?: Hex;
  contractAddress: Hex | null;
  logs: { address: Hex; topics: Hex[]; data: Hex; logIndex: Hex }[];
};

const RPC_URLS: Record<number, string | undefined> = {
  1: process.env.ETH_RPC_URL,
  137: process.env.POLYGON_RPC_URL,
  42161: process.env.ARB_RPC_URL,
};

async function jsonRpc<T>(url: string, method: string, params: unknown[]): Promise<T | null> {
  const r = await fetch(url, {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify({ jsonrpc: "2.0", id: 1, method, params }),
    cache: "no-store",
  });
  if (!r.ok) return null;
  const j = (await r.json()) as { result?: T; error?: unknown };
  if (j.error) return null;
  return (j.result ?? null) as T | null;
}

export async function fetchTxFromRpc(hash: string): Promise<{ chainId: number; tx: RawTx; receipt: RawReceipt | null } | null> {
  for (const [cidStr, url] of Object.entries(RPC_URLS)) {
    if (!url) continue;
    const tx = await jsonRpc<RawTx>(url, "eth_getTransactionByHash", [hash]);
    if (tx && tx.hash) {
      const receipt = await jsonRpc<RawReceipt>(url, "eth_getTransactionReceipt", [hash]);
      return { chainId: parseInt(cidStr, 10), tx, receipt };
    }
  }
  return null;
}

export type RawBlock = {
  number: Hex;
  hash: Hex;
  parentHash: Hex;
  timestamp: Hex;
  miner: Hex;
  gasLimit: Hex;
  gasUsed: Hex;
  size: Hex;
  baseFeePerGas?: Hex;
  transactions: { hash: Hex; from: Hex; to: Hex | null; value: Hex; gas: Hex }[];
};

export async function fetchBlockFromRpc(chainId: number, blockNumber: number): Promise<RawBlock | null> {
  const url = RPC_URLS[chainId];
  if (!url) return null;
  const hex = "0x" + blockNumber.toString(16);
  return jsonRpc<RawBlock>(url, "eth_getBlockByNumber", [hex, true]);
}
