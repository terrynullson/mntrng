import type { AuthUser } from "@/lib/api/types";

export function resolveCompanyScope(
  user: AuthUser | null,
  activeCompanyId: number | null
): number | null {
  if (!user) {
    return null;
  }

  if (user.company_id && user.company_id > 0) {
    return user.company_id;
  }

  return activeCompanyId;
}
