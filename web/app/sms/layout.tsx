"use client";

import { List, Settings2 } from "lucide-react";

import { AuthGate } from "@/components/auth/auth-gate";
import { PrivateTopbar } from "@/components/layout/private-topbar";
import { Sidebar } from "@/components/navigation/sidebar";

type SmsLayoutProps = {
  children: any;
};

const SMS_ITEMS = [
  { href: "/sms", label: "Overview", icon: List },
  { href: "/sms/settings", label: "Settings", icon: Settings2 }
];

export default function SmsLayout({ children }: SmsLayoutProps) {
  return (
    <AuthGate>
      <div className="section-root with-sidebar">
        <Sidebar title="SMS" storageKey="nav.sms.collapsed" items={SMS_ITEMS} />
        <div className="section-main">
          <PrivateTopbar title="SMS" />
          <main className="section-content">{children}</main>
        </div>
      </div>
    </AuthGate>
  );
}
