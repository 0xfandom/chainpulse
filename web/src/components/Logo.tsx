export function Logo({ size = 32 }: { size?: number }) {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 32 32"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      aria-label="ChainPulse"
      className="logo-mark group/logo"
    >
      {/* Tile flips with theme */}
      <rect width="32" height="32" rx="9" fill="var(--fg-strong)" />

      {/* Faint baseline guide */}
      <line x1="4" y1="16" x2="28" y2="16" stroke="#22c55e" strokeOpacity="0.18" strokeWidth="1" />

      {/* Animated heartbeat */}
      <path
        d="M4 16 L9 16 L11 11 L13 22 L15 8 L17 24 L19 13 L21 16 L28 16"
        stroke="#22c55e"
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
        fill="none"
        pathLength="100"
        className="logo-pulse"
      />

      {/* Trailing dot riding the line */}
      <circle r="1.5" fill="#22c55e" className="logo-bead">
        <animateMotion
          dur="3s"
          repeatCount="indefinite"
          rotate="auto"
          path="M4 16 L9 16 L11 11 L13 22 L15 8 L17 24 L19 13 L21 16 L28 16"
        />
      </circle>

    </svg>
  );
}
