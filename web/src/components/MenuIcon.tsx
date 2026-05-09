export function MenuIcon({ size = 18, open = false }: { size?: number; open?: boolean }) {
  if (open) {
    return (
      <svg width={size} height={size} viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
        <path d="M5 5 L19 19 M19 5 L5 19" stroke="currentColor" strokeWidth="2" strokeLinecap="round" />
      </svg>
    );
  }
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
      {/* top line */}
      <line x1="4" y1="7" x2="20" y2="7" stroke="currentColor" strokeWidth="2" strokeLinecap="round" />
      {/* middle line — short EKG spike, ChainPulse motif */}
      <path
        d="M4 12 L9 12 L11 9 L13 15 L15 11 L20 12"
        stroke="currentColor"
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
        fill="none"
      />
      {/* bottom line */}
      <line x1="4" y1="17" x2="20" y2="17" stroke="currentColor" strokeWidth="2" strokeLinecap="round" />
      {/* tiny accent dot at right of middle line */}
      <circle cx="20" cy="12" r="1.4" fill="#22c55e" />
    </svg>
  );
}
