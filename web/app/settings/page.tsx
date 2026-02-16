"use client";

import { FormEvent, useState } from "react";

import { useAuth } from "@/components/auth/auth-provider";
import { AppButton } from "@/components/ui/app-button";
import { StatePanel } from "@/components/ui/state-panel";
import { apiRequest, toErrorMessage } from "@/lib/api/client";
import type { TelegramLinkPayload } from "@/lib/api/types";

function parsePayload(rawValue: string): TelegramLinkPayload | null {
  try {
    const parsed = JSON.parse(rawValue) as unknown;
    if (typeof parsed !== "object" || parsed === null || Array.isArray(parsed)) {
      return null;
    }

    const payload: TelegramLinkPayload = {};
    Object.entries(parsed).forEach(([key, value]) => {
      payload[key] = typeof value === "string" ? value : String(value);
    });
    return payload;
  } catch {
    return null;
  }
}

export default function SettingsPage() {
  const { accessToken } = useAuth();

  const [payloadValue, setPayloadValue] = useState<string>("{}");
  const [isSubmitting, setIsSubmitting] = useState<boolean>(false);
  const [linkStatus, setLinkStatus] = useState<
    "idle" | "connected" | "reconnected"
  >("idle");
  const [error, setError] = useState<string | null>(null);

  const handleLink = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();

    if (!accessToken) {
      setError("Auth token is missing.");
      return;
    }

    const payload = parsePayload(payloadValue);
    if (!payload) {
      setError("Payload must be valid JSON object.");
      return;
    }

    setIsSubmitting(true);
    setError(null);

    try {
      await apiRequest<void>("/auth/telegram/link", {
        method: "POST",
        accessToken,
        body: payload
      });
      setLinkStatus((previous) =>
        previous === "connected" || previous === "reconnected"
          ? "reconnected"
          : "connected"
      );
    } catch (submitError) {
      setError(toErrorMessage(submitError));
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <section className="panel">
      <header className="page-header compact">
        <h2 className="page-title">Settings</h2>
        <p className="page-note">
          Telegram link flow for notifications and Telegram auth support.
        </p>
      </header>

      <div className="settings-card">
        <h3>Telegram Account Link</h3>
        <p>
          Paste Telegram login payload JSON (widget fields including <code>hash</code>)
          to connect or reconnect your account.
        </p>

        <form className="telegram-form" onSubmit={handleLink}>
          <label className="form-field" htmlFor="telegram-payload">
            <span>Telegram payload JSON</span>
            <textarea
              id="telegram-payload"
              value={payloadValue}
              onChange={(event) => setPayloadValue(event.target.value)}
              rows={8}
              disabled={isSubmitting}
            />
          </label>

          <AppButton type="submit" disabled={isSubmitting}>
            {isSubmitting
              ? "Processing..."
              : linkStatus === "idle"
                ? "Connect Telegram"
                : "Reconnect Telegram"}
          </AppButton>
        </form>

        {linkStatus === "connected" ? (
          <StatePanel>Telegram account successfully connected.</StatePanel>
        ) : null}
        {linkStatus === "reconnected" ? (
          <StatePanel>Telegram account successfully reconnected.</StatePanel>
        ) : null}
        {error ? <StatePanel kind="error">{error}</StatePanel> : null}
      </div>
    </section>
  );
}
