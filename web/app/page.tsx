"use client";

import Link from "next/link";

import { useAuth } from "@/components/auth/auth-provider";
import { StatePanel } from "@/components/ui/state-panel";

export default function OverviewPage() {
  const { user } = useAuth();

  return (
    <section className="panel">
      <header className="page-header">
        <h2 className="page-title">Overview</h2>
        <p className="page-note">
          Secure Admin UI v2. Tenant scope is derived from authenticated context.
        </p>
      </header>

      {user ? (
        <div className="overview-grid">
          <article className="overview-card">
            <h3>Account</h3>
            <p>
              {user.login} ({user.email})
            </p>
            <p>Role: {user.role}</p>
            <p>Status: {user.status}</p>
          </article>

          <article className="overview-card">
            <h3>Quick navigation</h3>
            <p>
              <Link className="stream-link" href="/streams">
                Streams
              </Link>
            </p>
            <p>
              <Link className="stream-link" href="/analytics">
                Analytics
              </Link>
            </p>
            <p>
              <Link className="stream-link" href="/settings">
                Settings
              </Link>
            </p>
          </article>
        </div>
      ) : (
        <StatePanel kind="error">Auth context is unavailable.</StatePanel>
      )}
    </section>
  );
}
