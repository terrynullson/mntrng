import { BrainCircuit } from "lucide-react";

export default function AiModelsPage() {
  return (
    <section className="panel">
      <h2 className="page-title">AI Models</h2>
      <p className="page-note">Заглушка под реестр моделей и их конфигурации.</p>
      <p className="status-row">
        <BrainCircuit size={16} aria-hidden /> В разработке
      </p>
    </section>
  );
}
