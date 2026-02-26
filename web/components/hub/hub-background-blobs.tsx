"use client";

import { useEffect, useState } from "react";

type BlobSpec = {
  size: number;
  duration: number;
  name: string;
  keyframes: string;
  left: number;
  top: number;
};

function randomBetween(min: number, max: number): number {
  return min + Math.random() * (max - min);
}

function generateKeyframes(name: string): string {
  const steps = [0, 20, 40, 60, 80, 100];
  const keyframeBlocks = steps
    .map((pct) => {
      const tx = randomBetween(-30, 30);
      const ty = randomBetween(-24, 24);
      const scale = randomBetween(0.9, 1.1);
      return `  ${pct}% { transform: translate3d(${tx}%, ${ty}%, 0) scale(${scale.toFixed(2)}); opacity: ${randomBetween(0.78, 1).toFixed(2)}; }`;
    })
    .join("\n");
  return `@keyframes ${name} { ${keyframeBlocks} }`;
}

function generateBlobConfig(): { keyframes: string; blobs: BlobSpec[] } {
  const keyframeNames = ["hub-orb-1", "hub-orb-2", "hub-orb-3", "hub-orb-4", "hub-orb-5"];
  const blobs: BlobSpec[] = keyframeNames.map((name) => ({
    size: Math.round(randomBetween(220, 520)),
    duration: randomBetween(9, 16),
    name,
    keyframes: generateKeyframes(name),
    left: randomBetween(0, 80),
    top: randomBetween(0, 75)
  }));
  const keyframes = blobs.map((b) => b.keyframes).join("\n");
  return { keyframes, blobs };
}

const ORB_COLORS = [
  "rgba(99, 102, 241, 0.35)",
  "rgba(79, 70, 229, 0.32)",
  "rgba(67, 56, 202, 0.3)",
  "rgba(129, 140, 248, 0.28)",
  "rgba(99, 102, 241, 0.25)"
];

export function HubBackgroundBlobs() {
  const [config, setConfig] = useState<ReturnType<typeof generateBlobConfig> | null>(null);

  useEffect(() => {
    setConfig(generateBlobConfig());
  }, []);

  if (!config) return null;

  return (
    <>
      <style dangerouslySetInnerHTML={{ __html: config.keyframes }} />
      <div className="hub-bg-blobs" aria-hidden>
        {config.blobs.map((blob, i) => (
          <div
            key={blob.name}
            className="hub-bg-blob"
            style={{
              width: blob.size,
              height: blob.size,
              left: `${blob.left}%`,
              top: `${blob.top}%`,
              background: `radial-gradient(circle at 50% 50%, ${ORB_COLORS[i] ?? ORB_COLORS[0]}, transparent 58%)`,
              animation: `${blob.name} ${blob.duration.toFixed(1)}s ease-in-out infinite`
            }}
          />
        ))}
      </div>
    </>
  );
}
