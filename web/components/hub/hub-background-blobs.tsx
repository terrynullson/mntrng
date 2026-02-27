"use client";

import { useEffect, useRef, useState } from "react";

const DARK_PALETTE = [
  "rgba(255, 77, 216, 0.22)",
  "rgba(255, 122, 182, 0.18)",
  "rgba(242, 181, 255, 0.14)",
  "rgba(86, 224, 255, 0.14)",
  "rgba(160, 174, 255, 0.1)"
];

const LIGHT_PALETTE = [
  "rgba(125, 211, 252, 0.28)",
  "rgba(244, 179, 230, 0.18)",
  "rgba(125, 211, 252, 0.2)",
  "rgba(244, 179, 230, 0.12)",
  "rgba(125, 211, 252, 0.18)"
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

function createBlobs(isDark: boolean): BlobState[] {
  const palette = isDark ? DARK_PALETTE : LIGHT_PALETTE;
  const now = performance.now();
  return SPREAD_POSITIONS.map(([sx, sy], i) => ({
    size: randomBetween(420, 680),
    color: palette[i]!,
    fromX: sx,
    fromY: sy,
    toX: randomBetween(15, 82),
    toY: randomBetween(15, 78),
    startTime: now,
    duration: randomBetween(1100, 2200)
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
          toX = randomBetween(15, 82);
          toY = randomBetween(15, 78);
          startTime = time;
          duration = randomBetween(1100, 2200);
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
            opacity: theme === "dark" ? 1 : 0.85,
            filter: theme === "dark" ? "blur(200px)" : "blur(160px)",
            zIndex: 0,
            transform: `translate3d(${blob.fromX}%, ${blob.fromY}%, 0)`
          }}
        />
      ))}
    </div>
  );
}
