"use client";

import { Sparkles } from "lucide-react";

export default function AiPage() {
  return (
    <section className="panel">
      <h2 className="page-title">AI модуль</h2>
      <p className="page-note">Плейсхолдер для будущих AI-инструментов и диагностики.</p>
      <p className="status-row">
        <Sparkles size={16} aria-hidden /> В разработке
      </p>
    </section>
  );
}
