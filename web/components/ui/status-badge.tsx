import type { CheckStatus } from "@/lib/api/types";

type StatusBadgeProps = {
  status: CheckStatus;
};

function normalizeStatus(status: string): CheckStatus {
  if (status === "WARN" || status === "FAIL") {
    return status;
  }

  return "OK";
}

export function StatusBadge({ status }: StatusBadgeProps) {
  const normalized = normalizeStatus(status);
  const className =
    normalized === "OK"
      ? "status-badge status-ok"
      : normalized === "WARN"
        ? "status-badge status-warn"
        : "status-badge status-fail";

  return <span className={className}>{normalized}</span>;
}
