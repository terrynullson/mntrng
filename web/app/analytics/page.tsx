"use client";

import type { FormEvent } from "react";
import { useEffect, useMemo, useState } from "react";

type Company = {
  id: number;
  name: string;
};

type Stream = {
  id: number;
  company_id: number;
  project_id: number;
  name: string;
  is_active: boolean;
};

type CheckStatus = "OK" | "WARN" | "FAIL";

type AtomicChecks = {
  playlist?: CheckStatus;
  freshness?: CheckStatus;
  segments?: CheckStatus;
  declared_bitrate?: CheckStatus;
  effective_bitrate?: CheckStatus;
  freeze?: CheckStatus;
  blackframe?: CheckStatus;
};

type CheckResult = {
  id: number;
  company_id: number;
  stream_id: number;
  status: CheckStatus;
  checks?: AtomicChecks;
  created_at: string;
};

type StatusFilter = "all" | CheckStatus;

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

function parseDateInput(value: string): string | null {
  if (!value) {
    return null;
  }
  const parsedDate = new Date(value);
  if (Number.isNaN(parsedDate.getTime())) {
    return null;
  }
  return parsedDate.toISOString();
}

function normalizeStatus(status: string): CheckStatus | null {
  if (status === "OK" || status === "WARN" || status === "FAIL") {
    return status;
  }
  return null;
}

function formatCheckStatus(status: string | undefined): string {
  const normalized = normalizeStatus(status ?? "");
  return normalized ?? "-";
}

export default function AnalyticsPage() {
  const [companies, setCompanies] = useState<Company[]>([]);
  const [companiesLoading, setCompaniesLoading] = useState<boolean>(true);
  const [companiesError, setCompaniesError] = useState<string | null>(null);

  const [streams, setStreams] = useState<Stream[]>([]);
  const [streamsLoading, setStreamsLoading] = useState<boolean>(false);
  const [streamsError, setStreamsError] = useState<string | null>(null);

  const [companyId, setCompanyId] = useState<string>("");
  const [streamId, setStreamId] = useState<string>("");
  const [fromValue, setFromValue] = useState<string>("");
  const [toValue, setToValue] = useState<string>("");
  const [statusFilter, setStatusFilter] = useState<StatusFilter>("all");

  const [results, setResults] = useState<CheckResult[]>([]);
  const [resultsLoading, setResultsLoading] = useState<boolean>(false);
  const [resultsError, setResultsError] = useState<string | null>(null);
  const [hasRequestedResults, setHasRequestedResults] = useState<boolean>(false);
  const [queryVersion, setQueryVersion] = useState<number>(0);

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
    setStreamId("");
    setStreams([]);
    setStreamsError(null);
    setResults([]);
    setResultsError(null);
    setHasRequestedResults(false);

    if (!companyId) {
      setStreamsLoading(false);
      return;
    }

    const abortController = new AbortController();

    setStreamsLoading(true);

    fetchJson<unknown>(`/companies/${companyId}/streams?limit=200`, abortController.signal)
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
  }, [companyId]);

  useEffect(() => {
    setResults([]);
    setResultsError(null);
    setHasRequestedResults(false);
  }, [streamId]);

  useEffect(() => {
    if (!hasRequestedResults || queryVersion === 0) {
      return;
    }

    if (!companyId || !streamId) {
      setResults([]);
      setResultsLoading(false);
      setResultsError("Select company and stream before applying filters.");
      return;
    }

    const fromIso = parseDateInput(fromValue);
    const toIso = parseDateInput(toValue);

    if (fromValue && !fromIso) {
      setResults([]);
      setResultsLoading(false);
      setResultsError("Invalid from date.");
      return;
    }

    if (toValue && !toIso) {
      setResults([]);
      setResultsLoading(false);
      setResultsError("Invalid to date.");
      return;
    }

    if (fromIso && toIso && new Date(fromIso).getTime() > new Date(toIso).getTime()) {
      setResults([]);
      setResultsLoading(false);
      setResultsError("From date must be earlier than To date.");
      return;
    }

    const abortController = new AbortController();
    const queryParams = new URLSearchParams();
    queryParams.set("limit", "200");

    if (fromIso) {
      queryParams.set("from", fromIso);
    }
    if (toIso) {
      queryParams.set("to", toIso);
    }
    if (statusFilter !== "all") {
      queryParams.set("status", statusFilter);
    }

    setResultsLoading(true);
    setResultsError(null);

    fetchJson<unknown>(
      `/companies/${companyId}/streams/${streamId}/check-results?${queryParams.toString()}`,
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
        setResults(sortedByDate);
      })
      .catch((error: unknown) => {
        if (abortController.signal.aborted) {
          return;
        }
        const message =
          error instanceof Error
            ? error.message
            : "Failed to load check results history";
        setResultsError(message);
      })
      .finally(() => {
        if (!abortController.signal.aborted) {
          setResultsLoading(false);
        }
      });

    return () => abortController.abort();
  }, [
    companyId,
    streamId,
    fromValue,
    toValue,
    statusFilter,
    hasRequestedResults,
    queryVersion
  ]);

  const summary = useMemo(() => {
    const counts: Record<CheckStatus, number> = {
      OK: 0,
      WARN: 0,
      FAIL: 0
    };

    results.forEach((result) => {
      const normalized = normalizeStatus(result.status);
      if (normalized) {
        counts[normalized] += 1;
      }
    });

    return counts;
  }, [results]);

  const handleApplyFilters = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setHasRequestedResults(true);
    setQueryVersion((previous) => previous + 1);
  };

  return (
    <section className="panel">
      <div className="page-header">
        <h2 className="page-title">Analytics</h2>
        <p className="page-note">
          Stream status history with period filters and aggregated status counts.
        </p>
      </div>

      <form className="analytics-filters" onSubmit={handleApplyFilters}>
        <label className="form-field" htmlFor="analytics-company-filter">
          <span>Company</span>
          <select
            id="analytics-company-filter"
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

        <label className="form-field" htmlFor="analytics-stream-filter">
          <span>Stream</span>
          <select
            id="analytics-stream-filter"
            value={streamId}
            onChange={(event) => setStreamId(event.target.value)}
            disabled={!companyId || streamsLoading}
          >
            <option value="">Select stream</option>
            {streams.map((stream) => (
              <option key={stream.id} value={stream.id}>
                {stream.name} ({stream.id})
              </option>
            ))}
          </select>
        </label>

        <label className="form-field" htmlFor="analytics-from">
          <span>From</span>
          <input
            id="analytics-from"
            type="datetime-local"
            value={fromValue}
            onChange={(event) => setFromValue(event.target.value)}
          />
        </label>

        <label className="form-field" htmlFor="analytics-to">
          <span>To</span>
          <input
            id="analytics-to"
            type="datetime-local"
            value={toValue}
            onChange={(event) => setToValue(event.target.value)}
          />
        </label>

        <label className="form-field" htmlFor="analytics-status-filter">
          <span>Status</span>
          <select
            id="analytics-status-filter"
            value={statusFilter}
            onChange={(event) =>
              setStatusFilter(event.target.value as StatusFilter)
            }
          >
            <option value="all">All</option>
            <option value="OK">OK</option>
            <option value="WARN">WARN</option>
            <option value="FAIL">FAIL</option>
          </select>
        </label>

        <div className="analytics-actions">
          <button
            type="submit"
            className="button-primary"
            disabled={resultsLoading || !companyId || !streamId}
          >
            {resultsLoading ? "Loading..." : "Apply filters"}
          </button>
        </div>
      </form>

      {companiesLoading ? (
        <p className="state state-info">Loading companies...</p>
      ) : null}
      {companiesError ? (
        <p className="state state-error">Failed to load companies: {companiesError}</p>
      ) : null}
      {!companiesLoading && !companiesError && companies.length === 0 ? (
        <p className="state state-info">No companies available.</p>
      ) : null}

      {companyId && streamsLoading ? (
        <p className="state state-info">Loading streams...</p>
      ) : null}
      {streamsError ? (
        <p className="state state-error">Failed to load streams: {streamsError}</p>
      ) : null}
      {companyId && !streamsLoading && !streamsError && streams.length === 0 ? (
        <p className="state state-info">No streams found for selected company.</p>
      ) : null}

      {!streamId ? (
        <p className="state state-info">
          Select company and stream, then apply filters to load analytics.
        </p>
      ) : null}

      {resultsError ? (
        <p className="state state-error">Failed to load analytics: {resultsError}</p>
      ) : null}
      {resultsLoading ? (
        <p className="state state-info">Loading analytics history...</p>
      ) : null}

      {hasRequestedResults && !resultsLoading && !resultsError ? (
        <div className="summary-grid">
          <div className="summary-card">
            <span>OK</span>
            <strong>{summary.OK}</strong>
          </div>
          <div className="summary-card">
            <span>WARN</span>
            <strong>{summary.WARN}</strong>
          </div>
          <div className="summary-card">
            <span>FAIL</span>
            <strong>{summary.FAIL}</strong>
          </div>
        </div>
      ) : null}

      {hasRequestedResults &&
      !resultsLoading &&
      !resultsError &&
      results.length === 0 ? (
        <p className="state state-info">
          No check-results found for selected period and filters.
        </p>
      ) : null}

      {hasRequestedResults &&
      !resultsLoading &&
      !resultsError &&
      results.length > 0 ? (
        <div className="table-wrap">
          <table>
            <thead>
              <tr>
                <th>Created at</th>
                <th>Status</th>
                <th>Playlist</th>
                <th>Freshness</th>
                <th>Segments</th>
                <th>Effective bitrate</th>
                <th>Freeze</th>
                <th>Blackframe</th>
              </tr>
            </thead>
            <tbody>
              {results.map((result) => (
                <tr key={result.id}>
                  <td>{formatTimestamp(result.created_at)}</td>
                  <td>{formatCheckStatus(result.status)}</td>
                  <td>{formatCheckStatus(result.checks?.playlist)}</td>
                  <td>{formatCheckStatus(result.checks?.freshness)}</td>
                  <td>{formatCheckStatus(result.checks?.segments)}</td>
                  <td>{formatCheckStatus(result.checks?.effective_bitrate)}</td>
                  <td>{formatCheckStatus(result.checks?.freeze)}</td>
                  <td>{formatCheckStatus(result.checks?.blackframe)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : null}
    </section>
  );
}
