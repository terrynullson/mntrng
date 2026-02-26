import type { LucideIcon } from "lucide-react";

type StatusCountBadgeProps = {
  icon: LucideIcon;
  count: number;
  label: string;
  name: string;
  tone?: "neutral" | "warn" | "fail";
};

export function StatusCountBadge({
  icon: Icon,
  count,
  label,
  name,
  tone = "neutral"
}: StatusCountBadgeProps) {
  const toneClass =
    tone === "fail"
      ? "status-count-badge tone-fail"
      : tone === "warn"
        ? "status-count-badge tone-warn"
        : "status-count-badge";

  return (
    <span className={toneClass} title={label} aria-label={label}>
      <Icon size={14} strokeWidth={1.75} aria-hidden />
      <span className="status-count-name">{name}</span>
      <span>{count}</span>
    </span>
  );
}
