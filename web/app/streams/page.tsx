"use client";

import Link from "next/link";
import { motion } from "framer-motion";
import { useEffect, useMemo, useState } from "react";

import { useAuth } from "@/components/auth/auth-provider";
import { AppButton } from "@/components/ui/app-button";
import { SkeletonBlock } from "@/components/ui/skeleton";
import { StatePanel } from "@/components/ui/state-panel";
import { StatusBadge } from "@/components/ui/status-badge";
import { apiRequest, toErrorMessage } from "@/lib/api/client";
import { resolveCompanyScope } from "@/lib/auth/tenant-scope";
import type {
  CheckResult,
  CheckStatus,
  EnqueueCheckJobResponse,
  Project,
  Stream,
  StreamCreateRequest,
  StreamPatchRequest,
  StreamWithFavorite
} from "@/lib/api/types";

type IsActiveFilter = "all" | "true" | "false";
type StatusFilter = "all" | CheckStatus;

type LatestStatusMap = Record<
  number,
  {
    status: CheckStatus | null;
    createdAt: string | null;
  }
>;

const STORAGE_LAST_SECTION_KEY = "core.lastSection";

function formatTimestamp(timestamp: string | null): string {
  if (!timestamp) {
    return "-";
  }

  const parsed = new Date(timestamp);
  return Number.isNaN(parsed.getTime()) ? timestamp : parsed.toLocaleString();
}

function normalizeStatus(value: string): CheckStatus | null {
  if (value === "OK" || value === "WARN" || value === "FAIL") {
    return value;
  }
  return null;
}

export default function StreamsPage() {
  const { user, accessToken, activeCompanyId } = useAuth();

  const scopeCompanyId = resolveCompanyScope(user, activeCompanyId);
  const isViewer = user?.role === "viewer";

  const [projects, setProjects] = useState<Project[]>([]);
  const [streams, setStreams] = useState<Stream[]>([]);
  const [favorites, setFavorites] = useState<StreamWithFavorite[]>([]);
  const [latestStatusMap, setLatestStatusMap] = useState<LatestStatusMap>({});

  const [projectId, setProjectId] = useState<string>("");
  const [isActiveFilter, setIsActiveFilter] = useState<IsActiveFilter>("all");
  const [statusFilter, setStatusFilter] = useState<StatusFilter>("all");
  const [search, setSearch] = useState<string>("");

  const [isLoading, setIsLoading] = useState<boolean>(false);
  const [error, setError] = useState<string | null>(null);
  const [runCheckError, setRunCheckError] = useState<string | null>(null);
  const [runCheckSuccess, setRunCheckSuccess] = useState<string | null>(null);
  const [formError, setFormError] = useState<string | null>(null);
  const [busyStreamID, setBusyStreamID] = useState<number | null>(null);
  const [busyFavoriteStreamID, setBusyFavoriteStreamID] = useState<number | null>(null);
  const [isFormOpen, setIsFormOpen] = useState(false);
  const [editingStream, setEditingStream] = useState<Stream | null>(null);
  const [isFormSubmitting, setIsFormSubmitting] = useState(false);
  const [formName, setFormName] = useState("");
  const [formURL, setFormURL] = useState("");
  const [formProjectID, setFormProjectID] = useState<string>("");
  const [formIsActive, setFormIsActive] = useState(true);

  const loadStreams = async () => {
    if (!accessToken || !scopeCompanyId) {
      setProjects([]);
      setStreams([]);
      setLatestStatusMap({});
      setIsLoading(false);
      return;
    }

    setIsLoading(true);
    setError(null);
    setRunCheckError(null);

    try {
      const projectResponse = await apiRequest<{ items: Project[] }>(
        `/companies/${scopeCompanyId}/projects?limit=200`,
        { accessToken }
      );
      setProjects(Array.isArray(projectResponse.items) ? projectResponse.items : []);

      const streamParams = new URLSearchParams();
      streamParams.set("limit", "200");
      if (projectId) {
        streamParams.set("project_id", projectId);
      }
      if (isActiveFilter !== "all") {
        streamParams.set("is_active", isActiveFilter);
      }

      const streamResponse = await apiRequest<{ items: Stream[] }>(
        `/companies/${scopeCompanyId}/streams?${streamParams.toString()}`,
        { accessToken }
      );

      const loadedStreams = Array.isArray(streamResponse.items)
        ? streamResponse.items
        : [];
      setStreams(loadedStreams);

      try {
        const favResponse = await apiRequest<{ items: StreamWithFavorite[] }>(
          `/companies/${scopeCompanyId}/streams/favorites`,
          { accessToken }
        );
        setFavorites(Array.isArray(favResponse.items) ? favResponse.items : []);
      } catch {
        setFavorites([]);
      }

      const checks = await Promise.all(
        loadedStreams.map(async (stream) => {
          try {
            const resultResponse = await apiRequest<{ items: CheckResult[] }>(
              `/companies/${scopeCompanyId}/streams/${stream.id}/check-results?limit=1`,
              { accessToken }
            );
            const latest = resultResponse.items?.[0];
            return {
              streamID: stream.id,
              status: latest ? normalizeStatus(latest.status) : null,
              createdAt: latest?.created_at ?? null
            };
          } catch {
            return {
              streamID: stream.id,
              status: null,
              createdAt: null
            };
          }
        })
      );

      const nextStatusMap: LatestStatusMap = {};
      checks.forEach((entry) => {
        nextStatusMap[entry.streamID] = {
          status: entry.status,
          createdAt: entry.createdAt
        };
      });
      setLatestStatusMap(nextStatusMap);
    } catch (loadError) {
      setError(toErrorMessage(loadError));
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    void loadStreams();
    // filters/scope should trigger reload
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [accessToken, scopeCompanyId, projectId, isActiveFilter]);

  useEffect(() => {
    if (typeof window === "undefined") {
      return;
    }
    window.localStorage.setItem(STORAGE_LAST_SECTION_KEY, "/streams");
  }, []);

  const favoriteMap = useMemo(() => {
    const map = new Map<
      number,
      { isPinned: boolean; sortOrder: number }
    >();
    favorites.forEach((fav) => {
      map.set(fav.stream.id, {
        isPinned: fav.is_pinned,
        sortOrder: fav.sort_order
      });
    });
    return map;
  }, [favorites]);

  const filteredStreams = useMemo(() => {
    const needle = search.trim().toLowerCase();

    const filtered = streams.filter((stream) => {
      const streamStatus = latestStatusMap[stream.id]?.status;

      if (statusFilter !== "all" && streamStatus !== statusFilter) {
        return false;
      }

      if (!needle) {
        return true;
      }

      return (
        stream.name.toLowerCase().includes(needle) ||
        String(stream.id).includes(needle) ||
        String(stream.project_id).includes(needle)
      );
    });

    return [...filtered].sort((a, b) => {
      const favA = favoriteMap.get(a.id);
      const favB = favoriteMap.get(b.id);
      const pinnedA = favA?.isPinned ?? false;
      const pinnedB = favB?.isPinned ?? false;
      if (pinnedA && !pinnedB) return -1;
      if (!pinnedA && pinnedB) return 1;
      if (pinnedA && pinnedB) {
        return (favA?.sortOrder ?? 0) - (favB?.sortOrder ?? 0);
      }
      const favOnlyA = favA != null;
      const favOnlyB = favB != null;
      if (favOnlyA && !favOnlyB) return -1;
      if (!favOnlyA && favOnlyB) return 1;
      return a.id - b.id;
    });
  }, [latestStatusMap, search, statusFilter, streams, favoriteMap]);

  const handleToggleFavorite = async (stream: Stream) => {
    if (!accessToken || !scopeCompanyId) return;
    const isFav = favoriteMap.has(stream.id);
    setBusyFavoriteStreamID(stream.id);
    try {
      if (isFav) {
        await apiRequest(
          `/companies/${scopeCompanyId}/streams/${stream.id}/favorite`,
          { method: "DELETE", accessToken }
        );
      } else {
        await apiRequest(
          `/companies/${scopeCompanyId}/streams/${stream.id}/favorite`,
          { method: "POST", accessToken }
        );
      }
      void loadStreams();
    } catch {
      setError(toErrorMessage(new Error("Не удалось изменить избранное")));
    } finally {
      setBusyFavoriteStreamID(null);
    }
  };

  const handleTogglePin = async (stream: Stream) => {
    if (!accessToken || !scopeCompanyId) return;
    const fav = favoriteMap.get(stream.id);
    const isPinned = fav?.isPinned ?? false;
    setBusyFavoriteStreamID(stream.id);
    try {
      if (isPinned) {
        await apiRequest(
          `/companies/${scopeCompanyId}/streams/${stream.id}/pin`,
          { method: "DELETE", accessToken }
        );
      } else {
        await apiRequest(
          `/companies/${scopeCompanyId}/streams/${stream.id}/pin`,
          { method: "POST", accessToken, body: {} }
        );
      }
      void loadStreams();
    } catch {
      setError(toErrorMessage(new Error("Не удалось изменить закрепление")));
    } finally {
      setBusyFavoriteStreamID(null);
    }
  };

  const handleRunCheck = async (stream: Stream) => {
    if (!accessToken || !scopeCompanyId || isViewer) {
      return;
    }

    setBusyStreamID(stream.id);
    setRunCheckError(null);
    setRunCheckSuccess(null);

    try {
      const response = await apiRequest<EnqueueCheckJobResponse>(
        `/companies/${scopeCompanyId}/streams/${stream.id}/check`,
        {
          method: "POST",
          accessToken,
        }
      );

      setRunCheckSuccess(`Check job #${response.job.id} queued for stream #${stream.id}.`);
      void loadStreams();
    } catch (runError) {
      setRunCheckError(toErrorMessage(runError));
    } finally {
      setBusyStreamID(null);
    }
  };

  const openCreateDialog = () => {
    setEditingStream(null);
    setFormName("");
    setFormURL("");
    setFormProjectID(projectId || "");
    setFormIsActive(true);
    setFormError(null);
    setIsFormOpen(true);
  };

  const openEditDialog = (stream: Stream) => {
    setEditingStream(stream);
    setFormName(stream.name);
    setFormURL(stream.url);
    setFormProjectID(String(stream.project_id));
    setFormIsActive(stream.is_active);
    setFormError(null);
    setIsFormOpen(true);
  };

  const closeDialog = () => {
    setIsFormOpen(false);
    setEditingStream(null);
    setFormError(null);
  };

  const handleSubmitStream = async () => {
    if (!accessToken || !scopeCompanyId || isViewer) {
      return;
    }
    const parsedProjectID = Number.parseInt(formProjectID, 10);
    if (!Number.isFinite(parsedProjectID) || parsedProjectID <= 0) {
      setFormError("Выберите проект.");
      return;
    }
    if (!formName.trim() || !formURL.trim()) {
      setFormError("Заполните название и URL m3u8.");
      return;
    }

    setIsFormSubmitting(true);
    setFormError(null);
    try {
      if (editingStream) {
        const payload: StreamPatchRequest = {
          name: formName.trim(),
          url: formURL.trim(),
          is_active: formIsActive
        };
        await apiRequest(`/companies/${scopeCompanyId}/streams/${editingStream.id}`, {
          method: "PATCH",
          accessToken,
          body: payload
        });
      } else {
        const payload: StreamCreateRequest = {
          project_id: parsedProjectID,
          name: formName.trim(),
          url: formURL.trim(),
          is_active: formIsActive
        };
        await apiRequest(`/companies/${scopeCompanyId}/streams`, {
          method: "POST",
          accessToken,
          body: payload
        });
      }
      closeDialog();
      await loadStreams();
    } catch (submitError) {
      setFormError(toErrorMessage(submitError));
    } finally {
      setIsFormSubmitting(false);
    }
  };

  const handleDeleteStream = async (stream: Stream) => {
    if (!accessToken || !scopeCompanyId || isViewer) {
      return;
    }
    if (!window.confirm(`Удалить поток «${stream.name}»?`)) {
      return;
    }
    setBusyStreamID(stream.id);
    try {
      await apiRequest(`/companies/${scopeCompanyId}/streams/${stream.id}`, {
        method: "DELETE",
        accessToken
      });
      await loadStreams();
    } catch (deleteError) {
      setError(toErrorMessage(deleteError));
    } finally {
      setBusyStreamID(null);
    }
  };

  return (
    <section className="panel">
      <header className="page-header compact">
        <h2 className="page-title">Мониторинг потоков</h2>
        <p className="page-note">
          CRUD потоков, статусы, избранное/закрепление и ручной запуск проверки.
        </p>
        <div style={{ marginTop: "8px" }}>
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

      <div className="filters-grid streams-v2-filters">
        <label className="form-field" htmlFor="streams-search">
          <span>Поиск</span>
          <input
            id="streams-search"
            value={search}
            onChange={(event) => setSearch(event.target.value)}
            placeholder="Название, ID потока или проекта"
            disabled={!scopeCompanyId || isLoading}
          />
        </label>

        <label className="form-field" htmlFor="streams-project-filter">
          <span>Проект</span>
          <select
            id="streams-project-filter"
            value={projectId}
            onChange={(event) => setProjectId(event.target.value)}
            disabled={!scopeCompanyId || isLoading}
          >
            <option value="">Все проекты</option>
            {projects.map((project) => (
              <option key={project.id} value={project.id}>
                {project.name} ({project.id})
              </option>
            ))}
          </select>
        </label>

        <label className="form-field" htmlFor="streams-active-filter">
          <span>Активен</span>
          <select
            id="streams-active-filter"
            value={isActiveFilter}
            onChange={(event) =>
              setIsActiveFilter(event.target.value as IsActiveFilter)
            }
            disabled={!scopeCompanyId || isLoading}
          >
            <option value="all">Все</option>
            <option value="true">Активные</option>
            <option value="false">Неактивные</option>
          </select>
        </label>

        <label className="form-field" htmlFor="streams-status-filter">
          <span>Статус</span>
          <select
            id="streams-status-filter"
            value={statusFilter}
            onChange={(event) => setStatusFilter(event.target.value as StatusFilter)}
            disabled={!scopeCompanyId || isLoading}
          >
            <option value="all">Все</option>
            <option value="OK">OK</option>
            <option value="WARN">WARN</option>
            <option value="FAIL">FAIL</option>
          </select>
        </label>
      </div>

      {isViewer ? (
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ duration: 0.24, ease: "easeOut" }}
        >
          <StatePanel>Роль «Зритель» — только просмотр. Запуск проверок недоступен.</StatePanel>
        </motion.div>
      ) : null}
      {runCheckSuccess ? (
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ duration: 0.24, ease: "easeOut" }}
        >
          <StatePanel>{runCheckSuccess}</StatePanel>
        </motion.div>
      ) : null}
      {runCheckError ? (
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ duration: 0.24, ease: "easeOut" }}
        >
          <StatePanel kind="error">{runCheckError}</StatePanel>
        </motion.div>
      ) : null}
      {error ? (
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ duration: 0.24, ease: "easeOut" }}
        >
          <StatePanel kind="error">{error}</StatePanel>
        </motion.div>
      ) : null}
      {isLoading ? (
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ duration: 0.2, ease: "easeOut" }}
          style={{ marginTop: "12px" }}
        >
          <SkeletonBlock lines={7} />
        </motion.div>
      ) : null}

      {!isLoading && !error && scopeCompanyId && filteredStreams.length === 0 ? (
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

      {!isLoading && !error && scopeCompanyId && filteredStreams.length > 0 ? (
        <motion.div
          className="table-wrap"
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ duration: 0.28, ease: "easeOut" }}
          style={{ marginTop: "12px" }}
        >
          <table>
            <thead>
              <tr>
                <th aria-label="Избранное и закрепление" />
                <th>ID</th>
                <th>Название</th>
                <th>Проект</th>
                <th>Статус</th>
                <th>Последняя проверка</th>
                <th>Активен</th>
                <th>Обновлён</th>
                <th>Действия</th>
              </tr>
            </thead>
            <tbody>
              {filteredStreams.map((stream) => {
                const latestStatus = latestStatusMap[stream.id]?.status ?? null;
                const lastCheckAt = latestStatusMap[stream.id]?.createdAt ?? null;
                const fav = favoriteMap.get(stream.id);
                const isPinned = fav?.isPinned ?? false;
                const isFavorite = fav != null;
                const busyFav = busyFavoriteStreamID === stream.id;

                return (
                  <tr
                    key={stream.id}
                    className={isPinned ? "stream-row-pinned" : undefined}
                  >
                    <td className="fav-pin-cell">
                      <button
                        type="button"
                        className="icon-btn"
                        onClick={() => void handleToggleFavorite(stream)}
                        disabled={isViewer || busyFav}
                        aria-pressed={isFavorite}
                        aria-label={isFavorite ? "Убрать из избранного" : "В избранное"}
                        title={isFavorite ? "Убрать из избранного" : "В избранное"}
                      >
                        ⭐
                      </button>
                      <button
                        type="button"
                        className="icon-btn"
                        onClick={() => void handleTogglePin(stream)}
                        disabled={isViewer || busyFav}
                        aria-pressed={isPinned}
                        aria-label={isPinned ? "Открепить" : "Закрепить"}
                        title={isPinned ? "Открепить" : "Закрепить"}
                      >
                        📌
                      </button>
                    </td>
                    <td>{stream.id}</td>
                    <td>
                      {isPinned ? (
                        <span className="pin-indicator" aria-hidden>📌 </span>
                      ) : null}
                      <Link className="stream-link" href={`/streams/${stream.id}`}>
                        {stream.name}
                      </Link>
                    </td>
                    <td>{stream.project_id}</td>
                    <td>
                      {latestStatus ? (
                        <StatusBadge status={latestStatus} />
                      ) : (
                        <span className="status-muted">Нет данных</span>
                      )}
                    </td>
                    <td>{formatTimestamp(lastCheckAt)}</td>
                    <td>{stream.is_active ? "Да" : "Нет"}</td>
                    <td>{formatTimestamp(stream.updated_at)}</td>
                    <td>
                      <div className="stream-actions">
                        <Link className="stream-link" href={`/watch?streamId=${stream.id}`}>
                          Смотреть
                        </Link>
                        <AppButton
                          type="button"
                          variant="secondary"
                          disabled={isViewer || busyStreamID === stream.id}
                          onClick={() => {
                            void handleRunCheck(stream);
                          }}
                          aria-label={busyStreamID === stream.id ? "В очереди" : `Запустить проверку: ${stream.name}`}
                        >
                          {busyStreamID === stream.id ? "В очереди…" : "Проверить сейчас"}
                        </AppButton>
                        <AppButton
                          type="button"
                          variant="secondary"
                          disabled={isViewer}
                          onClick={() => openEditDialog(stream)}
                        >
                          Редактировать
                        </AppButton>
                        <AppButton
                          type="button"
                          variant="danger"
                          disabled={isViewer || busyStreamID === stream.id}
                          onClick={() => {
                            void handleDeleteStream(stream);
                          }}
                        >
                          Удалить
                        </AppButton>
                      </div>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </motion.div>
      ) : null}

      {isFormOpen ? (
        <div className="overlay-backdrop" role="dialog" aria-modal="true">
          <div className="overlay-card">
            <h3>{editingStream ? "Редактировать поток" : "Добавить поток"}</h3>
            <div className="overlay-grid">
              <label className="form-field" htmlFor="stream-name">
                <span>Название</span>
                <input
                  id="stream-name"
                  value={formName}
                  onChange={(event) => setFormName(event.target.value)}
                  placeholder="Например: Main camera #1"
                />
              </label>
              <label className="form-field" htmlFor="stream-url">
                <span>URL m3u8</span>
                <input
                  id="stream-url"
                  value={formURL}
                  onChange={(event) => setFormURL(event.target.value)}
                  placeholder="https://example.com/live.m3u8"
                />
              </label>
              <label className="form-field" htmlFor="stream-project">
                <span>Проект</span>
                <select
                  id="stream-project"
                  value={formProjectID}
                  onChange={(event) => setFormProjectID(event.target.value)}
                >
                  <option value="">Выберите проект</option>
                  {projects.map((project) => (
                    <option key={project.id} value={project.id}>
                      {project.name} ({project.id})
                    </option>
                  ))}
                </select>
              </label>
              <label className="form-field form-check" htmlFor="stream-active">
                <input
                  id="stream-active"
                  type="checkbox"
                  checked={formIsActive}
                  onChange={(event) => setFormIsActive(event.target.checked)}
                />
                <span>Активный поток</span>
              </label>
            </div>
            {formError ? <StatePanel kind="error">{formError}</StatePanel> : null}
            <div className="overlay-actions">
              <AppButton
                type="button"
                variant="secondary"
                onClick={closeDialog}
                disabled={isFormSubmitting}
              >
                Отмена
              </AppButton>
              <AppButton
                type="button"
                onClick={() => {
                  void handleSubmitStream();
                }}
                disabled={isFormSubmitting}
              >
                {isFormSubmitting ? "Сохраняем…" : "Сохранить"}
              </AppButton>
            </div>
          </div>
        </div>
      ) : null}
    </section>
  );
}
