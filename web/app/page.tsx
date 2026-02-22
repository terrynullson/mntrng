"use client";

import Link from "next/link";
import { motion } from "framer-motion";

import { useAuth } from "@/components/auth/auth-provider";
import { SkeletonBlock } from "@/components/ui/skeleton";
import { StatePanel } from "@/components/ui/state-panel";

export default function OverviewPage() {
  const { user, isReady } = useAuth();

  return (
    <section className="panel">
      <header className="page-header">
        <h2 className="page-title">Overview</h2>
        <p className="page-note">
          Secure Admin UI v2. Tenant scope is derived from authenticated context.
        </p>
      </header>

      {!isReady ? (
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ duration: 0.2, ease: "easeOut" }}
          style={{ marginTop: "12px" }}
        >
          <SkeletonBlock lines={5} />
        </motion.div>
      ) : !user ? (
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ duration: 0.24, ease: "easeOut" }}
        >
          <StatePanel kind="error">Auth context is unavailable.</StatePanel>
        </motion.div>
      ) : (
        <motion.div
          className="overview-grid"
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ duration: 0.28, ease: "easeOut" }}
          style={{ marginTop: "12px" }}
        >
          <motion.article
            className="overview-card"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            transition={{ duration: 0.24, ease: "easeOut", delay: 0.05 }}
          >
            <h3>Account</h3>
            <p>
              {user.login} ({user.email})
            </p>
            <p>Role: {user.role}</p>
            <p>Status: {user.status}</p>
          </motion.article>

          <motion.article
            className="overview-card"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            transition={{ duration: 0.24, ease: "easeOut", delay: 0.1 }}
          >
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
          </motion.article>
        </motion.div>
      )}
    </section>
  );
}
