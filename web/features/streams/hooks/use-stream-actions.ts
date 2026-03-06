"use client";

import { useCallback, useState } from "react";
import { apiRequest, toErrorMessage } from "@/lib/api/client";
import type {
  EnqueueCheckJobResponse,
  Project,
  Stream,
  StreamCreateRequest,
  StreamPatchRequest
} from "@/lib/api/types";

type UseStreamActionsParams = {
  accessToken: string | null;
  scopeCompanyId: number | null;
  isViewer: boolean;
  reload: () => Promise<void>;
  getProjects: () => Project[];
};

export function useStreamActions({
  accessToken,
  scopeCompanyId,
  isViewer,
  reload,
  getProjects
}: UseStreamActionsParams) {
  const [runCheckError, setRunCheckError] = useState<string | null>(null);
  const [runCheckSuccess, setRunCheckSuccess] = useState<string | null>(null);
  const [busyStreamID, setBusyStreamID] = useState<number | null>(null);
  const [busyFavoriteStreamID, setBusyFavoriteStreamID] = useState<number | null>(null);
  const [screenError, setScreenError] = useState<string | null>(null);

  const handleToggleFavorite = useCallback(
    async (stream: Stream, favoriteMap: Map<number, { isPinned: boolean; sortOrder: number }>) => {
      if (!accessToken || !scopeCompanyId) return;
      const isFav = favoriteMap.has(stream.id);
      setBusyFavoriteStreamID(stream.id);
      try {
        if (isFav) {
          await apiRequest(
            `/companies/${scopeCompanyId}/streams/${stream.id}/favorite`,
            { method: "DELETE", accessToken }
          );
        } else {
          await apiRequest(
            `/companies/${scopeCompanyId}/streams/${stream.id}/favorite`,
            { method: "POST", accessToken }
          );
        }
        void reload();
      } catch {
        setScreenError(toErrorMessage(new Error("Не удалось изменить избранное")));
      } finally {
        setBusyFavoriteStreamID(null);
      }
    },
    [accessToken, scopeCompanyId, reload]
  );

  const handleTogglePin = useCallback(
    async (
      stream: Stream,
      favoriteMap: Map<number, { isPinned: boolean; sortOrder: number }>
    ) => {
      if (!accessToken || !scopeCompanyId) return;
      const fav = favoriteMap.get(stream.id);
      const isPinned = fav?.isPinned ?? false;
      setBusyFavoriteStreamID(stream.id);
      try {
        if (isPinned) {
          await apiRequest(
            `/companies/${scopeCompanyId}/streams/${stream.id}/pin`,
            { method: "DELETE", accessToken }
          );
        } else {
          await apiRequest(
            `/companies/${scopeCompanyId}/streams/${stream.id}/pin`,
            { method: "POST", accessToken, body: {} }
          );
        }
        void reload();
      } catch {
        setScreenError(toErrorMessage(new Error("Не удалось изменить закрепление")));
      } finally {
        setBusyFavoriteStreamID(null);
      }
    },
    [accessToken, scopeCompanyId, reload]
  );

  const handleRunCheck = useCallback(
    async (stream: Stream) => {
      if (!accessToken || !scopeCompanyId || isViewer) return;
      setBusyStreamID(stream.id);
      setRunCheckError(null);
      setRunCheckSuccess(null);
      try {
        const response = await apiRequest<EnqueueCheckJobResponse>(
          `/companies/${scopeCompanyId}/streams/${stream.id}/check`,
          { method: "POST", accessToken }
        );
        setRunCheckSuccess(`Check job #${response.job.id} queued for stream #${stream.id}.`);
        void reload();
      } catch (runError) {
        setRunCheckError(toErrorMessage(runError));
      } finally {
        setBusyStreamID(null);
      }
    },
    [accessToken, scopeCompanyId, isViewer, reload]
  );

  const handleDeleteStream = useCallback(
    async (stream: Stream) => {
      if (!accessToken || !scopeCompanyId || isViewer) return;
      if (!window.confirm(`Удалить поток «${stream.name}»?`)) return;
      setBusyStreamID(stream.id);
      try {
        await apiRequest(`/companies/${scopeCompanyId}/streams/${stream.id}`, {
          method: "DELETE",
          accessToken
        });
        await reload();
      } catch (deleteError) {
        setScreenError(toErrorMessage(deleteError));
      } finally {
        setBusyStreamID(null);
      }
    },
    [accessToken, scopeCompanyId, isViewer, reload]
  );

  const clearRunCheckFeedback = useCallback(() => {
    setRunCheckError(null);
    setRunCheckSuccess(null);
  }, []);

  return {
    runCheckError,
    runCheckSuccess,
    busyStreamID,
    busyFavoriteStreamID,
    screenError,
    setScreenError,
    handleToggleFavorite,
    handleTogglePin,
    handleRunCheck,
    handleDeleteStream,
    clearRunCheckFeedback,
    ensureCommonProject: useCallback(
      async (): Promise<Project> => {
        if (!accessToken || !scopeCompanyId) throw new Error("No scope");
        const projects = getProjects();
        const existing = projects.find((p) => p.name.trim().toLowerCase() === "общий");
        if (existing) return existing;
        return apiRequest<Project>(
          `/companies/${scopeCompanyId}/projects`,
          { method: "POST", accessToken, body: { name: "Общий" } }
        );
      },
      [accessToken, scopeCompanyId, getProjects]
    )
  };
}
