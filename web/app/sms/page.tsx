"use client";

import { MessageSquare } from "lucide-react";

export default function SmsPage() {
  return (
    <section className="panel">
      <h2 className="page-title">SMS модуль</h2>
      <p className="page-note">Плейсхолдер для будущей интеграции уведомлений.</p>
      <p className="status-row">
        <MessageSquare size={16} aria-hidden /> В разработке
      </p>
    </section>
  );
}
