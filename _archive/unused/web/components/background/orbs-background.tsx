"use client";

type OrbsBackgroundVariant = "auth" | "hub";

type OrbsBackgroundProps = {
  variant?: OrbsBackgroundVariant;
  className?: string;
};

export function OrbsBackground({
  variant = "hub",
  className = ""
}: OrbsBackgroundProps) {
  return (
    <div
      aria-hidden="true"
      className={`orbs-background orbs-background--${variant} ${className}`.trim()}
    >
      <div className="orbs-inner">
        <div className="auth-animated-blob auth-blob-1" />
        <div className="auth-animated-blob auth-blob-2" />
        <div className="auth-animated-blob auth-blob-3" />
        <div className="auth-animated-blob auth-blob-4" />
        <div className="auth-animated-blob auth-blob-5" />

        <svg
          className="orbs-noise-layer"
          xmlns="http://www.w3.org/2000/svg"
          preserveAspectRatio="none"
        >
          <filter id="orbs-noise-filter">
            <feTurbulence
              type="fractalNoise"
              baseFrequency="0.9"
              numOctaves={3}
              stitchTiles="noStitch"
            />
          </filter>
          <rect
            width="100%"
            height="100%"
            filter="url(#orbs-noise-filter)"
          />
        </svg>
      </div>
    </div>
  );
}



