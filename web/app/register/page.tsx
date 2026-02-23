"use client";

import Link from "next/link";
import { motion } from "framer-motion";
import { FormEvent, useState } from "react";

import { AppButton } from "@/components/ui/app-button";
import { apiRequest, toErrorMessage } from "@/lib/api/client";
import type { RegisterRequest, RegistrationRequest, Role } from "@/lib/api/types";

const REGISTRATION_ROLES: Array<{ value: Extract<Role, "company_admin" | "viewer">; label: string }> = [
  { value: "viewer", label: "Viewer" },
  { value: "company_admin", label: "Company admin" }
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

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();

    const parsedCompanyID = Number.parseInt(companyID, 10);
    if (!Number.isFinite(parsedCompanyID) || parsedCompanyID <= 0) {
      setError("company_id must be a positive number.");
      return;
    }

    if (!email.trim() || !login.trim() || password.length < 8) {
      setError("Fill all fields. Password must be at least 8 characters.");
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
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        transition={{ duration: 0.28, ease: "easeOut" }}
      >
        <h1>Registration Request</h1>
        <p>Create a pending request for super admin approval.</p>

        {pendingRequest ? (
          <motion.div
            className="pending-card"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            transition={{ duration: 0.24, ease: "easeOut" }}
          >
            <h2>Request submitted</h2>
            <p>
              Request #{pendingRequest.id} is in <strong>{pendingRequest.status}</strong>
              .
            </p>
            <p>Login will be available only after approval and activation.</p>
            <p>
              <Link href="/login" className="stream-link" aria-label="Back to login">
                Back to login
              </Link>
            </p>
          </motion.div>
        ) : (
          <motion.form
            className="auth-form"
            onSubmit={handleSubmit}
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            transition={{ duration: 0.2, ease: "easeOut" }}
          >
            <label className="form-field" htmlFor="register-company-id">
              <span>Company ID</span>
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
              <span>Login</span>
              <input
                id="register-login"
                value={login}
                onChange={(event) => setLogin(event.target.value)}
                disabled={isSubmitting}
              />
            </label>

            <label className="form-field" htmlFor="register-password">
              <span>Password</span>
              <input
                id="register-password"
                type="password"
                value={password}
                onChange={(event) => setPassword(event.target.value)}
                disabled={isSubmitting}
              />
            </label>

            <label className="form-field" htmlFor="register-role">
              <span>Requested role</span>
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

            {error ? (
              <motion.p
                className="state state-error"
                initial={{ opacity: 0 }}
                animate={{ opacity: 1 }}
                transition={{ duration: 0.2, ease: "easeOut" }}
              >
                {error}
              </motion.p>
            ) : null}

            <AppButton type="submit" disabled={isSubmitting} aria-label="Submit registration request">
              {isSubmitting ? "Submitting..." : "Submit request"}
            </AppButton>
          </motion.form>
        )}

        <p className="auth-secondary">
          Already approved? <Link href="/login" aria-label="Go to login">Login</Link>
        </p>
      </motion.section>
    </div>
  );
}
