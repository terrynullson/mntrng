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
  Stream
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
  const [latestStatusMap, setLatestStatusMap] = useState<LatestStatusMap>({});

  const [projectId, setProjectId] = useState<string>("");
  const [isActiveFilter, setIsActiveFilter] = useState<IsActiveFilter>("all");
  const [statusFilter, setStatusFilter] = useState<StatusFilter>("all");
  const [search, setSearch] = useState<string>("");

  const [isLoading, setIsLoading] = useState<boolean>(false);
  const [error, setError] = useState<string | null>(null);
  const [runCheckError, setRunCheckError] = useState<string | null>(null);
  const [runCheckSuccess, setRunCheckSuccess] = useState<string | null>(null);
  const [busyStreamID, setBusyStreamID] = useState<number | null>(null);

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

  const filteredStreams = useMemo(() => {
    const needle = search.trim().toLowerCase();

    return streams.filter((stream) => {
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
  }, [latestStatusMap, search, statusFilter, streams]);

  const handleRunCheck = async (stream: Stream) => {
    if (!accessToken || !scopeCompanyId || isViewer) {
      return;
    }

    setBusyStreamID(stream.id);
    setRunCheckError(null);
    setRunCheckSuccess(null);

    try {
      const response = await apiRequest<EnqueueCheckJobResponse>(
        `/companies/${scopeCompanyId}/streams/${stream.id}/check-jobs`,
        {
          method: "POST",
          accessToken,
          body: {
            planned_at: new Date().toISOString()
          }
        }
      );

      setRunCheckSuccess(`Check job #${response.job.id} queued for stream #${stream.id}.`);
    } catch (runError) {
      setRunCheckError(toErrorMessage(runError));
    } finally {
      setBusyStreamID(null);
    }
  };

  return (
    <section className="panel">
      <header className="page-header compact">
        <h2 className="page-title">Streams</h2>
        <p className="page-note">
          Tenant-scoped stream list with status badges and manual check trigger.
        </p>
      </header>

      {!scopeCompanyId ? (
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ duration: 0.24, ease: "easeOut" }}
        >
          <StatePanel>
            Select company scope in topbar to load streams.
          </StatePanel>
        </motion.div>
      ) : null}

      <div className="filters-grid streams-v2-filters">
        <label className="form-field" htmlFor="streams-search">
          <span>Search</span>
          <input
            id="streams-search"
            value={search}
            onChange={(event) => setSearch(event.target.value)}
            placeholder="Name, stream id, project id"
            disabled={!scopeCompanyId || isLoading}
          />
        </label>

        <label className="form-field" htmlFor="streams-project-filter">
          <span>Project</span>
          <select
            id="streams-project-filter"
            value={projectId}
            onChange={(event) => setProjectId(event.target.value)}
            disabled={!scopeCompanyId || isLoading}
          >
            <option value="">All projects</option>
            {projects.map((project) => (
              <option key={project.id} value={project.id}>
                {project.name} ({project.id})
              </option>
            ))}
          </select>
        </label>

        <label className="form-field" htmlFor="streams-active-filter">
          <span>Is active</span>
          <select
            id="streams-active-filter"
            value={isActiveFilter}
            onChange={(event) =>
              setIsActiveFilter(event.target.value as IsActiveFilter)
            }
            disabled={!scopeCompanyId || isLoading}
          >
            <option value="all">All</option>
            <option value="true">Active</option>
            <option value="false">Inactive</option>
          </select>
        </label>

        <label className="form-field" htmlFor="streams-status-filter">
          <span>Latest status</span>
          <select
            id="streams-status-filter"
            value={statusFilter}
            onChange={(event) => setStatusFilter(event.target.value as StatusFilter)}
            disabled={!scopeCompanyId || isLoading}
          >
            <option value="all">All</option>
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
          <StatePanel>Viewer role is read-only. Run check actions are disabled.</StatePanel>
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
          <StatePanel>No streams found for selected filters.</StatePanel>
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
                <th>ID</th>
                <th>Name</th>
                <th>Project</th>
                <th>Status</th>
                <th>Last check</th>
                <th>Is active</th>
                <th>Updated at</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {filteredStreams.map((stream) => {
                const latestStatus = latestStatusMap[stream.id]?.status ?? null;
                const lastCheckAt = latestStatusMap[stream.id]?.createdAt ?? null;

                return (
                  <tr key={stream.id}>
                    <td>{stream.id}</td>
                    <td>
                      <Link className="stream-link" href={`/streams/${stream.id}`}>
                        {stream.name}
                      </Link>
                    </td>
                    <td>{stream.project_id}</td>
                    <td>
                      {latestStatus ? (
                        <StatusBadge status={latestStatus} />
                      ) : (
                        <span className="status-muted">No data</span>
                      )}
                    </td>
                    <td>{formatTimestamp(lastCheckAt)}</td>
                    <td>{stream.is_active ? "true" : "false"}</td>
                    <td>{formatTimestamp(stream.updated_at)}</td>
                    <td>
                      <AppButton
                        type="button"
                        variant="secondary"
                        disabled={isViewer || busyStreamID === stream.id}
                        onClick={() => {
                          void handleRunCheck(stream);
                        }}
                        aria-label={busyStreamID === stream.id ? "Queueing check" : `Run check for stream ${stream.name}`}
                      >
                        {busyStreamID === stream.id ? "Queueing..." : "Run check"}
                      </AppButton>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </motion.div>
      ) : null}
    </section>
  );
}
