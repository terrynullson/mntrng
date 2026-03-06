"use client";

import { motion } from "framer-motion";
import { useEffect, useState } from "react";
import { useAuth } from "@/components/auth/auth-provider";
import { AppButton } from "@/components/ui/app-button";
import { SkeletonBlock } from "@/components/ui/skeleton";
import { StatePanel } from "@/components/ui/state-panel";
import { resolveCompanyScope } from "@/lib/auth/tenant-scope";
import {
  STORAGE_LAST_SECTION_KEY,
  StreamFormDialog,
  StreamsFilters,
  StreamsTable,
  useStreamActions,
  useStreamsData,
  useStreamsFilters,
  useStreamForm
} from "@/features/streams";
import type { IsActiveFilter } from "@/features/streams";

export default function StreamsPage() {
  const { user, accessToken, activeCompanyId } = useAuth();
  const scopeCompanyId = resolveCompanyScope(user, activeCompanyId);
  const isViewer = user?.role === "viewer";

  const [projectId, setProjectId] = useState<string>("");
  const [isActiveFilter, setIsActiveFilter] = useState<IsActiveFilter>("all");

  const data = useStreamsData({
    accessToken,
    scopeCompanyId,
    projectId,
    isActiveFilter
  });

  const filters = useStreamsFilters(
    data.streams,
    data.latestStatusMap,
    data.favorites
  );

  const actions = useStreamActions({
    accessToken,
    scopeCompanyId,
    isViewer,
    reload: data.reload,
    getProjects: () => data.projects
  });

  const form = useStreamForm({
    accessToken,
    scopeCompanyId,
    isViewer,
    projects: data.projects,
    reload: data.reload,
    ensureCommonProject: actions.ensureCommonProject
  });

  useEffect(() => {
    if (typeof window === "undefined") return;
    window.localStorage.setItem(STORAGE_LAST_SECTION_KEY, "/monitoring/streams");
  }, []);

  const openCreateDialog = () => {
    form.openCreate(projectId || undefined);
  };

  return (
    <section className="panel premium-panel">
      <header className="page-header compact">
        <div>
          <h2 className="page-title">Мониторинг потоков</h2>
          <p className="page-note">
            CRUD потоков, статусы, избранное/закрепление и ручной запуск проверки.
          </p>
        </div>
        <div className="page-header-actions">
          <AppButton
            type="button"
            disabled={isViewer || !scopeCompanyId}
            onClick={openCreateDialog}
          >
            + Добавить поток
          </AppButton>
        </div>
      </header>

      {!scopeCompanyId ? (
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ duration: 0.24, ease: "easeOut" }}
        >
          <StatePanel>
            Выберите компанию в шапке, чтобы загрузить потоки.
          </StatePanel>
        </motion.div>
      ) : null}

      <StreamsFilters
        search={filters.search}
        onSearchChange={filters.setSearch}
        projectId={projectId}
        onProjectIdChange={setProjectId}
        isActiveFilter={isActiveFilter}
        onIsActiveFilterChange={setIsActiveFilter}
        statusFilter={filters.statusFilter}
        onStatusFilterChange={filters.setStatusFilter}
        projects={data.projects}
        disabled={!scopeCompanyId || data.isLoading}
      />

      {isViewer ? (
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ duration: 0.24, ease: "easeOut" }}
        >
          <StatePanel>
            Роль «Зритель» — только просмотр. Запуск проверок недоступен.
          </StatePanel>
        </motion.div>
      ) : null}

      {actions.runCheckSuccess ? (
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ duration: 0.24, ease: "easeOut" }}
        >
          <StatePanel>{actions.runCheckSuccess}</StatePanel>
        </motion.div>
      ) : null}
      {actions.runCheckError ? (
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ duration: 0.24, ease: "easeOut" }}
        >
          <StatePanel kind="error">{actions.runCheckError}</StatePanel>
        </motion.div>
      ) : null}
      {data.error || actions.screenError ? (
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ duration: 0.24, ease: "easeOut" }}
        >
          <StatePanel kind="error">{data.error ?? actions.screenError}</StatePanel>
        </motion.div>
      ) : null}

      {data.isLoading ? (
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ duration: 0.2, ease: "easeOut" }}
          style={{ marginTop: "12px" }}
        >
          <SkeletonBlock lines={7} />
        </motion.div>
      ) : null}

      {!data.isLoading &&
        !data.error &&
        !actions.screenError &&
        scopeCompanyId &&
        filters.filteredStreams.length === 0 ? (
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ duration: 0.24, ease: "easeOut" }}
          style={{ marginTop: "12px" }}
        >
          <StatePanel>
            Потоков пока нет — добавь первый.
            {!isViewer ? (
              <>
                {" "}
                <button
                  type="button"
                  className="linklike-button"
                  onClick={openCreateDialog}
                >
                  Добавить поток
                </button>
              </>
            ) : null}
          </StatePanel>
        </motion.div>
      ) : null}

      {!data.isLoading &&
        !data.error &&
        !actions.screenError &&
        scopeCompanyId &&
        filters.filteredStreams.length > 0 ? (
        <StreamsTable
          streams={filters.filteredStreams}
          latestStatusMap={data.latestStatusMap}
          favoriteMap={filters.favoriteMap}
          isViewer={isViewer}
          busyStreamID={actions.busyStreamID}
          busyFavoriteStreamID={actions.busyFavoriteStreamID}
          onToggleFavorite={(stream) =>
            actions.handleToggleFavorite(stream, filters.favoriteMap)
          }
          onTogglePin={(stream) =>
            actions.handleTogglePin(stream, filters.favoriteMap)
          }
          onRunCheck={actions.handleRunCheck}
          onEdit={form.openEdit}
          onDelete={actions.handleDeleteStream}
        />
      ) : null}

      <StreamFormDialog
        isOpen={form.isFormOpen}
        editingStream={form.editingStream}
        initialProjectId={form.initialProjectId}
        projects={data.projects}
        formError={form.formError}
        isFormSubmitting={form.isFormSubmitting}
        onClose={form.close}
        onSubmit={form.submit}
      />
    </section>
  );
}
