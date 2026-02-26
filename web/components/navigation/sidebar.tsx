"use client";

import { PanelLeftClose, PanelLeftOpen } from "lucide-react";
import type { LucideIcon } from "lucide-react";
import { useEffect, useMemo, useState } from "react";

import { SidebarItem } from "@/components/navigation/sidebar-item";
import { IconButton } from "@/components/navigation/icon-button";

type Item = {
  href: string;
  label: string;
  icon: LucideIcon;
};

type SidebarProps = {
  title: string;
  storageKey: string;
  items: Item[];
};

export function Sidebar({ title, storageKey, items }: SidebarProps) {
  const [collapsed, setCollapsed] = useState(false);

  useEffect(() => {
    if (typeof window === "undefined") {
      return;
    }
    const raw = window.localStorage.getItem(storageKey);
    setCollapsed(raw === "1");
  }, [storageKey]);

  useEffect(() => {
    if (typeof window === "undefined") {
      return;
    }
    window.localStorage.setItem(storageKey, collapsed ? "1" : "0");
  }, [collapsed, storageKey]);

  const widthClass = useMemo(
    () => (collapsed ? "section-sidebar is-collapsed" : "section-sidebar"),
    [collapsed]
  );

  return (
    <aside className={widthClass}>
      <div className="section-sidebar-header">
        {!collapsed ? <h1>{title}</h1> : null}
        <IconButton
          onClick={() => setCollapsed((prev) => !prev)}
          label={collapsed ? "Развернуть меню" : "Свернуть меню"}
          tooltip={collapsed ? "Развернуть меню" : "Свернуть меню"}
        >
          {collapsed ? <PanelLeftOpen size={16} /> : <PanelLeftClose size={16} />}
        </IconButton>
      </div>
      <nav className="section-sidebar-nav" aria-label={title}>
        {items.map((item) => (
          <SidebarItem
            key={item.href}
            href={item.href}
            label={item.label}
            icon={item.icon}
            collapsed={collapsed}
          />
        ))}
      </nav>
    </aside>
  );
}
