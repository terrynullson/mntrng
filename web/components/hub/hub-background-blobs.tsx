"use client";

import { useEffect, useRef, useState } from "react";

const DARK_PALETTE = [
  // 2× холодный pink/magenta
  "rgba(244, 114, 182, 0.35)",
  "rgba(236, 72, 153, 0.28)",
  // 1× lilac/violet
  "rgba(196, 181, 253, 0.26)",
  // 2× ice cyan / cold violet
  "rgba(56, 189, 248, 0.28)",
  "rgba(129, 140, 248, 0.24)"
];

const LIGHT_PALETTE = [
  "rgba(191, 219, 254, 0.22)",
  "rgba(221, 214, 254, 0.18)",
  "rgba(186, 230, 253, 0.18)",
  "rgba(221, 239, 253, 0.12)",
  "rgba(224, 231, 255, 0.16)"
];

function randomBetween(min: number, max: number): number {
  return min + Math.random() * (max - min);
}

const SPREAD_POSITIONS: [number, number][] = [
  [8, 12],
  [78, 18],
  [15, 72],
  [82, 75],
  [48, 42]
];

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

function getLayer(index: number): "far" | "near" {
  return index <= 2 ? "far" : "near";
}

function pickTargetPosition(
  index: number,
  blobs: BlobState[]
): { x: number; y: number } {
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

function createBlobs(isDark: boolean): BlobState[] {
  const palette = isDark ? DARK_PALETTE : LIGHT_PALETTE;
  const now = performance.now();
  return SPREAD_POSITIONS.map(([sx, sy], i) => ({
    size: getLayer(i) === "far" ? randomBetween(520, 780) : randomBetween(380, 560),
    color: palette[i]!,
    fromX: sx,
    fromY: sy,
    toX: randomBetween(15, 82),
    toY: randomBetween(15, 78),
    startTime: now,
    duration:
      getLayer(i) === "far" ? randomBetween(5200, 7800) : randomBetween(3200, 5200)
  }));
}

function easeInOutQuad(t: number): number {
  return t < 0.5 ? 2 * t * t : -1 + (4 - 2 * t) * t;
}

export function HubBackgroundBlobs() {
  const blobRefs = useRef<(HTMLDivElement | null)[]>([]);
  const stateRef = useRef<BlobState[] | null>(null);
  const rafRef = useRef<number | null>(null);
  const [theme, setTheme] = useState<"light" | "dark">("dark");

  useEffect(() => {
    const root = document.documentElement;
    const sync = () => {
      const isDark = root.getAttribute("data-theme") === "dark";
      setTheme(isDark ? "dark" : "light");
      stateRef.current = createBlobs(isDark);
    };
    sync();
    const obs = new MutationObserver(sync);
    obs.observe(root, { attributes: true, attributeFilter: ["data-theme"] });
    return () => obs.disconnect();
  }, []);

  useEffect(() => {
    const reduced =
      typeof window !== "undefined" &&
      window.matchMedia("(prefers-reduced-motion: reduce)").matches;
    if (reduced || !stateRef.current) return;

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
          duration =
            getLayer(index) === "far"
              ? randomBetween(5200, 7800)
              : randomBetween(3200, 5200);
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
  }, [theme]);

  const blobs = stateRef.current ?? createBlobs(true);

  return (
    <div className="hub-bg-blobs" aria-hidden>
      {blobs.map((blob, index) => (
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
            opacity:
              theme === "dark"
                ? getLayer(index) === "far"
                  ? 0.42
                  : 0.65
                : getLayer(index) === "far"
                  ? 0.16
                  : 0.22,
            filter:
              theme === "dark"
                ? getLayer(index) === "far"
                  ? "blur(240px)"
                  : "blur(190px)"
                : getLayer(index) === "far"
                  ? "blur(210px)"
                  : "blur(180px)",
            zIndex: 0,
            transform: `translate3d(${blob.fromX}%, ${blob.fromY}%, 0)`
          }}
        />
      ))}
    </div>
  );
}
