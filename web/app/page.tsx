"use client";

import { motion } from "framer-motion";
import { useRouter } from "next/navigation";

import { useAuth } from "@/components/auth/auth-provider";
import { AppButton } from "@/components/ui/app-button";
import { SkeletonBlock } from "@/components/ui/skeleton";
import { StatePanel } from "@/components/ui/state-panel";

export default function OverviewPage() {
  const router = useRouter();
  const { user, isReady } = useAuth();

  return (
    <section className="panel">
      <header className="page-header">
        <h2 className="page-title">Главное меню</h2>
        <p className="page-note">
          Выберите раздел платформы после входа в систему.
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
            className="overview-card landing-primary-card"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            transition={{ duration: 0.24, ease: "easeOut", delay: 0.05 }}
          >
            <h3>МОНИТОРИНГ HLS</h3>
            <p>
              Панель мониторинга HLS-потоков, их статусов и ручного запуска
              проверок.
            </p>
            <AppButton
              type="button"
              className="landing-primary-button"
              aria-label="Открыть мониторинг HLS"
              onClick={() => {
                router.push("/streams");
              }}
            >
              МОНИТОРИНГ HLS
            </AppButton>
          </motion.article>
        </motion.div>
      )}
    </section>
  );
}
