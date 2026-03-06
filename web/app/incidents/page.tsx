"use client";

import { motion } from "framer-motion";
import { useEffect, useState } from "react";
import { useAuth } from "@/components/auth/auth-provider";
import { SkeletonBlock } from "@/components/ui/skeleton";
import { StatePanel } from "@/components/ui/state-panel";
import { resolveCompanyScope } from "@/lib/auth/tenant-scope";
import {
  IncidentsFilterChips,
  IncidentsFilters,
  IncidentsList,
  useIncidentsData
} from "@/features/incidents";
import type { SeverityFilter, StatusFilter } from "@/features/incidents";

const PAGE_SIZE = 20;
const SEARCH_DEBOUNCE_MS = 300;

export default function IncidentsPage() {
  const { user, accessToken, activeCompanyId } = useAuth();
  const scopeCompanyId = resolveCompanyScope(user, activeCompanyId);

  const [search, setSearch] = useState("");
  const [searchApplied, setSearchApplied] = useState("");
  const [statusFilter, setStatusFilter] = useState<StatusFilter>("");
  const [severityFilter, setSeverityFilter] = useState<SeverityFilter>("");
  const [page, setPage] = useState(0);

  const { data, isLoading, error } = useIncidentsData({
    accessToken,
    scopeCompanyId,
    statusFilter,
    severityFilter,
    searchApplied,
    page,
    pageSize: PAGE_SIZE
  });

  useEffect(() => {
    if (search.trim() === searchApplied.trim()) return;
    const t = setTimeout(() => {
      setSearchApplied(search);
      setPage(0);
    }, SEARCH_DEBOUNCE_MS);
    return () => clearTimeout(t);
  }, [search, searchApplied]);

  const handleStatusFilterChange = (value: StatusFilter) => {
    setStatusFilter(value);
    setPage(0);
  };

  const handleSeverityFilterChange = (value: SeverityFilter) => {
    setSeverityFilter(value);
    setPage(0);
  };

  return (
    <section className="panel premium-panel">
      <header className="page-header compact">
        <div>
          <h2 className="page-title">Инциденты</h2>
          <p className="page-note">
            Список инцидентов мониторинга: открытые и закрытые, с фильтрами по
            статусу и серьёзности.
          </p>
        </div>
      </header>

      {!scopeCompanyId ? (
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ duration: 0.24, ease: "easeOut" }}
        >
          <StatePanel>
            Выберите компанию в шапке, чтобы загрузить инциденты.
          </StatePanel>
        </motion.div>
      ) : null}

      {scopeCompanyId ? (
        <>
          <IncidentsFilters
            search={search}
            onSearchChange={setSearch}
            statusFilter={statusFilter}
            onStatusFilterChange={handleStatusFilterChange}
            severityFilter={severityFilter}
            onSeverityFilterChange={handleSeverityFilterChange}
            disabled={isLoading}
          />

          {data && !isLoading ? (
            <IncidentsFilterChips
              total={data.total}
              statusFilter={statusFilter}
              severityFilter={severityFilter}
              onStatusFilterChange={handleStatusFilterChange}
              onSeverityFilterChange={handleSeverityFilterChange}
              onPageReset={() => setPage(0)}
            />
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

          {!isLoading &&
            !error &&
            scopeCompanyId &&
            data &&
            data.items.length > 0 ? (
            <IncidentsList
              items={data.items}
              nextCursor={data.next_cursor ?? null}
              onLoadMore={() => setPage((p) => p + 1)}
            />
          ) : null}
        </>
      ) : null}
    </section>
  );
}
