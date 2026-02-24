"use client";

import Hls from "hls.js";
import { motion } from "framer-motion";
import { useSearchParams } from "next/navigation";
import { useEffect, useMemo, useRef, useState } from "react";

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
  EmbedWhitelistItem,
  EnqueueCheckJobResponse,
  Project,
  Stream
} from "@/lib/api/types";

type StatusFilter = "all" | CheckStatus | "none";

const STORAGE_LAST_SECTION_KEY = "core.lastSection";
const STORAGE_LAST_STREAM_ID_KEY = "core.lastStreamId";

function normalizeStatus(value: string): CheckStatus | null {
  if (value === "OK" || value === "WARN" || value === "FAIL") {
    return value;
  }
  return null;
}

function formatTimestamp(value: string | null): string {
  if (!value) {
    return "—";
  }
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString();
}

function extractHost(rawURL: string): string | null {
  try {
    const parsed = new URL(rawURL);
    return parsed.hostname.toLowerCase();
  } catch {
    return null;
  }
}

function isDomainAllowed(host: string, domains: string[]): boolean {
  const normalized = host.toLowerCase();
  return domains.some((domain) => normalized === domain || normalized.endsWith(`.${domain}`));
}

function buildEmbedSrc(rawURL: string): string | null {
  try {
    const parsed = new URL(rawURL);
    if (parsed.hostname.includes("youtube.com")) {
      const videoID = parsed.searchParams.get("v");
      if (videoID) {
        return `https://www.youtube.com/embed/${videoID}`;
      }
    }
    return parsed.toString();
  } catch {
    return null;
  }
}

export default function WatchPage() {
  const searchParams = useSearchParams();
  const { user, accessToken, activeCompanyId } = useAuth();
  const scopeCompanyId = resolveCompanyScope(user, activeCompanyId);
  const isViewer = user?.role === "viewer";

  const videoRef = useRef<HTMLVideoElement | null>(null);
  const hlsRef = useRef<Hls | null>(null);

  const [projects, setProjects] = useState<Project[]>([]);
  const [streams, setStreams] = useState<Stream[]>([]);
  const [selectedStreamId, setSelectedStreamId] = useState<number | null>(null);
  const [selectedStatus, setSelectedStatus] = useState<CheckStatus | null>(null);
  const [selectedLastCheckAt, setSelectedLastCheckAt] = useState<string | null>(null);
  const [latestStatusMap, setLatestStatusMap] = useState<Record<number, CheckStatus | null>>({});
  const [allowedDomains, setAllowedDomains] = useState<string[]>([]);

  const [projectFilter, setProjectFilter] = useState<string>("");
  const [statusFilter, setStatusFilter] = useState<StatusFilter>("all");
  const [search, setSearch] = useState("");

  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [playerError, setPlayerError] = useState<string | null>(null);
  const [checkMessage, setCheckMessage] = useState<string | null>(null);
  const [isChecking, setIsChecking] = useState(false);

  const selectedStream = useMemo(
    () => streams.find((item) => item.id === selectedStreamId) ?? null,
    [streams, selectedStreamId]
  );

  const cleanupPlayer = () => {
    if (hlsRef.current) {
      hlsRef.current.destroy();
      hlsRef.current = null;
    }
    if (videoRef.current) {
      videoRef.current.pause();
      videoRef.current.removeAttribute("src");
      videoRef.current.load();
    }
  };

  const loadSelectedStatus = async (streamId: number) => {
    if (!accessToken || !scopeCompanyId) {
      setSelectedStatus(null);
      setSelectedLastCheckAt(null);
      return;
    }
    try {
      const response = await apiRequest<{ items: CheckResult[] }>(
        `/companies/${scopeCompanyId}/streams/${streamId}/check-results?limit=1`,
        { accessToken }
      );
      const latest = response.items?.[0];
      setSelectedStatus(latest ? normalizeStatus(latest.status) : null);
      setSelectedLastCheckAt(latest?.created_at ?? null);
    } catch {
      setSelectedStatus(null);
      setSelectedLastCheckAt(null);
    }
  };

  const loadData = async () => {
    if (!accessToken || !scopeCompanyId) {
      setProjects([]);
      setStreams([]);
      setSelectedStreamId(null);
      return;
    }

    setIsLoading(true);
    setError(null);
    try {
      const [projectResponse, streamResponse, whitelistResponse] = await Promise.all([
        apiRequest<{ items: Project[] }>(
          `/companies/${scopeCompanyId}/projects?limit=200`,
          { accessToken }
        ),
        apiRequest<{ items: Stream[] }>(`/companies/${scopeCompanyId}/streams?limit=500`, {
          accessToken
        }),
        apiRequest<{ items: EmbedWhitelistItem[] }>(
          `/companies/${scopeCompanyId}/embed-whitelist`,
          { accessToken }
        )
      ]);

      const loadedStreams = Array.isArray(streamResponse.items) ? streamResponse.items : [];
      const loadedDomains = (whitelistResponse.items ?? [])
        .filter((item) => item.enabled)
        .map((item) => item.domain.toLowerCase());
      setProjects(Array.isArray(projectResponse.items) ? projectResponse.items : []);
      setStreams(loadedStreams);
      setAllowedDomains(loadedDomains);

      const checks = await Promise.all(
        loadedStreams.map(async (stream) => {
          try {
            const resultResponse = await apiRequest<{ items: CheckResult[] }>(
              `/companies/${scopeCompanyId}/streams/${stream.id}/check-results?limit=1`,
              { accessToken }
            );
            const latest = resultResponse.items?.[0];
            return {
              streamId: stream.id,
              status: latest ? normalizeStatus(latest.status) : null
            };
          } catch {
            return { streamId: stream.id, status: null };
          }
        })
      );
      const nextMap: Record<number, CheckStatus | null> = {};
      checks.forEach((item) => {
        nextMap[item.streamId] = item.status;
      });
      setLatestStatusMap(nextMap);

      const streamIdParam = Number.parseInt(searchParams.get("streamId") ?? "", 10);
      if (Number.isFinite(streamIdParam) && loadedStreams.some((item) => item.id === streamIdParam)) {
        setSelectedStreamId(streamIdParam);
        void loadSelectedStatus(streamIdParam);
      } else if (loadedStreams.length > 0) {
        setSelectedStreamId(loadedStreams[0].id);
        void loadSelectedStatus(loadedStreams[0].id);
      } else {
        setSelectedStreamId(null);
      }
    } catch (loadError) {
      setError(toErrorMessage(loadError));
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    void loadData();
    // dependent on auth/scope only
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [accessToken, scopeCompanyId]);

  useEffect(() => {
    if (!selectedStream) {
      cleanupPlayer();
      return;
    }

    setPlayerError(null);
    const streamURL = (selectedStream.source_url || selectedStream.url || "").trim();
    if (!streamURL) {
      setPlayerError("URL потока пустой.");
      cleanupPlayer();
      return;
    }

    if (selectedStream.source_type === "EMBED") {
      cleanupPlayer();
      const host = extractHost(streamURL);
      if (!host || !isDomainAllowed(host, allowedDomains)) {
        setPlayerError("Embed запрещён: домен не разрешён в whitelist.");
      }
      if (typeof window !== "undefined") {
        window.localStorage.setItem(STORAGE_LAST_SECTION_KEY, "/watch");
        window.localStorage.setItem(STORAGE_LAST_STREAM_ID_KEY, String(selectedStream.id));
      }
      return;
    }

    const video = videoRef.current;
    if (!video) {
      return;
    }

    cleanupPlayer();

    const handleVideoError = () => {
      setPlayerError("Не удалось воспроизвести поток (проверьте CORS/404/сеть).");
    };
    video.addEventListener("error", handleVideoError);

    if (video.canPlayType("application/vnd.apple.mpegurl")) {
      video.src = streamURL;
      void video.play().catch(() => {
        setPlayerError("Браузер отклонил autoplay. Нажмите play вручную.");
      });
    } else if (Hls.isSupported()) {
      const hls = new Hls({
        maxBufferLength: 30,
        enableWorker: true
      });
      hlsRef.current = hls;
      hls.loadSource(streamURL);
      hls.attachMedia(video);
      hls.on(Hls.Events.ERROR, (_, data) => {
        if (data.fatal) {
          setPlayerError("Ошибка HLS (timeout, CORS или недоступный плейлист).");
        }
      });
      void video.play().catch(() => {
        setPlayerError("Браузер отклонил autoplay. Нажмите play вручную.");
      });
    } else {
      setPlayerError("В этом браузере HLS не поддерживается.");
    }

    if (typeof window !== "undefined") {
      window.localStorage.setItem(STORAGE_LAST_SECTION_KEY, "/watch");
      window.localStorage.setItem(STORAGE_LAST_STREAM_ID_KEY, String(selectedStream.id));
    }

    return () => {
      video.removeEventListener("error", handleVideoError);
      cleanupPlayer();
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [allowedDomains, selectedStream?.id, selectedStream?.source_type, selectedStream?.source_url, selectedStream?.url]);

  const filteredStreams = useMemo(() => {
    const needle = search.trim().toLowerCase();
    return streams.filter((stream) => {
      if (projectFilter && String(stream.project_id) !== projectFilter) {
        return false;
      }
      if (statusFilter !== "all") {
        const streamStatus = latestStatusMap[stream.id] ?? null;
        if (statusFilter === "none" && streamStatus !== null) {
          return false;
        }
        if (statusFilter !== "none" && streamStatus !== statusFilter) {
          return false;
        }
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
  }, [latestStatusMap, projectFilter, search, statusFilter, streams]);

  const handleSelectStream = (stream: Stream) => {
    setSelectedStreamId(stream.id);
    setCheckMessage(null);
    void loadSelectedStatus(stream.id);
  };

  const handleRunCheck = async () => {
    if (!selectedStreamId || !accessToken || !scopeCompanyId || isViewer) {
      return;
    }
    setIsChecking(true);
    setCheckMessage(null);
    try {
      const response = await apiRequest<EnqueueCheckJobResponse>(
        `/companies/${scopeCompanyId}/streams/${selectedStreamId}/check`,
        {
          method: "POST",
          accessToken
        }
      );
      setCheckMessage(`Проверка поставлена в очередь: job #${response.job.id}.`);
      await loadSelectedStatus(selectedStreamId);
      void loadData();
    } catch (runError) {
      setCheckMessage(toErrorMessage(runError));
    } finally {
      setIsChecking(false);
    }
  };

  return (
    <section className="panel">
      <header className="page-header compact">
        <h2 className="page-title">СМОТРЕТЬ</h2>
        <p className="page-note">Операторский экран просмотра HLS-потоков.</p>
      </header>

      {!scopeCompanyId ? (
        <StatePanel>Выберите компанию в шапке, чтобы загрузить режим просмотра.</StatePanel>
      ) : null}
      {error ? <StatePanel kind="error">{error}</StatePanel> : null}
      {checkMessage ? <StatePanel>{checkMessage}</StatePanel> : null}
      {isLoading ? <SkeletonBlock lines={8} /> : null}

      {!isLoading && !error && scopeCompanyId ? (
        <motion.div
          className="watch-layout"
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ duration: 0.24, ease: "easeOut" }}
        >
          <div className="watch-main">
            <div className="watch-player-wrap">
              {selectedStream?.source_type === "EMBED" ? (
                (() => {
                  const iframeSrc = buildEmbedSrc(selectedStream.source_url || selectedStream.url);
                  const host = iframeSrc ? extractHost(iframeSrc) : null;
                  if (!iframeSrc || !host || !isDomainAllowed(host, allowedDomains)) {
                    return <StatePanel kind="error">Embed запрещён или URL некорректен.</StatePanel>;
                  }
                  return (
                    <iframe
                      className="watch-embed-frame"
                      src={iframeSrc}
                      title={`Embed: ${selectedStream.name}`}
                      allow="autoplay; encrypted-media; fullscreen; picture-in-picture"
                      referrerPolicy="strict-origin-when-cross-origin"
                      allowFullScreen
                    />
                  );
                })()
              ) : (
                <video ref={videoRef} controls playsInline muted className="watch-player" />
              )}
            </div>
            {playerError ? <StatePanel kind="error">{playerError}</StatePanel> : null}
          </div>

          <aside className="watch-sidebar">
            <div className="watch-filters">
              <label className="form-field" htmlFor="watch-search">
                <span>Поиск</span>
                <input
                  id="watch-search"
                  value={search}
                  onChange={(event) => setSearch(event.target.value)}
                  placeholder="Название, ID потока или проекта"
                />
              </label>
              <label className="form-field" htmlFor="watch-project">
                <span>Проект</span>
                <select
                  id="watch-project"
                  value={projectFilter}
                  onChange={(event) => setProjectFilter(event.target.value)}
                >
                  <option value="">Все проекты</option>
                  {projects.map((project) => (
                    <option key={project.id} value={project.id}>
                      {project.name} ({project.id})
                    </option>
                  ))}
                </select>
              </label>
              <label className="form-field" htmlFor="watch-status">
                <span>Статус</span>
                <select
                  id="watch-status"
                  value={statusFilter}
                  onChange={(event) => setStatusFilter(event.target.value as StatusFilter)}
                >
                  <option value="all">Все</option>
                  <option value="OK">OK</option>
                  <option value="WARN">WARN</option>
                  <option value="FAIL">FAIL</option>
                  <option value="none">Нет данных</option>
                </select>
              </label>
            </div>

            {filteredStreams.length === 0 ? (
              <StatePanel>Потоки по фильтрам не найдены.</StatePanel>
            ) : (
              <div className="watch-stream-list">
                {filteredStreams.map((stream) => (
                  <button
                    key={stream.id}
                    type="button"
                    className={`watch-stream-item ${stream.id === selectedStreamId ? "is-active" : ""}`}
                    onClick={() => handleSelectStream(stream)}
                  >
                    <strong>{stream.name}</strong>
                    <span>ID {stream.id}</span>
                    <span>{stream.source_type}</span>
                  </button>
                ))}
              </div>
            )}
          </aside>

          <aside className="watch-status">
            <h3>Статус потока</h3>
            <p>{selectedStream ? selectedStream.name : "Поток не выбран"}</p>
            <div>
              {selectedStatus ? (
                <StatusBadge status={selectedStatus} />
              ) : (
                <span className="status-muted">Нет данных</span>
              )}
            </div>
            <p>Последняя проверка: {formatTimestamp(selectedLastCheckAt)}</p>
            <AppButton
              type="button"
              variant="secondary"
              disabled={!selectedStream || selectedStream.source_type === "EMBED" || isViewer || isChecking}
              onClick={() => {
                void handleRunCheck();
              }}
            >
              {isChecking ? "Ставим в очередь…" : "Проверить сейчас"}
            </AppButton>
          </aside>
        </motion.div>
      ) : null}
    </section>
  );
}
