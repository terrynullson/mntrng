"use client";

import Link from "next/link";
import { motion } from "framer-motion";
import { useEffect, useMemo, useState } from "react";

import { useAuth } from "@/components/auth/auth-provider";
import { SkeletonBlock } from "@/components/ui/skeleton";
import { StatePanel } from "@/components/ui/state-panel";
import { apiRequest, toErrorMessage } from "@/lib/api/client";
import { resolveCompanyScope } from "@/lib/auth/tenant-scope";
import type { IncidentListResponse, Stream } from "@/lib/api/types";

const STORAGE_LAST_SECTION_KEY = "core.lastSection";
const STORAGE_LAST_STREAM_ID_KEY = "core.lastStreamId";

type LaunchpadSummary = {
  ok: number;
  warn: number;
  fail: number;
};

export default function OverviewPage() {
  const { user, isReady, accessToken, activeCompanyId } = useAuth();
  const scopeCompanyId = resolveCompanyScope(user, activeCompanyId);

  const [summary, setSummary] = useState<LaunchpadSummary>({
    ok: 0,
    warn: 0,
    fail: 0
  });
  const [isLoadingSummary, setIsLoadingSummary] = useState(false);
  const [summaryError, setSummaryError] = useState<string | null>(null);
  const [lastSection, setLastSection] = useState<string | null>(null);
  const [lastStreamId, setLastStreamId] = useState<string | null>(null);

  useEffect(() => {
    if (typeof window === "undefined") {
      return;
    }
    const section = window.localStorage.getItem(STORAGE_LAST_SECTION_KEY);
    const streamId = window.localStorage.getItem(STORAGE_LAST_STREAM_ID_KEY);
    setLastSection(section);
    setLastStreamId(streamId);
  }, []);

  useEffect(() => {
    if (!accessToken || !scopeCompanyId) {
      setSummary({ ok: 0, warn: 0, fail: 0 });
      setSummaryError(null);
      return;
    }

    const loadSummary = async () => {
      setIsLoadingSummary(true);
      setSummaryError(null);
      try {
        const [streamsResponse, warnResponse, failResponse] = await Promise.all([
          apiRequest<{ items: Stream[] }>(
            `/companies/${scopeCompanyId}/streams?limit=500`,
            { accessToken }
          ),
          apiRequest<IncidentListResponse>(
            `/companies/${scopeCompanyId}/incidents?status=open&severity=warn&page=0&page_size=1`,
            { accessToken }
          ),
          apiRequest<IncidentListResponse>(
            `/companies/${scopeCompanyId}/incidents?status=open&severity=fail&page=0&page_size=1`,
            { accessToken }
          )
        ]);

        const totalStreams = Array.isArray(streamsResponse.items)
          ? streamsResponse.items.length
          : 0;
        const warn = Number.isFinite(warnResponse.total) ? warnResponse.total : 0;
        const fail = Number.isFinite(failResponse.total) ? failResponse.total : 0;
        const ok = Math.max(0, totalStreams - (warn + fail));
        setSummary({ ok, warn, fail });
      } catch (error) {
        setSummaryError(toErrorMessage(error));
      } finally {
        setIsLoadingSummary(false);
      }
    };

    void loadSummary();
  }, [accessToken, scopeCompanyId]);

  const continueLink = useMemo(() => {
    if (lastSection === "/watch" && lastStreamId) {
      return `/watch?streamId=${encodeURIComponent(lastStreamId)}`;
    }
    if (lastSection) {
      return lastSection;
    }
    return null;
  }, [lastSection, lastStreamId]);

  return (
    <section className="panel">
      <header className="page-header">
        <h2 className="page-title">Домашняя</h2>
        <p className="page-note">
          Быстрый запуск режима просмотра и мониторинга после входа.
        </p>
      </header>

      {!isReady ? (
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ duration: 0.2, ease: "easeOut" }}
          style={{ marginTop: "12px" }}
        >
          <SkeletonBlock lines={5} />
        </motion.div>
      ) : !user || !scopeCompanyId ? (
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ duration: 0.24, ease: "easeOut" }}
        >
          <StatePanel kind="error">
            Не удалось определить компанию. Проверьте контекст авторизации.
          </StatePanel>
        </motion.div>
      ) : (
        <>
          <motion.div
            className="core-launchpad-grid"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            transition={{ duration: 0.28, ease: "easeOut" }}
            style={{ marginTop: "12px" }}
          >
            <Link className="core-launchpad-tile" href="/watch">
              <h3>СМОТРЕТЬ</h3>
              <p>Операторский экран HLS-плеера с быстрым переключением потоков.</p>
            </Link>
            <Link className="core-launchpad-tile" href="/streams">
              <h3>МОНИТОРИНГ</h3>
              <p>Таблица потоков, статусы, ручные проверки и управление потоками.</p>
            </Link>
          </motion.div>

          <motion.div
            className="core-launchpad-meta"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            transition={{ duration: 0.24, ease: "easeOut" }}
          >
            <article className="core-continue-card">
              <h4>Продолжить</h4>
              {continueLink ? (
                <Link href={continueLink}>Открыть последний раздел</Link>
              ) : (
                <p>История пока пустая — выберите раздел сверху.</p>
              )}
            </article>
            <article className="core-summary-card">
              <h4>Сводка статусов</h4>
              {isLoadingSummary ? (
                <p>Загрузка…</p>
              ) : (
                <div className="core-summary-row">
                  <span className="core-chip core-chip-ok">OK: {summary.ok}</span>
                  <span className="core-chip core-chip-warn">WARN: {summary.warn}</span>
                  <span className="core-chip core-chip-fail">FAIL: {summary.fail}</span>
                </div>
              )}
              {summaryError ? <p className="core-summary-error">{summaryError}</p> : null}
            </article>
          </motion.div>
        </>
      )}
    </section>
  );
}
