"use client";

import { motion } from "framer-motion";
import { FormEvent, useCallback, useEffect, useState } from "react";

import { useAuth } from "@/components/auth/auth-provider";
import { AppButton } from "@/components/ui/app-button";
import { SkeletonBlock } from "@/components/ui/skeleton";
import { StatePanel } from "@/components/ui/state-panel";
import { ApiError, apiRequest, toErrorMessage } from "@/lib/api/client";
import type {
  TelegramDeliverySettings,
  TelegramDeliverySettingsPatch,
  TelegramLinkPayload
} from "@/lib/api/types";
import { resolveCompanyScope } from "@/lib/auth/tenant-scope";

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
  const { user, accessToken, activeCompanyId } = useAuth();
  const scopeCompanyId = resolveCompanyScope(user, activeCompanyId);
  const canManageTelegram =
    user?.role === "company_admin" || user?.role === "super_admin";

  const [payloadValue, setPayloadValue] = useState<string>("{}");
  const [isSubmitting, setIsSubmitting] = useState<boolean>(false);
  const [linkStatus, setLinkStatus] = useState<
    "idle" | "connected" | "reconnected"
  >("idle");
  const [error, setError] = useState<string | null>(null);

  const [tgLoading, setTgLoading] = useState<boolean>(false);
  const [tgSettings, setTgSettings] = useState<TelegramDeliverySettings | null>(
    null
  );
  const [tgError, setTgError] = useState<string | null>(null);
  const [tgSubmitting, setTgSubmitting] = useState<boolean>(false);
  const [tgForm, setTgForm] = useState({
    is_enabled: false,
    chat_id: "",
    send_recovered: true
  });
  const [tgValidationError, setTgValidationError] = useState<string | null>(
    null
  );

  const loadTelegramSettings = useCallback(async () => {
    if (!accessToken || !scopeCompanyId) {
      setTgSettings(null);
      setTgError(null);
      return;
    }
    setTgLoading(true);
    setTgError(null);
    try {
      const data = await apiRequest<TelegramDeliverySettings>(
        `/companies/${scopeCompanyId}/telegram-delivery-settings`,
        { accessToken }
      );
      setTgSettings(data);
      setTgForm({
        is_enabled: data.is_enabled,
        chat_id: data.chat_id ?? "",
        send_recovered: data.send_recovered
      });
    } catch (err) {
      if (err instanceof ApiError && err.status === 404) {
        setTgSettings(null);
        setTgForm({ is_enabled: false, chat_id: "", send_recovered: true });
      } else {
        setTgError(toErrorMessage(err));
      }
    } finally {
      setTgLoading(false);
    }
  }, [accessToken, scopeCompanyId]);

  useEffect(() => {
    if (canManageTelegram && scopeCompanyId && accessToken) {
      void loadTelegramSettings();
    } else {
      setTgSettings(null);
      setTgError(null);
      setTgLoading(false);
    }
  }, [canManageTelegram, scopeCompanyId, accessToken, loadTelegramSettings]);

  const handleTelegramSettingsSubmit = async (
    event: FormEvent<HTMLFormElement>
  ) => {
    event.preventDefault();
    if (!accessToken || !scopeCompanyId) return;
    setTgValidationError(null);
    const trimmedChatId = tgForm.chat_id.trim();
    if (tgForm.is_enabled && !trimmedChatId) {
      setTgValidationError("Chat ID is required when alerts are enabled.");
      return;
    }
    setTgSubmitting(true);
    setTgError(null);
    try {
      const body: TelegramDeliverySettingsPatch = {
        is_enabled: tgForm.is_enabled,
        chat_id: trimmedChatId || undefined,
        send_recovered: tgForm.send_recovered
      };
      const data = await apiRequest<TelegramDeliverySettings>(
        `/companies/${scopeCompanyId}/telegram-delivery-settings`,
        { method: "PATCH", accessToken, body }
      );
      setTgSettings(data);
      setTgForm({
        is_enabled: data.is_enabled,
        chat_id: data.chat_id ?? "",
        send_recovered: data.send_recovered
      });
    } catch (err) {
      setTgError(toErrorMessage(err));
    } finally {
      setTgSubmitting(false);
    }
  };

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

          <AppButton type="submit" disabled={isSubmitting} aria-label={linkStatus === "idle" ? "Connect Telegram account" : "Reconnect Telegram account"}>
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

      {canManageTelegram ? (
        <motion.div
          className="settings-card"
          style={{ marginTop: "16px" }}
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ duration: 0.24, ease: "easeOut" }}
        >
          <h3>Telegram Alerts (Company)</h3>
          {!scopeCompanyId ? (
            <StatePanel>
              Select company in topbar to configure Telegram alerts.
            </StatePanel>
          ) : tgLoading ? (
            <SkeletonBlock lines={5} />
          ) : tgError ? (
            <>
              <StatePanel kind="error">{tgError}</StatePanel>
              <AppButton
                type="button"
                variant="secondary"
                onClick={() => void loadTelegramSettings()}
                style={{ marginTop: "10px" }}
                aria-label="Retry loading Telegram settings"
              >
                Retry
              </AppButton>
            </>
          ) : (
            <>
              {tgSettings === null && !tgError ? (
                <p>No company Telegram alerts configured. Use the form below to create.</p>
              ) : null}
              {tgSettings &&
              (tgSettings.created_at != null || tgSettings.updated_at != null) ? (
                <p className="form-field" style={{ marginTop: "8px", marginBottom: "0" }}>
                  <span style={{ fontSize: "0.85rem", color: "var(--text-muted)" }}>
                    {tgSettings.created_at
                      ? `Created: ${new Date(tgSettings.created_at).toLocaleString()}`
                      : ""}
                    {tgSettings.created_at && tgSettings.updated_at ? " · " : ""}
                    {tgSettings.updated_at
                      ? `Updated: ${new Date(tgSettings.updated_at).toLocaleString()}`
                      : ""}
                  </span>
                </p>
              ) : null}
              <form
                className="telegram-form"
                onSubmit={handleTelegramSettingsSubmit}
                style={{ marginTop: "12px" }}
              >
                <label className="form-field toggle-row" style={{ flexDirection: "row", alignItems: "center", gap: "10px" }}>
                  <input
                    type="checkbox"
                    checked={tgForm.is_enabled}
                    onChange={(e) =>
                      setTgForm((prev) => ({ ...prev, is_enabled: e.target.checked }))
                    }
                    disabled={tgSubmitting}
                  />
                  <span>Alerts enabled</span>
                </label>
                <label className="form-field" htmlFor="tg-chat-id">
                  <span>Chat ID</span>
                  <input
                    id="tg-chat-id"
                    type="text"
                    value={tgForm.chat_id}
                    onChange={(e) =>
                      setTgForm((prev) => ({ ...prev, chat_id: e.target.value }))
                    }
                    placeholder="-1001234567890"
                    disabled={tgSubmitting}
                  />
                </label>
                <label className="form-field toggle-row" style={{ flexDirection: "row", alignItems: "center", gap: "10px" }}>
                  <input
                    type="checkbox"
                    checked={tgForm.send_recovered}
                    onChange={(e) =>
                      setTgForm((prev) => ({ ...prev, send_recovered: e.target.checked }))
                    }
                    disabled={tgSubmitting}
                  />
                  <span>Send recovered notifications</span>
                </label>
                {tgValidationError ? (
                  <StatePanel kind="error">{tgValidationError}</StatePanel>
                ) : null}
                <AppButton type="submit" disabled={tgSubmitting} aria-label="Save Telegram alerts settings">
                  {tgSubmitting ? "Saving…" : tgSettings ? "Update" : "Create"}
                </AppButton>
              </form>
            </>
          )}
        </motion.div>
      ) : null}
    </section>
  );
}
