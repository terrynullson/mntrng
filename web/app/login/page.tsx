"use client";

import Link from "next/link";
import { motion } from "framer-motion";
import { useRouter } from "next/navigation";
import { FormEvent, useEffect, useState } from "react";

import { useAuth } from "@/components/auth/auth-provider";
import { AppButton } from "@/components/ui/app-button";
import { toErrorMessage } from "@/lib/api/client";

export default function LoginPage() {
  const router = useRouter();

  const { isReady, isAuthenticated, loginWithPassword } = useAuth();

  const [loginOrEmail, setLoginOrEmail] = useState<string>("");
  const [password, setPassword] = useState<string>("");
  const [isSubmitting, setIsSubmitting] = useState<boolean>(false);
  const [error, setError] = useState<string | null>(null);
  const [nextPath, setNextPath] = useState<string>("/");

  useEffect(() => {
    if (typeof window === "undefined") {
      return;
    }

    const value = new URLSearchParams(window.location.search).get("next");
    if (value && value.startsWith("/")) {
      setNextPath(value);
    }
  }, []);

  useEffect(() => {
    if (isReady && isAuthenticated) {
      router.replace("/");
    }
  }, [isAuthenticated, isReady, router]);

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();

    if (!loginOrEmail.trim() || !password) {
      setError("Enter login/email and password.");
      return;
    }

    setIsSubmitting(true);
    setError(null);

    try {
      await loginWithPassword({
        login_or_email: loginOrEmail.trim(),
        password
      });

      router.replace(nextPath);
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
        <h1>Login</h1>
        <p>Sign in to access secure admin routes.</p>

        <form className="auth-form" onSubmit={handleSubmit}>
          <label className="form-field" htmlFor="login-or-email">
            <span>Login or email</span>
            <input
              id="login-or-email"
              value={loginOrEmail}
              onChange={(event) => setLoginOrEmail(event.target.value)}
              autoComplete="username"
              disabled={isSubmitting}
            />
          </label>

          <label className="form-field" htmlFor="login-password">
            <span>Password</span>
            <input
              id="login-password"
              type="password"
              value={password}
              onChange={(event) => setPassword(event.target.value)}
              autoComplete="current-password"
              disabled={isSubmitting}
            />
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

          <AppButton type="submit" disabled={isSubmitting} aria-label="Sign in">
            {isSubmitting ? "Signing in..." : "Login"}
          </AppButton>
        </form>

        <p className="auth-secondary">
          No account? <Link href="/register" aria-label="Create registration request">Create registration request</Link>
        </p>
      </motion.section>
    </div>
  );
}
