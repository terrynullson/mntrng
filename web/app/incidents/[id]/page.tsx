"use client";

import { useEffect, useMemo, useState } from "react";
import { motion } from "framer-motion";
import { useParams } from "next/navigation";
import Image from "next/image";

import { useAuth } from "@/components/auth/auth-provider";
import { SkeletonBlock } from "@/components/ui/skeleton";
import { StatePanel } from "@/components/ui/state-panel";
import { apiRequest, toErrorMessage } from "@/lib/api/client";
import { resolveCompanyScope } from "@/lib/auth/tenant-scope";
import type { Incident } from "@/lib/api/types";

function formatTimestamp(ts: string | null | undefined): string {
  if (!ts) return "—";
  const d = new Date(ts);
  return Number.isNaN(d.getTime()) ? ts : d.toLocaleString();
}

function diagLabel(code: Incident["diag_code"]): string {
  switch (code) {
    case "BLACKFRAME":
      return "Чёрный экран";
    case "FREEZE":
      return "Фриз";
    case "CAPTURE_FAIL":
      return "Не удалось получить кадр";
    case "UNKNOWN":
      return "Неизвестно";
    default:
      return "—";
  }
}

export default function IncidentDetailsPage() {
  const params = useParams<{ id: string }>();
  const incidentID = Array.isArray(params.id) ? params.id[0] : params.id;
  const { user, accessToken, activeCompanyId } = useAuth();
  const scopeCompanyId = resolveCompanyScope(user, activeCompanyId);

  const [incident, setIncident] = useState<Incident | null>(null);
  const [screenshotUrl, setScreenshotUrl] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!accessToken || !scopeCompanyId || !incidentID) {
      setIncident(null);
      setScreenshotUrl(null);
      return;
    }

    const abort = new AbortController();
    let objectUrl: string | null = null;
    setIsLoading(true);
    setError(null);

    (async () => {
      try {
        const data = await apiRequest<Incident>(
          `/companies/${scopeCompanyId}/incidents/${incidentID}`,
          { accessToken, signal: abort.signal }
        );
        if (abort.signal.aborted) return;
        setIncident(data);
        if (data.has_screenshot) {
          const response = await fetch(
            `/api/v1/companies/${scopeCompanyId}/incidents/${incidentID}/screenshot`,
            {
              method: "GET",
              headers: { Authorization: `Bearer ${accessToken}` },
              signal: abort.signal,
              cache: "no-store"
            }
          );
          if (response.ok) {
            const blob = await response.blob();
            objectUrl = URL.createObjectURL(blob);
            if (!abort.signal.aborted) {
              setScreenshotUrl(objectUrl);
            }
          } else {
            setScreenshotUrl(null);
          }
        } else {
          setScreenshotUrl(null);
        }
      } catch (loadError) {
        if (!abort.signal.aborted) {
          setError(toErrorMessage(loadError));
          setIncident(null);
          setScreenshotUrl(null);
        }
      } finally {
        if (!abort.signal.aborted) {
          setIsLoading(false);
        }
      }
    })();

    return () => {
      abort.abort();
      if (objectUrl) {
        URL.revokeObjectURL(objectUrl);
      }
    };
  }, [accessToken, scopeCompanyId, incidentID]);

  const diagDetails = useMemo(() => incident?.diag_details ?? null, [incident]);

  return (
    <section className="panel">
      <header className="page-header compact">
        <h2 className="page-title">Инцидент #{incidentID}</h2>
      </header>

      {isLoading ? (
        <motion.div initial={{ opacity: 0 }} animate={{ opacity: 1 }} transition={{ duration: 0.2 }}>
          <SkeletonBlock lines={7} />
        </motion.div>
      ) : null}

      {error ? (
        <motion.div initial={{ opacity: 0 }} animate={{ opacity: 1 }} transition={{ duration: 0.24 }}>
          <StatePanel kind="error">{error}</StatePanel>
        </motion.div>
      ) : null}

      {!isLoading && !error && incident ? (
        <motion.div className="stream-details-grid" initial={{ opacity: 0 }} animate={{ opacity: 1 }} transition={{ duration: 0.24 }}>
          <article className="status-card">
            <h3 className="section-title">Состояние</h3>
            <p className="status-row">Статус: {incident.status === "open" ? "Открыт" : "Закрыт"}</p>
            <p className="status-row">Серьёзность: {incident.severity === "fail" ? "FAIL" : "WARN"}</p>
            <p className="status-row">Диагноз: {diagLabel(incident.diag_code)}</p>
            <p className="status-row">Начало: {formatTimestamp(incident.started_at)}</p>
            <p className="status-row">Последнее событие: {formatTimestamp(incident.last_event_at)}</p>
            <p className="status-row">Скриншот: {formatTimestamp(incident.screenshot_taken_at)}</p>
          </article>

          <article className="status-card">
            <h3 className="section-title">Детали диагноза</h3>
            {diagDetails ? (
              <pre className="incident-diag-json">{JSON.stringify(diagDetails, null, 2)}</pre>
            ) : (
              <StatePanel>Детали пока отсутствуют.</StatePanel>
            )}
          </article>

          <article className="status-card">
            <h3 className="section-title">Скриншот инцидента</h3>
            {screenshotUrl ? (
              <Image
                className="incident-screenshot"
                src={screenshotUrl}
                alt="Скриншот инцидента"
                width={640}
                height={360}
                unoptimized
              />
            ) : (
              <StatePanel>Скриншот отсутствует.</StatePanel>
            )}
          </article>
        </motion.div>
      ) : null}
    </section>
  );
}
