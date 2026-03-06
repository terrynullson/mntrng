"use client";

import { motion } from "framer-motion";
import type { ReactNode } from "react";
import { SkeletonBlock } from "./skeleton";
import { StatePanel } from "./state-panel";

const motionTransition = { duration: 0.24, ease: "easeOut" as const };
const motionTransitionShort = { duration: 0.2, ease: "easeOut" as const };

type AnimatedStatePanelProps = {
  kind?: "info" | "error";
  children?: ReactNode;
  style?: React.CSSProperties;
};

export function AnimatedStatePanel({
  kind = "info",
  children,
  style
}: AnimatedStatePanelProps) {
  return (
    <motion.div
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      transition={motionTransition}
      style={style}
    >
      <StatePanel kind={kind}>{children}</StatePanel>
    </motion.div>
  );
}

type AnimatedSkeletonProps = {
  lines?: number;
  style?: React.CSSProperties;
};

export function AnimatedSkeleton({ lines = 6, style }: AnimatedSkeletonProps) {
  return (
    <motion.div
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      transition={motionTransitionShort}
      style={{ marginTop: "12px", ...style }}
    >
      <SkeletonBlock lines={lines} />
    </motion.div>
  );
}
