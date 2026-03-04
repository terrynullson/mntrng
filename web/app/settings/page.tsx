"use client";

import { motion } from "framer-motion";
import { Power, Trash2 } from "lucide-react";
import { useCallback, useEffect, useState } from "react";

import { useAuth } from "@/components/auth/auth-provider";
import { IconButton } from "@/components/navigation/icon-button";
import { AppButton } from "@/components/ui/app-button";
import { SkeletonBlock } from "@/components/ui/skeleton";
import { StatePanel } from "@/components/ui/state-panel";
import { ApiError, apiRequest, toErrorMessage } from "@/lib/api/client";
import type {
  EmbedWhitelistItem,
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
  const canManageEmbedWhitelist =
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
  const [embedLoading, setEmbedLoading] = useState<boolean>(false);
  const [embedError, setEmbedError] = useState<string | null>(null);
  const [embedItems, setEmbedItems] = useState<EmbedWhitelistItem[]>([]);
  const [embedDomainInput, setEmbedDomainInput] = useState<string>("");
  const [embedSubmitting, setEmbedSubmitting] = useState<boolean>(false);
  const [embedBusyID, setEmbedBusyID] = useState<number | null>(null);

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

  const loadEmbedWhitelist = useCallback(async () => {
    if (!accessToken || !scopeCompanyId) {
      setEmbedItems([]);
      setEmbedError(null);
      return;
    }
    setEmbedLoading(true);
    setEmbedError(null);
    try {
      const data = await apiRequest<{ items: EmbedWhitelistItem[] }>(
        `/companies/${scopeCompanyId}/embed-whitelist`,
        { accessToken }
      );
      setEmbedItems(Array.isArray(data.items) ? data.items : []);
    } catch (err) {
      setEmbedError(toErrorMessage(err));
    } finally {
      setEmbedLoading(false);
    }
  }, [accessToken, scopeCompanyId]);

  useEffect(() => {
    if (canManageEmbedWhitelist && scopeCompanyId && accessToken) {
      void loadEmbedWhitelist();
    } else {
      setEmbedItems([]);
      setEmbedError(null);
      setEmbedLoading(false);
    }
  }, [canManageEmbedWhitelist, scopeCompanyId, accessToken, loadEmbedWhitelist]);

  const handleAddDomain = async () => {
    if (!accessToken || !scopeCompanyId) return;
    setEmbedSubmitting(true);
    setEmbedError(null);
    try {
      await apiRequest<EmbedWhitelistItem>(
        `/companies/${scopeCompanyId}/embed-whitelist`,
        {
          method: "POST",
          accessToken,
          body: { domain: embedDomainInput.trim() }
        }
      );
      setEmbedDomainInput("");
      await loadEmbedWhitelist();
    } catch (err) {
      setEmbedError(toErrorMessage(err));
    } finally {
      setEmbedSubmitting(false);
    }
  };

  const handleToggleDomain = async (item: EmbedWhitelistItem) => {
    if (!accessToken || !scopeCompanyId) return;
    setEmbedBusyID(item.id);
    setEmbedError(null);
    try {
      await apiRequest<EmbedWhitelistItem>(
        `/companies/${scopeCompanyId}/embed-whitelist/${item.id}`,
        {
          method: "PATCH",
          accessToken,
          body: { enabled: !item.enabled }
        }
      );
      await loadEmbedWhitelist();
    } catch (err) {
      setEmbedError(toErrorMessage(err));
    } finally {
      setEmbedBusyID(null);
    }
  };

  const handleDeleteDomain = async (item: EmbedWhitelistItem) => {
    if (!accessToken || !scopeCompanyId) return;
    if (!window.confirm(`Удалить домен ${item.domain}?`)) {
      return;
    }
    setEmbedBusyID(item.id);
    setEmbedError(null);
    try {
      await apiRequest<void>(`/companies/${scopeCompanyId}/embed-whitelist/${item.id}`, {
        method: "DELETE",
        accessToken
      });
      await loadEmbedWhitelist();
    } catch (err) {
      setEmbedError(toErrorMessage(err));
    } finally {
      setEmbedBusyID(null);
    }
  };

  const handleTelegramSettingsSubmit = async (
    event: { preventDefault: () => void }
  ) => {
    event.preventDefault();
    if (!accessToken || !scopeCompanyId) return;
    setTgValidationError(null);
    const trimmedChatId = tgForm.chat_id.trim();
    if (tgForm.is_enabled && !trimmedChatId) {
      setTgValidationError("Chat ID обязателен, когда оповещения включены.");
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

  const handleLink = async (event: { preventDefault: () => void }) => {
    event.preventDefault();

    if (!accessToken) {
      setError("Не найден auth token.");
      return;
    }

    const payload = parsePayload(payloadValue);
    if (!payload) {
      setError("Payload должен быть валидным JSON-объектом.");
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
    <section className="panel premium-panel">
      <header className="page-header compact">
        <div>
          <h2 className="page-title">Настройки</h2>
          <p className="page-note">
            Настройки Telegram и Embed whitelist.
          </p>
        </div>
      </header>

      <div className="settings-section">
        <h3>Привязка Telegram аккаунта</h3>
        <p>
          Вставьте JSON payload из Telegram Login Widget (включая поле <code>hash</code>),
          чтобы привязать или перепривязать аккаунт.
        </p>
        <form className="telegram-form" onSubmit={handleLink}>
          <label className="form-field" htmlFor="telegram-payload">
            <span>Telegram payload JSON</span>
            <textarea
              id="telegram-payload"
              value={payloadValue}
              onChange={(event) => setPayloadValue(event.target.value)}
              rows={5}
              disabled={isSubmitting}
              placeholder='{"id": 123, "hash": "…"}'
            />
          </label>
          <AppButton type="submit" disabled={isSubmitting} aria-label={linkStatus === "idle" ? "Подключить Telegram аккаунт" : "Переподключить Telegram аккаунт"}>
            {isSubmitting
              ? "Обработка…"
              : linkStatus === "idle"
                ? "Подключить Telegram"
                : "Переподключить"}
          </AppButton>
        </form>
        {linkStatus === "connected" ? (
          <StatePanel>Telegram аккаунт успешно подключен.</StatePanel>
        ) : null}
        {linkStatus === "reconnected" ? (
          <StatePanel>Telegram аккаунт успешно переподключен.</StatePanel>
        ) : null}
        {error ? <StatePanel kind="error">{error}</StatePanel> : null}
      </div>

      {canManageTelegram ? (
        <motion.div
          className="settings-section"
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ duration: 0.24, ease: "easeOut" }}
        >
          <h3>Telegram оповещения (компания)</h3>
          {!scopeCompanyId ? (
            <StatePanel>
              Выберите компанию в шапке, чтобы настроить Telegram оповещения.
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
                aria-label="Повторить загрузку настроек Telegram"
              >
                Повторить
              </AppButton>
            </>
          ) : (
            <>
              {tgSettings === null && !tgError ? (
                <p>Telegram оповещения ещё не настроены. Используйте форму ниже.</p>
              ) : null}
              {tgSettings &&
              (tgSettings.created_at != null || tgSettings.updated_at != null) ? (
                <p className="form-field" style={{ marginTop: "8px", marginBottom: "0" }}>
                  <span style={{ fontSize: "0.85rem", color: "var(--text-muted)" }}>
                    {tgSettings.created_at
                      ? `Создано: ${new Date(tgSettings.created_at).toLocaleString()}`
                      : ""}
                    {tgSettings.created_at && tgSettings.updated_at ? " · " : ""}
                    {tgSettings.updated_at
                      ? `Обновлено: ${new Date(tgSettings.updated_at).toLocaleString()}`
                      : ""}
                  </span>
                </p>
              ) : null}
              <form
                className="telegram-form"
                onSubmit={handleTelegramSettingsSubmit}
              >
                <label className="form-field form-check toggle-row">
                  <input
                    type="checkbox"
                    checked={tgForm.is_enabled}
                    onChange={(e) =>
                      setTgForm((prev) => ({ ...prev, is_enabled: e.target.checked }))
                    }
                    disabled={tgSubmitting}
                    aria-label="Оповещения включены"
                  />
                  <span>Оповещения включены</span>
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
                <label className="form-field form-check toggle-row">
                  <input
                    type="checkbox"
                    checked={tgForm.send_recovered}
                    onChange={(e) =>
                      setTgForm((prev) => ({ ...prev, send_recovered: e.target.checked }))
                    }
                    disabled={tgSubmitting}
                    aria-label="Отправлять уведомления о восстановлении"
                  />
                  <span>Отправлять уведомления о восстановлении</span>
                </label>
                {tgValidationError ? (
                  <StatePanel kind="error">{tgValidationError}</StatePanel>
                ) : null}
                <AppButton type="submit" disabled={tgSubmitting} aria-label="Сохранить настройки Telegram оповещений">
                  {tgSubmitting ? "Сохранение…" : tgSettings ? "Обновить" : "Создать"}
                </AppButton>
              </form>
            </>
          )}
        </motion.div>
      ) : null}

      {canManageEmbedWhitelist ? (
        <motion.div
          className="settings-section"
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ duration: 0.24, ease: "easeOut" }}
        >
          <h3>Embed whitelist</h3>
          {!scopeCompanyId ? (
            <StatePanel>Выберите компанию в шапке, чтобы управлять whitelist.</StatePanel>
          ) : embedLoading ? (
            <SkeletonBlock lines={5} />
          ) : embedError ? (
            <StatePanel kind="error">{embedError}</StatePanel>
          ) : (
            <>
              <div className="embed-whitelist-add">
                <label className="form-field" htmlFor="embed-domain-input">
                  <span>Домен</span>
                  <input
                    id="embed-domain-input"
                    type="text"
                    placeholder="youtube.com"
                    value={embedDomainInput}
                    onChange={(event) => setEmbedDomainInput(event.target.value)}
                    disabled={embedSubmitting}
                  />
                </label>
                <AppButton
                  type="button"
                  disabled={embedSubmitting || embedDomainInput.trim() === ""}
                  onClick={() => {
                    void handleAddDomain();
                  }}
                >
                  {embedSubmitting ? "Добавляем…" : "Добавить домен"}
                </AppButton>
              </div>

              {embedItems.length === 0 ? (
                <StatePanel>Whitelist пуст — добавьте первый домен.</StatePanel>
              ) : (
                <div className="card-table-wrap" style={{ marginTop: "12px" }}>
                  <table>
                    <thead>
                      <tr>
                        <th>ID</th>
                        <th>Домен</th>
                        <th>Статус</th>
                        <th>Создан</th>
                        <th>Действия</th>
                      </tr>
                    </thead>
                    <tbody>
                      {embedItems.map((item) => (
                        <tr key={item.id}>
                          <td>{item.id}</td>
                          <td>{item.domain}</td>
                          <td>{item.enabled ? "Вкл" : "Выкл"}</td>
                          <td>{new Date(item.created_at).toLocaleString()}</td>
                          <td>
                            <div className="stream-actions">
                              <IconButton
                                disabled={embedBusyID === item.id}
                                onClick={() => {
                                  void handleToggleDomain(item);
                                }}
                                label={`${item.enabled ? "Отключить" : "Включить"} домен ${item.domain}`}
                                tooltip={item.enabled ? "Отключить домен" : "Включить домен"}
                              >
                                <Power size={16} />
                              </IconButton>
                              <IconButton
                                disabled={embedBusyID === item.id}
                                onClick={() => {
                                  void handleDeleteDomain(item);
                                }}
                                label={`Удалить домен ${item.domain}`}
                                tooltip="Удалить домен"
                                destructive
                              >
                                <Trash2 size={16} />
                              </IconButton>
                            </div>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </>
          )}
        </motion.div>
      ) : null}
    </section>
  );
}
