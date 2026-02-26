"use client";

import { useRouter } from "next/navigation";
import { useEffect } from "react";

import { useAuth } from "@/components/auth/auth-provider";
import { SkeletonBlock } from "@/components/ui/skeleton";

type AuthGateProps = {
  children: any;
};

export function AuthGate({ children }: AuthGateProps) {
  const router = useRouter();
  const { isReady, isAuthenticated } = useAuth();

  useEffect(() => {
    if (!isReady) {
      return;
    }
    if (!isAuthenticated) {
      router.replace("/auth/login");
    }
  }, [isAuthenticated, isReady, router]);

  if (!isReady || !isAuthenticated) {
    return (
      <div className="protected-loading" role="status" aria-live="polite">
        <SkeletonBlock lines={6} className="protected-loading-card" />
      </div>
    );
  }

  return <>{children as any}</>;
}
