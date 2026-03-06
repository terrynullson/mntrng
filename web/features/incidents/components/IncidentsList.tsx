"use client";

import Link from "next/link";
import { motion } from "framer-motion";
import { AlertTriangle, Eye } from "lucide-react";
import type { Incident } from "@/lib/api/types";
import { diagLabel } from "../types";
import { formatTimestamp } from "../utils/format";

type IncidentsListProps = {
  items: Incident[];
  nextCursor: string | null;
  onLoadMore: () => void;
};

export function IncidentsList({
  items,
  nextCursor,
  onLoadMore
}: IncidentsListProps) {
  return (
    <motion.div
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      transition={{ duration: 0.28 }}
    >
      <div className="incident-list">
        {items.map((inc) => (
          <div key={inc.id} className="incident-card">
            <div
              className={`incident-card-severity ${inc.severity === "fail" ? "fail" : "warn"}`}
              aria-hidden
            >
              <AlertTriangle size={14} strokeWidth={2.5} />
            </div>
            <div className="incident-card-body">
              <div className="incident-card-title">
                {diagLabel(inc.diag_code)}
                {inc.fail_reason ? ` — ${inc.fail_reason}` : ""}
              </div>
              <div className="incident-card-meta">
                <Link
                  className="stream-link"
                  href={`/monitoring/streams/${inc.stream_id}`}
                >
                  {inc.stream_name ?? `Поток #${inc.stream_id}`}
                </Link>
                {" · "}
                {formatTimestamp(inc.started_at)}
                {inc.status === "resolved" && inc.resolved_at
                  ? ` · Закрыт ${formatTimestamp(inc.resolved_at)}`
                  : ""}
              </div>
            </div>
            <div className="incident-card-actions">
              <Link
                href={`/monitoring/incidents/${inc.id}`}
                className="icon-button"
                aria-label={`Открыть инцидент #${inc.id}`}
                title="Открыть"
              >
                <Eye size={16} aria-hidden />
              </Link>
            </div>
          </div>
        ))}
      </div>
      {nextCursor ? (
        <div style={{ marginTop: "12px" }}>
          <button
            type="button"
            className="button-secondary"
            onClick={onLoadMore}
          >
            Ещё
          </button>
        </div>
      ) : null}
    </motion.div>
  );
}
