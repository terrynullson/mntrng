"use client";

import type { ReactNode } from "react";

import { AuthProvider } from "@/components/auth/auth-provider";
import { ProtectedShell } from "@/components/layout/protected-shell";
import { ThemeProvider } from "@/components/theme/theme-provider";

export function AppRoot({ children }: { children: ReactNode }) {
  return (
    <ThemeProvider>
      <AuthProvider>
        <ProtectedShell>{children}</ProtectedShell>
      </AuthProvider>
    </ThemeProvider>
  );
}
