"use client";

import Link from "next/link";
import { motion } from "framer-motion";
import { useEffect, useMemo, useState } from "react";

import { useAuth } from "@/components/auth/auth-provider";
import { SkeletonBlock } from "@/components/ui/skeleton";
import { StatePanel } from "@/components/ui/state-panel";
import { apiRequest, toErrorMessage } from "@/lib/api/client";
import { resolveCompanyScope } from "@/lib/auth/tenant-scope";
import type { Incident, IncidentListResponse } from "@/lib/api/types";

type StatusFilter = "" | "open" | "resolved";
type SeverityFilter = "" | "warn" | "fail";

function formatTimestamp(ts: string | null | undefined): string {
  if (!ts) return "—";
  const d = new Date(ts);
  return Number.isNaN(d.getTime()) ? ts : d.toLocaleString();
}

export default function IncidentsPage() {
  const { user, accessToken, activeCompanyId } = useAuth();
  const scopeCompanyId = resolveCompanyScope(user, activeCompanyId);

  const [data, setData] = useState<IncidentListResponse | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [search, setSearch] = useState("");
  const [searchApplied, setSearchApplied] = useState("");
  const [statusFilter, setStatusFilter] = useState<StatusFilter>("");
  const [severityFilter, setSeverityFilter] = useState<SeverityFilter>("");
  const [page, setPage] = useState(0);
  const pageSize = 20;

  const loadIncidents = async () => {
    if (!accessToken || !scopeCompanyId) {
      setData(null);
      setIsLoading(false);
      return;
    }
    setIsLoading(true);
    setError(null);
    try {
      const params = new URLSearchParams();
      if (statusFilter) params.set("status", statusFilter);
      if (severityFilter) params.set("severity", severityFilter);
      if (searchApplied.trim()) params.set("q", searchApplied.trim());
      params.set("page", String(page));
      params.set("page_size", String(pageSize));

      const response = await apiRequest<IncidentListResponse>(
        `/companies/${scopeCompanyId}/incidents?${params.toString()}`,
        { accessToken }
      );
      setData(response);
    } catch (e) {
      setError(toErrorMessage(e));
      setData(null);
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    void loadIncidents();
  }, [accessToken, scopeCompanyId, statusFilter, severityFilter, page, searchApplied]);

  useEffect(() => {
    if (search.trim() === searchApplied.trim()) return;
    const t = setTimeout(() => {
      setSearchApplied(search);
      setPage(0);
    }, 300);
    return () => clearTimeout(t);
  }, [search, searchApplied]);

  return (
    <section className="panel">
      <header className="page-header compact">
        <h2 className="page-title">Инциденты</h2>
        <p className="page-note">
          Список инцидентов мониторинга: открытые и закрытые, с фильтрами по статусу и серьёзности.
        </p>
      </header>

      {!scopeCompanyId ? (
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ duration: 0.24, ease: "easeOut" }}
        >
          <StatePanel>Выберите компанию в шапке, чтобы загрузить инциденты.</StatePanel>
        </motion.div>
      ) : null}

      {scopeCompanyId ? (
        <>
          <div className="filters-grid streams-v2-filters">
            <label className="form-field" htmlFor="incidents-search">
              <span>Поиск</span>
              <input
                id="incidents-search"
                type="search"
                value={search}
                onChange={(e: { target: { value: string } }) => setSearch(e.target.value)}
                placeholder="Название потока или причина"
                disabled={isLoading}
                aria-label="Поиск по названию потока или причине"
              />
            </label>
            <label className="form-field" htmlFor="incidents-status">
              <span>Статус</span>
              <select
                id="incidents-status"
                value={statusFilter}
                onChange={(e: { target: { value: string } }) => {
                  setStatusFilter(e.target.value as StatusFilter);
                  setPage(0);
                }}
                disabled={isLoading}
                aria-label="Фильтр по статусу инцидента"
              >
                <option value="">Все</option>
                <option value="open">Открыт</option>
                <option value="resolved">Закрыт</option>
              </select>
            </label>
            <label className="form-field" htmlFor="incidents-severity">
              <span>Серьёзность</span>
              <select
                id="incidents-severity"
                value={severityFilter}
                onChange={(e: { target: { value: string } }) => {
                  setSeverityFilter(e.target.value as SeverityFilter);
                  setPage(0);
                }}
                disabled={isLoading}
                aria-label="Фильтр по серьёзности"
              >
                <option value="">Все</option>
                <option value="warn">WARN</option>
                <option value="fail">FAIL</option>
              </select>
            </label>
          </div>

          {data && !isLoading ? (
            <motion.div
              className="incidents-summary"
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              transition={{ duration: 0.2 }}
              style={{ display: "flex", gap: "12px", flexWrap: "wrap", marginTop: "12px", alignItems: "center" }}
            >
              <span className="summary-total">Всего: {data.total}</span>
              <button
                type="button"
                className="summary-card"
                onClick={() => {
                  setStatusFilter("open");
                  setPage(0);
                }}
                aria-pressed={statusFilter === "open"}
              >
                <span className="summary-label">Открытые</span>
              </button>
              <button
                type="button"
                className="summary-card"
                onClick={() => {
                  setStatusFilter("resolved");
                  setPage(0);
                }}
                aria-pressed={statusFilter === "resolved"}
              >
                <span className="summary-label">Закрытые</span>
              </button>
              <button
                type="button"
                className="summary-card summary-warn"
                onClick={() => {
                  setSeverityFilter("warn");
                  setPage(0);
                }}
                aria-pressed={severityFilter === "warn"}
              >
                <span className="summary-label">WARN</span>
              </button>
              <button
                type="button"
                className="summary-card summary-fail"
                onClick={() => {
                  setSeverityFilter("fail");
                  setPage(0);
                }}
                aria-pressed={severityFilter === "fail"}
              >
                <span className="summary-label">FAIL</span>
              </button>
            </motion.div>
          ) : null}

          {error ? (
            <motion.div
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              transition={{ duration: 0.24 }}
              style={{ marginTop: "12px" }}
            >
              <StatePanel kind="error">{error}</StatePanel>
            </motion.div>
          ) : null}

          {isLoading ? (
            <motion.div
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              transition={{ duration: 0.2 }}
              style={{ marginTop: "12px" }}
            >
              <SkeletonBlock lines={6} />
            </motion.div>
          ) : null}

          {!isLoading && !error && scopeCompanyId && data?.items?.length === 0 ? (
            <motion.div
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              transition={{ duration: 0.24 }}
              style={{ marginTop: "12px" }}
            >
              <StatePanel>Инцидентов по выбранным фильтрам нет.</StatePanel>
            </motion.div>
          ) : null}

          {!isLoading && !error && scopeCompanyId && data && data.items.length > 0 ? (
            <motion.div
              className="table-wrap"
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              transition={{ duration: 0.28 }}
              style={{ marginTop: "12px" }}
            >
              <table>
                <thead>
                  <tr>
                    <th>ID</th>
                    <th>Статус</th>
                    <th>Серьёзность</th>
                    <th>Поток</th>
                    <th>Начало</th>
                    <th>Последнее событие</th>
                    <th>Закрыт</th>
                    <th>Причина</th>
                  </tr>
                </thead>
                <tbody>
                  {data.items.map((inc) => (
                    <tr key={inc.id}>
                      <td>{inc.id}</td>
                      <td>
                        <span
                          className={
                            inc.status === "open"
                              ? "incident-badge incident-badge-open"
                              : "incident-badge incident-badge-resolved"
                          }
                        >
                          {inc.status === "open" ? "Открыт" : "Закрыт"}
                        </span>
                      </td>
                      <td>
                        <span
                          className={
                            inc.severity === "fail"
                              ? "incident-badge incident-badge-fail"
                              : "incident-badge incident-badge-warn"
                          }
                        >
                          {inc.severity === "fail" ? "FAIL" : "WARN"}
                        </span>
                      </td>
                      <td>
                        <Link
                          className="stream-link"
                          href={`/streams/${inc.stream_id}`}
                        >
                          {inc.stream_name ?? `Поток #${inc.stream_id}`}
                        </Link>
                      </td>
                      <td>{formatTimestamp(inc.started_at)}</td>
                      <td>{formatTimestamp(inc.last_event_at)}</td>
                      <td>{formatTimestamp(inc.resolved_at)}</td>
                      <td>{inc.fail_reason ?? "—"}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
              {data.next_cursor ? (
                <div style={{ marginTop: "12px" }}>
                  <button
                    type="button"
                    className="button-secondary"
                    onClick={() => setPage((p) => p + 1)}
                  >
                    Ещё
                  </button>
                </div>
              ) : null}
            </motion.div>
          ) : null}
        </>
      ) : null}
    </section>
  );
}
