"use client";

import Link from "next/link";
import { Home } from "lucide-react";
import { AuthGate } from "@/components/auth/auth-gate";
import { PrivateTopbar } from "@/components/layout/private-topbar";

type WatchLayoutProps = {
  children: any;
};

export default function WatchLayout({ children }: WatchLayoutProps) {
  return (
    <AuthGate>
      <div className="section-root no-sidebar">
        <PrivateTopbar title="Watch" />
        <div className="watch-back-row">
          <Link
            href="/hub"
            className="icon-button"
            aria-label="Вернуться в Hub"
            title="Вернуться в Hub"
          >
            <Home size={16} />
          </Link>
        </div>
        <main className="section-content">{children}</main>
      </div>
    </AuthGate>
  );
}
