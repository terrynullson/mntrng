"use client";

import { Bell, Activity, AlertTriangle, Bot, ChevronDown, Gauge, LogOut, Radar, Settings, Shield, Tv, XCircle } from "lucide-react";
import { useCallback, useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";

import { AuthGate } from "@/components/auth/auth-gate";
import { ModuleCard } from "@/components/navigation/module-card";
import { StatusCountBadge } from "@/components/navigation/status-count-badge";
import { ThemeToggleButton } from "@/components/theme/theme-toggle-button";
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
  const router = useRouter();
  const { user, companies, accessToken, activeCompanyId, setActiveCompanyId, logout } = useAuth();
  const scopeCompanyId = resolveCompanyScope(user, activeCompanyId);
  const [summary, setSummary] = useState<MonitoringSummary>({ total: 0, warn: 0, fail: 0 });
  const [summaryError, setSummaryError] = useState<string | null>(null);
  const [isUserMenuOpen, setIsUserMenuOpen] = useState<boolean>(false);
  const userMenuRef = useRef<HTMLDivElement | null>(null);

  const closeUserMenu = useCallback(() => setIsUserMenuOpen(false), []);

  useEffect(() => {
    if (!isUserMenuOpen) {
      return;
    }
    const handleClick = (event: MouseEvent) => {
      const root = userMenuRef.current;
      if (root && !root.contains(event.target as Node)) {
        closeUserMenu();
      }
    };
    document.addEventListener("click", handleClick, true);
    return () => document.removeEventListener("click", handleClick, true);
  }, [closeUserMenu, isUserMenuOpen]);

  const handleLogout = async () => {
    await logout();
    router.replace("/login");
  };

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
        <main className="hub-content">
          <div className="hub-floating-topbar">
            <div className="hub-topbar-zone hub-topbar-left">
              <span className="hub-brand-mark" aria-hidden />
              <span className="hub-brand-text">SHOZA PORTAL</span>
            </div>

            <div className="hub-topbar-zone hub-topbar-center">
              <div className="hub-floating-control hub-theme-control">
                <ThemeToggleButton />
              </div>
            </div>

            <div className="hub-topbar-zone hub-topbar-right">
              {user?.role === "super_admin" ? (
                <label className="hub-floating-control hub-company-switcher" htmlFor="hub-active-company-switcher">
                  <select
                    id="hub-active-company-switcher"
                    value={activeCompanyId ?? ""}
                    onChange={(event) => {
                      const value = Number.parseInt(event.target.value, 10);
                      setActiveCompanyId(Number.isFinite(value) ? value : null);
                    }}
                    aria-label="Выбор компании (контекст)"
                  >
                    {companies.length === 0 ? <option value="">Нет компаний</option> : null}
                    {companies.map((company) => (
                      <option key={company.id} value={company.id}>
                        {company.name}
                      </option>
                    ))}
                  </select>
                </label>
              ) : null}

              <button
                type="button"
                className="hub-floating-control hub-icon-control hub-icon-bell"
                aria-label="Уведомления"
                title="Уведомления"
              >
                <Bell size={16} strokeWidth={1.75} aria-hidden />
              </button>

              <div className="hub-user-menu" ref={userMenuRef}>
                <button
                  type="button"
                  className="hub-floating-control hub-user-trigger"
                  onClick={() => setIsUserMenuOpen((previous) => !previous)}
                  aria-expanded={isUserMenuOpen}
                  aria-haspopup="true"
                  aria-controls="hub-user-menu-panel"
                >
                  <span>{user?.login ?? "user"}</span>
                  <ChevronDown size={14} strokeWidth={1.75} aria-hidden />
                </button>

                {isUserMenuOpen ? (
                  <div id="hub-user-menu-panel" className="hub-user-menu-panel" role="menu">
                    <p>{user?.email}</p>
                    <button type="button" className="hub-user-logout" onClick={() => void handleLogout()}>
                      <LogOut size={14} strokeWidth={1.75} aria-hidden />
                      <span>Выход</span>
                    </button>
                  </div>
                ) : null}
              </div>
            </div>
          </div>

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
                        icon={Activity}
                        name="Activity"
                        count={summary.total}
                        label={`${summary.total} потоков всего`}
                        iconNumberOnly
                      />
                      <StatusCountBadge
                        icon={AlertTriangle}
                        name="AlertTriangle"
                        count={summary.warn}
                        tone="warn"
                        label={`${summary.warn} потоков требуют внимания (WARN)`}
                        iconNumberOnly
                      />
                      <StatusCountBadge
                        icon={XCircle}
                        name="XCircle"
                        count={summary.fail}
                        tone="fail"
                        label={`${summary.fail} потоков недоступны (FAIL)`}
                        iconNumberOnly
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
