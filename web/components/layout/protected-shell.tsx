"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { AnimatePresence, motion } from "framer-motion";
import {
  type PropsWithChildren,
  useEffect,
  useMemo,
  useState
} from "react";

import { useAuth } from "@/components/auth/auth-provider";
import { AnimatedGradientBackground } from "@/components/ui/AnimatedGradientBackground";
import { AppButton } from "@/components/ui/app-button";
import { SkeletonBlock } from "@/components/ui/skeleton";
import { ThemeToggleButton } from "@/components/theme/theme-toggle-button";
import type { Role } from "@/lib/api/types";

const PUBLIC_ROUTES = new Set(["/login", "/register"]);

const NAV_ITEMS_BY_ROLE: Record<Role, Array<{ href: string; label: string }>> = {
  super_admin: [
    { href: "/", label: "Overview" },
    { href: "/streams", label: "Streams" },
    { href: "/analytics", label: "Analytics" },
    { href: "/admin/users", label: "Users" },
    { href: "/companies", label: "Companies" },
    { href: "/admin/requests", label: "Requests" },
    { href: "/settings", label: "Settings" }
  ],
  company_admin: [
    { href: "/", label: "Overview" },
    { href: "/streams", label: "Streams" },
    { href: "/analytics", label: "Analytics" },
    { href: "/settings", label: "Settings" }
  ],
  viewer: [
    { href: "/", label: "Overview" },
    { href: "/streams", label: "Streams" },
    { href: "/analytics", label: "Analytics" }
  ]
};

const PATH_LABELS: Array<{ pattern: RegExp; title: string }> = [
  { pattern: /^\/$/, title: "Overview" },
  { pattern: /^\/streams$/, title: "Streams" },
  { pattern: /^\/streams\/.+/, title: "Stream Details" },
  { pattern: /^\/analytics$/, title: "Analytics" },
  { pattern: /^\/settings$/, title: "Settings" },
  { pattern: /^\/admin\/requests$/, title: "Registration Requests" },
  { pattern: /^\/admin\/users$/, title: "Users" },
  { pattern: /^\/companies$/, title: "Companies" }
];

function isPublicPath(pathname: string): boolean {
  return PUBLIC_ROUTES.has(pathname);
}

function resolvePageTitle(pathname: string): string {
  const match = PATH_LABELS.find((item) => item.pattern.test(pathname));
  return match?.title ?? "Admin";
}

function isActiveRoute(pathname: string, href: string): boolean {
  if (href === "/") {
    return pathname === href;
  }
  return pathname === href || pathname.startsWith(`${href}/`);
}

export function ProtectedShell({ children }: PropsWithChildren) {
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

  const isPublicRoute = isPublicPath(pathname);

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
        {children}
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

        <nav className="secure-nav" aria-label="Main navigation">
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
              aria-label="Toggle sidebar"
            >
              {isNavCollapsed ? "Expand" : "Collapse"}
            </AppButton>
            <div>
              <p className="secure-page-title">{pageTitle}</p>
              <p className="secure-page-note">Role: {user.role}</p>
            </div>
          </div>

          <div className="secure-topbar-right">
            <ThemeToggleButton />
            {user.role === "super_admin" ? (
              <label className="company-switcher" htmlFor="active-company-switcher">
                <span>Company scope</span>
                <select
                  id="active-company-switcher"
                  value={activeCompanyId ?? ""}
                  onChange={(event) => {
                    const value = Number.parseInt(event.target.value, 10);
                    setActiveCompanyId(Number.isFinite(value) ? value : null);
                  }}
                >
                  {companies.length === 0 ? (
                    <option value="">No companies</option>
                  ) : null}
                  {companies.map((company) => (
                    <option key={company.id} value={company.id}>
                      {company.name} ({company.id})
                    </option>
                  ))}
                </select>
              </label>
            ) : null}

            <div className="user-menu-root">
              <AppButton
                type="button"
                variant="secondary"
                className="user-menu-trigger"
                onClick={() => setIsUserMenuOpen((previous) => !previous)}
                aria-expanded={isUserMenuOpen}
              >
                {user.login}
              </AppButton>

              <AnimatePresence>
                {isUserMenuOpen ? (
                  <motion.div
                    className="user-menu-panel"
                    initial={{ opacity: 0, y: -8 }}
                    animate={{ opacity: 1, y: 0 }}
                    exit={{ opacity: 0, y: -8 }}
                    transition={{ duration: 0.16, ease: "easeOut" }}
                  >
                    <p>{user.email}</p>
                    <AppButton type="button" variant="danger" onClick={handleLogout}>
                      Logout
                    </AppButton>
                  </motion.div>
                ) : null}
              </AnimatePresence>
            </div>
          </div>
        </header>

        <main className="secure-content">{children}</main>
      </div>
    </div>
  );
}
