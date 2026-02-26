"use client";

import { ChevronDown, LogOut } from "lucide-react";
import { useState } from "react";
import { useRouter } from "next/navigation";

import { useAuth } from "@/components/auth/auth-provider";
import { ThemeToggleButton } from "@/components/theme/theme-toggle-button";
import { IconButton } from "@/components/navigation/icon-button";

type PrivateTopbarProps = {
  title: string;
};

export function PrivateTopbar({ title }: PrivateTopbarProps) {
  const router = useRouter();
  const { user, companies, activeCompanyId, setActiveCompanyId, logout } = useAuth();
  const [isOpen, setIsOpen] = useState(false);

  const handleLogout = async () => {
    await logout();
    router.replace("/auth/login");
  };

  return (
    <header className="section-topbar">
      <div>
        <h2 className="section-topbar-title">{title}</h2>
      </div>
      <div className="section-topbar-actions">
        {user?.role === "super_admin" ? (
          <label className="company-switcher" htmlFor="active-company-switcher">
            <span>Компания</span>
            <select
              id="active-company-switcher"
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
                  {company.name} ({company.id})
                </option>
              ))}
            </select>
          </label>
        ) : null}
        <ThemeToggleButton />
        <div className="user-menu-inline">
          <button
            type="button"
            className="user-pill"
            aria-label="Меню пользователя"
            title="Меню пользователя"
            onClick={() => setIsOpen((prev) => !prev)}
          >
            <span>{user?.login ?? "user"}</span>
            <ChevronDown size={14} />
          </button>
          {isOpen ? (
            <div className="user-menu-panel">
              <p>{user?.email}</p>
              <IconButton
                onClick={() => void handleLogout()}
                label="Выйти"
                tooltip="Выйти"
                destructive
              >
                <LogOut size={16} />
              </IconButton>
            </div>
          ) : null}
        </div>
      </div>
    </header>
  );
}
