import type { LucideIcon } from "lucide-react";

type StatusCountBadgeProps = {
  icon: LucideIcon;
  count: number;
  label: string;
  tone?: "neutral" | "warn" | "fail";
};

export function StatusCountBadge({
  icon: Icon,
  count,
  label,
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
      <Icon size={14} aria-hidden />
      <span>{count}</span>
    </span>
  );
}
