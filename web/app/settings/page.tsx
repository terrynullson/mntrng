"use client";

import { StatePanel } from "@/components/ui/state-panel";

export default function SettingsPage() {
  return (
    <section className="panel">
      <header className="page-header compact">
        <h2 className="page-title">Settings</h2>
        <p className="page-note">
          Baseline account settings screen for secure admin shell.
        </p>
      </header>

      <StatePanel>
        Telegram connect/reconnect flow is scheduled for the next UI phase.
      </StatePanel>
    </section>
  );
}
