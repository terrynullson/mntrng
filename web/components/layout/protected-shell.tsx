"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { AnimatePresence, motion } from "framer-motion";
import type { ReactNode } from "react";
import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState
} from "react";

import { useAuth } from "@/components/auth/auth-provider";
import {
  AnimatedGradientBackground as AnimatedGradientBackgroundSource
} from "@/components/ui/AnimatedGradientBackground";

/// eslint-disable-next-line @typescript-eslint/no-explicit-any — совместимость типов memo/ReactNode (shim и @types/react)
const AnimatedGradientBackground = AnimatedGradientBackgroundSource as any;
import { AppButton } from "@/components/ui/app-button";
import { SkeletonBlock } from "@/components/ui/skeleton";
import { ThemeToggleButton } from "@/components/theme/theme-toggle-button";
import type { Role } from "@/lib/api/types";

const PUBLIC_ROUTES = new Set(["/login", "/register"]);

const NAV_ITEMS_BY_ROLE: Record<Role, Array<{ href: string; label: string }>> = {
  super_admin: [
    { href: "/", label: "Домашняя" },
    { href: "/watch", label: "Смотреть" },
    { href: "/streams", label: "Потоки" },
    { href: "/incidents", label: "Инциденты" },
    { href: "/analytics", label: "Аналитика" },
    { href: "/admin/users", label: "Пользователи" },
    { href: "/companies", label: "Компании" },
    { href: "/admin/requests", label: "Заявки" },
    { href: "/settings", label: "Настройки" }
  ],
  company_admin: [
    { href: "/", label: "Домашняя" },
    { href: "/watch", label: "Смотреть" },
    { href: "/streams", label: "Потоки" },
    { href: "/incidents", label: "Инциденты" },
    { href: "/analytics", label: "Аналитика" },
    { href: "/settings", label: "Настройки" }
  ],
  viewer: [
    { href: "/", label: "Домашняя" },
    { href: "/watch", label: "Смотреть" },
    { href: "/streams", label: "Потоки" },
    { href: "/incidents", label: "Инциденты" },
    { href: "/analytics", label: "Аналитика" }
  ]
};

const PATH_LABELS: Array<{ pattern: RegExp; title: string }> = [
  { pattern: /^\/$/, title: "Домашняя" },
  { pattern: /^\/watch$/, title: "Смотреть" },
  { pattern: /^\/streams$/, title: "Потоки" },
  { pattern: /^\/streams\/.+/, title: "Поток" },
  { pattern: /^\/incidents$/, title: "Инциденты" },
  { pattern: /^\/analytics$/, title: "Аналитика" },
  { pattern: /^\/settings$/, title: "Настройки" },
  { pattern: /^\/admin\/requests$/, title: "Заявки на регистрацию" },
  { pattern: /^\/admin\/users$/, title: "Пользователи" },
  { pattern: /^\/companies$/, title: "Компании" }
];

function isPublicPath(pathname: string): boolean {
  return PUBLIC_ROUTES.has(pathname);
}

function resolvePageTitle(pathname: string): string {
  const match = PATH_LABELS.find((item) => item.pattern.test(pathname));
  return match?.title ?? "Панель";
}

function isActiveRoute(pathname: string, href: string): boolean {
  if (href === "/") {
    return pathname === href;
  }
  return pathname === href || pathname.startsWith(`${href}/`);
}

export function ProtectedShell({ children }: { children?: ReactNode }) {
  const router = useRouter();
  const pathname = usePathname();

  const {
    isReady,
    isAuthenticated,
    user,
    companies,
    activeCompanyId,
    setActiveCompanyId,
    logout
  } = useAuth();

  const [isNavCollapsed, setIsNavCollapsed] = useState<boolean>(false);
  const [isUserMenuOpen, setIsUserMenuOpen] = useState<boolean>(false);
  const userMenuRef = useRef<HTMLDivElement | null>(null);

  const isPublicRoute = isPublicPath(pathname);

  const closeUserMenu = useCallback(() => setIsUserMenuOpen(false), []);

  useEffect(() => {
    if (!isUserMenuOpen) return;
    const handleClick = (event: MouseEvent) => {
      const el = userMenuRef.current;
      if (el && !el.contains(event.target as Node)) {
        closeUserMenu();
      }
    };
    document.addEventListener("click", handleClick, true);
    return () => document.removeEventListener("click", handleClick, true);
  }, [isUserMenuOpen, closeUserMenu]);

  useEffect(() => {
    if (!isReady) {
      return;
    }

    if (!isAuthenticated && !isPublicRoute) {
      router.replace(`/login?next=${encodeURIComponent(pathname)}`);
      return;
    }

    if (isAuthenticated && isPublicRoute) {
      router.replace("/");
    }
  }, [isAuthenticated, isPublicRoute, isReady, pathname, router]);

  const navItems = useMemo(() => {
    if (!user) {
      return [];
    }
    return NAV_ITEMS_BY_ROLE[user.role];
  }, [user]);

  const pageTitle = resolvePageTitle(pathname);

  const handleLogout = async () => {
    await logout();
    router.replace("/login");
  };

  if (isPublicRoute) {
    return (
      <div className="public-root">
        <div className="public-theme-toggle">
          <ThemeToggleButton />
        </div>
        <AnimatedGradientBackground />
        {children as any}
      </div>
    );
  }

  if (!isReady || !isAuthenticated || !user) {
    return (
      <div className="protected-loading" role="status" aria-live="polite">
        <SkeletonBlock lines={6} className="protected-loading-card" />
      </div>
    );
  }

  return (
    <div className="secure-shell">
      <motion.aside
        className="secure-sidebar"
        animate={{ width: isNavCollapsed ? 78 : 248 }}
        transition={{ duration: 0.22, ease: "easeOut" }}
      >
        <div className="secure-brand">
          <p className="secure-brand-kicker">HLS Monitoring</p>
          {!isNavCollapsed ? <h1>Admin v2</h1> : <h1>A2</h1>}
        </div>

        <nav className="secure-nav" aria-label="Основная навигация">
          {navItems.map((item) => (
            <Link
              key={item.href}
              href={item.href}
              className={isActiveRoute(pathname, item.href) ? "is-active" : ""}
              title={item.label}
            >
              {isNavCollapsed ? item.label.slice(0, 1) : item.label}
            </Link>
          ))}
        </nav>
      </motion.aside>

      <div className="secure-main">
        <header className="secure-topbar">
          <div className="secure-topbar-left">
            <AppButton
              type="button"
              variant="ghost"
              className="burger-button"
              onClick={() => setIsNavCollapsed((previous) => !previous)}
              aria-label={isNavCollapsed ? "Развернуть боковую панель" : "Свернуть боковую панель"}
            >
              {isNavCollapsed ? "Развернуть" : "Свернуть"}
            </AppButton>
            <div>
              <p className="secure-page-title">{pageTitle}</p>
              <p className="secure-page-note">Роль: {user.role}</p>
            </div>
          </div>

          <div className="secure-topbar-right">
            <ThemeToggleButton />
            {user.role === "super_admin" ? (
              <label className="company-switcher" htmlFor="active-company-switcher">
                <span>Компания</span>
                <select
                  id="active-company-switcher"
                  value={activeCompanyId ?? ""}
                  onChange={(event: { target: { value: string } }) => {
                    const value = Number.parseInt(event.target.value, 10);
                    setActiveCompanyId(Number.isFinite(value) ? value : null);
                  }}
                  aria-label="Выбор компании (контекст)"
                >
                  {companies.length === 0 ? (
                    <option value="">Нет компаний</option>
                  ) : null}
                  {companies.map((company) => (
                    <option key={company.id} value={company.id}>
                      {company.name} ({company.id})
                    </option>
                  ))}
                </select>
              </label>
            ) : null}

            <div className="user-menu-root" ref={userMenuRef}>
              <AppButton
                type="button"
                variant="secondary"
                className="user-menu-trigger"
                onClick={() => setIsUserMenuOpen((previous) => !previous)}
                aria-expanded={isUserMenuOpen}
                aria-haspopup="true"
                aria-controls="user-menu-panel"
              >
                {user.login}
              </AppButton>

              <AnimatePresence>
                {isUserMenuOpen ? (
                  <motion.div
                    id="user-menu-panel"
                    className="user-menu-panel"
                    initial={{ opacity: 0, y: -8 }}
                    animate={{ opacity: 1, y: 0 }}
                    exit={{ opacity: 0, y: -8 }}
                    transition={{ duration: 0.16, ease: "easeOut" }}
                    role="menu"
                  >
                    <p>{user.email}</p>
                    <AppButton type="button" variant="danger" onClick={handleLogout}>
                      Выход
                    </AppButton>
                  </motion.div>
                ) : null}
              </AnimatePresence>
            </div>
          </div>
        </header>

        <main className="secure-content">{children as any}</main>
      </div>
    </div>
  );
}
