"use client";

import { useCallback, useState } from "react";
import { apiRequest, toErrorMessage } from "@/lib/api/client";
import type { Project, Stream, StreamCreateRequest, StreamPatchRequest } from "@/lib/api/types";

export type StreamFormState = {
  name: string;
  sourceURL: string;
  sourceType: "HLS" | "EMBED";
  projectID: string;
  isActive: boolean;
};

const defaultFormState: StreamFormState = {
  name: "",
  sourceURL: "",
  sourceType: "HLS",
  projectID: "",
  isActive: true
};

type UseStreamFormParams = {
  accessToken: string | null;
  scopeCompanyId: number | null;
  isViewer: boolean;
  projects: Project[];
  reload: () => Promise<void>;
  ensureCommonProject: () => Promise<Project>;
};

export function useStreamForm({
  accessToken,
  scopeCompanyId,
  isViewer,
  projects,
  reload,
  ensureCommonProject
}: UseStreamFormParams) {
  const [isFormOpen, setIsFormOpen] = useState(false);
  const [editingStream, setEditingStream] = useState<Stream | null>(null);
  const [formError, setFormError] = useState<string | null>(null);
  const [isFormSubmitting, setIsFormSubmitting] = useState(false);
  const [initialProjectId, setInitialProjectId] = useState("");

  const openCreate = useCallback((defaultProjectId?: string) => {
    setEditingStream(null);
    setFormError(null);
    setInitialProjectId(defaultProjectId ?? "");
    setIsFormOpen(true);
  }, []);

  const openEdit = useCallback((stream: Stream) => {
    setEditingStream(stream);
    setFormError(null);
    setInitialProjectId(String(stream.project_id));
    setIsFormOpen(true);
  }, []);

  const close = useCallback(() => {
    setIsFormOpen(false);
    setEditingStream(null);
    setFormError(null);
  }, []);

  const submit = useCallback(
    async (values: StreamFormState) => {
      if (!accessToken || !scopeCompanyId || isViewer) return;
      if (!values.name.trim() || !values.sourceURL.trim()) {
        setFormError("Заполните название и URL источника.");
        return;
      }

      setIsFormSubmitting(true);
      setFormError(null);

      try {
        if (editingStream) {
          const payload: StreamPatchRequest = {
            name: values.name.trim(),
            source_type: values.sourceType,
            source_url: values.sourceURL.trim(),
            is_active: values.isActive
          };
          await apiRequest(`/companies/${scopeCompanyId}/streams/${editingStream.id}`, {
            method: "PATCH",
            accessToken,
            body: payload
          });
        } else {
          let parsedProjectID = Number.parseInt(values.projectID, 10);
          if (!Number.isFinite(parsedProjectID) || parsedProjectID <= 0) {
            const fallback = await ensureCommonProject();
            parsedProjectID = fallback.id;
          }
          const payload: StreamCreateRequest = {
            project_id: parsedProjectID,
            name: values.name.trim(),
            source_type: values.sourceType,
            source_url: values.sourceURL.trim(),
            is_active: values.isActive
          };
          await apiRequest(`/companies/${scopeCompanyId}/streams`, {
            method: "POST",
            accessToken,
            body: payload
          });
        }
        close();
        await reload();
      } catch (submitError) {
        setFormError(toErrorMessage(submitError));
      } finally {
        setIsFormSubmitting(false);
      }
    },
    [
      accessToken,
      scopeCompanyId,
      isViewer,
      editingStream,
      ensureCommonProject,
      close,
      reload
    ]
  );

  return {
    isFormOpen,
    editingStream,
    formError,
    isFormSubmitting,
    initialProjectId,
    openCreate,
    openEdit,
    close,
    submit,
    defaultFormState
  };
}
