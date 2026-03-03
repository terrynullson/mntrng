"use client";

import { useEffect, useRef, useState } from "react";

type OrbsBackgroundMode = "auth" | "hub";
type OrbsBackgroundIntensity = "calm" | "vivid";

type OrbsBackgroundProps = {
  mode?: OrbsBackgroundMode;
  intensity?: OrbsBackgroundIntensity;
  className?: string;
};

type BlobState = {
  size: number;
  color: string;
  fromX: number;
  fromY: number;
  toX: number;
  toY: number;
  startTime: number;
  duration: number;
};

type Layer = "far" | "near";

const HUB_DARK_PALETTE = [
  // cold pink / magenta
  "rgba(244, 114, 182, 0.35)",
  "rgba(236, 72, 153, 0.28)",
  // lilac / violet
  "rgba(196, 181, 253, 0.26)",
  // ice cyan / cold violet
  "rgba(56, 189, 248, 0.28)",
  "rgba(129, 140, 248, 0.24)"
];

const HUB_LIGHT_PALETTE = [
  "rgba(191, 219, 254, 0.22)",
  "rgba(221, 214, 254, 0.18)",
  "rgba(186, 230, 253, 0.18)",
  "rgba(221, 239, 253, 0.12)",
  "rgba(224, 231, 255, 0.16)"
];

const HUB_SPREAD_POSITIONS: [number, number][] = [
  [8, 12],
  [78, 18],
  [15, 72],
  [82, 75],
  [48, 42]
];

function randomBetween(min: number, max: number): number {
  return min + Math.random() * (max - min);
}

function getLayer(index: number): Layer {
  return index <= 2 ? "far" : "near";
}

function easeInOutQuad(t: number): number {
  return t < 0.5 ? 2 * t * t : -1 + (4 - 2 * t) * t;
}

function pickTargetPosition(index: number, blobs: BlobState[]): { x: number; y: number } {
  const CENTER_X = 50;
  const CENTER_Y = 50;
  const MIN_CENTER_DISTANCE = 18;
  const MIN_BLOB_DISTANCE = 18;

  for (let attempt = 0; attempt < 8; attempt += 1) {
    const candidateX = randomBetween(14, 86);
    const candidateY = randomBetween(12, 88);

    const dxCenter = candidateX - CENTER_X;
    const dyCenter = candidateY - CENTER_Y;
    const centerDistance = Math.hypot(dxCenter, dyCenter);
    if (centerDistance < MIN_CENTER_DISTANCE) continue;

    let spacedEnough = true;
    for (let j = 0; j < blobs.length; j += 1) {
      if (j === index) continue;
      const other = blobs[j]!;
      const otherX = other.toX ?? other.fromX;
      const otherY = other.toY ?? other.fromY;
      const distance = Math.hypot(candidateX - otherX, candidateY - otherY);
      if (distance < MIN_BLOB_DISTANCE) {
        spacedEnough = false;
        break;
      }
    }

    if (spacedEnough) {
      return { x: candidateX, y: candidateY };
    }
  }

  return {
    x: randomBetween(15, 82),
    y: randomBetween(15, 78)
  };
}

function createHubBlobs(isDark: boolean, intensity: OrbsBackgroundIntensity): BlobState[] {
  const palette = isDark ? HUB_DARK_PALETTE : HUB_LIGHT_PALETTE;
  const now = performance.now();

  const farDurationMin = intensity === "calm" ? 5200 : 4200;
  const farDurationMax = intensity === "calm" ? 7800 : 6400;
  const nearDurationMin = intensity === "calm" ? 3200 : 2600;
  const nearDurationMax = intensity === "calm" ? 5200 : 4200;

  return HUB_SPREAD_POSITIONS.map(([sx, sy], index) => {
    const layer = getLayer(index);
    const size =
      layer === "far" ? randomBetween(520, 780) : randomBetween(380, 560);

    return {
      size,
      color: palette[index]!,
      fromX: sx,
      fromY: sy,
      toX: randomBetween(15, 82),
      toY: randomBetween(15, 78),
      startTime: now,
      duration:
        layer === "far"
          ? randomBetween(farDurationMin, farDurationMax)
          : randomBetween(nearDurationMin, nearDurationMax)
    };
  });
}

function shouldReduceMotion(): boolean {
  if (typeof window === "undefined") return false;
  return window.matchMedia("(prefers-reduced-motion: reduce)").matches;
}

export function OrbsBackground({
  mode = "hub",
  intensity = "calm",
  className
}: OrbsBackgroundProps) {
  const blobRefs = useRef<(HTMLDivElement | null)[]>([]);
  const stateRef = useRef<BlobState[] | null>(null);
  const rafRef = useRef<number | null>(null);
  const [theme, setTheme] = useState<"light" | "dark">("dark");

  useEffect(() => {
    const root = document.documentElement;

    const sync = () => {
      const isDark = root.getAttribute("data-theme") === "dark";
      setTheme(isDark ? "dark" : "light");
      stateRef.current = createHubBlobs(isDark, intensity);
    };

    sync();

    const observer = new MutationObserver(sync);
    observer.observe(root, { attributes: true, attributeFilter: ["data-theme"] });

    return () => observer.disconnect();
  }, [intensity]);

  useEffect(() => {
    if (shouldReduceMotion() || !stateRef.current) return;

    const frame = () => {
      const blobs = stateRef.current;
      if (!blobs) return;

      const time = performance.now();

      stateRef.current = blobs.map((blob, index) => {
        let { fromX, fromY, toX, toY, startTime, duration } = blob;
        let t = (time - startTime) / duration;

        if (t >= 1) {
          fromX = toX;
          fromY = toY;
          const next = pickTargetPosition(index, blobs);
          toX = next.x;
          toY = next.y;
          startTime = time;

          const layer = getLayer(index);
          const farDurationMin = intensity === "calm" ? 5200 : 4200;
          const farDurationMax = intensity === "calm" ? 7800 : 6400;
          const nearDurationMin = intensity === "calm" ? 3200 : 2600;
          const nearDurationMax = intensity === "calm" ? 5200 : 4200;

          duration =
            layer === "far"
              ? randomBetween(farDurationMin, farDurationMax)
              : randomBetween(nearDurationMin, nearDurationMax);

          t = 0;
        }

        const eased = easeInOutQuad(Math.max(0, Math.min(1, t)));
        const x = fromX + (toX - fromX) * eased;
        const y = fromY + (toY - fromY) * eased;

        const el = blobRefs.current?.[index];
        if (el) {
          el.style.transform = `translate3d(${x}%, ${y}%, 0)`;
        }

        return { ...blob, fromX, fromY, toX, toY, startTime, duration };
      });

      rafRef.current = requestAnimationFrame(frame);
    };

    rafRef.current = requestAnimationFrame(frame);

    return () => {
      if (rafRef.current != null) cancelAnimationFrame(rafRef.current);
    };
  }, [intensity, mode, theme]);

  const blobs = stateRef.current ?? createHubBlobs(theme === "dark", intensity);

  const containerClassName =
    mode === "hub" ? "hub-bg-blobs" : "hub-bg-blobs orbs-background-auth";

  return (
    <div
      className={[containerClassName, className].filter(Boolean).join(" ")}
      aria-hidden
    >
      {blobs.map((blob, index) => {
        const layer = getLayer(index);
        const isDark = theme === "dark";

        const opacity = isDark
          ? layer === "far"
            ? 0.42
            : 0.65
          : layer === "far"
            ? 0.16
            : 0.22;

        const blur = isDark
          ? layer === "far"
            ? "blur(240px)"
            : "blur(190px)"
          : layer === "far"
            ? "blur(210px)"
            : "blur(180px)";

        return (
          <div
            key={index}
            ref={(el) => {
              const arr = blobRefs.current ?? [];
              blobRefs.current = arr;
              arr[index] = el;
            }}
            className="hub-bg-blob"
            style={{
              width: blob.size,
              height: blob.size,
              left: 0,
              top: 0,
              marginLeft: -blob.size / 2,
              marginTop: -blob.size / 2,
              background: `radial-gradient(circle, ${blob.color} 0%, transparent 70%)`,
              opacity,
              filter: blur,
              zIndex: 0,
              transform: `translate3d(${blob.fromX}%, ${blob.fromY}%, 0)`
            }}
          />
        );
      })}
    </div>
  );
}

