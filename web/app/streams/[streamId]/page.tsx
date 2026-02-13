"use client";

import Link from "next/link";
import { useParams, useSearchParams } from "next/navigation";
import { useEffect, useMemo, useRef, useState } from "react";
import type Hls from "hls.js";

type Stream = {
  id: number;
  company_id: number;
  project_id: number;
  name: string;
  url: string;
  is_active: boolean;
  updated_at: string;
};

type AtomicCheckStatus = "OK" | "WARN" | "FAIL";

type AtomicChecks = {
  playlist?: AtomicCheckStatus;
  freshness?: AtomicCheckStatus;
  segments?: AtomicCheckStatus;
  declared_bitrate?: AtomicCheckStatus;
  effective_bitrate?: AtomicCheckStatus;
  freeze?: AtomicCheckStatus;
  blackframe?: AtomicCheckStatus;
};

type CheckResult = {
  id: number;
  company_id: number;
  stream_id: number;
  status: AtomicCheckStatus;
  checks?: AtomicChecks;
  created_at: string;
};

const API_BASE_PATH = "/api/v1";

const ATOMIC_CHECK_ORDER: Array<keyof AtomicChecks> = [
  "playlist",
  "freshness",
  "segments",
  "declared_bitrate",
  "effective_bitrate",
  "freeze",
  "blackframe"
];

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}

function extractItems<T>(payload: unknown): T[] {
  if (Array.isArray(payload)) {
    return payload as T[];
  }

  if (isRecord(payload) && Array.isArray(payload.items)) {
    return payload.items as T[];
  }

  return [];
}

function buildApiErrorMessage(status: number, payload: unknown): string {
  if (isRecord(payload)) {
    const message = payload.message;
    const code = payload.code;
    if (typeof message === "string" && message.length > 0) {
      if (typeof code === "string" && code.length > 0) {
        return `${message} (${code})`;
      }
      return message;
    }
  }

  return `Request failed with status ${status}`;
}

async function fetchJson<T>(path: string, signal: AbortSignal): Promise<T> {
  const response = await fetch(`${API_BASE_PATH}${path}`, {
    method: "GET",
    headers: { Accept: "application/json" },
    cache: "no-store",
    signal
  });

  let payload: unknown = null;
  try {
    payload = (await response.json()) as T;
  } catch {
    payload = null;
  }

  if (!response.ok) {
    throw new Error(buildApiErrorMessage(response.status, payload));
  }

  return payload as T;
}

function formatTimestamp(timestamp: string): string {
  const parsedDate = new Date(timestamp);
  if (Number.isNaN(parsedDate.getTime())) {
    return timestamp;
  }

  return parsedDate.toLocaleString();
}

function normalizeStatus(status: string): AtomicCheckStatus | null {
  if (status === "OK" || status === "WARN" || status === "FAIL") {
    return status;
  }
  return null;
}

function statusBadgeClass(status: AtomicCheckStatus): string {
  if (status === "OK") {
    return "status-badge status-ok";
  }
  if (status === "WARN") {
    return "status-badge status-warn";
  }
  return "status-badge status-fail";
}

export default function StreamDetailsPage() {
  const params = useParams<{ streamId: string }>();
  const searchParams = useSearchParams();
  const streamId = Array.isArray(params.streamId)
    ? params.streamId[0]
    : params.streamId;
  const companyId = searchParams.get("companyId") ?? "";

  const [stream, setStream] = useState<Stream | null>(null);
  const [streamLoading, setStreamLoading] = useState<boolean>(true);
  const [streamError, setStreamError] = useState<string | null>(null);

  const [latestResult, setLatestResult] = useState<CheckResult | null>(null);
  const [resultsLoading, setResultsLoading] = useState<boolean>(true);
  const [resultsError, setResultsError] = useState<string | null>(null);
  const [resultsLoaded, setResultsLoaded] = useState<boolean>(false);

  const [playerError, setPlayerError] = useState<string | null>(null);
  const [fallbackUrl, setFallbackUrl] = useState<string | null>(null);
  const videoRef = useRef<HTMLVideoElement | null>(null);

  useEffect(() => {
    setStream(null);
    setStreamError(null);

    if (!companyId || !streamId) {
      setStreamLoading(false);
      return;
    }

    const abortController = new AbortController();
    setStreamLoading(true);

    fetchJson<Stream>(
      `/companies/${companyId}/streams/${streamId}`,
      abortController.signal
    )
      .then((payload) => {
        setStream(payload);
      })
      .catch((error: unknown) => {
        if (abortController.signal.aborted) {
          return;
        }
        const message =
          error instanceof Error ? error.message : "Failed to load stream";
        setStreamError(message);
      })
      .finally(() => {
        if (!abortController.signal.aborted) {
          setStreamLoading(false);
        }
      });

    return () => abortController.abort();
  }, [companyId, streamId]);

  useEffect(() => {
    setLatestResult(null);
    setResultsError(null);
    setResultsLoaded(false);

    if (!companyId || !streamId) {
      setResultsLoading(false);
      return;
    }

    const abortController = new AbortController();
    setResultsLoading(true);

    fetchJson<unknown>(
      `/companies/${companyId}/streams/${streamId}/check-results?limit=10`,
      abortController.signal
    )
      .then((payload) => {
        const items = extractItems<CheckResult>(payload);
        const sortedByDate = [...items].sort((left, right) => {
          return (
            new Date(right.created_at).getTime() -
            new Date(left.created_at).getTime()
          );
        });
        setLatestResult(sortedByDate[0] ?? null);
      })
      .catch((error: unknown) => {
        if (abortController.signal.aborted) {
          return;
        }
        const message =
          error instanceof Error
            ? error.message
            : "Failed to load check results";
        setResultsError(message);
      })
      .finally(() => {
        if (!abortController.signal.aborted) {
          setResultsLoading(false);
          setResultsLoaded(true);
        }
      });

    return () => abortController.abort();
  }, [companyId, streamId]);

  useEffect(() => {
    const video = videoRef.current;
    if (!video || !stream?.url) {
      return;
    }

    setPlayerError(null);
    setFallbackUrl(null);

    const canPlayNativeHls =
      video.canPlayType("application/vnd.apple.mpegurl") !== "";

    if (canPlayNativeHls) {
      video.src = stream.url;
      return () => {
        video.removeAttribute("src");
        video.load();
      };
    }

    let hlsInstance: Hls | null = null;
    let isDisposed = false;

    const setupHlsFallback = async () => {
      try {
        const hlsModule = await import("hls.js");
        const HlsCtor = hlsModule.default;

        if (isDisposed || !videoRef.current) {
          return;
        }

        if (!HlsCtor.isSupported()) {
          setFallbackUrl(stream.url);
          setPlayerError(
            "This browser does not support HLS playback. Open stream URL directly."
          );
          return;
        }

        hlsInstance = new HlsCtor();
        hlsInstance.loadSource(stream.url);
        hlsInstance.attachMedia(videoRef.current);
      } catch {
        if (!isDisposed) {
          setFallbackUrl(stream.url);
          setPlayerError("Failed to initialize HLS fallback player.");
        }
      }
    };

    void setupHlsFallback();

    return () => {
      isDisposed = true;
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
    if (!latestResult?.checks) {
      return [];
    }

    return ATOMIC_CHECK_ORDER.reduce<
      Array<{ key: keyof AtomicChecks; status: AtomicCheckStatus }>
    >((accumulator, key) => {
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
    }, []);
  }, [latestResult]);

  return (
    <section className="panel">
      <div className="page-header">
        <h2 className="page-title">Stream Player</h2>
        <p className="page-note">
          <Link href="/streams" className="stream-link">
            Back to streams list
          </Link>
        </p>
      </div>

      {!companyId ? (
        <p className="state state-error">
          Missing companyId in URL. Open this page from the Streams table.
        </p>
      ) : null}

      {streamLoading ? <p className="state state-info">Loading stream...</p> : null}
      {streamError ? (
        <p className="state state-error">Failed to load stream: {streamError}</p>
      ) : null}

      {stream && !streamLoading && !streamError ? (
        <div className="stream-details-grid">
          <div className="player-card">
            <h3 className="section-title">{stream.name}</h3>
            <p className="section-meta">
              Stream ID: {stream.id} | Project ID: {stream.project_id} | Active:{" "}
              {stream.is_active ? "true" : "false"}
            </p>

            <video
              ref={videoRef}
              className="stream-player"
              controls
              playsInline
              muted
            />

            {playerError ? (
              <p className="state state-info">
                {playerError}{" "}
                {fallbackUrl ? (
                  <a href={fallbackUrl} target="_blank" rel="noreferrer">
                    Open stream URL
                  </a>
                ) : null}
              </p>
            ) : null}
          </div>

          <div className="status-card">
            <h3 className="section-title">Latest Status</h3>

            {resultsLoading ? (
              <p className="state state-info">Loading latest check result...</p>
            ) : null}
            {resultsError ? (
              <p className="state state-error">
                Failed to load latest check result: {resultsError}
              </p>
            ) : null}
            {!resultsLoading &&
            !resultsError &&
            resultsLoaded &&
            !latestResult ? (
              <p className="state state-info">No check results available yet.</p>
            ) : null}

            {!resultsLoading && !resultsError && latestResult ? (
              <>
                <p className="status-row">
                  Status:{" "}
                  <span className={statusBadgeClass(latestResult.status)}>
                    {latestResult.status}
                  </span>
                </p>
                <p className="status-row">
                  Last check at: {formatTimestamp(latestResult.created_at)}
                </p>

                {atomicRows.length > 0 ? (
                  <div className="atomic-checks">
                    <h4 className="section-title section-title-small">
                      Atomic Checks
                    </h4>
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
                              <span className={statusBadgeClass(row.status)}>
                                {row.status}
                              </span>
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                ) : null}
              </>
            ) : null}
          </div>
        </div>
      ) : null}
    </section>
  );
}
