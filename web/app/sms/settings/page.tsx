import { SlidersHorizontal } from "lucide-react";

export default function SmsSettingsPage() {
  return (
    <section className="panel">
      <h2 className="page-title">SMS Settings</h2>
      <p className="page-note">Заглушка для будущих настроек SMS-модуля.</p>
      <p className="status-row">
        <SlidersHorizontal size={16} aria-hidden /> В разработке
      </p>
    </section>
  );
}
