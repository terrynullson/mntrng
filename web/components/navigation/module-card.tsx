"use client";

import Link from "next/link";
import { motion } from "framer-motion";
import type { LucideIcon } from "lucide-react";

type ModuleCardProps = {
  href: string;
  title: string;
  subtitle?: string;
  icon: LucideIcon;
  meta?: any;
};

export function ModuleCard({ href, title, subtitle, icon: Icon, meta }: ModuleCardProps) {
  return (
    <motion.div whileHover={{ y: -4 }} transition={{ duration: 0.18, ease: "easeOut" }}>
      <Link href={href} className="module-card">
        <div className="module-card-head">
          <span className="module-card-icon" aria-hidden>
            <Icon size={20} />
          </span>
          <div>
            <h3>{title}</h3>
            {subtitle ? <p>{subtitle}</p> : null}
          </div>
        </div>
        {meta ? <div className="module-card-meta">{meta as any}</div> : null}
      </Link>
    </motion.div>
  );
}
