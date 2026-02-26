import type { LucideIcon } from "lucide-react";

type StatusCountBadgeProps = {
  icon: LucideIcon;
  count: number;
  label: string;
  name: string;
  tone?: "neutral" | "warn" | "fail";
  /** Hide name text, show only icon + number (tooltip still uses label) */
  iconNumberOnly?: boolean;
};

export function StatusCountBadge({
  icon: Icon,
  count,
  label,
  name,
  tone = "neutral",
  iconNumberOnly = false
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
      {!iconNumberOnly ? <span className="status-count-name">{name}</span> : null}
      <span>{count}</span>
    </span>
  );
}
