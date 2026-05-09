const CHAIN_ICONS: Record<number, string> = {
  1: "/chains/ethereum.png",
  137: "/chains/polygon.png",
  42161: "/chains/arbitrum.png",
};

const CHAIN_ICON_BY_NAME: Record<string, string> = {
  ethereum: "/chains/ethereum.png",
  polygon: "/chains/polygon.png",
  arbitrum: "/chains/arbitrum.png",
};

export function ChainIcon({
  id,
  name,
  size = 16,
  className = "",
  fallbackColor,
}: {
  id?: number | string;
  name?: string;
  size?: number;
  className?: string;
  fallbackColor?: string;
}) {
  const numericId = typeof id === "string" ? parseInt(id, 10) : id;
  const src =
    (numericId !== undefined ? CHAIN_ICONS[numericId] : undefined) ??
    (name ? CHAIN_ICON_BY_NAME[name.toLowerCase()] : undefined);
  if (!src) {
    if (fallbackColor) {
      return (
        <span
          className={`inline-block rounded-full ${className}`}
          style={{ width: size, height: size, background: fallbackColor }}
        />
      );
    }
    return null;
  }
  return (
    <img
      src={src}
      alt=""
      width={size}
      height={size}
      className={`inline-block rounded-full object-contain ${className}`}
      style={{ width: size, height: size }}
    />
  );
}
