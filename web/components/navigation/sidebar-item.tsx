"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import type { LucideIcon } from "lucide-react";

type SidebarItemProps = {
  href: string;
  label: string;
  icon: LucideIcon;
  collapsed: boolean;
};

export function SidebarItem({ href, label, icon: Icon, collapsed }: SidebarItemProps) {
  const pathname = usePathname();
  const isActive = pathname === href || pathname.startsWith(`${href}/`);

  return (
    <Link
      href={href}
      className={isActive ? "section-sidebar-item is-active" : "section-sidebar-item"}
      title={collapsed ? label : undefined}
      aria-label={label}
    >
      <Icon size={18} aria-hidden />
      {!collapsed ? <span>{label}</span> : null}
    </Link>
  );
}
