"use client";

type NoiseLayerProps = {
  className?: string;
};

export function NoiseLayer({ className }: NoiseLayerProps) {
  return (
    <div
      aria-hidden="true"
      className={["hub-noise-layer", className].filter(Boolean).join(" ")}
      style={{
        pointerEvents: "none",
        backgroundImage:
          "linear-gradient(0deg, rgba(15,23,42,0.12) 1px, transparent 1px), linear-gradient(90deg, rgba(148,163,184,0.08) 1px, transparent 1px)",
        backgroundSize: "3px 3px",
        mixBlendMode: "soft-light"
      }}
    />
  );
}

