"use client";

import { Brain, List } from "lucide-react";

import { AuthGate } from "@/components/auth/auth-gate";
import { PrivateTopbar } from "@/components/layout/private-topbar";
import { Sidebar } from "@/components/navigation/sidebar";

type AiLayoutProps = {
  children: any;
};

const AI_ITEMS = [
  { href: "/ai", label: "Overview", icon: List },
  { href: "/ai/models", label: "Models", icon: Brain }
];

export default function AiLayout({ children }: AiLayoutProps) {
  return (
    <AuthGate>
      <div className="section-root with-sidebar">
        <Sidebar title="AI" storageKey="nav.ai.collapsed" items={AI_ITEMS} />
        <div className="section-main">
          <PrivateTopbar title="AI" />
          <main className="section-content">{children}</main>
        </div>
      </div>
    </AuthGate>
  );
}
