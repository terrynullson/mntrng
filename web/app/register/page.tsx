"use client";

import Link from "next/link";
import { motion, useReducedMotion } from "framer-motion";
import { useState } from "react";

import { AppButton } from "@/components/ui/app-button";
import { apiRequest, toErrorMessage } from "@/lib/api/client";
import type { RegisterRequest, RegistrationRequest, Role } from "@/lib/api/types";

const REGISTRATION_ROLES: Array<{ value: Extract<Role, "company_admin" | "viewer">; label: string }> = [
  { value: "viewer", label: "Просмотр (viewer)" },
  { value: "company_admin", label: "Администратор компании" }
];

export default function RegisterPage() {
  const [companyID, setCompanyID] = useState<string>("");
  const [email, setEmail] = useState<string>("");
  const [login, setLogin] = useState<string>("");
  const [password, setPassword] = useState<string>("");
  const [requestedRole, setRequestedRole] =
    useState<Extract<Role, "company_admin" | "viewer">>("viewer");

  const [isSubmitting, setIsSubmitting] = useState<boolean>(false);
  const [error, setError] = useState<string | null>(null);
  const [pendingRequest, setPendingRequest] =
    useState<RegistrationRequest | null>(null);

  const prefersReducedMotion = useReducedMotion();

  const handleSubmit = async (event: { preventDefault: () => void }) => {
    event.preventDefault();

    const parsedCompanyID = Number.parseInt(companyID, 10);
    if (!Number.isFinite(parsedCompanyID) || parsedCompanyID <= 0) {
      setError("company_id должен быть положительным числом.");
      return;
    }

    if (!email.trim() || !login.trim() || password.length < 8) {
      setError("Заполните все поля. Пароль — минимум 8 символов.");
      return;
    }

    const payload: RegisterRequest = {
      company_id: parsedCompanyID,
      email: email.trim(),
      login: login.trim(),
      password,
      requested_role: requestedRole
    };

    setIsSubmitting(true);
    setError(null);

    try {
      const response = await apiRequest<RegistrationRequest>("/auth/register", {
        method: "POST",
        body: payload
      });
      setPendingRequest(response);
    } catch (submitError) {
      setError(toErrorMessage(submitError));
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div className="auth-page">
      <motion.section
        className="auth-card"
        initial={prefersReducedMotion ? undefined : { opacity: 0, y: 18 }}
        animate={prefersReducedMotion ? undefined : { opacity: 1, y: 0 }}
        transition={
          prefersReducedMotion
            ? undefined
            : {
                duration: 0.3,
                ease: "easeOut"
              }
        }
      >
        <div className="auth-card-header">
          <h1>Регистрация</h1>
          <p>Заявка попадёт на подтверждение администратора</p>
        </div>

        {pendingRequest ? (
          <div className="pending-card">
            <h2>Заявка отправлена</h2>
            <p>
              Заявка №{pendingRequest.id} со статусом{" "}
              <strong>{pendingRequest.status}</strong>.
            </p>
            <p>Вход станет доступен после одобрения и активации учётной записи.</p>
            <p>
              <Link href="/login" className="stream-link" aria-label="Вернуться к входу">
                Вернуться к входу
              </Link>
            </p>
          </div>
        ) : (
          <form className="auth-form" onSubmit={handleSubmit}>
            <label className="form-field" htmlFor="register-company-id">
              <span>ID компании</span>
              <input
                id="register-company-id"
                type="number"
                value={companyID}
                onChange={(event) => setCompanyID(event.target.value)}
                disabled={isSubmitting}
              />
            </label>

            <label className="form-field" htmlFor="register-email">
              <span>Email</span>
              <input
                id="register-email"
                type="email"
                value={email}
                onChange={(event) => setEmail(event.target.value)}
                disabled={isSubmitting}
              />
            </label>

            <label className="form-field" htmlFor="register-login">
              <span>Логин</span>
              <input
                id="register-login"
                value={login}
                onChange={(event) => setLogin(event.target.value)}
                disabled={isSubmitting}
              />
            </label>

            <label className="form-field" htmlFor="register-password">
              <span>Пароль</span>
              <input
                id="register-password"
                type="password"
                value={password}
                onChange={(event) => setPassword(event.target.value)}
                disabled={isSubmitting}
              />
            </label>

            <label className="form-field" htmlFor="register-role">
              <span>Запрашиваемая роль</span>
              <select
                id="register-role"
                value={requestedRole}
                onChange={(event) =>
                  setRequestedRole(
                    event.target.value as Extract<Role, "company_admin" | "viewer">
                  )
                }
                disabled={isSubmitting}
              >
                {REGISTRATION_ROLES.map((role) => (
                  <option key={role.value} value={role.value}>
                    {role.label}
                  </option>
                ))}
              </select>
            </label>

            {error ? <p className="state state-error">{error}</p> : null}

            <AppButton
              type="submit"
              isLoading={isSubmitting}
              aria-label="Отправить заявку на регистрацию"
            >
              {isSubmitting ? "Отправляем…" : "Отправить заявку"}
            </AppButton>
          </form>
        )}

        <p className="auth-secondary">
          Уже есть аккаунт?{" "}
          <Link href="/login" aria-label="Перейти к входу">
            Войти
          </Link>
        </p>
      </motion.section>
    </div>
  );
}
