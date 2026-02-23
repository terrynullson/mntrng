"use client";

import type Hls from "hls.js";
import Link from "next/link";
import { motion } from "framer-motion";
import { useParams, useRouter } from "next/navigation";
import { useEffect, useMemo, useRef, useState } from "react";

import { useAuth } from "@/components/auth/auth-provider";
import { AppButton } from "@/components/ui/app-button";
import { SkeletonBlock } from "@/components/ui/skeleton";
import { StatePanel } from "@/components/ui/state-panel";
import { StatusBadge } from "@/components/ui/status-badge";
import { ApiError, apiRequest, toErrorMessage } from "@/lib/api/client";
import { resolveCompanyScope } from "@/lib/auth/tenant-scope";
import type { AiIncident, CheckResult, CheckStatus, Stream } from "@/lib/api/types";

const ATOMIC_CHECK_ORDER: Array<keyof CheckResult["checks"]> = [
  "playlist",
  "freshness",
  "segments",
  "declared_bitrate",
  "effective_bitrate",
  "freeze",
  "blackframe"
];

function formatTimestamp(timestamp: string | null): string {
  if (!timestamp) {
    return "-";
  }
  const parsed = new Date(timestamp);
  return Number.isNaN(parsed.getTime()) ? timestamp : parsed.toLocaleString();
}

function normalizeStatus(status: string): CheckStatus | null {
  if (status === "OK" || status === "WARN" || status === "FAIL") {
    return status;
  }
  return null;
}

export default function StreamDetailsPageV2() {
  const params = useParams<{ streamId: string }>();
  const router = useRouter();
  const streamID = Array.isArray(params.streamId)
    ? params.streamId[0]
    : params.streamId;

  const { user, accessToken, activeCompanyId } = useAuth();
  const scopeCompanyId = resolveCompanyScope(user, activeCompanyId);

  const [stream, setStream] = useState<Stream | null>(null);
  const [latestResult, setLatestResult] = useState<CheckResult | null>(null);

  const [isLoading, setIsLoading] = useState<boolean>(false);
  const [error, setError] = useState<string | null>(null);

  const [aiIncident, setAiIncident] = useState<AiIncident | null>(null);
  const [aiIncidentLoading, setAiIncidentLoading] = useState<boolean>(false);
  const [aiIncidentError, setAiIncidentError] = useState<string | null>(null);

  const [companyStreams, setCompanyStreams] = useState<Stream[]>([]);
  const [isLoadingStreams, setIsLoadingStreams] = useState<boolean>(false);
  const [streamsError, setStreamsError] = useState<string | null>(null);

  const [playerError, setPlayerError] = useState<string | null>(null);
  const [fallbackURL, setFallbackURL] = useState<string | null>(null);

  const videoRef = useRef<HTMLVideoElement | null>(null);
  const playerWrapRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    if (!accessToken || !scopeCompanyId || !streamID) {
      setStream(null);
      setLatestResult(null);
      setIsLoading(false);
      return;
    }

    const abortController = new AbortController();

    setIsLoading(true);
    setError(null);

    Promise.all([
      apiRequest<Stream>(`/companies/${scopeCompanyId}/streams/${streamID}`, {
        accessToken,
        signal: abortController.signal
      }),
      apiRequest<{ items: CheckResult[] }>(
        `/companies/${scopeCompanyId}/streams/${streamID}/check-results?limit=1`,
        {
          accessToken,
          signal: abortController.signal
        }
      )
    ])
      .then(([streamPayload, resultPayload]) => {
        setStream(streamPayload);
        setLatestResult(resultPayload.items?.[0] ?? null);
      })
      .catch((loadError) => {
        if (abortController.signal.aborted) {
          return;
        }
        setError(toErrorMessage(loadError));
      })
      .finally(() => {
        if (!abortController.signal.aborted) {
          setIsLoading(false);
        }
      });

    return () => abortController.abort();
  }, [accessToken, scopeCompanyId, streamID]);

  useEffect(() => {
    if (!accessToken || !scopeCompanyId) {
      setCompanyStreams([]);
      setIsLoadingStreams(false);
      setStreamsError(null);
      return;
    }

    const abort = new AbortController();
    setIsLoadingStreams(true);
    setStreamsError(null);

    apiRequest<{ items: Stream[] }>(
      `/companies/${scopeCompanyId}/streams?limit=200`,
      { accessToken, signal: abort.signal }
    )
      .then((response) => {
        if (abort.signal.aborted) return;
        const items = Array.isArray(response.items) ? response.items : [];
        setCompanyStreams(items);
      })
      .catch((loadError) => {
        if (abort.signal.aborted) return;
        setStreamsError(toErrorMessage(loadError));
      })
      .finally(() => {
        if (!abort.signal.aborted) {
          setIsLoadingStreams(false);
        }
      });

    return () => abort.abort();
  }, [accessToken, scopeCompanyId]);

  useEffect(() => {
    if (
      !accessToken ||
      !scopeCompanyId ||
      !streamID ||
      !latestResult?.job_id
    ) {
      setAiIncident(null);
      setAiIncidentLoading(false);
      setAiIncidentError(null);
      return;
    }

    const jobId = latestResult.job_id;
    const abort = new AbortController();
    setAiIncidentLoading(true);
    setAiIncidentError(null);
    setAiIncident(null);

    apiRequest<AiIncident>(
      `/companies/${scopeCompanyId}/streams/${streamID}/check-jobs/${jobId}/ai-incident`,
      { accessToken, signal: abort.signal }
    )
      .then((data) => {
        if (!abort.signal.aborted) {
          setAiIncident(data);
        }
      })
      .catch((err) => {
        if (abort.signal.aborted) return;
        if (err instanceof ApiError && err.status === 404) {
          setAiIncident(null);
          return;
        }
        setAiIncidentError(toErrorMessage(err));
      })
      .finally(() => {
        if (!abort.signal.aborted) {
          setAiIncidentLoading(false);
        }
      });

    return () => abort.abort();
  }, [accessToken, scopeCompanyId, streamID, latestResult?.job_id]);

  useEffect(() => {
    const video = videoRef.current;
    if (!video || !stream?.url) {
      return;
    }

    setPlayerError(null);
    setFallbackURL(null);

    const canPlayNative = video.canPlayType("application/vnd.apple.mpegurl") !== "";
    if (canPlayNative) {
      video.src = stream.url;
      return () => {
        video.removeAttribute("src");
        video.load();
      };
    }

    let hlsInstance: Hls | null = null;
    let disposed = false;

    const setupFallback = async () => {
      try {
        const hlsModule = await import("hls.js");
        const HlsCtor = hlsModule.default;

        if (disposed || !videoRef.current) {
          return;
        }

        if (!HlsCtor.isSupported()) {
          setFallbackURL(stream.url);
          setPlayerError("Native HLS is unavailable in this browser.");
          return;
        }

        hlsInstance = new HlsCtor();
        hlsInstance.loadSource(stream.url);
        hlsInstance.attachMedia(videoRef.current);
      } catch {
        if (!disposed) {
          setFallbackURL(stream.url);
          setPlayerError("Failed to initialize hls.js fallback.");
        }
      }
    };

    void setupFallback();

    return () => {
      disposed = true;
      if (hlsInstance) {
        hlsInstance.destroy();
      }
      if (videoRef.current) {
        videoRef.current.removeAttribute("src");
        videoRef.current.load();
      }
    };
  }, [stream?.url]);

  const atomicRows = useMemo(() => {
    if (!latestResult) {
      return [] as Array<{ key: string; status: CheckStatus }>;
    }

    return ATOMIC_CHECK_ORDER.reduce<Array<{ key: string; status: CheckStatus }>>(
      (accumulator, key) => {
        const value = latestResult.checks?.[key];
        if (!value) {
          return accumulator;
        }

        const normalized = normalizeStatus(value);
        if (!normalized) {
          return accumulator;
        }

        accumulator.push({ key, status: normalized });
        return accumulator;
      },
      []
    );
  }, [latestResult]);

  return (
    <section className="panel">
      <header className="page-header compact">
        <h2 className="page-title">Stream Player</h2>
        <p className="page-note">
          <Link className="stream-link" href="/streams">
            Back to streams
          </Link>
        </p>
        {scopeCompanyId ? (
          <div className="page-note">
            <label className="form-field" htmlFor="stream-switcher">
              <span>Switch stream</span>
              <select
                id="stream-switcher"
                value={streamID ?? ""}
                onChange={(event) => {
                  const targetId = event.target.value;
                  if (targetId && targetId !== streamID) {
                    router.push(`/streams/${targetId}`);
                  }
                }}
                disabled={isLoadingStreams || companyStreams.length === 0}
              >
                <option value="">Select stream</option>
                {companyStreams.map((item) => (
                  <option key={item.id} value={item.id}>
                    {item.name} ({item.id})
                  </option>
                ))}
              </select>
            </label>
          </div>
        ) : null}
      </header>

      {!scopeCompanyId ? (
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ duration: 0.24, ease: "easeOut" }}
        >
          <StatePanel>Select company scope in topbar to open stream details.</StatePanel>
        </motion.div>
      ) : null}

      {streamsError ? (
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ duration: 0.24, ease: "easeOut" }}
        >
          <StatePanel kind="error">{streamsError}</StatePanel>
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

      {!isLoading && !error && stream ? (
        <motion.div
          className="stream-details-grid"
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ duration: 0.28, ease: "easeOut" }}
          style={{ marginTop: "12px" }}
        >
          <article className="player-card">
            <h3 className="section-title">{stream.name}</h3>
            <p className="section-meta">
              Stream #{stream.id} | Project #{stream.project_id} | Active: {" "}
              {stream.is_active ? "true" : "false"}
            </p>

            <div ref={playerWrapRef} className="stream-player-wrap">
              <video
                ref={videoRef}
                className="stream-player"
                controls
                playsInline
                muted
              />
              <AppButton
                type="button"
                variant="secondary"
                className="stream-player-fullscreen-btn"
                onClick={() => {
                  const el = playerWrapRef.current;
                  if (el?.requestFullscreen) {
                    el.requestFullscreen();
                  }
                }}
              >
                Fullscreen
              </AppButton>
            </div>

            {playerError ? (
              <motion.div
                initial={{ opacity: 0 }}
                animate={{ opacity: 1 }}
                transition={{ duration: 0.24, ease: "easeOut" }}
              >
                <StatePanel>
                  {playerError}{" "}
                  {fallbackURL ? (
                    <a href={fallbackURL} target="_blank" rel="noreferrer">
                      Open stream URL
                    </a>
                  ) : null}
                </StatePanel>
              </motion.div>
            ) : null}
          </article>

          <article className="status-card">
            <h3 className="section-title">Latest Status</h3>

            {!latestResult ? (
              <motion.div
                initial={{ opacity: 0 }}
                animate={{ opacity: 1 }}
                transition={{ duration: 0.24, ease: "easeOut" }}
              >
                <StatePanel>No check results available yet.</StatePanel>
              </motion.div>
            ) : (
              <>
                <p className="status-row">
                  Status: <StatusBadge status={latestResult.status} />
                </p>
                <p className="status-row">
                  Last check at: {formatTimestamp(latestResult.created_at)}
                </p>

                {atomicRows.length > 0 ? (
                  <motion.div
                    className="atomic-checks"
                    initial={{ opacity: 0 }}
                    animate={{ opacity: 1 }}
                    transition={{ duration: 0.24, ease: "easeOut" }}
                  >
                    <h4 className="section-title section-title-small">Atomic checks</h4>
                    <table>
                      <thead>
                        <tr>
                          <th>Check</th>
                          <th>Status</th>
                        </tr>
                      </thead>
                      <tbody>
                        {atomicRows.map((row) => (
                          <tr key={row.key}>
                            <td>{row.key}</td>
                            <td>
                              <StatusBadge status={row.status} />
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </motion.div>
                ) : null}
              </>
            )}
          </article>

          <article className="status-card">
            <h3 className="section-title">AI incident</h3>
            {aiIncidentLoading ? (
              <motion.div
                initial={{ opacity: 0 }}
                animate={{ opacity: 1 }}
                transition={{ duration: 0.2, ease: "easeOut" }}
              >
                <SkeletonBlock lines={3} />
              </motion.div>
            ) : aiIncidentError ? (
              <motion.div
                initial={{ opacity: 0 }}
                animate={{ opacity: 1 }}
                transition={{ duration: 0.24, ease: "easeOut" }}
              >
                <StatePanel kind="error">{aiIncidentError}</StatePanel>
              </motion.div>
            ) : aiIncident ? (
              <motion.div
                className="ai-incident-content"
                initial={{ opacity: 0 }}
                animate={{ opacity: 1 }}
                transition={{ duration: 0.24, ease: "easeOut" }}
              >
                <p className="section-meta">
                  <strong>Cause:</strong> {aiIncident.cause}
                </p>
                <p className="section-meta">
                  <strong>Summary:</strong> {aiIncident.summary}
                </p>
              </motion.div>
            ) : (
              <motion.div
                initial={{ opacity: 0 }}
                animate={{ opacity: 1 }}
                transition={{ duration: 0.24, ease: "easeOut" }}
              >
                <StatePanel>No AI incident analysis for this check.</StatePanel>
              </motion.div>
            )}
          </article>
        </motion.div>
      ) : null}

      {!isLoading && !error && scopeCompanyId && !stream && streamID ? (
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ duration: 0.24, ease: "easeOut" }}
          style={{ marginTop: "12px" }}
        >
          <StatePanel>Stream not found or not accessible.</StatePanel>
        </motion.div>
      ) : null}
    </section>
  );
}
