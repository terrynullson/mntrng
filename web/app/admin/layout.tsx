"use client";

import { Building2, Settings, ShieldCheck, Users } from "lucide-react";
import { AuthGate } from "@/components/auth/auth-gate";
import { PrivateTopbar } from "@/components/layout/private-topbar";
import { Sidebar } from "@/components/navigation/sidebar";

type AdminLayoutProps = {
  children: any;
};

const ADMIN_ITEMS = [
  { href: "/admin/users", label: "Users", icon: Users },
  { href: "/admin/companies", label: "Companies", icon: Building2 },
  { href: "/admin/requests", label: "Requests", icon: ShieldCheck },
  { href: "/admin/settings", label: "Settings", icon: Settings }
];

export default function AdminLayout({ children }: AdminLayoutProps) {
  return (
    <AuthGate>
      <div className="section-root with-sidebar">
        <Sidebar title="Admin" storageKey="nav.admin.collapsed" items={ADMIN_ITEMS} />
        <div className="section-main">
          <PrivateTopbar title="Admin" />
          <main className="section-content">{children}</main>
        </div>
      </div>
    </AuthGate>
  );
}
