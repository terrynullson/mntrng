"use client";

import { useCallback, useEffect, useState } from "react";
import { apiRequest, toErrorMessage } from "@/lib/api/client";
import type {
  CheckResult,
  CheckStatus,
  Project,
  Stream,
  StreamWithFavorite
} from "@/lib/api/types";
import type { IsActiveFilter, LatestStatusMap } from "../types";

function normalizeStatus(value: string): CheckStatus | null {
  if (value === "OK" || value === "WARN" || value === "FAIL") {
    return value;
  }
  return null;
}

type UseStreamsDataParams = {
  accessToken: string | null;
  scopeCompanyId: number | null;
  projectId: string;
  isActiveFilter: IsActiveFilter;
};

export function useStreamsData({
  accessToken,
  scopeCompanyId,
  projectId,
  isActiveFilter
}: UseStreamsDataParams) {
  const [projects, setProjects] = useState<Project[]>([]);
  const [streams, setStreams] = useState<Stream[]>([]);
  const [favorites, setFavorites] = useState<StreamWithFavorite[]>([]);
  const [latestStatusMap, setLatestStatusMap] = useState<LatestStatusMap>({});
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    if (!accessToken || !scopeCompanyId) {
      setProjects([]);
      setStreams([]);
      setLatestStatusMap({});
      setIsLoading(false);
      return;
    }

    setIsLoading(true);
    setError(null);

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
  }, [accessToken, scopeCompanyId, projectId, isActiveFilter]);

  useEffect(() => {
    void load();
  }, [load]);

  return {
    projects,
    streams,
    favorites,
    latestStatusMap,
    isLoading,
    error,
    reload: load
  };
}
