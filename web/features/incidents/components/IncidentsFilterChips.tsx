"use client";

import { motion } from "framer-motion";
import type { SeverityFilter, StatusFilter } from "../types";

type IncidentsFilterChipsProps = {
  total: number;
  statusFilter: StatusFilter;
  severityFilter: SeverityFilter;
  onStatusFilterChange: (value: StatusFilter) => void;
  onSeverityFilterChange: (value: SeverityFilter) => void;
  onPageReset: () => void;
};

export function IncidentsFilterChips({
  total,
  statusFilter,
  severityFilter,
  onStatusFilterChange,
  onSeverityFilterChange,
  onPageReset
}: IncidentsFilterChipsProps) {
  const setStatus = (v: StatusFilter) => {
    onStatusFilterChange(v);
    onPageReset();
  };
  const setSeverity = (v: SeverityFilter) => {
    onSeverityFilterChange(v);
    onPageReset();
  };

  return (
    <motion.div
      className="severity-chips"
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      transition={{ duration: 0.2 }}
    >
      <span className="summary-total">Всего: {total}</span>
      <button
        type="button"
        className="severity-chip"
        onClick={() => setStatus("open")}
        aria-pressed={statusFilter === "open"}
      >
        Открытые
      </button>
      <button
        type="button"
        className="severity-chip"
        onClick={() => setStatus("resolved")}
        aria-pressed={statusFilter === "resolved"}
      >
        Закрытые
      </button>
      <button
        type="button"
        className="severity-chip severity-warn"
        onClick={() => setSeverity("warn")}
        aria-pressed={severityFilter === "warn"}
      >
        WARN
      </button>
      <button
        type="button"
        className="severity-chip severity-fail"
        onClick={() => setSeverity("fail")}
        aria-pressed={severityFilter === "fail"}
      >
        FAIL
      </button>
    </motion.div>
  );
}
