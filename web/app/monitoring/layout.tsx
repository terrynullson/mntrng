"use client";

import { ActivitySquare, AlertTriangle, Radio } from "lucide-react";
import { AuthGate } from "@/components/auth/auth-gate";
import { PrivateTopbar } from "@/components/layout/private-topbar";
import { Sidebar } from "@/components/navigation/sidebar";

type MonitoringLayoutProps = {
  children: any;
};

const MONITORING_ITEMS = [
  { href: "/monitoring/streams", label: "Streams", icon: Radio },
  { href: "/monitoring/incidents", label: "Incidents", icon: AlertTriangle },
  { href: "/monitoring/analytics", label: "Analytics", icon: ActivitySquare }
];

export default function MonitoringLayout({ children }: MonitoringLayoutProps) {
  return (
    <AuthGate>
      <div className="section-root with-sidebar">
        <Sidebar title="Monitoring" storageKey="nav.monitoring.collapsed" items={MONITORING_ITEMS} />
        <div className="section-main">
          <PrivateTopbar title="Monitoring" />
          <main className="section-content">{children}</main>
        </div>
      </div>
    </AuthGate>
  );
}
