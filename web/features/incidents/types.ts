import type { Incident } from "@/lib/api/types";

export type StatusFilter = "" | "open" | "resolved";
export type SeverityFilter = "" | "warn" | "fail";

export function diagLabel(code: Incident["diag_code"]): string {
  switch (code) {
    case "BLACKFRAME":
      return "Чёрный экран";
    case "FREEZE":
      return "Фриз";
    case "CAPTURE_FAIL":
      return "Не удалось получить кадр";
    case "UNKNOWN":
      return "Неизвестно";
    default:
      return "—";
  }
}
