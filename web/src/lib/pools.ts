// Uniswap V3 pool registry: pool address -> {token0, token1} per chain
// Token0 is the lower-sorted address. amount0/amount1 in swap event correspond.
// Only top pools by volume across major chains. Expand as needed.

export type Pair = { t0: string; t1: string };

export const POOLS: Record<string, Pair> = {
  // Ethereum (1) — chain prefix
  "1:0x88e6a0c2ddd26feeb64f039a2c41296fcb3f5640": { t0: "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", t1: "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2" }, // USDC/WETH 0.05
  "1:0x8ad599c3a0ff1de082011efddc58f1908eb6e6d8": { t0: "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", t1: "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2" }, // USDC/WETH 0.3
  "1:0xc2e9f25be6257c210d7adf0d4cd6e3e881ba25f8": { t0: "0x6b175474e89094c44da98b954eedeac495271d0f", t1: "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2" }, // DAI/WETH
  "1:0x4585fe77225b41b697c938b018e2ac67ac5a20c0": { t0: "0x2260fac5e5542a773aa44fbcfedf7c193bc2c599", t1: "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2" }, // WBTC/WETH
  "1:0xcba27c8e7115b4eb50aa14999bc0866674a96ecb": { t0: "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", t1: "0xdac17f958d2ee523a2206206994597c13d831ec7" }, // USDC/USDT

  // Base (8453)
  "8453:0xd0b53d9277642d899df5c87a3966a349a798f224": { t0: "0x4200000000000000000000000000000000000006", t1: "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913" }, // WETH/USDC
  "8453:0x4e962bb3889bf030368f56810a9c96b83cb3e778": { t0: "0x4200000000000000000000000000000000000006", t1: "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913" }, // WETH/USDC alt fee
  "8453:0x70acdf2ad0bf2402c957154f944c19ef4e1cbae1": { t0: "0x4200000000000000000000000000000000000006", t1: "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913" }, // WETH/USDC alt fee

  // Arbitrum (42161)
  "42161:0xc6962004f452be9203591991d15f6b388e09e8d0": { t0: "0xaf88d065e77c8cc2239327c5edb3a432268e5831", t1: "0x82af49447d8a07e3bd95bd0d56f35241523fbab1" }, // USDC/WETH
  "42161:0xd13040d4fe917ee704158cfcb3338dcd2838b245": { t0: "0xaf88d065e77c8cc2239327c5edb3a432268e5831", t1: "0x82af49447d8a07e3bd95bd0d56f35241523fbab1" }, // USDC/WETH alt
  "42161:0xc31e54c7a869b9fcbecc14363cf510d1c41fa443": { t0: "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8", t1: "0x82af49447d8a07e3bd95bd0d56f35241523fbab1" }, // USDC.e/WETH

  // Optimism (10)
  "10:0x4dc22588ade05c40338a9d95a6da9dcee68bcd60": { t0: "0x0b2c639c533813f4aa9d7837caf62653d097ff85", t1: "0x4200000000000000000000000000000000000006" }, // USDC/WETH
  "10:0x68f5c0a2de713a54991e01858fd27a3832401849": { t0: "0x4200000000000000000000000000000000000006", t1: "0x4200000000000000000000000000000000000042" }, // WETH/OP
  "10:0xb589969d38ce76d3d7aa319de7133bc9755fd840": { t0: "0x0b2c639c533813f4aa9d7837caf62653d097ff85", t1: "0x68f180fcce6836688e9084f035309e29bf0a2095" }, // USDC/WBTC

  // Polygon (137)
  "137:0xa374094527e1673a86de625aa59517c5de346d32": { t0: "0x3c499c542cef5e3811e1192ce70d8cc03d5c3359", t1: "0x0d500b1d8e8ef31e21c99d1db9a6444d3adf1270" }, // USDC/WMATIC
  "137:0x45dda9cb7c25131df268515131f647d726f50608": { t0: "0x2791bca1f2de4661ed88a30c99a7a9449aa84174", t1: "0x0d500b1d8e8ef31e21c99d1db9a6444d3adf1270" }, // USDC.e/WMATIC

  // BSC (56)
  "56:0x7905e0d7442c784ca41822b50dbcc007dafa634a": { t0: "0x55d398326f99059ff775485246999027b3197955", t1: "0xbb4cdb9cbd36b01bd1cbaebf2de08d9173bc095c" }, // USDT/WBNB

  // Avalanche (43114)
  "43114:0xfae3f424a0a47706811521e3ee268f00cfb5c45e": { t0: "0xb97ef9ef8734c71904d8002f8b6bc66dd9c48a6e", t1: "0xb31f66aa3c1e785363f0875a1b74e27b85fd66c7" }, // USDC/WAVAX
  "43114:0x11476e10eb79ddffa6f2585be526d2bd840c3e20": { t0: "0xb97ef9ef8734c71904d8002f8b6bc66dd9c48a6e", t1: "0xb31f66aa3c1e785363f0875a1b74e27b85fd66c7" }, // USDC/WAVAX alt
  "43114:0xc13fd9599f0bedd2da2c307f35bc03ba54a64332": { t0: "0xb97ef9ef8734c71904d8002f8b6bc66dd9c48a6e", t1: "0x49d5c2bdffac6ce2bfdb6640f4f80f226bc10bab" }, // USDC/WETH.e

  // zkSync (324)
  "324:0x3d7264539e6e3f596bb485e3091f3ae02ad01ef8": { t0: "0x1d17cbcf0d6d143135ae902365d2e5e2a16538d4", t1: "0x5aea5775959fbc2557cc8789bc1bf90a239d9a91" }, // USDC/WETH

  // Scroll (534352)
  "534352:0x04566bf83399e4f750728d1ef57008aedda00e71": { t0: "0x06efdbff2a14a7c8e15944d1f4a48f9f95f663a4", t1: "0x5300000000000000000000000000000000000004" }, // USDC/WETH
};

export function lookupPool(chainId: number | string | undefined | null, pool: string | null | undefined): Pair | null {
  if (!pool || chainId === undefined || chainId === null) return null;
  const cid = typeof chainId === "string" ? parseInt(chainId, 10) : chainId;
  const key = `${cid}:${pool.toLowerCase()}`;
  return POOLS[key] ?? null;
}
