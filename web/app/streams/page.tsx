"use client";

import Link from "next/link";
import { useEffect, useMemo, useState } from "react";

type Company = {
  id: number;
  name: string;
};

type Project = {
  id: number;
  company_id: number;
  name: string;
};

type Stream = {
  id: number;
  company_id: number;
  project_id: number;
  name: string;
  is_active: boolean;
  updated_at: string;
};

type IsActiveFilterValue = "all" | "true" | "false";

const API_BASE_PATH = "/api/v1";

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

export default function StreamsPage() {
  const [companies, setCompanies] = useState<Company[]>([]);
  const [companiesLoading, setCompaniesLoading] = useState<boolean>(true);
  const [companiesError, setCompaniesError] = useState<string | null>(null);

  const [companyId, setCompanyId] = useState<string>("");

  const [projects, setProjects] = useState<Project[]>([]);
  const [projectsLoading, setProjectsLoading] = useState<boolean>(false);
  const [projectsError, setProjectsError] = useState<string | null>(null);
  const [projectId, setProjectId] = useState<string>("");

  const [isActiveFilter, setIsActiveFilter] =
    useState<IsActiveFilterValue>("all");

  const [streams, setStreams] = useState<Stream[]>([]);
  const [streamsLoading, setStreamsLoading] = useState<boolean>(false);
  const [streamsError, setStreamsError] = useState<string | null>(null);

  useEffect(() => {
    const abortController = new AbortController();

    setCompaniesLoading(true);
    setCompaniesError(null);

    fetchJson<unknown>("/companies", abortController.signal)
      .then((payload) => {
        setCompanies(extractItems<Company>(payload));
      })
      .catch((error: unknown) => {
        if (abortController.signal.aborted) {
          return;
        }

        const message =
          error instanceof Error ? error.message : "Failed to load companies";
        setCompaniesError(message);
      })
      .finally(() => {
        if (!abortController.signal.aborted) {
          setCompaniesLoading(false);
        }
      });

    return () => abortController.abort();
  }, []);

  useEffect(() => {
    setProjectId("");
    setProjects([]);
    setProjectsError(null);

    if (!companyId) {
      setProjectsLoading(false);
      return;
    }

    const abortController = new AbortController();

    setProjectsLoading(true);

    fetchJson<unknown>(
      `/companies/${companyId}/projects`,
      abortController.signal
    )
      .then((payload) => {
        setProjects(extractItems<Project>(payload));
      })
      .catch((error: unknown) => {
        if (abortController.signal.aborted) {
          return;
        }

        const message =
          error instanceof Error ? error.message : "Failed to load projects";
        setProjectsError(message);
      })
      .finally(() => {
        if (!abortController.signal.aborted) {
          setProjectsLoading(false);
        }
      });

    return () => abortController.abort();
  }, [companyId]);

  const streamQuery = useMemo(() => {
    const params = new URLSearchParams();

    if (projectId) {
      params.set("project_id", projectId);
    }

    if (isActiveFilter !== "all") {
      params.set("is_active", isActiveFilter);
    }

    return params.toString();
  }, [projectId, isActiveFilter]);

  useEffect(() => {
    setStreams([]);
    setStreamsError(null);

    if (!companyId) {
      setStreamsLoading(false);
      return;
    }

    const abortController = new AbortController();
    const querySuffix = streamQuery ? `?${streamQuery}` : "";

    setStreamsLoading(true);

    fetchJson<unknown>(
      `/companies/${companyId}/streams${querySuffix}`,
      abortController.signal
    )
      .then((payload) => {
        setStreams(extractItems<Stream>(payload));
      })
      .catch((error: unknown) => {
        if (abortController.signal.aborted) {
          return;
        }

        const message =
          error instanceof Error ? error.message : "Failed to load streams";
        setStreamsError(message);
      })
      .finally(() => {
        if (!abortController.signal.aborted) {
          setStreamsLoading(false);
        }
      });

    return () => abortController.abort();
  }, [companyId, streamQuery]);

  return (
    <section className="panel">
      <div className="page-header">
        <h2 className="page-title">Streams</h2>
        <p className="page-note">
          Streams are loaded only after selecting a company.
        </p>
      </div>

      <div className="filters-grid">
        <label className="form-field" htmlFor="company-filter">
          <span>Company</span>
          <select
            id="company-filter"
            value={companyId}
            onChange={(event) => setCompanyId(event.target.value)}
            disabled={companiesLoading}
          >
            <option value="">Select company</option>
            {companies.map((company) => (
              <option key={company.id} value={company.id}>
                {company.name} ({company.id})
              </option>
            ))}
          </select>
        </label>

        <label className="form-field" htmlFor="project-filter">
          <span>Project</span>
          <select
            id="project-filter"
            value={projectId}
            onChange={(event) => setProjectId(event.target.value)}
            disabled={!companyId || projectsLoading}
          >
            <option value="">All projects</option>
            {projects.map((project) => (
              <option key={project.id} value={project.id}>
                {project.name} ({project.id})
              </option>
            ))}
          </select>
        </label>

        <label className="form-field" htmlFor="is-active-filter">
          <span>Active status</span>
          <select
            id="is-active-filter"
            value={isActiveFilter}
            onChange={(event) =>
              setIsActiveFilter(event.target.value as IsActiveFilterValue)
            }
            disabled={!companyId}
          >
            <option value="all">All</option>
            <option value="true">Active only</option>
            <option value="false">Inactive only</option>
          </select>
        </label>
      </div>

      {companiesLoading ? (
        <p className="state state-info">Loading companies...</p>
      ) : null}
      {companiesError ? (
        <p className="state state-error">Failed to load companies: {companiesError}</p>
      ) : null}
      {!companiesLoading && !companiesError && companies.length === 0 ? (
        <p className="state state-info">No companies available.</p>
      ) : null}
      {projectsError ? (
        <p className="state state-error">Failed to load projects: {projectsError}</p>
      ) : null}
      {companyId && projectsLoading ? (
        <p className="state state-info">Loading projects...</p>
      ) : null}
      {companyId &&
      !projectsLoading &&
      !projectsError &&
      projects.length === 0 ? (
        <p className="state state-info">No projects found for selected company.</p>
      ) : null}

      {!companyId ? (
        <p className="state state-info">
          Select a company to request stream data.
        </p>
      ) : null}

      {companyId && streamsLoading ? (
        <p className="state state-info">Loading streams...</p>
      ) : null}

      {companyId && streamsError ? (
        <p className="state state-error">Failed to load streams: {streamsError}</p>
      ) : null}

      {companyId && !streamsLoading && !streamsError && streams.length === 0 ? (
        <p className="state state-info">No streams found for selected filters.</p>
      ) : null}

      {companyId && !streamsLoading && !streamsError && streams.length > 0 ? (
        <div className="table-wrap">
          <table>
            <thead>
              <tr>
                <th>ID</th>
                <th>Name</th>
                <th>Project ID</th>
                <th>Is active</th>
                <th>Updated at</th>
              </tr>
            </thead>
            <tbody>
              {streams.map((stream) => (
                <tr key={stream.id}>
                  <td>{stream.id}</td>
                  <td>
                    <Link
                      className="stream-link"
                      href={`/streams/${stream.id}?companyId=${encodeURIComponent(companyId)}`}
                    >
                      {stream.name}
                    </Link>
                  </td>
                  <td>{stream.project_id}</td>
                  <td>{stream.is_active ? "true" : "false"}</td>
                  <td>{formatTimestamp(stream.updated_at)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : null}
    </section>
  );
}
