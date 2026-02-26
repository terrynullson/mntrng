"use client";

import {
  AnimatedGradientBackground as AnimatedGradientBackgroundSource
} from "@/components/ui/AnimatedGradientBackground";
import { ThemeToggleButton } from "@/components/theme/theme-toggle-button";

/// eslint-disable-next-line @typescript-eslint/no-explicit-any -- совместимость react-типов (shim и @types/react)
const AnimatedGradientBackground = AnimatedGradientBackgroundSource as any;

type AuthShellProps = {
  children: any;
};

export function AuthShell({ children }: AuthShellProps) {
  return (
    <div className="public-root">
      <div className="public-theme-toggle">
        <ThemeToggleButton />
      </div>
      <AnimatedGradientBackground />
      {children}
    </div>
  );
}
