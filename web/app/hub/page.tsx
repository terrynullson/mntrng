"use client";

import { Activity, AlertTriangle, Bot, Gauge, Radar, Settings, Shield, Tv } from "lucide-react";
import { useEffect, useState } from "react";

import { AuthGate } from "@/components/auth/auth-gate";
import { PrivateTopbar } from "@/components/layout/private-topbar";
import { ModuleCard } from "@/components/navigation/module-card";
import { StatusCountBadge } from "@/components/navigation/status-count-badge";
import { apiRequest, toErrorMessage } from "@/lib/api/client";
import { resolveCompanyScope } from "@/lib/auth/tenant-scope";
import { useAuth } from "@/components/auth/auth-provider";
import type { IncidentListResponse, Stream } from "@/lib/api/types";

type MonitoringSummary = {
  total: number;
  warn: number;
  fail: number;
};

const MODULES = [
  { href: "/watch", title: "Watch", subtitle: "Операторский режим", icon: Tv },
  { href: "/monitoring/streams", title: "Monitoring", subtitle: "Потоки и инциденты", icon: Radar },
  { href: "/admin/users", title: "Admin", subtitle: "Пользователи и настройки", icon: Shield },
  { href: "/sms", title: "SMS", subtitle: "Модуль уведомлений", icon: Settings },
  { href: "/ai", title: "AI", subtitle: "AI-инструменты", icon: Bot },
  { href: "/monitoring/analytics", title: "Reports", subtitle: "Сводная аналитика", icon: Gauge }
] as const;

export default function HubPage() {
  const { user, accessToken, activeCompanyId } = useAuth();
  const scopeCompanyId = resolveCompanyScope(user, activeCompanyId);
  const [summary, setSummary] = useState<MonitoringSummary>({ total: 0, warn: 0, fail: 0 });
  const [summaryError, setSummaryError] = useState<string | null>(null);

  useEffect(() => {
    if (!accessToken || !scopeCompanyId) {
      return;
    }
    const loadSummary = async () => {
      try {
        const [streamsResponse, warnResponse, failResponse] = await Promise.all([
          apiRequest<{ items: Stream[] }>(
            `/companies/${scopeCompanyId}/streams?limit=500`,
            { accessToken }
          ),
          apiRequest<IncidentListResponse>(
            `/companies/${scopeCompanyId}/incidents?status=open&severity=warn&page=0&page_size=1`,
            { accessToken }
          ),
          apiRequest<IncidentListResponse>(
            `/companies/${scopeCompanyId}/incidents?status=open&severity=fail&page=0&page_size=1`,
            { accessToken }
          )
        ]);
        const total = Array.isArray(streamsResponse.items) ? streamsResponse.items.length : 0;
        const warn = Number.isFinite(warnResponse.total) ? warnResponse.total : 0;
        const fail = Number.isFinite(failResponse.total) ? failResponse.total : 0;
        setSummary({ total, warn, fail });
      } catch (error) {
        setSummaryError(toErrorMessage(error));
      }
    };
    void loadSummary();
  }, [accessToken, scopeCompanyId]);

  return (
    <AuthGate>
      <div className="hub-page">
        <PrivateTopbar title="Gateway" />
        <main className="hub-content">
          <header className="page-header" style={{ marginBottom: "16px" }}>
            <h2 className="page-title">Добро пожаловать, {user?.login ?? "оператор"}.</h2>
            <p className="page-note">Выберите рабочий модуль платформы.</p>
          </header>
          <div className="hub-grid">
            {MODULES.map((moduleItem) => (
              <ModuleCard
                key={moduleItem.href}
                href={moduleItem.href}
                title={moduleItem.title}
                subtitle={moduleItem.subtitle}
                icon={moduleItem.icon}
                meta={
                  moduleItem.href === "/monitoring/streams" ? (
                    <div className="hub-status-row">
                      <StatusCountBadge
                        icon={AlertTriangle}
                        count={summary.warn}
                        tone="warn"
                        label={`${summary.warn} потоков требуют внимания (WARN)`}
                      />
                      <StatusCountBadge
                        icon={AlertTriangle}
                        count={summary.fail}
                        tone="fail"
                        label={`${summary.fail} потоков недоступны (FAIL)`}
                      />
                      <StatusCountBadge
                        icon={Activity}
                        count={summary.total}
                        label={`${summary.total} потоков всего`}
                      />
                    </div>
                  ) : null
                }
              />
            ))}
          </div>
          {summaryError ? <p className="core-summary-error">{summaryError}</p> : null}
        </main>
      </div>
    </AuthGate>
  );
}
