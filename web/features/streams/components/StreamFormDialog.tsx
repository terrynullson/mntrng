"use client";

import { useEffect, useState } from "react";
import { AppButton } from "@/components/ui/app-button";
import { StatePanel } from "@/components/ui/state-panel";
import type { Project, Stream } from "@/lib/api/types";
import type { StreamFormState } from "../hooks/use-stream-form";

type StreamFormDialogProps = {
  isOpen: boolean;
  editingStream: Stream | null;
  initialProjectId: string;
  projects: Project[];
  formError: string | null;
  isFormSubmitting: boolean;
  onClose: () => void;
  onSubmit: (values: StreamFormState) => void;
};

export function StreamFormDialog({
  isOpen,
  editingStream,
  initialProjectId,
  projects,
  formError,
  isFormSubmitting,
  onClose,
  onSubmit
}: StreamFormDialogProps) {
  const [name, setName] = useState("");
  const [sourceURL, setSourceURL] = useState("");
  const [sourceType, setSourceType] = useState<"HLS" | "EMBED">("HLS");
  const [projectID, setProjectID] = useState("");
  const [isActive, setIsActive] = useState(true);

  useEffect(() => {
    if (!isOpen) return;
    if (editingStream) {
      setName(editingStream.name);
      setSourceURL(editingStream.source_url || editingStream.url);
      setSourceType(editingStream.source_type ?? "HLS");
      setProjectID(String(editingStream.project_id));
      setIsActive(editingStream.is_active);
    } else {
      setName("");
      setSourceURL("");
      setSourceType("HLS");
      setProjectID(initialProjectId || "");
      setIsActive(true);
    }
  }, [isOpen, editingStream, initialProjectId]);

  const handleSubmit = () => {
    onSubmit({
      name,
      sourceURL,
      sourceType,
      projectID,
      isActive
    });
  };

  if (!isOpen) return null;

  return (
    <div className="overlay-backdrop" role="dialog" aria-modal="true">
      <div className="overlay-card">
        <h3>{editingStream ? "Редактировать поток" : "Добавить поток"}</h3>
        <div className="overlay-grid">
          <label className="form-field" htmlFor="stream-name">
            <span>Название</span>
            <input
              id="stream-name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="Например: Main camera #1"
            />
          </label>
          <label className="form-field" htmlFor="stream-url">
            <span>URL источника</span>
            <input
              id="stream-url"
              value={sourceURL}
              onChange={(e) => setSourceURL(e.target.value)}
              placeholder={
                sourceType === "HLS"
                  ? "https://example.com/live.m3u8"
                  : "https://youtube.com/watch?v=..."
              }
            />
          </label>
          <label className="form-field" htmlFor="stream-source-type">
            <span>Тип источника</span>
            <select
              id="stream-source-type"
              value={sourceType}
              onChange={(e) => setSourceType(e.target.value as "HLS" | "EMBED")}
            >
              <option value="HLS">HLS</option>
              <option value="EMBED">Embed</option>
            </select>
          </label>
          <label className="form-field" htmlFor="stream-project">
            <span>Проект</span>
            <select
              id="stream-project"
              value={projectID}
              onChange={(e) => setProjectID(e.target.value)}
            >
              <option value="">Авто: создать/выбрать «Общий»</option>
              {projects.map((project) => (
                <option key={project.id} value={project.id}>
                  {project.name} ({project.id})
                </option>
              ))}
            </select>
          </label>
          <label className="form-field form-check" htmlFor="stream-active">
            <input
              id="stream-active"
              type="checkbox"
              checked={isActive}
              onChange={(e) => setIsActive(e.target.checked)}
            />
            <span>Активный поток</span>
          </label>
        </div>
        {formError ? <StatePanel kind="error">{formError}</StatePanel> : null}
        {sourceType === "EMBED" ? (
          <StatePanel>Домен должен быть в whitelist.</StatePanel>
        ) : null}
        <div className="overlay-actions">
          <AppButton
            type="button"
            variant="secondary"
            onClick={onClose}
            disabled={isFormSubmitting}
          >
            Отмена
          </AppButton>
          <AppButton
            type="button"
            onClick={handleSubmit}
            disabled={isFormSubmitting}
          >
            {isFormSubmitting ? "Сохраняем…" : "Сохранить"}
          </AppButton>
        </div>
      </div>
    </div>
  );
}
