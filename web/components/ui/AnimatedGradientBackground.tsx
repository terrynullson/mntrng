 "use client";

import { memo } from "react";

type AnimatedGradientBackgroundProps = {
  className?: string;
};

function AnimatedGradientBackgroundComponent({
  className = ""
}: AnimatedGradientBackgroundProps) {
  return (
    <div
      aria-hidden="true"
      className={`auth-animated-bg ${className}`.trim()}
    >
      <div className="auth-animated-blob auth-blob-1" />
      <div className="auth-animated-blob auth-blob-2" />
      <div className="auth-animated-blob auth-blob-3" />
      <div className="auth-animated-blob auth-blob-4" />

      <svg
        className="auth-noise-layer"
        xmlns="http://www.w3.org/2000/svg"
        preserveAspectRatio="none"
      >
        <filter id="auth-noise-filter">
          <feTurbulence
            type="fractalNoise"
            baseFrequency="0.9"
            numOctaves="3"
            stitchTiles="noStitch"
          />
        </filter>
        <rect
          width="100%"
          height="100%"
          filter="url(#auth-noise-filter)"
        />
      </svg>
    </div>
  );
}

export const AnimatedGradientBackground = memo(AnimatedGradientBackgroundComponent);

