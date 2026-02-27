"use client";

import { useEffect, useRef } from "react";

type BlobLayer = "near" | "far";

type BlobAnimState = {
  layer: BlobLayer;
  size: number;
  color: string;
  // animation state
  fromX: number;
  fromY: number;
  toX: number;
  toY: number;
  startTime: number;
  duration: number;
};

const BLOB_COLORS: string[] = [
  "rgba(99, 102, 241, 0.5)",
  "rgba(79, 70, 229, 0.45)",
  "rgba(56, 189, 248, 0.4)",
  "rgba(59, 130, 246, 0.42)",
  "rgba(99, 102, 241, 0.38)"
];

function randomBetween(min: number, max: number): number {
  return min + Math.random() * (max - min);
}

function createInitialBlobs(now: number): BlobAnimState[] {
  // 3 дальних, 2 ближних
  const layers: BlobLayer[] = ["far", "far", "far", "near", "near"];

  return layers.map((layer, index) => {
    const size = randomBetween(240, 520);
    const x = randomBetween(-20, 20);
    const y = randomBetween(-16, 16);
    const duration =
      layer === "far" ? randomBetween(5500, 9000) : randomBetween(3200, 6500);

    return {
      layer,
      size,
      color: BLOB_COLORS[index % BLOB_COLORS.length],
      fromX: x,
      fromY: y,
      toX: randomBetween(-26, 26),
      toY: randomBetween(-20, 20),
      startTime: now,
      duration
    };
  });
}

function easeInOutQuad(t: number): number {
  return t < 0.5 ? 2 * t * t : -1 + (4 - 2 * t) * t;
}

export function HubBackgroundBlobs() {
  const blobRefs = useRef<(HTMLDivElement | null)[]>([]);
  const animStateRef = useRef<BlobAnimState[] | null>(null);
  const rafRef = useRef<number | null>(null);

  useEffect(() => {
    const now = performance.now();
    animStateRef.current = createInitialBlobs(now);

    const frame = () => {
      const blobs = animStateRef.current;
      if (!blobs) {
        return;
      }

      const time = performance.now();

      animStateRef.current = blobs.map((blob, index) => {
        let { fromX, fromY, toX, toY, startTime, duration } = blob;
        let t = (time - startTime) / duration;

        if (t >= 1) {
          // новая цель каждые ~3–7 секунд
          const nextFromX = toX;
          const nextFromY = toY;
          const nextToX = randomBetween(-26, 26);
          const nextToY = randomBetween(-20, 20);
          const nextDuration =
            blob.layer === "far"
              ? randomBetween(5500, 9000)
              : randomBetween(3200, 6500);

          fromX = nextFromX;
          fromY = nextFromY;
          toX = nextToX;
          toY = nextToY;
          startTime = time;
          duration = nextDuration;
          t = 0;
        }

        const eased = easeInOutQuad(Math.max(0, Math.min(1, t)));
        const x = fromX + (toX - fromX) * eased;
        const y = fromY + (toY - fromY) * eased;

        const el = blobRefs.current![index];
        if (el) {
          const scale = blob.layer === "near" ? 1.04 : 1;
          el.style.transform = `translate3d(${x}%, ${y}%, 0) scale(${scale})`;
        }

        return {
          ...blob,
          fromX,
          fromY,
          toX,
          toY,
          startTime,
          duration
        };
      });

      rafRef.current = requestAnimationFrame(frame);
    };

    rafRef.current = requestAnimationFrame(frame);

    return () => {
      if (rafRef.current != null) {
        cancelAnimationFrame(rafRef.current);
      }
    };
  }, []);

  const blobs = animStateRef.current ?? createInitialBlobs(performance.now());

  return (
    <div className="hub-bg-blobs" aria-hidden>
      {blobs.map((blob, index) => (
        <div
          // ровно 5 элементов
          // eslint-disable-next-line react/no-array-index-key
          key={index}
          ref={(el) => {
            blobRefs.current![index] = el;
          }}
          className="hub-bg-blob"
          style={{
            width: blob.size,
            height: blob.size,
            background: `radial-gradient(circle at 50% 50%, ${blob.color}, transparent 60%)`,
            opacity: blob.layer === "near" ? 0.5 : 0.38,
            filter:
              blob.layer === "near"
                ? "blur(140px)"
                : "blur(180px)",
            zIndex: blob.layer === "near" ? 0 : -1
          }}
        />
      ))}
    </div>
  );
}

