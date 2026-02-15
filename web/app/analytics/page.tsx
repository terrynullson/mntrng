"use client";

import type { FormEvent } from "react";
import { useEffect, useMemo, useState } from "react";

import { useAuth } from "@/components/auth/auth-provider";
import { AppButton } from "@/components/ui/app-button";
import { SkeletonBlock } from "@/components/ui/skeleton";
import { StatePanel } from "@/components/ui/state-panel";
import { StatusBadge } from "@/components/ui/status-badge";
import { apiRequest, toErrorMessage } from "@/lib/api/client";
import { resolveCompanyScope } from "@/lib/auth/tenant-scope";
import type { CheckResult, CheckStatus, Stream } from "@/lib/api/types";

type StatusFilter = "all" | CheckStatus;

function parseDateInput(value: string): string | null {
  if (!value) {
    return null;
  }

  const parsed = new Date(value);
  return Number.isNaN(parsed.getTime()) ? null : parsed.toISOString();
}

function formatTimestamp(timestamp: string): string {
  const parsed = new Date(timestamp);
  return Number.isNaN(parsed.getTime()) ? timestamp : parsed.toLocaleString();
}

function normalizeStatus(status: string): CheckStatus | null {
  if (status === "OK" || status === "WARN" || status === "FAIL") {
    return status;
  }
  return null;
}

export default function AnalyticsPage() {
  const { user, accessToken, activeCompanyId } = useAuth();
  const scopeCompanyId = resolveCompanyScope(user, activeCompanyId);

  const [streams, setStreams] = useState<Stream[]>([]);
  const [streamID, setStreamID] = useState<string>("");
  const [fromValue, setFromValue] = useState<string>("");
  const [toValue, setToValue] = useState<string>("");
  const [statusFilter, setStatusFilter] = useState<StatusFilter>("all");

  const [results, setResults] = useState<CheckResult[]>([]);
  const [isLoadingStreams, setIsLoadingStreams] = useState<boolean>(false);
  const [isLoadingResults, setIsLoadingResults] = useState<boolean>(false);
  const [error, setError] = useState<string | null>(null);
  const [hasRequested, setHasRequested] = useState<boolean>(false);

  useEffect(() => {
    if (!accessToken || !scopeCompanyId) {
      setStreams([]);
      setStreamID("");
      setIsLoadingStreams(false);
      return;
    }

    const abortController = new AbortController();

    setIsLoadingStreams(true);
    setError(null);

    apiRequest<{ items: Stream[] }>(`/companies/${scopeCompanyId}/streams?limit=200`, {
      accessToken,
      signal: abortController.signal
    })
      .then((response) => {
        const loaded = Array.isArray(response.items) ? response.items : [];
        setStreams(loaded);
        setStreamID((current) => {
          if (current && loaded.some((stream) => String(stream.id) === current)) {
            return current;
          }
          return loaded[0] ? String(loaded[0].id) : "";
        });
      })
      .catch((loadError) => {
        if (abortController.signal.aborted) {
          return;
        }
        setError(toErrorMessage(loadError));
      })
      .finally(() => {
        if (!abortController.signal.aborted) {
          setIsLoadingStreams(false);
        }
      });

    return () => abortController.abort();
  }, [accessToken, scopeCompanyId]);

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

  const handleApply = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();

    if (!accessToken || !scopeCompanyId || !streamID) {
      setError("Select stream before requesting analytics.");
      return;
    }

    const fromISO = parseDateInput(fromValue);
    const toISO = parseDateInput(toValue);

    if (fromValue && !fromISO) {
      setError("Invalid from date.");
      return;
    }
    if (toValue && !toISO) {
      setError("Invalid to date.");
      return;
    }

    if (fromISO && toISO && new Date(fromISO).getTime() > new Date(toISO).getTime()) {
      setError("From date must be before To date.");
      return;
    }

    const params = new URLSearchParams();
    params.set("limit", "200");
    if (fromISO) {
      params.set("from", fromISO);
    }
    if (toISO) {
      params.set("to", toISO);
    }
    if (statusFilter !== "all") {
      params.set("status", statusFilter);
    }

    setHasRequested(true);
    setIsLoadingResults(true);
    setError(null);

    try {
      const response = await apiRequest<{ items: CheckResult[] }>(
        `/companies/${scopeCompanyId}/streams/${streamID}/check-results?${params.toString()}`,
        { accessToken }
      );

      const items = Array.isArray(response.items) ? response.items : [];
      const sorted = [...items].sort((left, right) => {
        return new Date(right.created_at).getTime() - new Date(left.created_at).getTime();
      });
      setResults(sorted);
    } catch (requestError) {
      setResults([]);
      setError(toErrorMessage(requestError));
    } finally {
      setIsLoadingResults(false);
    }
  };

  return (
    <section className="panel">
      <header className="page-header compact">
        <h2 className="page-title">Analytics</h2>
        <p className="page-note">
          History of stream check results with status summary by selected period.
        </p>
      </header>

      {!scopeCompanyId ? (
        <StatePanel>Select company scope in topbar to load analytics.</StatePanel>
      ) : null}

      <form className="analytics-filters" onSubmit={handleApply}>
        <label className="form-field" htmlFor="analytics-stream-filter">
          <span>Stream</span>
          <select
            id="analytics-stream-filter"
            value={streamID}
            onChange={(event) => setStreamID(event.target.value)}
            disabled={!scopeCompanyId || isLoadingStreams}
          >
            <option value="">Select stream</option>
            {streams.map((stream) => (
              <option key={stream.id} value={stream.id}>
                {stream.name} ({stream.id})
              </option>
            ))}
          </select>
        </label>

        <label className="form-field" htmlFor="analytics-from-filter">
          <span>From</span>
          <input
            id="analytics-from-filter"
            type="datetime-local"
            value={fromValue}
            onChange={(event) => setFromValue(event.target.value)}
            disabled={!scopeCompanyId || isLoadingResults}
          />
        </label>

        <label className="form-field" htmlFor="analytics-to-filter">
          <span>To</span>
          <input
            id="analytics-to-filter"
            type="datetime-local"
            value={toValue}
            onChange={(event) => setToValue(event.target.value)}
            disabled={!scopeCompanyId || isLoadingResults}
          />
        </label>

        <label className="form-field" htmlFor="analytics-status-filter">
          <span>Status</span>
          <select
            id="analytics-status-filter"
            value={statusFilter}
            onChange={(event) => setStatusFilter(event.target.value as StatusFilter)}
            disabled={!scopeCompanyId || isLoadingResults}
          >
            <option value="all">All</option>
            <option value="OK">OK</option>
            <option value="WARN">WARN</option>
            <option value="FAIL">FAIL</option>
          </select>
        </label>

        <div className="analytics-actions">
          <AppButton type="submit" disabled={!streamID || isLoadingResults}>
            {isLoadingResults ? "Loading..." : "Apply filters"}
          </AppButton>
        </div>
      </form>

      {isLoadingStreams ? <SkeletonBlock lines={4} /> : null}
      {error ? <StatePanel kind="error">{error}</StatePanel> : null}
      {scopeCompanyId && !isLoadingStreams && streams.length === 0 ? (
        <StatePanel>No streams available for selected company scope.</StatePanel>
      ) : null}

      {isLoadingResults ? <SkeletonBlock lines={6} /> : null}

      {hasRequested && !isLoadingResults && !error ? (
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

      {hasRequested && !isLoadingResults && !error && results.length === 0 ? (
        <StatePanel>No check-results for selected period and filters.</StatePanel>
      ) : null}

      {hasRequested && !isLoadingResults && !error && results.length > 0 ? (
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
                  <td>
                    {normalizeStatus(result.status) ? (
                      <StatusBadge status={result.status} />
                    ) : (
                      <span className="status-muted">-</span>
                    )}
                  </td>
                  <td>{result.checks?.playlist ?? "-"}</td>
                  <td>{result.checks?.freshness ?? "-"}</td>
                  <td>{result.checks?.segments ?? "-"}</td>
                  <td>{result.checks?.effective_bitrate ?? "-"}</td>
                  <td>{result.checks?.freeze ?? "-"}</td>
                  <td>{result.checks?.blackframe ?? "-"}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : null}
    </section>
  );
}
