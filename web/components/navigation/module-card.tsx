"use client";

import Link from "next/link";
import { motion } from "framer-motion";
import type { LucideIcon } from "lucide-react";
import type { ReactNode } from "react";

type ModuleCardProps = {
  href: string;
  title: string;
  subtitle?: string;
  icon: LucideIcon;
  meta?: ReactNode;
  ctaLabel?: string;
};

export function ModuleCard({ href, title, subtitle, icon: Icon, meta, ctaLabel = "Открыть" }: ModuleCardProps) {
  return (
    <motion.div whileHover={{ scale: 1.02 }} transition={{ duration: 0.2, ease: "easeOut" }}>
      <Link href={href} className="module-card">
        <div className="module-card-top">
          <span className="module-card-icon-wrap" aria-hidden>
            <span className="module-card-icon-glow" />
            <span className="module-card-icon">
              <Icon size={20} strokeWidth={1.75} />
            </span>
          </span>
        </div>

        <div className="module-card-middle">
          <div className="module-card-text">
            <h3>{title}</h3>
            {subtitle ? <p>{subtitle}</p> : null}
          </div>
        </div>

        <div className="module-card-bottom">
          {meta ? (
            <div className="module-card-meta">{meta}</div>
          ) : (
            <span className="module-card-cta" aria-hidden>
              {ctaLabel}
            </span>
          )}
        </div>
      </Link>
    </motion.div>
  );
}
